// Package messages serves GET /messages/daily and GET /messages/mode
// (backend/app/Http/Controllers/Api/V1/MessageController.php) on top of messages/manager.
package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/usertz"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/postpartum"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Handlers are the MessageController actions. Mount them behind auth RequireUser and the
// locale middleware.
type Handlers struct {
	q        *store.Queries
	pq       *pstore.Queries
	clock    clock.Clock
	accounts *companion.Accounts // B-N4-03: a companion (male) account has no daily cycle messages → 409
	pp       PostpartumSignals   // B-N5-01: the postpartum engine's facts
	states   state.UserStates    // T-M9-38: the v6 engine state source (nil = CYCLE_ENGINE=v1)
	zones    zoneStore           // T-M9-38b: a user's stored zone (admin preview of another user's day)
	feed     *feed.Service       // T-M10-06: the message feed (GET /messages/feed, /messages/selected)
	writer   *events.Writer      // T-M10-06: the buffered event log (nil = synchronous inserts)
	marks    *events.Marks       // T-M10-06: once-a-day marks of server-logged events (nil = DB check)
	segs     SegmentResolver     // T-M10-32: the `segment` fact (nil = empty set)
}

// WithSegments wires the saved admin segments behind the `segment` fact (T-M10-32).
func (h *Handlers) WithSegments(r SegmentResolver) *Handlers {
	h.segs = r
	return h
}

// WithFeed wires the feed's event log: the buffered writer, the Redis marks and the app module switches (T-M10-06).
// Without it the feed still answers but logs nothing and client events are inserted synchronously.
func (h *Handlers) WithFeed(w *events.Writer, marks *events.Marks, mods feed.Modules) *Handlers {
	h.writer, h.marks = w, marks
	h.feed = feed.New(h.q, w, marks, mods)
	return h
}

// zoneStore reads user_profiles.timezone (*profilestore.Queries).
type zoneStore interface {
	GetUserTimezone(ctx context.Context, userID uint64) (sql.NullString, error)
}

// WithEngine reads the cycle facts from the v6 engine (CYCLE_ENGINE=shadow|v6, T-M9-38, D-130); nil keeps v1.
func (h *Handlers) WithEngine(states state.UserStates) *Handlers {
	h.states = states
	return h
}

// NewHandlers wires the handlers; base is the fallback clock.
func NewHandlers(db store.DBTX, base clock.Clock) *Handlers {
	q := store.New(db)
	return &Handlers{
		q: q, pq: pstore.New(db), clock: base, accounts: companion.NewAccounts(db), pp: postpartum.NewService(db),
		zones: profilestore.New(db), feed: feed.New(q, nil, nil, nil),
	}
}

// UserToday is userID's calendar day at now for a caller that is not that user (the admin rule preview): under v6 the
// user's local day from her stored zone (D-126; no request header to read — Tehran when none is stored), the same day
// her own /messages/daily and /messages/selected resolve without a header; v1 keeps the Tehran day (T-M9-38b).
func (h *Handlers) UserToday(ctx context.Context, userID uint64, now time.Time) (civildate.Date, error) {
	if h.states == nil {
		return civildate.InTehran(now), nil
	}
	name, err := h.zones.GetUserTimezone(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return civildate.Date{}, fmt.Errorf("messages: user zone: %w", err)
	}
	return civildate.FromTime(now.In(usertz.ZoneOf(name))), nil
}

// UserZone is userID's stored zone (user_profiles.timezone; Tehran when none) — the admin simulator (T-M10-11).
func (h *Handlers) UserZone(ctx context.Context, userID uint64) (*time.Location, error) {
	name, err := h.zones.GetUserTimezone(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("messages: user zone: %w", err)
	}
	return usertz.ZoneOf(name), nil
}

// request is one request's manager and its source.
func (h *Handlers) request(c fiber.Ctx) (*manager.Manager, *StoreSource, string, civildate.Date, error) {
	return h.requestOf(c, false)
}

// requestOf is request; deferCompanion leaves the companion check to the caller (the v6 feed reads the account kind
// off the fact profile row it reads anyway, T-M10-06).
func (h *Handlers) requestOf(c fiber.Ctx, deferCompanion bool) (*manager.Manager, *StoreSource, string, civildate.Date, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return nil, nil, "", civildate.Date{}, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	if !deferCompanion {
		is, err := h.accounts.IsCompanion(c.Context(), uid)
		if err != nil {
			return nil, nil, "", civildate.Date{}, err
		}
		if is { // bloom B-N4-03: no cycle engine for a companion account
			return nil, nil, "", civildate.Date{}, companion.AccountConflict(i18n.Locale(c))
		}
	}
	locale := i18n.ResolveLocale(c, "")
	today := civildate.Today(clock.FromContext(c, h.clock))
	if h.states != nil { // v6: the user-local day, as every engine consumer (D-126)
		today = civildate.LocalToday(c, clock.FromContext(c, h.clock))
	}
	src := NewStoreSource(h.q, h.pq, uid, today).WithEngine(h.states).WithSegments(h.segs)
	return manager.New(WithPostpartum(src, h.pp), content.NewRepository(h.q), locale, today), src, locale, today, nil
}

