package healthlog

// CycleHistoryService (backend/app/Services/HealthEngine/CycleHistoryService.php).
//
// Live code (T-M2-12 finding): DailyHealthLogController::store is its only caller, and it
// uses every method — getSpottingWarning (→ isLikelyLutealSpotting) and
// checkAndUpdatePeriodStart (→ updatePreviousPeriodEndDate, updateProfileLMP). The spotting warning is ported
// here; the period reconciliation is the engine v6 bleeding-log bridge in bridge.go (T-M9-28, D-124).

import (
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// spottingMessages are getSpottingWarning's hard-coded texts ($locale === 'fa' ? fa : en).
const (
	spottingMessageFa = "این خون\u200cریزی ممکن است لکه\u200cبینی لوتئال باشد نه شروع پریود. آیا مطمئن هستید؟"
	spottingMessageEn = "This bleeding might be luteal spotting, not the start of your period. Are you sure?"
)

// IsLikelyLutealSpotting is CycleHistoryService::isLikelyLutealSpotting: spotting or low
// bleeding on cycle day 15 … cycle_duration−2 (default 28) counted from the profile LMP.
func IsLikelyLutealSpotting(profile *store.UserProfile, l *model.DailyHealthLog) bool {
	spotting := l.Spotting()
	bleeding := l.BleedingIntensity()
	if (spotting == nil || !*spotting) && (bleeding == nil || *bleeding != "low") {
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
	msg := spottingMessageEn
	if locale == "fa" {
		msg = spottingMessageFa
	}
	return jsonx.Obj("type", "luteal_spotting_warning", "message", msg)
}
