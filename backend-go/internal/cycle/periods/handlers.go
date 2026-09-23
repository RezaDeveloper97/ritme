package periods

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Handlers are the PeriodLogController actions. Mount them behind auth RequireUser and the
// locale middleware.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (the request clock wins).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

func (h *Handlers) now(c fiber.Ctx) time.Time { return clock.FromContext(c, h.clock).Now() }

func userID(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// updatedMessage is __('profile.updated') in the request locale (app()->setLocale($locale)).
func updatedMessage(locale string) string {
	return lang.Default().Trans("profile.updated", nil, locale)
}

// Status is GET /cycle/period/status.
func (h *Handlers) Status(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	active, latest, err := h.svc.Status(c.Context(), uid)
	if err != nil {
		return err
	}
	var start, end any
	if latest != nil {
		start, end = latest.PeriodStartDate, latest.PeriodEndDate
	}
	return httpx.OK(c, jsonx.Obj("active", active, "period_start_date", start, "period_end_date", end))
}

// History is GET /cycle/period/history.
func (h *Handlers) History(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	rows, err := h.svc.History(c.Context(), uid)
	if err != nil {
		return err
	}
	periods := make([]any, len(rows))
	for i, r := range rows {
		periods[i] = jsonx.Obj("id", r.ID, "period_start_date", r.PeriodStartDate,
			"period_end_date", r.PeriodEndDate, "source", r.Source)
	}
	return httpx.OK(c, jsonx.Obj("periods", periods))
}

// Start is POST /cycle/period/start.
func (h *Handlers) Start(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	locale, now := i18n.ResolveLocale(c, ""), h.now(c)
	input := validation.Input(c)
	if err := validate(input, locale, now, validation.Rules{validation.F("date", "nullable|date|before_or_equal:today")}); err != nil {
		return err
	}
	start := civildate.InTehran(now)
	if d, ok, err := filledDate(input, "date", now); err != nil {
		return err
	} else if ok {
		start = d
	}
	res, err := h.svc.Start(c.Context(), uid, start, now)
	if err != nil {
		return refusal(err, locale)
	}
	r := res.Row
	return httpx.OK(c, jsonx.Obj(
		"active", true,
		"period_start_date", r.PeriodStartDate,
		"period_end_date", r.PeriodEndDate,
		"warnings", Warnings(intPtr(r.CycleLength), intPtr(r.BleedingLength), locale),
	), updatedMessage(locale))
}

// End is POST /cycle/period/end.
func (h *Handlers) End(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	locale, now := i18n.ResolveLocale(c, ""), h.now(c)
	input := validation.Input(c)
	if err := validate(input, locale, now, validation.Rules{validation.F("date", "nullable|date|before_or_equal:today")}); err != nil {
		return err
	}
	var end *civildate.Date
	if d, ok, err := filledDate(input, "date", now); err != nil {
		return err
	} else if ok {
		end = &d
	}
	res, err := h.svc.End(c.Context(), uid, end, now)
	if err != nil {
		return refusal(err, locale)
	}
	r := res.Row
	return httpx.OK(c, jsonx.Obj(
		"active", false,
		"period_start_date", r.PeriodStartDate,
		"period_end_date", r.PeriodEndDate,
		"warnings", Warnings(intPtr(r.CycleLength), intPtr(r.BleedingLength), locale),
	), updatedMessage(locale))
}

// rangeRules are the store / update rules (the end may lie in the future).
var rangeRules = validation.Rules{
	validation.F("start_date", "required|date|before_or_equal:today"),
	validation.F("end_date", "nullable|date|after_or_equal:start_date"),
}

// Store is POST /cycle/period.
func (h *Handlers) Store(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	locale, now := i18n.ResolveLocale(c, ""), h.now(c)
	start, end, err := parseRange(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := h.svc.Store(c.Context(), uid, start, end, now)
	if err != nil {
		return refusal(err, locale)
	}
	r := res.Row
	return httpx.OK(c, jsonx.Obj(
		"id", r.ID,
		"active", !r.PeriodEndDate.Valid,
		"period_start_date", r.PeriodStartDate,
		"period_end_date", r.PeriodEndDate,
		"warnings", Warnings(intPtr(r.CycleLength), intPtr(r.BleedingLength), locale),
	), updatedMessage(locale))
}

// Update is PUT /cycle/period/{period} (no `active` key).
func (h *Handlers) Update(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	id, err := periodParam(c)
	if err != nil {
		return err
	}
	locale, now := i18n.ResolveLocale(c, ""), h.now(c)
	start, end, err := parseRange(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	res, err := h.svc.Update(c.Context(), uid, id, start, end, now)
	if err != nil {
		return refusal(err, locale)
	}
	r := res.Row
	return httpx.OK(c, jsonx.Obj(
		"id", r.ID,
		"period_start_date", r.PeriodStartDate,
		"period_end_date", r.PeriodEndDate,
		"warnings", Warnings(intPtr(r.CycleLength), intPtr(r.BleedingLength), locale),
	), updatedMessage(locale))
}

// Destroy is DELETE /cycle/period/{period}.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	id, err := periodParam(c)
	if err != nil {
		return err
	}
	locale, now := i18n.ResolveLocale(c, ""), h.now(c)
	if err := h.svc.Destroy(c.Context(), uid, id, now); err != nil {
		return refusal(err, locale)
	}
	return httpx.OK(c, jsonx.Obj("deleted", true), updatedMessage(locale))
}

// periodParam is the `int $period` route argument; non-numeric → 404 (D-02; Laravel 500s).
func periodParam(c fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(c.Params("period"), 10, 64)
	if err != nil {
		return 0, httpx.NotFound()
	}
	return id, nil
}

// validate is Validator::make(...)->fails() → 422 {success:false, message: first, errors}.
func validate(input phpval.Map, locale string, now time.Time, rules validation.Rules) error {
	v := validation.Make(lang.Default(), locale, input, rules, validation.Now(now))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, v.First(), "errors", v.ErrorBag())
	}
	return nil
}

