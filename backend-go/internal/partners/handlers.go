package partners

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/partners/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the owner routes under /api/v1/partners (locale, auth RequireUser, writes throttled). Every request
// lookup is scoped by the caller's id: another user's request answers the same 404 as a missing one.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers.
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

func user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

func failValidation(locale string, errs any) error {
	return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.validation_failed", locale), "errors", errs)
}

func fail(status int, msgKey, code, locale string) error {
	return httpx.Fail(status, T("messages."+msgKey, locale), "error_code", code)
}

// mapErr turns a service error into its response.
func mapErr(err error, locale string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return fail(fiber.StatusNotFound, "not_found", CodeNotFound, locale)
	case errors.Is(err, ErrServiceUnavail):
		return httpx.Fail(fiber.StatusUnprocessableEntity, T("messages.service_unavailable", locale), "error_code", CodeServiceUnavail,
			"errors", jsonx.Obj("service_id", []string{T("messages.service_unavailable", locale)}))
	case errors.Is(err, ErrTooManyOpen):
		return fail(fiber.StatusConflict, "too_many_open", CodeTooManyOpen, locale)
	case errors.Is(err, ErrPartnerUnavail):
		return fail(fiber.StatusServiceUnavailable, "partner_unavailable", CodePartnerUnavail, locale)
	case errors.Is(err, ErrStorage):
		return fail(fiber.StatusServiceUnavailable, "storage_unavailable", CodeStorage, locale)
	case errors.Is(err, ErrNotCancellable):
		return fail(fiber.StatusConflict, "not_cancellable", CodeNotCancellable, locale)
	case errors.Is(err, ErrNoResult):
		return fail(fiber.StatusConflict, "no_result", CodeNoResult, locale)
	case errors.Is(err, ErrAlreadyImported):
		return fail(fiber.StatusConflict, "already_imported", CodeAlreadyImported, locale)
	case errors.Is(err, ErrImportConsent):
		return fail(fiber.StatusUnprocessableEntity, "import_consent", CodeImportConsent, locale)
	}
	return err
}

func serviceJSON(s store.PartnerLabService, locale, def string) *jsonx.OrderedMap {
	desc := ""
	if s.Description.Valid {
		desc = PickText(s.Description.String, locale, def)
	}
	return jsonx.Obj("id", s.ID, "code", s.Code, "kind", s.Kind, "title", PickText(s.Title, locale, def),
		"description", desc, "partner_name", s.PartnerName)
}

// Services is GET /partners/lab-services: the active catalog. The Today «پیشنهاد» cards render from this alone (an
// empty list = no card); it is never computed from health data.
func (h *Handlers) Services(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	rows, err := h.svc.Services(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, s := range rows {
		items = append(items, serviceJSON(s, locale, def))
	}
	return httpx.OK(c, jsonx.Obj("services", items, "max_open", MaxOpenRequests, "max_days_ahead", MaxDaysAhead,
		"purge_days", int(PurgeAfter/(24*time.Hour))))
}

func (h *Handlers) requestJSON(r store.ListUserRequestsRow, locale, def string, details bool) *jsonx.OrderedMap {
	o := jsonx.Obj("id", r.ID, "reference", r.Reference, "status", r.Status,
		"service", jsonx.Obj("id", r.ServiceID, "kind", r.ServiceKind, "title", PickText(r.ServiceTitle, locale, def),
			"partner_name", r.PartnerName),
		"preferred_date", r.PreferredDate.String(), "preferred_window", r.PreferredWindow,
		"consent_import", r.ConsentImport, "imported", r.ImportedLabID.Valid, "can_cancel", CanCancel(r.Status),
		"can_import", r.Status == StatusResultReady && r.ConsentImport && !r.ImportedLabID.Valid,
		"purged", r.PurgedAt.Valid, "created_at", isoOrNil(r.CreatedAt.Time, r.CreatedAt.Valid))
	if r.ImportedLabID.Valid && r.ImportedLabID.Int64 > 0 {
		o.Set("lab_id", r.ImportedLabID.Int64)
	} else {
		o.Set("lab_id", nil)
	}
	if details {
		if p, ok := h.svc.Details(r.UserID, r.Reference, r.PayloadEnc); ok {
			o.Set("details", jsonx.Obj("full_name", p.FullName, "phone", p.Phone, "city", p.City, "address", p.Address, "note", p.Note))
		} else {
			o.Set("details", nil)
		}
	}
	return o
}

func isoOrNil(t time.Time, ok bool) any {
	if !ok {
		return nil
	}
	return jsonx.ISO8601(t)
}

// Index is GET /partners/lab-requests: the caller's requests, newest first (no personal details in the list).
func (h *Handlers) Index(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.List(c, userID)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	items := make([]*jsonx.OrderedMap, 0, len(rows))
	for _, r := range rows {
		items = append(items, h.requestJSON(r, locale, def, false))
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.OK(c, jsonx.Obj("requests", items))
}

func idParam(c fiber.Ctx, locale string) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, mapErr(ErrNotFound, locale)
	}
	return id, nil
}

