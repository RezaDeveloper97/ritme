package join

import (
	"embed"
	"errors"
	"io/fs"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/files"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// HTTP (Go only; OpenAPI tag CityServiceApplications), auth:api, localized, no-store, writes throttled per user:
//
//	GET    /api/v1/city-services/applications                       the caller's applications (summaries)
//	POST   /api/v1/city-services/applications                       open a draft (201) or return the open one (200)
//	GET    /api/v1/city-services/applications/{id}                  the full application
//	DELETE /api/v1/city-services/applications/{id}                  delete a draft (+ its files)
//	GET    /api/v1/city-services/applications/{id}/status           status + JoinDone timeline
//	PUT    /api/v1/city-services/applications/{id}/steps/{step}     auto-save step 1..4
//	POST   /api/v1/city-services/applications/{id}/photos           multipart `file` → private photo (public on approval)
//	DELETE /api/v1/city-services/applications/{id}/photos/{fileId}
//	PUT    /api/v1/city-services/applications/{id}/documents/{slot} multipart `file` → private document
//	DELETE /api/v1/city-services/applications/{id}/documents/{slot}
//	POST   /api/v1/city-services/applications/{id}/submit           submit (also rate-limited per user)

// Error codes.
const (
	CodeNotFound = "application_not_found"
	CodeLocked   = "application_locked"
	CodeTooMany  = "too_many_applications"
	CodeMissing  = "application_incomplete"
)

//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	t, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return t
})

// T is the join line for key ("messages.saved") in locale.
func T(key, locale string, params map[string]string) string {
	return translator().Trans("join."+key, params, locale)
}

func attributes(locale string) []string {
	v, ok := translator().Get("join.attributes", locale)
	m, isMap := v.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if s, ok := m.Get(k); ok {
			if str, ok := s.(string); ok {
				kv = append(kv, k, str)
			}
		}
	}
	return kv
}

// Handlers serve the applicant routes.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers (clock nil = real).
func NewHandlers(svc *Service, base clock.Clock) *Handlers {
	if base == nil {
		base = clock.Real{}
	}
	return &Handlers{svc: svc, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return id, nil
}

func fail(status int, key, code, locale string) error {
	return httpx.Fail(status, T("messages."+key, locale, nil), "error_code", code)
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale, nil), "errors", errs)
}

// mapErr turns a service error into its response; files errors are answered like POST /files.
func mapErr(err error, p files.Purpose, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
	case errors.Is(err, ErrLocked):
		return fail(fiber.StatusConflict, "locked", CodeLocked, locale)
	case errors.Is(err, ErrTooMany):
		return fail(fiber.StatusConflict, "too_many", CodeTooMany, locale)
	case errors.Is(err, ErrPhotoLimit):
		return failValidation(locale, jsonx.Obj("file", []string{T("validation.max_photos", locale, map[string]string{"max": strconv.Itoa(MaxPhotos)})}))
	case errors.Is(err, files.ErrDisabled), errors.Is(err, files.ErrType), errors.Is(err, files.ErrTooLarge),
		errors.Is(err, files.ErrQuota), errors.Is(err, files.ErrPurpose), errors.Is(err, files.ErrBusy):
		return files.PutFailure(err, p, locale)
	}
	return err
}

func idParam(c fiber.Ctx, name, locale string) (uint64, error) {
	s := c.Params(name)
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != s {
		return 0, fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
	}
	return id, nil
}

// ---------------------------------------------------------------------------------------------------------------
// JSON

