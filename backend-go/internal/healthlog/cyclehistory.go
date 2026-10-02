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

// Period reconciliation bounds (B-N3-14b, deviations.md D-55).
const (
	// adjacentDays: a bleeding day at most this many days after a period's end (or before its start)
	// belongs to that period — one day without a log in between does not split it.
	adjacentDays = 2
	// openPeriodMaxDays: an open period (no end, no bleeding length) still covers a bleeding day this
	// many days after its start, minus one (the cycle engine's hard cap of a missing period end).
	openPeriodMaxDays = 12
)

// checkAndUpdatePeriodStart is CycleHistoryService::checkAndUpdatePeriodStart, fixed for back-dated
// logs (B-N3-14b, deviations.md D-55). A bleeding day
//   - inside a period, or up to adjacentDays after its logged end, extends that period (never a new one);
//   - whose previous day had bleeding continues a period (Laravel);
//   - up to adjacentDays before the next period's start moves that start back;
//   - otherwise starts a new (unconfirmed) cycle_histories row: its cycle_length counts from the period
//     before the day (never negative), an open previous period is closed at its last bleeding day (a
//     logged end is never cut), and the profile LMP moves only when the new row is the latest one — a
//     back-dated row between two periods is closed on the day instead of left open.
//
// profile is updated in memory too ($user->profile is the same cached instance markRecalculatedIfNeeded
// reads).
func (s *Service) checkAndUpdatePeriodStart(ctx context.Context, userID uint64, l *model.DailyHealthLog, profile *store.UserProfile, now time.Time) error {
	if b := l.BleedingIntensity(); b == nil || *b == "" {
		return nil
	}
	logDate := l.Row.LogDate
	ts := sql.NullTime{Time: now, Valid: true}

	before, hasBefore, err := s.historyRow(s.q.GetCycleHistoryOnOrBefore(ctx, store.GetCycleHistoryOnOrBeforeParams{UserID: userID, OnDate: logDate}))
	if err != nil {
		return fmt.Errorf("healthlog: period before: %w", err)
	}
	if hasBefore {
		if before.PeriodStartDate == logDate {
			return nil // already recorded
		}
		if covered, end := periodCovers(before, logDate); covered {
			if end.Valid && logDate.After(end.Date) {
				return s.setPeriodEnd(ctx, before, logDate, ts)
			}
			return nil
		}
	}

	yesterday, err := s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: logDate.AddDays(-1)})
	switch {
	case err == nil:
		if b := model.FromRow(yesterday).BleedingIntensity(); b != nil && *b != "" {
			return nil // not a new period start
		}
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("healthlog: yesterday's log: %w", err)
	}

	next, hasNext, err := s.historyRow(s.q.GetCycleHistoryAfter(ctx, store.GetCycleHistoryAfterParams{UserID: userID, AfterDate: logDate}))
	if err != nil {
		return fmt.Errorf("healthlog: period after: %w", err)
	}
	if hasNext && logDate.DiffDays(next.PeriodStartDate) <= adjacentDays {
		return s.moveStartBack(ctx, userID, next, logDate, profile, ts)
	}

	var cycleLength sql.NullInt32
	if hasBefore {
		cycleLength = sql.NullInt32{Int32: int32(before.PeriodStartDate.DiffDays(logDate)), Valid: true} //nolint:gosec // G115: day counts fit int32
		if !before.PeriodEndDate.Valid {
			if err := s.updatePreviousPeriodEndDate(ctx, userID, before, logDate, ts); err != nil {
				return err
			}
		}
	}
	id, err := s.q.InsertCycleHistory(ctx, store.InsertCycleHistoryParams{
		UserID: userID, PeriodStartDate: logDate, CycleLength: cycleLength, CreatedAt: ts, UpdatedAt: ts,
	})
	if err != nil {
		return fmt.Errorf("healthlog: create period: %w", err)
	}
	if hasNext { // back-dated between two periods: a closed one-day period, later days extend it
		if err := s.q.UpdateCycleHistoryEnd(ctx, store.UpdateCycleHistoryEndParams{
			PeriodEndDate: civildate.NullDate{Date: logDate, Valid: true}, BleedingLength: sql.NullInt32{Int32: 1, Valid: true},
			UpdatedAt: ts, ID: uint64(id), //nolint:gosec // G115: auto-increment ids are positive
		}); err != nil {
			return fmt.Errorf("healthlog: close back-dated period: %w", err)
		}
		return nil
	}
	return s.updateProfileLMP(ctx, profile, logDate, ts)
}

