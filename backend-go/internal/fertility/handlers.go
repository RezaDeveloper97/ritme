package fertility

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/validation"
)

// Handlers are the /api/v1/fertility actions. Mount them behind the locale middleware and auth
// RequireUser.
type Handlers struct {
	svc   *Service
	clock clock.Clock
}

// NewHandlers wires the handlers; base is the fallback clock (tests pin it per request).
func NewHandlers(db DB, base clock.Clock) *Handlers {
	return &Handlers{svc: NewService(db), clock: base}
}

func (h *Handlers) user(c fiber.Ctx) (uint64, error) {
	id, ok := auth.CurrentUserID(c)
	if !ok {
		return 0, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	return id, nil
}

// now is the request time, Tehran wall-clock, whole seconds.
func (h *Handlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// WithEngine makes the handlers read the v6 engine (CYCLE_ENGINE shadow|v6, T-M9-35); nil keeps v1 unchanged.
func (h *Handlers) WithEngine(states state.UserStates) *Handlers {
	h.svc.WithEngine(states)
	return h
}

// today is the user-local day under v6 (D-126) and the Tehran day under v1 (unchanged).
func (h *Handlers) today(c fiber.Ctx) civildate.Date {
	if !h.svc.Engine() {
		return civildate.InTehran(h.now(c))
	}
	return civildate.LocalToday(c, clock.FromContext(c, h.clock))
}

// notModified tags a v6 engine GET (D-132): user, engine revision, local day, locale, engine params, route part and
// build. Every table these routes read (fertility_logs, daily_health_logs, health_log_entries, the engine inputs)
// bumps engine_rev (T-M9-30), so the revision covers the body. True = 304 already sent. Never under v1.
func (h *Handlers) notModified(c fiber.Ctx, uid uint64, today civildate.Date, part string) bool {
	if !h.svc.Engine() {
		return false
	}
	var rev uint64
	if r, ok := auth.CurrentRevisions(c); ok {
		rev = r.Engine
	}
	return etag.NotModified(c, etag.Engine{
		UserID: uid, Rev: rev, Today: today, Locale: i18n.Locale(c), ParamsFP: engineparams.Fingerprint(today),
		ConfigFP: "fert-" + part, Build: etag.Build(),
	}.Tag())
}

// engineState is uid's v6 state on today. A read uses the request's engine revision from the auth query on the
// shared provider (no query on an L1 hit); after a write (fresh) or without it, Source.ForUser reads user_revisions.
func (h *Handlers) engineState(c fiber.Ctx, uid uint64, today civildate.Date, fresh bool) (*engine.State, error) {
	if src, ok := h.svc.states.(state.Source); ok && src.P != nil && !fresh {
		if r, ok := auth.CurrentRevisions(c); ok {
			st, err := src.P.For(c.Context(), uid, r.Engine, state.Params{Engine: engineparams.For(today), FP: engineparams.Fingerprint(today)})
			if err != nil {
				return nil, fmt.Errorf("fertility: engine state: %w", err)
			}
			return st, nil
		}
	}
	return h.svc.engineState(c.Context(), uid, today)
}

// load is the merged day of date with its cycle context (v1: Service.Load; v6: the engine state).
func (h *Handlers) load(c fiber.Ctx, uid uint64, date, today civildate.Date, fresh bool) (Day, Context, error) {
	if !h.svc.Engine() {
		return h.svc.Load(c, uid, date, today)
	}
	st, err := h.engineState(c, uid, today, fresh)
	if err != nil {
		return Day{}, Context{}, err
	}
	return h.svc.LoadV6(c, st, uid, date, today)
}

// Today is GET /fertility/today: the home tiles (LH, BBT, intercourse), the chance and the
// cycle day of today (Tehran; the user-local day and an ETag under v6), in three queries.
func (h *Handlers) Today(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	today := h.today(c)
	if h.notModified(c, userID, today, "today") {
		return nil
	}
	day, cx, err := h.load(c, userID, today, today, false)
	if err != nil {
		return err
	}
	return httpx.OK(c, day.TodayJSON(cx, i18n.Locale(c)))
}

// ShowDay is GET /fertility/days/{date}: the merged day (any date; a future one carries the
// predicted chance).
func (h *Handlers) ShowDay(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	date, err := parseDate(dateParam(c), locale, now)
	if err != nil {
		return err
	}
	day, cx, err := h.load(c, userID, date, h.today(c), false)
	if err != nil {
		return err
	}
	return httpx.OK(c, day.JSON(cx, locale))
}

// UpdateDay is PUT /fertility/days/{date}: a partial update (omitted keys stay, null clears),
// then the merged day. A luteal-spotting warning from the health log rides along as the
// top-level `warning`, like POST /health-logs.
func (h *Handlers) UpdateDay(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	in, err := validateDay(dateParam(c), validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	warning, err := h.svc.Save(c, userID, in, locale, now)
	if err != nil {
		return err
	}
	day, cx, err := h.load(c, userID, in.Date, h.today(c), true)
	if err != nil {
		return err
	}
	body := httpx.Envelope(day.JSON(cx, locale), T("messages.day_saved", locale))
	if warning != nil {
		body.Set("warning", warning)
	}
	return httpx.JSON(c, fiber.StatusOK, body)
}

// dateParam is the {date} route segment, URL-decoded.
func dateParam(c fiber.Ctx) string {
	raw := c.Params("date")
	if s, err := url.PathUnescape(raw); err == nil {
		return s
	}
	return raw
}

// BBT is GET /fertility/bbt?range=1|3|6: the basal-temperature chart of the current cycle and,
// for range 3 / 6, the previous cycles aligned by cycle day (internal/fertility/bbt).
func (h *Handlers) BBT(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	locale, now := i18n.Locale(c), h.now(c)
	rangeSize, err := parseRange(validation.Input(c), locale, now)
	if err != nil {
		return err
	}
	if !h.svc.Engine() {
		res, err := h.svc.BBT(c, userID, civildate.InTehran(now), rangeSize)
		if err != nil {
			return err
		}
		return httpx.OK(c, BBTJSON(res, rangeSize, locale))
	}
	today := h.today(c)
	if h.notModified(c, userID, today, "bbt"+strconv.Itoa(rangeSize)) {
		return nil
	}
	st, err := h.engineState(c, userID, today, false)
	if err != nil {
		return err
	}
	return httpx.OK(c, BBTJSON(BBTV6(st, today, rangeSize), rangeSize, locale))
}

// Insights is GET /fertility/insights: the cycle view's fertile window with the confidence,
// evidence rows, past ovulation days and tips that explain it (insights.go).
func (h *Handlers) Insights(c fiber.Ctx) error {
	userID, err := h.user(c)
	if err != nil {
		return err
	}
	if !h.svc.Engine() {
		ins, err := h.svc.Insights(c, userID, civildate.InTehran(h.now(c)))
		if err != nil {
			return err
		}
		return httpx.OK(c, InsightsJSON(ins, i18n.Locale(c)))
	}
	today := h.today(c)
	if h.notModified(c, userID, today, "insights") {
		return nil
	}
	st, err := h.engineState(c, userID, today, false)
	if err != nil {
		return err
	}
	ins, err := h.svc.InsightsV6(c, st, userID, today)
	if err != nil {
		return err
	}
	return httpx.OK(c, InsightsJSON(ins, i18n.Locale(c)))
}
