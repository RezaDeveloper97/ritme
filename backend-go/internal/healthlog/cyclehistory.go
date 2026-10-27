package healthlog

// CycleHistoryService (backend/app/Services/HealthEngine/CycleHistoryService.php).
//
// Live code (T-M2-12 finding): DailyHealthLogController::store is its only caller, and it
// uses every method — getSpottingWarning (→ isLikelyLutealSpotting) and
// checkAndUpdatePeriodStart (→ updatePreviousPeriodEndDate, updateProfileLMP). The spotting warning is ported
// here; the period reconciliation is the engine v6 bleeding-log bridge in bridge.go (T-M9-28, D-124).

import (
	"context"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// CycleStates hands out a user's prepared v6 engine state for the user-local day (state.Source satisfies it). nil
// means CYCLE_ENGINE=v1: the spotting warning keeps the profile-LMP rule.
type CycleStates interface {
	ForUser(ctx context.Context, uid uint64, today civildate.Date) (*engine.State, error)
}

// WithEngine makes the luteal-spotting warning read the v6 engine phase (CYCLE_ENGINE shadow / v6, T-M9-37, D-136);
// nil keeps v1. Services bound to a transaction by this one inherit it.
func (s *Service) WithEngine(states CycleStates) *Service {
	s.cycle = states
	return s
}

// lutealSpottingMinDaysToPeriod: the v6 rule keeps v1's margin before the period — v1 stopped at cycle day
// cycle_duration − 2 (the 3rd day before the next start), so a bleed in the last two days before the predicted
// start is a likely period start, never "luteal spotting".
const lutealSpottingMinDaysToPeriod = 3

// IsLikelyLutealSpottingV6 is the v6 rule: spotting or low bleeding on a day the engine resolves as luteal (main
// phase luteal: post-ovulation through pms_possible) at least lutealSpottingMinDaysToPeriod days before the predicted
// next start. Unknown, suppressed (pill, pregnancy, postpartum), late or follicular days never warn.
func IsLikelyLutealSpottingV6(st *engine.State, l *model.DailyHealthLog, today civildate.Date) bool {
	if !spottingOrLow(l) || st == nil {
		return false
	}
	s := st.Resolve(engine.DayOf(l.Row.LogDate), engine.DayOf(today))
	if s.MainPhase != engine.MainPhaseLuteal {
		return false
	}
	d, ok := s.DaysToPeriod.Get()
	return ok && d >= lutealSpottingMinDaysToPeriod
}

// spottingWarning is SpottingWarning, with the engine's phase under CYCLE_ENGINE shadow / v6 (the state of the
// revision before this write: the day's own bleeding is not a period yet).
func (s *Service) spottingWarning(ctx context.Context, userID uint64, profile *store.UserProfile, l *model.DailyHealthLog,
	locale string, now time.Time,
) (*jsonx.OrderedMap, error) {
	if s.cycle == nil {
		return SpottingWarning(profile, l, locale), nil
	}
	if !spottingOrLow(l) {
		return nil, nil
	}
	today := civildate.FromTime(now.In(civildate.Zone(ctx)))
	st, err := s.cycle.ForUser(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("healthlog: engine state: %w", err)
	}
	if !IsLikelyLutealSpottingV6(st, l, today) {
		return nil, nil
	}
	return spottingJSON(locale), nil
}

func spottingOrLow(l *model.DailyHealthLog) bool {
	spotting := l.Spotting()
	bleeding := l.BleedingIntensity()
	return (spotting != nil && *spotting) || (bleeding != nil && *bleeding == "low")
}

// spottingMessages are getSpottingWarning's hard-coded texts ($locale === 'fa' ? fa : en).
const (
	spottingMessageFa = "این خون\u200cریزی ممکن است لکه\u200cبینی لوتئال باشد نه شروع پریود. آیا مطمئن هستید؟"
	spottingMessageEn = "This bleeding might be luteal spotting, not the start of your period. Are you sure?"
)

// IsLikelyLutealSpotting is CycleHistoryService::isLikelyLutealSpotting: spotting or low
// bleeding on cycle day 15 … cycle_duration−2 (default 28) counted from the profile LMP.
func IsLikelyLutealSpotting(profile *store.UserProfile, l *model.DailyHealthLog) bool {
	if !spottingOrLow(l) {
		return false
	}
	if profile == nil || !profile.LastPeriodStart.Valid {
		return false
	}
	cycleDay := profile.LastPeriodStart.Date.DiffDays(l.Row.LogDate) + 1
	cycleLength := 28
	if profile.CycleDuration.Valid {
		cycleLength = int(profile.CycleDuration.Int16)
	}
	return cycleDay >= 15 && cycleDay <= cycleLength-2
}

// SpottingWarning is CycleHistoryService::getSpottingWarning: the top-level `warning` of
// POST /health-logs, or nil.
func SpottingWarning(profile *store.UserProfile, l *model.DailyHealthLog, locale string) *jsonx.OrderedMap {
	if !IsLikelyLutealSpotting(profile, l) {
		return nil
	}
	return spottingJSON(locale)
}

func spottingJSON(locale string) *jsonx.OrderedMap {
	msg := spottingMessageEn
	if locale == "fa" {
		msg = spottingMessageFa
	}
	return jsonx.Obj("type", "luteal_spotting_warning", "message", msg)
}