func parseRange(input phpval.Map, locale string, now time.Time) (civildate.Date, *civildate.Date, error) {
	if err := validate(input, locale, now, rangeRules); err != nil {
		return civildate.Date{}, nil, err
	}
	start, _, err := filledDate(input, "start_date", now)
	if err != nil {
		return civildate.Date{}, nil, err
	}
	end, ok, err := filledDate(input, "end_date", now)
	if err != nil || !ok {
		return start, nil, err
	}
	return start, &end, nil
}

// filledDate is `$request->filled($key) ? Carbon::parse($request->input($key))->startOfDay()`:
// ok=false when the value is missing, null or blank.
func filledDate(input phpval.Map, key string, now time.Time) (civildate.Date, bool, error) {
	v, _ := input.Get(key)
	if v == nil {
		return civildate.Date{}, false, nil
	}
	s := phpval.ToString(v)
	if validation.TrimString(s) == "" {
		return civildate.Date{}, false, nil
	}
	t, err := civildate.ParseLenient(s, now, civildate.Tehran)
	if err != nil {
		// The `date` rule passed but the supported Carbon subset cannot read it (D-09).
		return civildate.Date{}, false, httpx.ServerError()
	}
	return civildate.FromTime(t), true, nil
}

// refusal renders a Refusal with the controller's copy; other errors pass through.
func refusal(err error, locale string) error {
	var r *Refusal
	if !errors.As(err, &r) {
		return err
	}
	switch r.Kind {
	case refusePreviousOpen:
		return httpx.Fail(r.Status, msgPreviousOpen(locale), "code", r.Code,
			"data", jsonx.Obj("open_period_start", r.OpenPeriodStart))
	case refuseStartOverlap:
		return httpx.Fail(r.Status, msgStartOverlap(locale), "code", r.Code)
	case refuseRangeOverlap:
		return httpx.Fail(r.Status, msgRangeOverlap(locale), "code", r.Code)
	case refuseNoOngoing:
		return httpx.Fail(r.Status, msgNoOngoing(locale))
	case refuseEndBeforeStart:
		return httpx.Fail(r.Status, msgEndBeforeStart(locale))
	default:
		return httpx.Fail(r.Status, msgNotFound(locale))
	}
}
