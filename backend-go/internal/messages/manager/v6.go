package manager

// The message system on the v6 engine (T-M9-38, D-130, DECISIONS.md I-11). Under CYCLE_ENGINE=shadow|v6 a Source that
// also implements EngineSource feeds the cycle context from the engine status of the day instead of the legacy
// calculation (Source.Cycle is then never called):
//
//	phase            I-11 content phase: menstrual → menstruation; follicular and fertile_rising → follicular;
//	                 high_fertility and ovulation_likely → ovulation; luteal and period_expected → luteal;
//	                 unknown → no phase (neutral content only: the "default" base copy, no phase supplements)
//	cycle_day        the engine's cycle day, which never wraps: a late cycle keeps counting (day 36 of a 28-day cycle),
//	                 it is never "day N of a new cycle"
//	is_fertile       the display window: a cycle day in the fertile main phase (O−5..O), fertility not suppressed —
//	                 O+1 is post_ovulation (luteal), never fertile
//	is_pms_window    pms_possible days only (next − 3 .. next − 1)
//
// Fertility is stripped (no fertile window, no ovulation day, no ovulation content, no fertility-worded subphase) when
// the engine suppresses it (hormonal contraception, …) or the mode has no fertility copy (teen, menopause).

import (
	"context"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// EngineDay is the v6 engine's view of one day: the resolved status and the engine mode.
type EngineDay struct {
	Status engine.Status
	Mode   engine.Mode
	// HasAnchor is true when the engine has a real period or the declared onboarding start (I-01): a day without a
	// content phase is then neutral, not «enter your last period».
	HasAnchor bool
}

// EngineSource is an optional Source extension (T-M9-38): the v6 status of date. ok is false when the source has no
// engine (CYCLE_ENGINE=v1, or a source that never had one), and the manager keeps the legacy Cycle path.
type EngineSource interface {
	EngineDay(ctx context.Context, date civildate.Date) (EngineDay, bool, error)
}

// NoFertility reports whether the day's fertility content is dropped: the engine suppresses fertility or the engine
// mode has no fertility copy (teen, menopause).
func (d *EngineDay) NoFertility() bool {
	return d.Status.Limits.FertilitySuppressed || view.NoFertilityCopy(d.Mode)
}

// HasCycleDay reports whether the status has a cycle day.
func (d *EngineDay) HasCycleDay() bool { return d.Status.CycleDay.OK() }

// ContentPhase is the I-11 content phase of the day ("" = unknown → neutral content only), before any fertility strip.
func (d *EngineDay) ContentPhase() enums.CyclePhase {
	if !d.HasCycleDay() {
		return ""
	}
	p, ok := view.ContentPhase(d.Status.Subphase)
	if !ok {
		return ""
	}
	return p
}

// Fertile is the display fertile window (O−5..O, never O+1), false when fertility is dropped.
func (d *EngineDay) Fertile() bool {
	return d.HasCycleDay() && d.Status.MainPhase == engine.MainPhaseFertile && !d.NoFertility()
}

// PMS reports whether the day is a pms_possible day.
func (d *EngineDay) PMS() bool {
	return d.HasCycleDay() && d.Status.Subphase == engine.SubphasePMSPossible
}

// DaysToPeriod is the engine's days to the next period, 0 when the period is late; ok is false without a prediction.
func (d *EngineDay) DaysToPeriod() (int, bool) {
	if late, ok := d.Status.DaysLate.Get(); ok && late > 0 {
		return 0, true
	}
	return d.Status.DaysToPeriod.Get()
}

// strippedPhase is a content phase without fertility: the ovulation content days (high_fertility, ovulation_likely)
// precede or are the ovulation, so they read as follicular.
func strippedPhase(p enums.CyclePhase) enums.CyclePhase {
	if p == enums.CyclePhaseOvulation {
		return enums.CyclePhaseFollicular
	}
	return p
}

// engineDay returns the source's engine day of date (nil = the legacy path).
func (m *Manager) engineDay(ctx context.Context, date civildate.Date) (*EngineDay, error) {
	es, ok := m.src.(EngineSource)
	if !ok {
		return nil, nil
	}
	d, ok, err := es.EngineDay(ctx, date)
	if err != nil || !ok {
		return nil, err
	}
	return &d, nil
}

// cycleContextV6 is the cycle context of an engine day.
func cycleContextV6(mc *Context, d *EngineDay) {
	mc.V6, mc.V6Anchor = true, d.HasAnchor
	s := &d.Status
	cd, ok := s.CycleDay.Get()
	if !ok {
		return
	}
	length := s.Metrics.EffectiveCycleLength
	mc.CycleDay, mc.CycleLength = &cd, &length
	if p := d.ContentPhase(); p != "" {
		mc.CyclePhase, mc.CycleSubphase = string(p), s.Subphase.String()
	}
	if ov, start := s.Anchors.EstimatedOvulationDate, s.Anchors.CurrentPeriodStart; ov.IsSet() && start.IsSet() && !d.NoFertility() {
		o := int(ov-start) + 1
		mc.EstimatedOvulationDay = &o
	}
	mc.IsFertileWindow, mc.IsPmsWindow = d.Fertile(), d.PMS()
	if d.NoFertility() { // no fertility content of any kind, TTC copy included (a pill user with goal TTC)
		mc.withoutFertility()
		mc.UserGoal = string(enums.UserGoalNonTtc)
	}
}

// neutral reports whether a v6 day has no content phase (unknown: neutral content only).
func (mc *Context) neutral() bool { return mc.V6 && mc.CyclePhase == "" }

// neutralBase is the base message of a v6 day without a content phase: the "default" (phase-free) copy of the base
// group, no phase. A day without any cycle data keeps getDefaultMessage («enter your last period»).
func (e cycleEngine) neutralBase(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error) {
	group := "cycle_base_non_ttc"
	if mc.IsTTC() {
		group = "cycle_base_ttc"
	}
	p, err := e.content.Resolve(ctx, group, "default", e.locale)
	if err != nil {
		return nil, err
	}
	out := jsonx.Obj("phase", nil, "phase_label", nil, "cycle_day", intOrNil(mc.CycleDay))
	if mc.IsTTC() {
		out.Set("is_fertile_window", false).Set("fertility_info", nil)
	}
	out.Set("short_message", p.Or("short", "")).
		Set("long_message", p.Or("long", "")).
		Set("action_suggestion", p.Or("action", "")).
		Set("dos", p.Or("dos", emptyList())).
		Set("donts", p.Or("donts", emptyList()))
	if mc.IsTTC() {
		out.Set("ttc_tips", p.Or("ttc_tips", emptyList()))
	}
	return out, nil
}

// EngineFacts are the smart-message v2 facts of an engine day (T-M9-38): the cycle numbers from the engine, the content
// phase (I-11), lateness, data quality and the pregnancy-test suggestion. noFertilityMode is the stored life mode's
// rule (teen, menopause: no fertility content) on top of the engine's own. Fertility facts (is_fertile_window, the
// ovulation content phase, fertility-worded subphases, the fertile main phase) are stripped when fertility is
// suppressed or the mode has none; fertility_suppressed is then true.
func EngineFacts(d *EngineDay, noFertilityMode bool) map[string]any {
	s := &d.Status
	modeNoFert := noFertilityMode || view.NoFertilityCopy(d.Mode)
	strip := modeNoFert || s.Limits.FertilitySuppressed
	f := map[string]any{
		"data_quality":           s.DataQuality.String(),
		"confidence":             s.Confidence.String(),
		"fertility_suppressed":   strip,
		"suggest_pregnancy_test": s.SuggestPregnancyTest && !modeNoFert,
	}
	main := s.MainPhase
	if strip && main == engine.MainPhaseFertile {
		main = engine.MainPhaseFollicular
	}
	f["main_phase"] = main.String()
	if late, ok := s.DaysLate.Get(); ok {
		f["days_late"] = float64(late)
	} else if d.HasCycleDay() {
		f["days_late"] = 0.0
	}
	if n, ok := d.DaysToPeriod(); ok {
		f["days_to_period"] = float64(max(n, 0))
	}
	cd, ok := s.CycleDay.Get()
	if !ok {
		return f
	}
	f["cycle_day"] = float64(cd)
	f["cycle_length"] = float64(s.Metrics.EffectiveCycleLength)
	f["is_pms_window"] = d.PMS()
	if p := d.ContentPhase(); p != "" {
		if strip {
			p = strippedPhase(p)
		}
		f["phase"] = string(p)
	}
	sub := enums.CycleSubphase(s.Subphase.String())
	if fertileWorded := slices.Contains(fertileSubphases, sub); s.Subphase != engine.SubphaseUnknown && (!strip || !fertileWorded) {
		f["subphase"] = string(sub)
	}
	if !strip {
		f["is_fertile_window"] = d.Fertile()
	}
	return f
}