func isoOrNil(t time.Time, ok bool) any {
	if !ok {
		return nil
	}
	return jsonx.ISO8601(t.In(civildate.Tehran))
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func coord(s string) any {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return f
}

func list[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}

func (h *Handlers) fileJSON(f files.File, now time.Time) (*jsonx.OrderedMap, error) {
	link, err := h.svc.Link(f, now)
	if err != nil {
		return nil, err
	}
	var exp any
	if link.ExpiresAt != nil {
		exp = jsonx.ISO8601(link.ExpiresAt.In(civildate.Tehran))
	}
	return jsonx.Obj("id", f.ID, "mime", f.MIME, "size_bytes", f.Size, "url", link.URL, "url_expires_at", exp), nil
}

func summaryJSON(a Application) *jsonx.OrderedMap {
	r := a.Row
	return jsonx.Obj("id", r.ID, "status", r.Status, "step", r.Step, "tracking_code", strOrNil(r.TrackingCode.String),
		"business_name", strOrNil(a.Draft.BusinessName), "editable", Editable(r.Status),
		"submitted_at", isoOrNil(r.SubmittedAt.Time, r.SubmittedAt.Valid),
		"updated_at", isoOrNil(r.UpdatedAt.Time, r.UpdatedAt.Valid))
}

// applicationJSON is the owner's full view (live files with fresh links; no account id, no review internals).
func (h *Handlers) applicationJSON(c fiber.Ctx, userID uint64, a Application, now time.Time) (*jsonx.OrderedMap, error) {
	photos, docs, err := h.svc.LiveFiles(c, userID, a.Draft)
	if err != nil {
		return nil, err
	}
	pj := make([]*jsonx.OrderedMap, 0, len(photos))
	for _, f := range photos {
		o, err := h.fileJSON(f, now)
		if err != nil {
			return nil, err
		}
		pj = append(pj, o)
	}
	dj := jsonx.NewObject()
	for _, slot := range Slots {
		var v any
		if f, ok := docs[slot]; ok {
			o, err := h.fileJSON(f, now)
			if err != nil {
				return nil, err
			}
			v = o
		}
		dj.Set(slot, v)
	}
	r, d := a.Row, a.Draft
	var city any
	if d.CityID > 0 {
		city = d.CityID
	}
	return jsonx.Obj(
		"id", r.ID,
		"status", r.Status,
		"step", r.Step,
		"tracking_code", strOrNil(r.TrackingCode.String),
		"editable", Editable(r.Status),
		"can_add_photos", PhotosOpen(r.Status),
		"business_name", strOrNil(d.BusinessName),
		"category_slugs", list(d.CategorySlugs),
		"city_id", city,
		"district", strOrNil(d.District),
		"contact_name", strOrNil(d.ContactName),
		"contact_phone", strOrNil(d.ContactPhone),
		"owner_confirmed", d.OwnerConfirmed,
		"lat", coord(d.Lat),
		"lng", coord(d.Lng),
		"address", strOrNil(d.Address),
		"about", strOrNil(d.About),
		"age_ranges", list(d.AgeRanges),
		"amenities", list(d.Amenities),
		"weekly_hours", list(d.WeeklyHours),
		"photos", pj,
		"services", list(d.Services),
		"booking_mode", strOrNil(d.BookingMode),
		"terms_accepted_at", isoOrNil(r.TermsAcceptedAt.Time, r.TermsAcceptedAt.Valid),
		"documents", dj,
		"requirements", jsonx.Obj("min_photos", MinPhotos, "max_photos", MaxPhotos,
			"health_cert_category", HealthCertCategory),
		"submitted_at", isoOrNil(r.SubmittedAt.Time, r.SubmittedAt.Valid),
		"reviewed_at", isoOrNil(r.ReviewedAt.Time, r.ReviewedAt.Valid),
		"review_note", strOrNil(r.ReviewNote.String),
		"created_at", isoOrNil(r.CreatedAt.Time, r.CreatedAt.Valid),
		"updated_at", isoOrNil(r.UpdatedAt.Time, r.UpdatedAt.Valid),
	), nil
}

// Timeline states of the JoinDone steps.
const (
	stateDone    = "done"
	stateCurrent = "current"
	statePending = "pending"
	stateFailed  = "failed"
)

// timeline is JoinDone's «درخواست ثبت شد → بررسی مدارک → تماس کارشناس → انتشار صفحه» for status.
func timeline(a Application) []*jsonx.OrderedMap {
	r := a.Row
	states := map[string][4]string{
		StatusDraft:     {stateCurrent, statePending, statePending, statePending},
		StatusSubmitted: {stateDone, stateCurrent, statePending, statePending},
		StatusInReview:  {stateDone, stateDone, stateCurrent, statePending},
		StatusNeedsInfo: {stateDone, stateDone, stateCurrent, statePending},
		StatusApproved:  {stateDone, stateDone, stateDone, stateDone},
		StatusRejected:  {stateDone, stateDone, stateDone, stateFailed},
	}[r.Status]
	if states[0] == "" {
		states = [4]string{statePending, statePending, statePending, statePending}
	}
	at := [4]any{isoOrNil(r.SubmittedAt.Time, r.SubmittedAt.Valid), nil, nil, nil}
	if r.Status == StatusApproved || r.Status == StatusRejected {
		at[3] = isoOrNil(r.ReviewedAt.Time, r.ReviewedAt.Valid)
	}
	keys := [4]string{"submitted", "documents_review", "expert_contact", "published"}
	out := make([]*jsonx.OrderedMap, 0, 4)
	for i, k := range keys {
		out = append(out, jsonx.Obj("key", k, "state", states[i], "at", at[i]))
	}
	return out
}

// ---------------------------------------------------------------------------------------------------------------
// Routes

// Index is GET /applications.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.List(c, userID)
	if err != nil {
		return err
	}
	out := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, a := range rows {
		out = append(out, summaryJSON(a))
	}
	return httpx.OK(c, jsonx.Obj("applications", out, "max_applications", MaxApplications))
}