// Show is GET /partners/lab-requests/{id}: one request with the details she typed (until they are purged).
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	id, err := idParam(c, locale)
	if err != nil {
		return err
	}
	r, err := h.svc.Get(c, userID, id)
	if err != nil {
		return mapErr(err, locale)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return httpx.OK(c, h.requestJSON(r, locale, def, true))
}

var rePhone = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

func asciiDigits(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == ' ' || r == '-':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func text(m phpval.Map, key string) string {
	v, ok := m.Get(key)
	if !ok || v == nil {
		return ""
	}
	return strings.Join(strings.Fields(phpval.ToString(v)), " ")
}

// parseInput validates the booking form.
func parseInput(body phpval.Map, locale string, now time.Time) (Input, error) {
	data := phpval.NewMap()
	for _, k := range []string{"service_id", "full_name", "phone", "city", "address", "preferred_date", "preferred_window",
		"note", "consent_share", "consent_import"} {
		if v, ok := body.Get(k); ok {
			data.Set(k, v)
		}
	}
	rules := validation.Rules{
		validation.F("service_id", "required", "integer", "min:1"),
		validation.F("full_name", "required", "string", "max:80"),
		validation.F("phone", "required", "string", "max:24"),
		validation.F("city", "required", "string", "max:60"),
		validation.F("address", "required", "string", "max:300"),
		validation.F("preferred_date", "required", "date_format:Y-m-d"),
		validation.F("preferred_window", "required", "in:"+strings.Join(Windows, ",")),
		validation.F("note", "nullable", "string", "max:300"),
		validation.F("consent_share", "required", "accepted"),
		validation.F("consent_import", "nullable", "boolean"),
	}
	v := validation.Make(lang.Default(), locale, data, rules, validation.Now(now), validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return Input{}, failValidation(locale, v.ErrorBag())
	}
	phone := asciiDigits(text(data, "phone"))
	if !rePhone.MatchString(phone) {
		return Input{}, failValidation(locale, jsonx.Obj("phone", []string{T("validation.phone", locale)}))
	}
	day, err := civildate.Parse(text(data, "preferred_date"))
	today := civildate.InTehran(now)
	if err != nil || day.Before(today) || day.After(today.AddDays(MaxDaysAhead)) {
		return Input{}, failValidation(locale, jsonx.Obj("preferred_date", []string{T("validation.date_range", locale)}))
	}
	id, _ := strconv.ParseUint(text(data, "service_id"), 10, 64)
	in := Input{ServiceID: id, PreferredDate: day, Window: text(data, "preferred_window"), ConsentShare: true,
		Payload: Payload{FullName: text(data, "full_name"), Phone: phone, City: text(data, "city"),
			Address: text(data, "address"), Note: text(data, "note")}}
	if x, ok := data.Get("consent_import"); ok {
		in.ConsentImport = phpval.Truthy(x)
	}
	return in, nil
}

// Store is POST /partners/lab-requests → 201: validates the form, seals the personal fields, sends the allow-listed
// payload to the partner. 409 too many open requests, 503 partner / storage unavailable.
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale, def := h.now(c), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	in, err := parseInput(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	r, err := h.svc.Create(c, userID, in, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return httpx.Created(c, h.requestJSON(r, locale, def, true), T("messages.requested", locale))
}

// Cancel is POST /partners/lab-requests/{id}/cancel (requested or accepted only).
func (h *Handlers) Cancel(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale, def := h.now(c), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	id, err := idParam(c, locale)
	if err != nil {
		return err
	}
	if err := h.svc.Cancel(c, userID, id, now); err != nil {
		return mapErr(err, locale)
	}
	r, err := h.svc.Get(c, userID, id)
	if err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, h.requestJSON(r, locale, def, true), T("messages.cancelled", locale))
}

// ImportResult is POST /partners/lab-requests/{id}/import-result {consent: true} → {lab_id}: copies the result into
// the user's labs (consent ticked on the request AND confirmed now).
func (h *Handlers) ImportResult(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale, def := h.now(c), i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	id, err := idParam(c, locale)
	if err != nil {
		return err
	}
	data := phpval.NewMap()
	if v, ok := validation.Input(c).Get("consent"); ok {
		data.Set("consent", v)
	}
	v := validation.Make(lang.Default(), locale, data, validation.Rules{validation.F("consent", "required", "boolean")},
		validation.Attributes(attributes(locale)...))
	if v.Fails() {
		return failValidation(locale, v.ErrorBag())
	}
	x, _ := data.Get("consent")
	labID, err := h.svc.Import(c, userID, id, phpval.Truthy(x), locale, def, now)
	if err != nil {
		return mapErr(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("lab_id", labID), T("messages.imported", locale))
}