// historyRow folds sql.ErrNoRows into ok = false.
func (s *Service) historyRow(row store.CycleHistory, err error) (store.CycleHistory, bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return store.CycleHistory{}, false, nil
	}
	if err != nil {
		return store.CycleHistory{}, false, err
	}
	return row, true, nil
}

// periodCovers reports whether a bleeding day on d (after h's start) belongs to period h, and h's end
// (logged, else from its bleeding length; invalid for an open period).
func periodCovers(h store.CycleHistory, d civildate.Date) (bool, civildate.NullDate) {
	end := h.PeriodEndDate
	if !end.Valid && h.BleedingLength.Valid && h.BleedingLength.Int32 > 0 {
		end = civildate.NullDate{Date: h.PeriodStartDate.AddDays(int(h.BleedingLength.Int32) - 1), Valid: true}
	}
	if end.Valid {
		return end.Date.DiffDays(d) <= adjacentDays, end
	}
	return h.PeriodStartDate.DiffDays(d) < openPeriodMaxDays, end
}

// setPeriodEnd extends period h to end on d.
func (s *Service) setPeriodEnd(ctx context.Context, h store.CycleHistory, d civildate.Date, ts sql.NullTime) error {
	if err := s.q.UpdateCycleHistoryEnd(ctx, store.UpdateCycleHistoryEndParams{
		PeriodEndDate:  civildate.NullDate{Date: d, Valid: true},
		BleedingLength: sql.NullInt32{Int32: int32(h.PeriodStartDate.DiffDays(d) + 1), Valid: true}, //nolint:gosec // G115: day counts fit int32
		UpdatedAt:      ts,
		ID:             h.ID,
	}); err != nil {
		return fmt.Errorf("healthlog: extend period: %w", err)
	}
	return nil
}

// moveStartBack moves period h's start back to d (a bleeding day right before it); the profile LMP
// follows when h is the latest period.
func (s *Service) moveStartBack(ctx context.Context, userID uint64, h store.CycleHistory, d civildate.Date, profile *store.UserProfile, ts sql.NullTime) error {
	bleeding := h.BleedingLength
	if bleeding.Valid {
		bleeding.Int32 += int32(d.DiffDays(h.PeriodStartDate)) //nolint:gosec // G115: day counts fit int32
	}
	if err := s.q.UpdateCycleHistoryStart(ctx, store.UpdateCycleHistoryStartParams{
		PeriodStartDate: d, BleedingLength: bleeding, UpdatedAt: ts, ID: h.ID,
	}); err != nil {
		return fmt.Errorf("healthlog: move period start: %w", err)
	}
	latest, err := s.q.GetLatestCycleHistory(ctx, userID)
	if err != nil {
		return fmt.Errorf("healthlog: latest period: %w", err)
	}
	if latest.ID != h.ID {
		return nil
	}
	return s.updateProfileLMP(ctx, profile, d, ts)
}

// updateProfileLMP is CycleHistoryService::updateProfileLMP: $profile->update(['last_period_start' =>
// $logDate]) (no-op when equal or without a profile).
func (s *Service) updateProfileLMP(ctx context.Context, profile *store.UserProfile, d civildate.Date, ts sql.NullTime) error {
	if profile == nil || (profile.LastPeriodStart.Valid && profile.LastPeriodStart.Date == d) {
		return nil
	}
	if err := s.q.UpdateProfileLastPeriodStart(ctx, store.UpdateProfileLastPeriodStartParams{
		LastPeriodStart: civildate.NullDate{Date: d, Valid: true}, UpdatedAt: ts, ID: profile.ID,
	}); err != nil {
		return fmt.Errorf("healthlog: update LMP: %w", err)
	}
	profile.LastPeriodStart = civildate.NullDate{Date: d, Valid: true}
	profile.UpdatedAt = ts
	return nil
}

// updatePreviousPeriodEndDate closes the previous (open) period at its last bleeding day before
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