// Store is POST /applications: the open draft (200) or a new one (201).
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	a, created, err := h.svc.Start(c, userID, now)
	if err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	body, err := h.applicationJSON(c, userID, a, now)
	if err != nil {
		return err
	}
	if created {
		return httpx.Created(c, jsonx.Obj("application", body), T("messages.created", locale, nil))
	}
	return httpx.OK(c, jsonx.Obj("application", body))
}

// Show is GET /applications/{id}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	a, err := h.svc.Get(c, userID, id)
	if err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	body, err := h.applicationJSON(c, userID, a, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("application", body))
}

// Status is GET /applications/{id}/status (JoinDone).
func (h *Handlers) Status(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	a, err := h.svc.Get(c, userID, id)
	if err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	r := a.Row
	return httpx.OK(c, jsonx.Obj("id", r.ID, "status", r.Status, "tracking_code", strOrNil(r.TrackingCode.String),
		"business_name", strOrNil(a.Draft.BusinessName),
		"submitted_at", isoOrNil(r.SubmittedAt.Time, r.SubmittedAt.Valid),
		"reviewed_at", isoOrNil(r.ReviewedAt.Time, r.ReviewedAt.Valid),
		"review_note", strOrNil(r.ReviewNote.String), "timeline", timeline(a)))
}

// Destroy is DELETE /applications/{id} (drafts only).
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c, userID, id); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	return httpx.Message(c, T("messages.deleted", locale, nil))
}

// SaveStep is PUT /applications/{id}/steps/{step}: the whole step form (absent = empty), lenient on completeness.
func (h *Handlers) SaveStep(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	step, err := strconv.Atoi(c.Params("step"))
	if err != nil || step < 1 || step > 4 || c.Params("step") != strconv.Itoa(step) {
		return fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
	}
	// Ownership first: a foreign id is 404 whatever the body holds.
	if _, err := h.svc.Get(c, userID, id); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	input, err := h.parseStep(c, step, locale, now)
	if err != nil {
		return err
	}
	a, err := h.svc.SaveStep(c, userID, id, input, now)
	if err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	body, err := h.applicationJSON(c, userID, a, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("application", body), T("messages.saved", locale, nil))
}

// upload reads the multipart `file` of the request (bounded, no temp file); the caller clears it.
func upload(c fiber.Ctx, locale string) ([]byte, error) {
	raw := c.Request().Body()
	defer clear(raw) // the document never outlives the request in memory
	_, data, err := files.ReadUpload(c, raw, locale)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, files.FileRequired(locale)
	}
	return data, nil
}

