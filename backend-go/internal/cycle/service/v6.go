package service

// The v6 engine behind /cycle/today and /cycle/date/{date} (T-M9-32, D-120, DECISIONS.md X-03/X-05):
//
//	auth (1 query: token ⋈ user ⋈ user_revisions → engine_rev)
//	 → ETag W/"u…e{rev}-{localToday}-{locale}-{paramsFP}-{tipsFP}-{build}"; If-None-Match → 304
//	 → state.Provider.For(uid, rev, params of the local day)   L1 hit = no query
//	 → Resolve(target, LocalToday) + FutureCycles(LocalToday) + TodayUI
//	 → §2.35 encoder (engine.AppendJSON) + the legacy keys rebuilt from the status (view.LegacyFromV6)
//	   + the per-day `calculation` block rebuilt from the status (calculationV6) — daily tips from the
//	   recommendations process snapshot (no query when warm)
//
// CYCLE_ENGINE=v1 never reaches this file; shadow serves v1 and hands the same day to the shadow pool after the
// response (shadowSubmit).

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/shadow"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n"
	pcache "github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Engine is the CYCLE_ENGINE rollout of /cycle/today and /cycle/date. A nil *Engine (or mode v1) serves v1.
type Engine struct {
	mode   config.CycleEngine
	states *state.Provider
	tips   recommendation.Source
	pool   *shadow.Pool
	build  string
}

// EngineDeps are NewEngine's inputs. States is required for shadow and v6; Redis (samples) may be nil.
type EngineDeps struct {
	Mode   config.CycleEngine
	States *state.Provider
	DB     store.DBTX
	Redis  *pcache.Client
	Build  string // etag.Build() in production
}

// NewEngine wires the rollout. Mode v1 returns nil (nothing to do).
func NewEngine(d EngineDeps, pool func(shadow.Compute) *shadow.Pool) (*Engine, error) {
	if d.Mode == "" || d.Mode == config.CycleEngineV1 {
		return nil, nil
	}
	if d.States == nil {
		return nil, fmt.Errorf("cycle: CYCLE_ENGINE=%s needs the engine state provider", d.Mode)
	}
	e := &Engine{mode: d.Mode, states: d.States, tips: RecommendationSource{Q: store.New(d.DB)}, build: d.Build}
	if d.Mode == config.CycleEngineShadow {
		if pool == nil {
			pool = func(c shadow.Compute) *shadow.Pool { return shadow.New(c, d.Redis, nil) }
		}
		e.pool = pool(e.shadowCompute)
	}
	return e, nil
}

// Mode is the rollout mode (v1 for a nil engine).
func (e *Engine) Mode() config.CycleEngine {
	if e == nil {
		return config.CycleEngineV1
	}
	return e.mode
}

// Shadow is the shadow pool (nil unless CYCLE_ENGINE=shadow).
func (e *Engine) Shadow() *shadow.Pool {
	if e == nil {
		return nil
	}
	return e.pool
}

// WithEngine attaches the rollout to the handlers (nil = v1).
func (h *Handlers) WithEngine(e *Engine) *Handlers {
	h.eng = e
	return h
}

func (h *Handlers) servesV6() bool { return h.eng.Mode() == config.CycleEngineV6 }

// stateFor is the prepared state of uid at rev with the params valid on the user-local day.
func (e *Engine) stateFor(ctx context.Context, uid, rev uint64, today civildate.Date) (*engine.State, error) {
	st, err := e.states.For(ctx, uid, rev, state.Params{Engine: engineparams.For(today), FP: engineparams.Fingerprint(today)})
	if err != nil {
		return nil, fmt.Errorf("cycle: engine state: %w", err)
	}
	return st, nil
}

// tipsFingerprint is the config part of the ETag: the recommendations snapshot the daily tips come from.
func tipsFingerprint() string { return configcache.Fingerprint(recommendation.Table) }

// Tag is the ETag of uid's v6 day payload for the local day and locale.
func (e *Engine) Tag(uid, rev uint64, today civildate.Date, locale string) string {
	return etag.Engine{
		UserID: uid, Rev: rev, Today: today, Locale: locale,
		ParamsFP: engineparams.Fingerprint(today), ConfigFP: tipsFingerprint(), Build: e.build,
	}.Tag()
}

// Day is the v6 status of target for the local day today: Resolve + today's future_cycles + the status's today_ui.
func (e *Engine) Day(ctx context.Context, uid, rev uint64, target, today civildate.Date) (*engine.State, engine.Status, engine.TodayUI, error) {
	st, err := e.stateFor(ctx, uid, rev, today)
	if err != nil {
		return nil, engine.Status{}, engine.TodayUI{}, err
	}
	t := engine.DayOf(today)
	status := st.Resolve(engine.DayOf(target), t)
	status.FutureCycles = st.FutureCycles(t)
	return st, status, st.TodayUI(status), nil
}

