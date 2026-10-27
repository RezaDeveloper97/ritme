package http

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/pregnancy"
	pregnancystore "github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/shop/checklist"
	shopstore "github.com/ritme/backend-go/internal/shop/store"
	"github.com/ritme/backend-go/internal/teen"
)

// Layette checklist + period-supply reminder (CB-SHOP-04), Go only. auth:api, localized by Accept-Language; teen-mode
// accounts get 404 (no shop). The shop package stays free of health imports: the due date, the cycle prediction and
// the companion share check are adapted here.
func init() {
	Register("shop_checklist", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		var predict checklist.PeriodPredictor = shopPeriodPredictor{cycle: cycleservice.New(d.DB, nil)}
		if states := engineStates(r, d); states != nil {
			predict = shopEnginePredictor{states: states, revs: cyclestore.New(d.DB)}
		}
		svc := checklist.NewService(shopstore.New(d.DB), shopDueDate{q: pregnancystore.New(d.DB)}, predict)
		h := checklist.NewHandlers(svc, teen.NewPolicy(d.DB),
			shopChecklistShare{d: companion.NewDelegation(d.DB, clock.Real{}, d.Logger), preg: pregnancystore.New(d.DB)}, clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/shop/checklist", locale, guard, delegatedReadThrottle(d), h.Show)
		r.Post("/api/v1/shop/checklist/custom", locale, guard, writes, h.StoreCustom)
		r.Patch("/api/v1/shop/checklist/custom/:id", locale, guard, writes, h.UpdateCustom)
		r.Delete("/api/v1/shop/checklist/custom/:id", locale, guard, writes, h.DestroyCustom)
		r.Put("/api/v1/shop/checklist/items/:id", locale, guard, writes, h.SetState)
		r.Get("/api/v1/shop/supply-reminder", locale, guard, h.ShowSupply)
		r.Put("/api/v1/shop/supply-reminder", locale, guard, writes, h.UpdateSupply)
	})
}

// shopDueDate is the active pregnancy's due date as the pregnancy calculator resolves it.
type shopDueDate struct{ q pregnancystore.Querier }

func (s shopDueDate) DueDate(ctx context.Context, userID uint64, today civildate.Date) (civildate.Date, bool, error) {
	p, err := pregnancy.ActiveProfile(ctx, s.q, userID)
	if errors.Is(err, pregnancy.ErrNotPregnant) || errors.Is(err, sql.ErrNoRows) {
		return civildate.Date{}, false, nil
	}
	if err != nil {
		return civildate.Date{}, false, err
	}
	due, ok := pregnancy.CurrentDue(p, today)
	return due, ok, nil
}

// shopPeriodPredictor is the engine's predicted next period start, only for the cycle-tracking life modes (none,
// cycle, TTC): pregnancy, postpartum and menopause modes get no supply reminder. Never logs the date.
type shopPeriodPredictor struct{ cycle *cycleservice.Service }

func (p shopPeriodPredictor) NextPeriodStart(ctx context.Context, userID uint64, today civildate.Date) (checklist.Prediction, error) {
	sn, err := p.cycle.Load(ctx, userID, today, today, today)
	if err != nil {
		return checklist.Prediction{}, err
	}
	switch sn.LifeMode {
	case "", enums.LifeModeCycle, enums.LifeModeTTC:
	default:
		return checklist.Prediction{}, nil
	}
	if len(sn.Histories) == 0 && sn.Profile == nil {
		return checklist.Prediction{}, nil // nothing logged and nothing declared
	}
	// T-M9-27 (D-123): an onboarding-only user has no period row any more; the declared LMP on the profile still
	// predicts (onboarding_based), as the old seed row did.
	profile := sn.EngineProfile()
	st := resolver.Resolve(sn.Histories, profile, today, today, metrics.Calculate(sn.Histories, profile))
	next := st.PredictedNextPeriodStart
	return checklist.Prediction{Next: next, Known: !next.IsZero()}, nil
}

// Revision is unknown under v1: every read re-syncs the reminder.
func (shopPeriodPredictor) Revision(context.Context, uint64) (uint64, bool, error) {
	return 0, false, nil
}

// shopEnginePredictor is the v6 engine's predicted_next_period_start (CYCLE_ENGINE shadow / v6, T-M9-37, D-136):
// gated on the engine's resolved mode (cycle or ttc — a pregnant user without a stored mode is pregnancy here) and on
// its limits (pregnancy and postpartum-not-returned predict nothing). The prediction carries the engine revision it
// was computed from (I-10), so the stored reminder is re-synced only when that revision moves. Never logs the date.
type shopEnginePredictor struct {
	states state.UserStates
	revs   state.RevisionReader
}

func (p shopEnginePredictor) NextPeriodStart(ctx context.Context, userID uint64, today civildate.Date) (checklist.Prediction, error) {
	rev, _, err := p.Revision(ctx, userID)
	if err != nil {
		return checklist.Prediction{}, err
	}
	st, err := p.states.ForUser(ctx, userID, today)
	if err != nil {
		return checklist.Prediction{}, err
	}
	out := checklist.Prediction{Rev: rev, Revisioned: true}
	if m := st.Mode(); m != engine.ModeCycle && m != engine.ModeTTC {
		return out, nil
	}
	t := engine.DayOf(today)
	s := st.Resolve(t, t)
	if s.Limits.EngineSuspended || s.Limits.SuppressionReason == engine.SuppressionPostpartumNotReturned {
		return out, nil
	}
	if next := s.Anchors.PredictedNextPeriodStart; next != engine.NoDay {
		out.Next, out.Known = next.Civil(), true
	}
	return out, nil
}

// Revision is the user's engine revision (user_revisions.engine_rev, 0 without a row).
func (p shopEnginePredictor) Revision(ctx context.Context, userID uint64) (uint64, bool, error) {
	r, err := p.revs.GetUserRevisions(ctx, userID)
	switch {
	case err == nil:
		return r.EngineRev, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return 0, true, nil
	}
	return 0, false, fmt.Errorf("shop: engine revision: %w", err)
}

// shopChecklistShare is the family share: a companion with a live `pregnancy` grant (view or edit) reads the
// owner's checklist while she is pregnant — the same pregnancy check as the companion pregnancy section (T-M9-02):
// postpartum, after a loss or between pregnancies nothing is shared, and a grant reset by
// companion.ResetModeScopedGrants opens nothing (so no due date or weeks left reach a member). The read is audited
// by the companion delegation before any data is touched; a refused one is not.
type shopChecklistShare struct {
	d    *companion.Delegation
	preg pregnancystore.Querier
}

func (s shopChecklistShare) CanView(ctx context.Context, ownerID, viewerID uint64) (bool, error) {
	if ownerID != viewerID {
		_, err := pregnancy.ActiveProfile(ctx, s.preg, ownerID)
		if errors.Is(err, pregnancy.ErrNotPregnant) || errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}
	return s.d.Authorize(ctx, ownerID, viewerID, companion.SectionPregnancy, false)
}