// AddPhoto is POST /applications/{id}/photos (multipart `file`) → 201 the photo.
func (h *Handlers) AddPhoto(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	if _, err := h.svc.Get(c, userID, id); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	data, err := upload(c, locale)
	if err != nil {
		return err
	}
	defer clear(data)
	p, _ := h.svc.files.Purpose(files.PurposePlacePhotoPending)
	f, err := h.svc.AddPhoto(c, userID, id, data, now)
	if err != nil {
		return mapErr(err, p, locale)
	}
	body, err := h.fileJSON(f, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, jsonx.Obj("photo", body), T("messages.photo_added", locale, nil))
}

// RemovePhoto is DELETE /applications/{id}/photos/{fileId}.
func (h *Handlers) RemovePhoto(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	fileID, err := idParam(c, "fileId", locale)
	if err != nil {
		return err
	}
	if err := h.svc.RemovePhoto(c, userID, id, fileID, h.now(c)); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	return httpx.Message(c, T("messages.photo_removed", locale, nil))
}

func slotParam(c fiber.Ctx, locale string) (string, error) {
	s := c.Params("slot")
	for _, v := range Slots {
		if v == s {
			return s, nil
		}
	}
	return "", fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
}

// PutDocument is PUT /applications/{id}/documents/{slot} (multipart `file`) → 200 the private document.
func (h *Handlers) PutDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	slot, err := slotParam(c, locale)
	if err != nil {
		return err
	}
	if _, err := h.svc.Get(c, userID, id); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	data, err := upload(c, locale)
	if err != nil {
		return err
	}
	defer clear(data)
	p, _ := h.svc.files.Purpose(files.PurposePlaceLicence)
	f, err := h.svc.PutDocument(c, userID, id, slot, data, now)
	if err != nil {
		return mapErr(err, p, locale)
	}
	body, err := h.fileJSON(f, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj("slot", slot, "document", body), T("messages.document_uploaded", locale, nil))
}

// RemoveDocument is DELETE /applications/{id}/documents/{slot}.
func (h *Handlers) RemoveDocument(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	slot, err := slotParam(c, locale)
	if err != nil {
		return err
	}
	if err := h.svc.RemoveDocument(c, userID, id, slot, h.now(c)); err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	return httpx.Message(c, T("messages.document_removed", locale, nil))
}

// Submit is POST /applications/{id}/submit: 422 `application_incomplete` with per-field errors, else 200 with the
// tracking code.
func (h *Handlers) Submit(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	id, err := idParam(c, "id", locale)
	if err != nil {
		return err
	}
	res, err := h.svc.Submit(c, userID, id, now)
	if err != nil {
		return mapErr(err, files.Purpose{}, locale)
	}
	if len(res.Missing) > 0 {
		errs := jsonx.NewObject()
		for _, f := range missingOrder {
			if key, ok := res.Missing[f]; ok {
				errs.Set(f, []string{T("validation."+key, locale, map[string]string{"min": strconv.Itoa(MinPhotos)})})
			}
		}
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.incomplete", locale, nil),
			"error_code", CodeMissing, "errors", errs)
	}
	body, err := h.applicationJSON(c, userID, res.Application, now)
	if err != nil {
		return err
	}
	code := res.Application.Row.TrackingCode.String
	return httpx.OK(c, jsonx.Obj("application", body),
		T("messages.submitted", locale, map[string]string{"code": code}))
}

// missingOrder is the board order of the submit errors.
var missingOrder = []string{"business_name", "category_slugs", "city_id", "contact_name", "contact_phone",
	"owner_confirmed", "location", "address", "age_ranges", "weekly_hours", "photos", "services", "booking_mode",
	"terms", "documents." + SlotLicence, "documents." + SlotHealthCert}

// TooManySubmits is the localized 429 of the submit throttle.
func TooManySubmits(locale string, retryAfter int, headers map[string]string) *httpx.FailError {
	e := httpx.Fail(fiber.StatusTooManyRequests, T("messages.too_many_submits", locale, nil),
		"error_code", "too_many_requests", "retry_after", retryAfter)
	for k, v := range headers {
		e = e.WithHeader(k, v)
	}
	return e
}