var bufPool = sync.Pool{New: func() any { b := make([]byte, 0, 8<<10); return &b }}

// DayBody is the whole 200 body of a v6 /cycle/today or /cycle/date:
// {"success":true,"data":{"calculation":…,"cycle_view":…,"calculation_status":"completed","is_recalculating":false}}.
// The engine has no asynchronous recalculation, so the status is always completed (D-120).
func (e *Engine) DayBody(ctx context.Context, uid, rev uint64, target, today civildate.Date, locale string) ([]byte, error) {
	st, status, ui, err := e.Day(ctx, uid, rev, target, today)
	if err != nil {
		return nil, err
	}
	calc, err := e.calculationV6(ctx, st, &status, locale)
	if err != nil {
		return nil, err
	}
	legacyJSON, err := jsonx.Marshal(view.LegacyFromV6(view.V6Input{
		State: st, Status: &status, UI: &ui, Today: engine.DayOf(today), Locale: locale,
	}), jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return nil, fmt.Errorf("cycle: encode legacy keys: %w", err)
	}
	calcJSON, err := jsonx.Marshal(calc, jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	if err != nil {
		return nil, fmt.Errorf("cycle: encode calculation: %w", err)
	}

	bp := bufPool.Get().(*[]byte)
	b := (*bp)[:0]
	b = append(b, `{"success":true,"data":{"calculation":`...)
	b = append(b, calcJSON...)
	b = append(b, `,"cycle_view":`...)
	b = engine.AppendJSON(b, &status, &ui)
	// Splice the legacy keys into the §2.35 object: drop its closing brace and the legacy object's opening one.
	b = append(b[:len(b)-1], ',')
	b = append(b, legacyJSON[1:]...)
	b = append(b, `,"calculation_status":"completed","is_recalculating":false}}`...)
	out := append([]byte(nil), b...)
	*bp = b
	bufPool.Put(bp)
	return out, nil
}

// calculationV6 is the per-day `calculation` block rebuilt from the v6 status (X-05). Fields without a v6 value are
// null: is_luteal_spotting, the scores (cycle_score, age_factor, base_probability, symptom_score, final_probability),
// uncertainty_range, text_flags, source_profile_data and source_daily_log_data. daily_tips are the recommendations of
// the I-11 content phase without symptom triggers (the engine does not load the day's log); a day without a cycle day
// is the empty calculation (every core field null, no tips).
func (e *Engine) calculationV6(ctx context.Context, st *engine.State, s *engine.Status, locale string) (*jsonx.OrderedMap, error) {
	m := jsonx.NewObject()
	m.Set("calculation_date", s.Date.Civil())
	cd, complete := s.CycleDay.Get()
	phase, hasPhase := view.ContentPhase(s.Subphase)
	if complete {
		m.Set("cycle_day", cd)
		if hasPhase {
			m.Set("phase", phase)
			m.Set("subphase", s.Subphase.String())
		} else {
			m.Set("phase", nil).Set("subphase", nil)
		}
		if ov, start := s.Anchors.EstimatedOvulationDate, s.Anchors.CurrentPeriodStart; ov.IsSet() && start.IsSet() {
			m.Set("estimated_ovulation_day", int(ov-start)+1)
		} else {
			m.Set("estimated_ovulation_day", nil)
		}
		m.Set("cycle_length_used", s.Metrics.EffectiveCycleLength)
	} else {
		for _, k := range []string{"cycle_day", "phase", "subphase", "estimated_ovulation_day", "cycle_length_used"} {
			m.Set(k, nil)
		}
	}
	m.Set("is_fertile_window", complete && s.MainPhase == engine.MainPhaseFertile && !s.Limits.FertilitySuppressed).
		Set("is_pms_window", complete && s.Subphase == engine.SubphasePMSPossible).
		Set("is_period_tomorrow", complete && s.DaysToPeriod.Or(0) == 1 && s.DaysToPeriod.OK())
	for _, k := range []string{"is_luteal_spotting", "cycle_score", "age_factor", "base_probability", "symptom_score", "final_probability"} {
		m.Set(k, nil)
	}
	switch s.Metrics.CycleRegularity {
	case engine.RegularityRegular:
		m.Set("cycle_variability", enums.CycleVariabilityRegular)
	case engine.RegularityIrregular:
		m.Set("cycle_variability", enums.CycleVariabilityIrregular)
	default:
		m.Set("cycle_variability", nil)
	}
	m.Set("uncertainty_range", nil).Set("text_flags", nil)

	tips := []recommendation.LocalizedTip{}
	if complete {
		t, err := e.dailyTips(ctx, phase, s.Subphase, view.NoFertilityCopy(st.Mode()))
		if err != nil {
			return nil, err
		}
		tips = recommendation.Localize(t, locale)
	}
	m.Set("daily_tips", tips)
	m.Set("source_profile_data", nil).Set("source_daily_log_data", nil)
	return m, nil
}

// dailyTips are the admin recommendations of the content phase (I-11; "" = neutral rows only), fertility tips dropped
// for a mode without fertility content. An install without any recommendation row gets none (the legacy built-in copy
// lives in the legacy engine and is not part of v6).
func (e *Engine) dailyTips(ctx context.Context, phase enums.CyclePhase, sub engine.Subphase, noFertility bool) ([]recommendation.Tip, error) {
	repo := e.tipsRepo(ctx)
	has, err := repo.HasContent(ctx)
	if err != nil || !has {
		return nil, err
	}
	subphase := enums.CycleSubphase("")
	if phase != "" {
		subphase = enums.CycleSubphase(sub.String())
	}
	tips, err := repo.ForDay(ctx, phase, subphase, nil)
	if err != nil {
		return nil, err
	}
	if !noFertility {
		return tips, nil
	}
	out := make([]recommendation.Tip, 0, len(tips))
	for _, t := range tips {
		if t.Type != "fertility" {
			out = append(out, t)
		}
	}
	return out, nil
}

type tipsKey struct{}

// withTips carries a request-scoped recommendation repository to dailyTips.
func withTips(ctx context.Context, r *recommendation.Repository) context.Context {
	return context.WithValue(ctx, tipsKey{}, r)
}

// tipsRepo is the request's repository (withTips), else a fresh one.
func (e *Engine) tipsRepo(ctx context.Context) *recommendation.Repository {
	if r, ok := ctx.Value(tipsKey{}).(*recommendation.Repository); ok {
		return r
	}
	return recommendation.New(e.tips)
}

// revision is the request's engine revision from the auth query (0 without a user_revisions row).
func revision(c fiber.Ctx) uint64 {
	r, _ := auth.CurrentRevisions(c)
	return r.Engine
}

// dayV6 answers a v6 day: 304 on a matching If-None-Match (nothing else runs), else the v6 body.
func (h *Handlers) dayV6(c fiber.Ctx, target, today civildate.Date) error {
	uid, err := userID(c)
	if err != nil {
		return err
	}
	rev := revision(c)
	locale := i18n.ResolveLocale(c, "")
	// The tag names the recommendations snapshot, so it is loaded first (a process-snapshot hit when warm): a cold
	// process would otherwise tag its first answer without it and never revalidate that body.
	// One request-scoped repository for the tag and the daily tips: it memoises, so a cold process (or caches off)
	// reads the table once per request.
	ctx := withTips(c.Context(), recommendation.New(h.eng.tips))
	if _, err := h.eng.tipsRepo(ctx).HasContent(ctx); err != nil {
		return err
	}
	if etag.NotModified(c, h.eng.Tag(uid, rev, today, locale)) {
		return nil
	}
	body, err := h.eng.DayBody(ctx, uid, rev, target, today, locale)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(fiber.StatusOK).Send(body)
}

// shadowCompute is the shadow pool's v6 computation (the same state cache as v6 serving).
func (e *Engine) shadowCompute(ctx context.Context, uid, rev uint64, target, today engine.Day) (engine.Status, error) {
	_, s, _, err := e.Day(ctx, uid, rev, target.Civil(), today.Civil())
	return s, err
}

// shadowSubmit hands the served v1 day ({calculation, cycle_view}) to the shadow pool. The job context carries the request's zone and clock only
// (the fiber context is recycled after the response); a full pool drops the job.
func (h *Handlers) shadowSubmit(c fiber.Ctx, uid uint64, target, today civildate.Date, v1Data []byte) {
	pool := h.eng.Shadow()
	if pool == nil {
		return
	}
	loc := civildate.Zone(c.Context())
	ctx := clock.WithClock(context.Background(), clock.At(clock.FromContext(c, h.clock).Now()))
	ctx = civildate.WithZone(ctx, func(context.Context) *time.Location { return loc })
	pool.Submit(shadow.Job{
		Ctx: ctx, UserID: uid, Rev: revision(c),
		Target: engine.DayOf(target), Today: engine.DayOf(today),
		V1: append([]byte(nil), v1Data...),
	})
}
