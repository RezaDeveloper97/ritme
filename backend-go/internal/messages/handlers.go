// Package messages serves GET /messages/daily and GET /messages/mode
// (backend/app/Http/Controllers/Api/V1/MessageController.php) on top of messages/manager.
package messages

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/postpartum"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Handlers are the MessageController actions. Mount them behind auth RequireUser and the
// locale middleware.
type Handlers struct {
	q        *store.Queries
	pq       *pstore.Queries
	clock    clock.Clock
	accounts *companion.Accounts // B-N4-03: a companion (male) account has no daily cycle messages → 409
	pp       PostpartumSignals   // B-N5-01: the postpartum engine's facts
}

// NewHandlers wires the handlers; base is the fallback clock.
func NewHandlers(db store.DBTX, base clock.Clock) *Handlers {
	return &Handlers{q: store.New(db), pq: pstore.New(db), clock: base, accounts: companion.NewAccounts(db), pp: postpartum.NewService(db)}
}

// request is one request's manager and its source.
func (h *Handlers) request(c fiber.Ctx) (*manager.Manager, *StoreSource, string, civildate.Date, error) {
	uid, ok := auth.CurrentUserID(c)
	if !ok {
		return nil, nil, "", civildate.Date{}, &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	is, err := h.accounts.IsCompanion(c.Context(), uid)
	if err != nil {
		return nil, nil, "", civildate.Date{}, err
	}
	if is { // bloom B-N4-03: no cycle engine for a companion account
		return nil, nil, "", civildate.Date{}, companion.AccountConflict(i18n.Locale(c))
	}
	locale := i18n.ResolveLocale(c, "")
	today := civildate.Today(clock.FromContext(c, h.clock))
	src := NewStoreSource(h.q, h.pq, uid, today)
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
		p, err := src.Profile(ctx)
		if err != nil {
			return err
		}
		if p == nil || !p.HasLastPeriodStart {
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
