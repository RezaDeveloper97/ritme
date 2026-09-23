package healthlog

// CycleHistoryService (backend/app/Services/HealthEngine/CycleHistoryService.php).
//
// Live code (T-M2-12 finding): DailyHealthLogController::store is its only caller, and it
// uses every method — getSpottingWarning (→ isLikelyLutealSpotting) and
// checkAndUpdatePeriodStart (→ updatePreviousPeriodEndDate, updateProfileLMP). Nothing else
// in the app references the class, so the whole service is ported here and nowhere else.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
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

// checkAndUpdatePeriodStart is CycleHistoryService::checkAndUpdatePeriodStart: bleeding on a
// day whose previous day had none starts a new (unconfirmed) cycle_histories row, closes the
// previous period at its last bleeding day and moves the profile LMP. profile is updated in
// memory too ($user->profile is the same cached instance markRecalculatedIfNeeded reads).
func (s *Service) checkAndUpdatePeriodStart(ctx context.Context, userID uint64, l *model.DailyHealthLog, profile *store.UserProfile, now time.Time) error {
	if b := l.BleedingIntensity(); b == nil || *b == "" {
		return nil
	}
	logDate := l.Row.LogDate

	yesterday, err := s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: logDate.AddDays(-1)})
	switch {
	case err == nil:
		if b := model.FromRow(yesterday).BleedingIntensity(); b != nil && *b != "" {
			return nil // not a new period start
		}
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("healthlog: yesterday's log: %w", err)
	}

	if _, err := s.q.GetCycleHistoryByStart(ctx, store.GetCycleHistoryByStartParams{UserID: userID, PeriodStartDate: logDate}); err == nil {
		return nil // already recorded
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("healthlog: existing period: %w", err)
	}

	ts := sql.NullTime{Time: now, Valid: true}
	var cycleLength sql.NullInt32
	previous, err := s.q.GetLatestCycleHistory(ctx, userID)
	switch {
	case err == nil:
		// Carbon diffInDays is signed: a back-dated log before the latest period gives a
		// negative length, stored as is.
		cycleLength = sql.NullInt32{Int32: int32(previous.PeriodStartDate.DiffDays(logDate)), Valid: true} //nolint:gosec // G115: day counts fit int32
		if err := s.updatePreviousPeriodEndDate(ctx, userID, previous, logDate, ts); err != nil {
			return err
		}
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("healthlog: previous period: %w", err)
	}

	if _, err := s.q.InsertCycleHistory(ctx, store.InsertCycleHistoryParams{
		UserID: userID, PeriodStartDate: logDate, CycleLength: cycleLength, CreatedAt: ts, UpdatedAt: ts,
	}); err != nil {
		return fmt.Errorf("healthlog: create period: %w", err)
	}

	// updateProfileLMP: $profile->update(['last_period_start' => $logDate]) (no-op when equal).
	if profile != nil && (!profile.LastPeriodStart.Valid || profile.LastPeriodStart.Date != logDate) {
		if err := s.q.UpdateProfileLastPeriodStart(ctx, store.UpdateProfileLastPeriodStartParams{
			LastPeriodStart: civildate.NullDate{Date: logDate, Valid: true}, UpdatedAt: ts, ID: profile.ID,
		}); err != nil {
			return fmt.Errorf("healthlog: update LMP: %w", err)
		}
		profile.LastPeriodStart = civildate.NullDate{Date: logDate, Valid: true}
		profile.UpdatedAt = ts
	}
	return nil
}

// updatePreviousPeriodEndDate closes the previous period at its last bleeding day before
// the new start (bleeding_length = days + 1); saved only when a value changes.
func (s *Service) updatePreviousPeriodEndDate(ctx context.Context, userID uint64, previous store.CycleHistory, newStart civildate.Date, ts sql.NullTime) error {
	last, err := s.q.LastBleedingDateBetween(ctx, store.LastBleedingDateBetweenParams{
		UserID: userID, FromDate: previous.PeriodStartDate, BeforeDate: newStart,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("healthlog: last bleeding day: %w", err)
	}
	length := int32(previous.PeriodStartDate.DiffDays(last) + 1) //nolint:gosec // G115: day counts fit int32
	if previous.PeriodEndDate.Valid && previous.PeriodEndDate.Date == last &&
		previous.BleedingLength.Valid && previous.BleedingLength.Int32 == length {
		return nil
	}
	if err := s.q.UpdateCycleHistoryEnd(ctx, store.UpdateCycleHistoryEndParams{
		PeriodEndDate:  civildate.NullDate{Date: last, Valid: true},
		BleedingLength: sql.NullInt32{Int32: length, Valid: true},
		UpdatedAt:      ts,
		ID:             previous.ID,
	}); err != nil {
		return fmt.Errorf("healthlog: close previous period: %w", err)
	}
	return nil
}
