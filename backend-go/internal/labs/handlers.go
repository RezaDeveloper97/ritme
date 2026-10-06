package labs

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/ai/access"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/labs/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Error codes of the non-validation failures.
const (
	ErrorCodeNotFound       = "lab_not_found"
	ErrorCodeMarkerNotFound = "marker_not_found"
	ErrorCodeFileNotFound   = "file_not_found"
	ErrorCodeBusy           = "lab_busy"
	ErrorCodeNotReviewable  = "lab_not_reviewable"
	ErrorCodeNoMarkers      = "no_markers"
	ErrorCodeTooManyMarkers = "too_many_markers"
	ErrorCodeStorage        = "lab_storage_unavailable"
	ErrorCodeDailyLimit     = "lab_daily_limit"
)

// Handlers are the /api/v1/labs actions. Mount them behind the locale middleware and auth RequireUser; POST /labs
// also behind the AI gate (access.Guard.Chain(ai.FeatureLabAnalysis): throttles, plus.lab_ai, consent, caps,
// reserve-first quota).
type Handlers struct {
	svc   *Service
	langs *i18n.Registry
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, langs *i18n.Registry, base clock.Clock) *Handlers {
	return &Handlers{svc: svc, langs: langs, clock: base}
}

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func (h *Handlers) loc(c fiber.Ctx) loc {
	def := ""
	if h.langs != nil {
		def = h.langs.DefaultCode(c.Context())
	}
	return loc{Locale: i18n.Locale(c), Default: def}
}

// fail maps the domain errors (a foreign or missing lab is a uniform 404).
func fail(err error, locale string) error {
	f := func(status int, key, code string) error {
		return httpx.Fail(status, T("messages."+key, locale, nil), "error_code", code)
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return f(fiber.StatusNotFound, "not_found", ErrorCodeNotFound)
	case errors.Is(err, ErrMarkerNotFound):
		return f(fiber.StatusNotFound, "marker_not_found", ErrorCodeMarkerNotFound)
	case errors.Is(err, ErrFileNotFound):
		return f(fiber.StatusNotFound, "file_not_found", ErrorCodeFileNotFound)
	case errors.Is(err, ErrBusy):
		return f(fiber.StatusConflict, "busy", ErrorCodeBusy)
	case errors.Is(err, ErrNotReviewable):
		return f(fiber.StatusConflict, "not_reviewable", ErrorCodeNotReviewable)
	case errors.Is(err, ErrNoMarkers):
		return f(fiber.StatusUnprocessableEntity, "no_markers", ErrorCodeNoMarkers)
	case errors.Is(err, ErrTooManyMarkers):
		return f(fiber.StatusUnprocessableEntity, "too_many_markers", ErrorCodeTooManyMarkers)
	case errors.Is(err, ErrStorage):
		return f(fiber.StatusServiceUnavailable, "storage_unavailable", ErrorCodeStorage)
	case errors.Is(err, ErrDailyLimit):
		return f(fiber.StatusTooManyRequests, "daily_limit", ErrorCodeDailyLimit)
	}
	return err
}

