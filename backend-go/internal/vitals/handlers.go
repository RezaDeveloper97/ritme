package vitals

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
)

// ErrorCodeNotFound is the 404 code of an unknown or foreign reading.
const ErrorCodeNotFound = "vital_not_found"

// HubRecent is the number of «ثبت‌های اخیر» on the hub; HubRecentDays their window.
const (
	HubRecent     = 5
	HubRecentDays = 30
)

// Handlers are the /api/v1/vitals actions. Mount them behind the locale middleware and auth RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests and the contract suite pin it per request).
func NewHandlers(svc *Service, base clock.Clock) *Handlers { return &Handlers{svc: svc, clock: base} }

// service is the service with the request clock (the pregnancy range ends on the request's today).
func (h *Handlers) service(c fiber.Ctx) *Service {
	return h.svc.WithClock(clock.FromContext(c, h.clock).Now)
}

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

func notFound(locale string) error {
	return httpx.Fail(fiber.StatusNotFound, T("messages.not_found", locale), "error_code", ErrorCodeNotFound)
}

func fail(err error, locale string) error {
	if errors.Is(err, ErrNotFound) {
		return notFound(locale)
	}
	return err
}

func readingID(c fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

func notificationsJSON(enabled bool) *jsonx.OrderedMap {
	return jsonx.Obj("category", string(ReminderCategory), "enabled", enabled)
}

// planSlotsJSON is the slots a plan item may use per type.
func planSlotsJSON() *jsonx.OrderedMap {
	m := jsonx.Obj()
	for _, t := range Types {
		m.Set(t, PlanSlots[t])
	}
	return m
}

// Hub is GET /vitals (nbl_Vitals_Hub): the latest reading of each type, this week's plan progress and the recent
// readings (timed + log sheet, newest first).
func (h *Handlers) Hub(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now := h.now(c)
	today := civildate.InTehran(now)
	latest, err := h.service(c).Latest(c, userID, today)
	if err != nil {
		return err
	}
	latestJSON := jsonx.Obj()
	for _, t := range Types {
		if r := latest[t]; r != nil {
			latestJSON.Set(t, ReadingJSON(*r))
		} else {
			latestJSON.Set(t, nil)
		}
	}
	recent, err := h.service(c).Merged(c, userID, "", today.AddDays(1-HubRecentDays), today)
	if err != nil {
		return err
	}
	if len(recent) > HubRecent {
		recent = recent[:HubRecent]
	}
	plan, err := h.service(c).Plan(c, userID)
	if err != nil {
		return err
	}
	week, err := h.service(c).Timed(c, userID, "", today.StartOfWeek(), today.EndOfWeek())
	if err != nil {
		return err
	}
	prefs, err := h.service(c).Notifications(c, userID)
	if err != nil {
		return err
	}
	return httpx.OK(c, jsonx.Obj(
		"date", today.String(),
		"latest", latestJSON,
		"plan", WeekJSON(plan, week, today),
		"recent", ReadingsJSON(recent),
		"notifications", notificationsJSON(prefs.Enabled(ReminderCategory)),
	))
}

// Thresholds is GET /vitals/thresholds: the classification bands with their sources.
func (h *Handlers) Thresholds(c fiber.Ctx) error {
	if _, err := user(c); err != nil {
		return err
	}
	return httpx.OK(c, ThresholdsJSON(ThresholdsOn(civildate.InTehran(h.now(c)))))
}

// Readings is GET /vitals/readings?type&from&to: every reading of the range (timed + log sheet), newest first.
func (h *Handlers) Readings(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	q, err := ValidateList(validation.Query(c), locale, now)
	if err != nil {
		return err
	}
	rs, err := h.service(c).Merged(c, userID, q.Type, q.From, q.To)
	if err != nil {
		return err
	}
	var typ any
	if q.Type != "" {
		typ = q.Type
	}
	return httpx.OK(c, jsonx.Obj("type", typ, "from", q.From.String(), "to", q.To.String(), "count", len(rs),
		"items", ReadingsJSON(rs)))
}

// saved is the body of a stored / updated reading: the reading and the urgent modal (null under the thresholds). A
// pregnant user's reading goes to the pregnancy alert engine first (T-M9-05): its urgent alert is the modal, so one
// reading never raises two alerts; the vitals copy covers what the engine does not (glucose < 54, BP > 180/120).
func (h *Handlers) saved(c fiber.Ctx, userID uint64, r Reading, now time.Time) (*jsonx.OrderedMap, error) {
	svc := h.service(c)
	locale, def := i18n.Locale(c), i18n.LanguagesOf(c).DefaultCode()
	pregnancy, err := svc.PregnancyAlerts(c, userID, r, now, v2.Lang{Locale: locale, Default: def})
	if err != nil {
		return nil, err
	}
	var alert any
	if len(pregnancy) > 0 || AlertRule(r) != "" {
		cp, err := svc.AlertCopy(c)
		if err != nil {
			return nil, err
		}
		for _, a := range pregnancy {
			if alert = cp.PregnancyModalJSON(a, locale, def); alert != nil {
				break
			}
		}
		if alert == nil && AlertRule(r) != "" {
			alert = cp.AlertJSON(r, locale, def)
		}
	}
	return jsonx.Obj("reading", ReadingJSON(r), "alert", alert), nil
}

// Store is POST /vitals/readings (201 {reading, alert}).
func (h *Handlers) Store(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	in, err := ValidateReading(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	r, err := h.service(c).Create(c, userID, in, now)
	if err != nil {
		return err
	}
	body, err := h.saved(c, userID, r, now)
	if err != nil {
		return err
	}
	return httpx.Created(c, body, T("messages.saved", locale))
}

// Show is GET /vitals/readings/{id}.
func (h *Handlers) Show(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := readingID(c)
	if !ok {
		return notFound(locale)
	}
	r, err := h.service(c).Get(c, userID, id)
	if err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, ReadingJSON(r))
}

// Update is PUT /vitals/readings/{id}: a full replace ({reading, alert}).
func (h *Handlers) Update(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	id, ok := readingID(c)
	if !ok {
		return notFound(locale)
	}
	if _, err := h.service(c).Get(c, userID, id); err != nil {
		return fail(err, locale)
	}
	in, err := ValidateReading(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	r, err := h.service(c).Update(c, userID, id, in, now)
	if err != nil {
		return fail(err, locale)
	}
	body, err := h.saved(c, userID, r, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, T("messages.updated", locale))
}

// Destroy is DELETE /vitals/readings/{id}.
func (h *Handlers) Destroy(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	locale := i18n.Locale(c)
	id, ok := readingID(c)
	if !ok {
		return notFound(locale)
	}
	if err := h.service(c).Delete(c, userID, id); err != nil {
		return fail(err, locale)
	}
	return httpx.OK(c, nil, T("messages.deleted", locale))
}

// Report is GET /vitals/reports/{type}?range=7d|14d|30d|90d[&filter=] (nbl_Vitals_BPReport / GlucoseReport).
func (h *Handlers) Report(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	typ := c.Params("type")
	if _, known := DefaultRange[typ]; !known {
		return httpx.NotFound()
	}
	q, err := ValidateReport(typ, validation.Query(c), locale, now)
	if err != nil {
		return err
	}
	rng := NewRange(q.Range, civildate.InTehran(now))
	rs, err := h.service(c).Merged(c, userID, typ, rng.From, rng.To)
	if err != nil {
		return err
	}
	return httpx.OK(c, Report(typ, rs, rng, q.Filter))
}

func (h *Handlers) planJSON(c fiber.Ctx, userID uint64, now time.Time) (*jsonx.OrderedMap, error) {
	today := civildate.InTehran(now)
	plan, err := h.service(c).Plan(c, userID)
	if err != nil {
		return nil, err
	}
	week, err := h.service(c).Timed(c, userID, "", today.StartOfWeek(), today.EndOfWeek())
	if err != nil {
		return nil, err
	}
	prefs, err := h.service(c).Notifications(c, userID)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(plan))
	for _, it := range plan {
		items = append(items, PlanItemJSON(it))
	}
	return jsonx.Obj(
		"items", items,
		"week", WeekJSON(plan, week, today),
		"slots", planSlotsJSON(),
		"notifications", notificationsJSON(prefs.Enabled(ReminderCategory)),
	), nil
}

// Plan is GET /vitals/plan.
func (h *Handlers) Plan(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	body, err := h.planJSON(c, userID, h.now(c))
	if err != nil {
		return err
	}
	return httpx.OK(c, body)
}

// SavePlan is PUT /vitals/plan {items}: replaces the plan.
func (h *Handlers) SavePlan(c fiber.Ctx) error {
	userID, err := user(c)
	if err != nil {
		return err
	}
	now, locale := h.now(c), i18n.Locale(c)
	items, err := ValidatePlan(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if err := h.service(c).SavePlan(c, userID, items, now); err != nil {
		return err
	}
	body, err := h.planJSON(c, userID, now)
	if err != nil {
		return err
	}
	return httpx.OK(c, body, T("messages.plan_saved", locale))
}