func pick(locale, fa, en string) string {
	if locale == "fa" {
		return fa
	}
	return en
}

// Daily is MessageController::daily.
func (h *Handlers) Daily(c fiber.Ctx) error {
	m, src, locale, today, err := h.request(c)
	if err != nil {
		return err
	}
	ctx := c.Context()

	query := validation.Query(c)
	F := validation.F
	v := validation.Make(lang.Default(), i18n.Locale(c), query, validation.Rules{
		F("date", "nullable", "date_format:Y-m-d"),
		F("mode", "nullable", validation.In(enums.MessageModeValues()...)),
	}, validation.Now(clock.FromContext(c, h.clock).Now()))
	if v.Fails() {
		errs, _ := v.Errors().Body().Get("errors")
		return httpx.Fail(fiber.StatusUnprocessableEntity,
			pick(locale, "پارامترهای ورودی نامعتبر است", "Invalid query parameters"), "errors", errs)
	}

	date := today
	if raw, _ := query.Get("date"); phpval.Truthy(raw) {
		if date, err = civildate.Parse(phpval.ToString(raw)); err != nil {
			return err // unreachable: date_format:Y-m-d passed
		}
	}
	var force enums.MessageMode
	if raw, _ := query.Get("mode"); phpval.Truthy(raw) {
		force = enums.MessageMode(phpval.ToString(raw))
	}
	if force == enums.MessageModePregnancy {
		if force, err = h.pregnancyEnded(c, src, force); err != nil {
			return err
		}
	}

	detected := force
	if detected == "" {
		if detected, err = m.DetectMode(ctx); err != nil {
			return err
		}
	}
	switch detected {
	case enums.MessageModeCycle:
		has, err := hasCycleAnchor(ctx, src, today)
		if err != nil {
			return err
		}
		if !has {
			return httpx.Fail(fiber.StatusBadRequest, pick(locale,
				"لطفاً ابتدا پروفایل خود را تکمیل کنید (تاریخ آخرین پریود)",
				"Please complete your profile first (last period date)"))
		}
	case enums.MessageModePregnancy:
		p, err := src.PregnancyProfile(ctx)
		if err != nil {
			return err
		}
		if p == nil || !p.OnboardingCompleted {
			return httpx.Fail(fiber.StatusBadRequest, pick(locale,
				"لطفاً ابتدا آنبوردینگ بارداری را تکمیل کنید",
				"Please complete pregnancy onboarding first"))
		}
	}

	res, err := m.Generate(ctx, date, force)
	if err != nil {
		return err
	}
	return httpx.OK(c, res.JSON())
}

// hasCycleAnchor is the daily's "has cycle data" gate: v1 keeps Laravel's `$profile->last_period_start`; under v6
// (T-M9-38b) the engine has an anchor — a logged period or the declared onboarding start (I-01) — so a user with
// logged periods and an empty profile field gets her message, as Home does (Context.HasCycleData).
func hasCycleAnchor(ctx context.Context, src *StoreSource, today civildate.Date) (bool, error) {
	d, ok, err := src.EngineDay(ctx, today)
	if err != nil {
		return false, err
	}
	if ok {
		return d.HasAnchor, nil
	}
	p, err := src.Profile(ctx)
	if err != nil {
		return false, err
	}
	return p != nil && p.HasLastPeriodStart, nil
}

// pregnancyEnded drops a forced ?mode=pregnancy when her pregnancy has ended (a pregnancy profile with pregnancy mode
// off — a pregnancy loss, CB-LOSS-01, or a deactivation), so the detected mode answers instead: no pregnancy message
// reaches her after the pregnancy, durably (it does not depend on the loss record, which she may erase). Without any
// pregnancy profile Laravel's 400 «complete pregnancy onboarding first» is kept.
func (h *Handlers) pregnancyEnded(c fiber.Ctx, src *StoreSource, force enums.MessageMode) (enums.MessageMode, error) {
	p, err := src.PregnancyProfile(c)
	if err != nil || p == nil || p.PregnancyMode {
		return force, err
	}
	return "", nil
}

// Mode is MessageController::mode.
func (h *Handlers) Mode(c fiber.Ctx) error {
	m, src, locale, _, err := h.request(c)
	if err != nil {
		return err
	}
	ctx := c.Context()
	p, err := src.Profile(ctx)
	if err != nil {
		return err
	}
	mode, err := m.DetectMode(ctx)
	if err != nil {
		return err
	}
	goal, sub := "non_ttc", "free"
	if p != nil {
		goal, sub = p.UserGoal, p.SubscriptionType
	}
	return httpx.OK(c, jsonx.Obj(
		"mode", string(mode),
		"mode_label", mode.Label(locale),
		"user_goal", goal,
		"subscription_type", sub,
		"is_ttc", p != nil && p.UserGoal == string(enums.UserGoalTtc),
		"is_premium", p != nil && p.SubscriptionType == string(enums.SubscriptionTypePremium),
	))
}