func idParam(c fiber.Ctx, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	return id, err == nil && id > 0
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// lab resolves :id to the user's lab (a malformed id is the same 404).
func (h *Handlers) lab(c fiber.Ctx) (uint64, store.LabReport, error) {
	userID, err := user(c)
	if err != nil {
		return 0, store.LabReport{}, err
	}
	id, ok := idParam(c, "id")
	if !ok {
		return userID, store.LabReport{}, fail(ErrNotFound, i18n.Locale(c))
	}
	lab, err := h.svc.Get(c, userID, id)
	if err != nil {
		return userID, lab, fail(err, i18n.Locale(c))
	}
	return userID, lab, nil
}

// detail renders a lab with its markers, files and interpretation.
func (h *Handlers) detail(c fiber.Ctx, userID uint64, lab store.LabReport) (*jsonx.OrderedMap, error) {
	cat, err := h.svc.Catalog(c)
	if err != nil {
		return nil, err
	}
	rows, err := h.svc.Markers(c, userID, lab.ID)
	if err != nil {
		return nil, err
	}
	fs, err := h.svc.Files(c, userID, lab.ID)
	if err != nil {
		return nil, err
	}
	return h.loc(c).labJSON(lab, evaluateAll(rows, cat), fs), nil
}

func (h *Handlers) reload(c fiber.Ctx, userID, id uint64, status int, msg string) error {
	lab, err := h.svc.Get(c, userID, id)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	body, err := h.detail(c, userID, lab)
	if err != nil {
		return err
	}
	return httpx.JSON(c, status, httpx.Envelope(body, msg))
}

// Index is GET /labs: the user's labs, newest first (nbl_Lab_Intro «تحلیل‌های قبلی»), with marker / attention counts.
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	labs, err := h.svc.List(c, userID)
	if err != nil {
		return err
	}
	cat, err := h.svc.Catalog(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.UserMarkers(c, userID)
	if err != nil {
		return err
	}
	byLab := map[uint64][]Evaluated{}
	for _, r := range rows {
		byLab[r.LabID] = append(byLab[r.LabID], evaluate(rowMarker(r), cat))
	}
	l := h.loc(c)
	list := make([]*jsonx.OrderedMap, 0, len(labs))
	for _, lab := range labs {
		list = append(list, l.listItemJSON(lab, countStates(byLab[lab.ID])))
	}
	return httpx.OK(c, jsonx.Obj("labs", list, "limits", jsonx.Obj("max_files", MaxFiles, "max_image_kb", MaxImageBytes/1024,
		"max_pdf_kb", MaxPDFBytes/1024, "categories", Categories)))
}

// Upload is POST /labs (multipart: files ≤ 5 photos / PDFs, category, title?, taken_on?, fasting?). The Plus use is
// reserved by the AI gate before this runs and refunded by the gate when this fails, or by the job when the
// extraction fails for good. 202 with the lab (status queued; in sync mode already needs_review / failed).
func (h *Handlers) Upload(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	raw := c.Request().Body()
	defer clear(raw) // the sheet never outlives the request in memory
	if h.svc.FilesDisabled() {
		return fail(ErrStorage, locale)
	}
	up, err := readUpload(c, raw, locale)
	if err != nil {
		return err
	}
	defer up.wipe()
	now := h.now(c)
	meta, err := validateMeta(up.fields, locale, now)
	if err != nil {
		return err
	}
	var quotaAt *time.Time
	if access.ReservationFrom(c.Context()) != nil {
		quotaAt = &now
	}
	id, err := h.svc.CreateUpload(c.Context(), userID, meta, up.files, quotaAt, locale, now)
	if err != nil {
		return fail(err, locale)
	}
	return h.reload(c, userID, id, fiber.StatusAccepted, T("messages.uploaded", locale, nil))
}

// StoreManual is POST /labs/manual: a typed-in lab (no file, no AI, not Plus): verified, rules interpretation. 201.
func (h *Handlers) StoreManual(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	l, now := h.loc(c), h.now(c)
	meta, markers, err := validateManual(validation.Input(c), l.Locale, now)
	if err != nil {
		return err
	}
	id, err := h.svc.CreateManual(c.Context(), userID, meta, markers, l.Locale, l.Default, now)
	if err != nil {
		return fail(err, l.Locale)
	}
	return h.reload(c, userID, id, fiber.StatusCreated, T("messages.saved", l.Locale, nil))
}

// Show is GET /labs/{id}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	body, err := h.detail(c, userID, lab)
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

// Status is GET /labs/{id}/status: the processing screen's poll.
func (h *Handlers) Status(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.Markers(c, userID, lab.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, statusJSON(lab, len(rows), h.loc(c)))
}

// Update is PUT /labs/{id} {category, title?, taken_on?, fasting?}.
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	meta, err := validateMeta(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateMeta(c.Context(), userID, lab.ID, meta, now); err != nil {
		return fail(err, locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusOK, T("messages.saved", locale, nil))
}

// Destroy is DELETE /labs/{id}: the lab, its markers and its files.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.Delete(c.Context(), userID, lab.ID); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("id", lab.ID), T("messages.deleted", locale, nil))
}

// Verify is POST /labs/{id}/verify: the values are confirmed; the interpretation is prepared (202).
func (h *Handlers) Verify(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.Verify(c.Context(), userID, lab.ID, locale, h.now(c)); err != nil {
		return fail(err, locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusAccepted, T("messages.verified", locale, nil))
}

// Feedback is POST /labs/{id}/feedback {helpful, note?} («این تحلیل مفید بود؟ 👍 👎»).
func (h *Handlers) Feedback(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	helpful, note, err := validateFeedback(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.Feedback(c.Context(), userID, lab.ID, helpful, note, now); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("id", lab.ID, "helpful", helpful), T("messages.feedback_saved", locale, nil))
}

// StoreMarker is POST /labs/{id}/markers (nbl_Lab_Verify «افزودن شاخص جاافتاده»). 201 with the lab.
func (h *Handlers) StoreMarker(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	l, now := h.loc(c), h.now(c)
	in, err := validateMarker(validation.Input(c), l.Locale, now)
	if err != nil {
		return err
	}
	if _, err := h.svc.AddMarker(c.Context(), userID, lab.ID, in, l, now); err != nil {
		return fail(err, l.Locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusCreated, T("messages.saved", l.Locale, nil))
}

func (h *Handlers) markerID(c fiber.Ctx) (uint64, error) {
	id, ok := idParam(c, "mid")
	if !ok {
		return 0, fail(ErrMarkerNotFound, i18n.Locale(c))
	}
	return id, nil
}

// UpdateMarker is PUT /labs/{id}/markers/{mid}.
func (h *Handlers) UpdateMarker(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	mid, err := h.markerID(c)
	if err != nil {
		return err
	}
	l, now := h.loc(c), h.now(c)
	in, err := validateMarker(validation.Input(c), l.Locale, now)
	if err != nil {
		return err
	}
	if err := h.svc.UpdateMarker(c.Context(), userID, lab.ID, mid, in, l, now); err != nil {
		return fail(err, l.Locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusOK, T("messages.saved", l.Locale, nil))
}

// DestroyMarker is DELETE /labs/{id}/markers/{mid}.
func (h *Handlers) DestroyMarker(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	mid, err := h.markerID(c)
	if err != nil {
		return err
	}
	l := h.loc(c)
	if err := h.svc.DeleteMarker(c.Context(), userID, lab.ID, mid, l, h.now(c)); err != nil {
		return fail(err, l.Locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusOK, T("messages.deleted", l.Locale, nil))
}

// ShowMarker is GET /labs/{id}/markers/{mid} (nbl_Lab_Marker): the value against its range, what the marker is,
// related factors, when to see a doctor, personal context notes and the trend across the user's labs.
func (h *Handlers) ShowMarker(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	mid, err := h.markerID(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.Markers(c, userID, lab.ID)
	if err != nil {
		return err
	}
	var row *store.LabMarker
	for i := range rows {
		if rows[i].ID == mid {
			row = &rows[i]
		}
	}
	l := h.loc(c)
	if row == nil {
		return fail(ErrMarkerNotFound, l.Locale)
	}
	trends, cat, err := h.svc.Trends(c, userID)
	if err != nil {
		return err
	}
	e := evaluate(*row, cat)
	uc, err := h.svc.userContext(c, userID, civildate.InTehran(h.now(c)))
	if err != nil {
		return err
	}
	key := seriesKey(row.Code.String, row.Name)
	series := Series{Key: key, Latest: e, Points: []Point{}}
	for _, s := range trends.Series {
		if s.Key == key {
			series.Points = s.Points
		}
	}
	about, factors, seeDoctor := l.aboutJSON(e)
	var flag any
	if f, ok := redFlag(e, l.name(e)); ok {
		flag = l.redFlagsJSON([]RedFlag{f})[0]
	}
	disclaimer := T("disclaimer_rules", l.Locale, nil)
	if lab.Source == SourceUpload {
		disclaimer = T("disclaimer", l.Locale, nil)
	}
	return httpx.OK(c, jsonx.Obj(
		"lab", jsonx.Obj("id", lab.ID, "display_title", l.title(lab), "date", labDate(lab.TakenOn, lab.CreatedAt).String(), "status", lab.Status),
		"marker", l.markerJSON(e),
		"about", about,
		"factors", factors,
		"see_doctor", seeDoctor,
		"context_notes", l.contextNotes(e, uc),
		"red_flag", flag,
		"trend", l.trendJSON(series),
		"disclaimer", disclaimer,
	))
}

// Trends is GET /labs/trends (nbl_An_Labs): every marker of the user's verified labs with its latest value, state
// against the sheet's range and its series (a direction from MinTrendPoints labs).
func (h *Handlers) Trends(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	t, _, err := h.svc.Trends(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, h.loc(c).trendsJSON(t))
}

// Catalog is GET /labs/markers: the marker catalog (names, units, typical ranges) for the «add marker» picker.
func (h *Handlers) Catalog(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	cat, err := h.svc.Catalog(c)
	if err != nil {
		return err
	}
	l := h.loc(c)
	list := make([]*jsonx.OrderedMap, 0, len(cat.Items()))
	for _, m := range cat.Items() {
		list = append(list, l.catalogJSON(m))
	}
	return httpx.OK(c, jsonx.Obj("markers", list))
}

func (h *Handlers) fileID(c fiber.Ctx) (uint64, error) {
	id, ok := idParam(c, "fid")
	if !ok {
		return 0, fail(ErrFileNotFound, i18n.Locale(c))
	}
	return id, nil
}

// File is GET /labs/{id}/files/{fid}: the decrypted page, to its owner only, as an attachment that is never
// cached, sniffed or rendered as a page.
func (h *Handlers) File(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	fid, err := h.fileID(c)
	if err != nil {
		return err
	}
	f, data, err := h.svc.File(c.Context(), userID, lab.ID, fid)
	if err != nil {
		return fail(err, i18n.Locale(c))
	}
	defer clear(data)
	ext := "webp"
	if f.Mime == "application/pdf" {
		ext = "pdf"
	}
	c.Set(fiber.HeaderContentType, f.Mime)
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="lab-`+strconv.FormatUint(lab.ID, 10)+"-"+strconv.Itoa(int(f.Page))+"."+ext+`"`)
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	c.Set(fiber.HeaderContentSecurityPolicy, "sandbox; default-src 'none'")
	return c.Status(fiber.StatusOK).Send(append([]byte(nil), data...))
}

// DestroyFile is DELETE /labs/{id}/files/{fid}: one page (the values read from it stay).
func (h *Handlers) DestroyFile(c fiber.Ctx) error {
	userID, lab, err := h.lab(c)
	if err != nil {
		return err
	}
	fid, err := h.fileID(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	if err := h.svc.DeleteFile(c.Context(), userID, lab.ID, fid); err != nil {
		return fail(err, locale)
	}
	return h.reload(c, userID, lab.ID, fiber.StatusOK, T("messages.deleted", locale, nil))
}
