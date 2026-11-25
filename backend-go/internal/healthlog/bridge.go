package healthlog

// The bleeding-log → period bridge, engine v6 (T-M9-28; SPEC v6 §1.4, §2.2, §2.9, §2.39.9; docs/engine-v6/DECISIONS.md
// I-02, I-05, H-14; deviations D-124, D-138 — supersedes D-55 and Laravel's CycleHistoryService::
// checkAndUpdatePeriodStart).
//
// A day's `flow` (daily_health_logs.bleeding_intensity) is actual period data:
//   - Adding flow on a day: a day inside a bleeding_log period, or up to adjacentDays after its end, extends it (and
//     merges it with the next bleeding_log period when they now touch); the previous day's flow continues a period;
//     a day up to adjacentDays before the next period moves that period's start back; otherwise a new confirmed
//     bleeding_log row starts — closed on the day when a later period exists (back-dated), else open, and an open
//     previous bleeding_log period is then closed at its last bleeding day. The end rule is unchanged: the newest
//     period stays open until the user confirms an end or the next period starts; no log is never "bleeding stopped".
//   - Removing flow (clearing it, deleting the log or the day): the bleeding_log period around the day is rebuilt
//     from the remaining bleeding days with the same adjacency (shrink, split, or delete when none is left).
//   - Rows of the period button (any source but bleeding_log) are never edited, moved, closed or deleted; a flow day
//     they cover is just a log.
//   - Every write recomputes the stored cycle_length of every row and re-anchors the profile LMP
//     (periods.RecomputeCycleLengths / periods.Reanchor) in the same transaction, under the per-user lock
//     (periods.LockUser, D-131) the period button takes too.
//   - Mode guard: in effective mode pregnancy, flow never touches cycle_histories or the LMP; a new or heavier flow
//     is the day's bleeding symptom and raises the preg.critical_symptom alert of the message engine (T-M10-08b).
//     In postpartum mode, flow before birth + postpartum.flow_from_day (42) is lochia and builds nothing; from that
//     day the bridge works as in cycle mode.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/cycle/periods"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
	pregalerts "github.com/ritme/backend-go/internal/pregnancy/alerts"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
	pregv2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Bridge bounds (unchanged from D-55).
const (
	// adjacentDays: a bleeding day at most this many days after a period's end (or before its start) belongs to
	// that period — one day without a log in between does not split it. Two bleeding days further apart than this
	// are two periods (the rebuild splits there).
	adjacentDays = 2
	// openPeriodMaxDays: an open period (no end, no bleeding length) still covers a bleeding day this many days
	// after its start, minus one (the cycle engine's hard cap of a missing period end).
	openPeriodMaxDays = 12
)

// sourceBleedingLog is cycle_histories.source of a bridge row (D-138).
var sourceBleedingLog = string(enums.DataSourceBleedingLog)

// bleedingChange is what one write did to one day's flow ("" = none).
type bleedingChange struct {
	date          civildate.Date
	before, after string
}

func isTrue(p *bool) bool { return p != nil && *p }

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// lockUser takes the per-user period lock on the Service's connection (the write transaction).
func (s *Service) lockUser(ctx context.Context, userID uint64) error {
	return periods.LockUser(ctx, s.raw, userID)
}

// postpartumFlowFrom is the first day a postpartum user's flow counts as a period (birth +
// postpartum.flow_from_day of the version valid on day); ok false without a postpartum profile.
func (s *Service) postpartumFlowFrom(ctx context.Context, userID uint64, day civildate.Date) (civildate.Date, bool, error) {
	birth, err := s.q.GetLogPostpartumBirth(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return civildate.Date{}, false, nil
	}
	if err != nil {
		return civildate.Date{}, false, fmt.Errorf("healthlog: postpartum birth: %w", err)
	}
	return birth.AddDays(clinicalconfig.Int(clinicalconfig.KeyPostpartumFlowFromDay, "day", day)), true, nil
}

// FlowGates are the taxonomy gates of a day for a user in mode: postpartum `flow` opens on and after birth +
// postpartum.flow_from_day (T-M9-28, I-05).
func (s *Service) FlowGates(ctx context.Context, userID uint64, mode string, day civildate.Date) (taxonomy.Gates, error) {
	if mode != taxonomy.ModePostpartum {
		return nil, nil
	}
	from, ok, err := s.postpartumFlowFrom(ctx, userID, day)
	if err != nil || !ok || day.Before(from) {
		return nil, err
	}
	return taxonomy.Gates{taxonomy.GatePostpartumFlow: true}, nil
}

// bridge runs the bridge for one changed day; s is transactional and holds the user lock. It reports whether
// cycle_histories changed (the profile has then been re-anchored and marked recalculated).
func (s *Service) bridge(ctx context.Context, userID uint64, ch bleedingChange, locale string, now time.Time) (bool, error) {
	if ch.before == "" && ch.after == "" {
		return false, nil
	}
	mode, err := s.LifeMode(ctx, userID)
	if err != nil {
		return false, err
	}
	var flowFrom civildate.Date // zero: every day may hold a period
	switch mode {
	case taxonomy.ModePregnancy:
		if pregalerts.FlowRank(ch.after) > pregalerts.FlowRank(ch.before) {
			return false, s.raisePregnancyFlowAlert(ctx, userID, ch.date, ch.after, locale, now)
		}
		return false, nil
	case taxonomy.ModePostpartum:
		from, ok, err := s.postpartumFlowFrom(ctx, userID, ch.date)
		if err != nil {
			return false, err
		}
		if ok {
			if ch.date.Before(from) {
				return false, nil // lochia
			}
			flowFrom = from
		}
	}
	var changed bool
	if ch.after != "" {
		changed, err = s.addBleedingDay(ctx, userID, ch.date, flowFrom, now)
	} else {
		changed, err = s.removeBleedingDay(ctx, userID, ch.date, now)
	}
	if err != nil || !changed {
		return false, err
	}
	cq, pq := cyclestore.New(s.raw), profilestore.New(s.raw)
	if err := periods.RecomputeCycleLengths(ctx, cq, userID, now); err != nil {
		return false, err
	}
	if err := periods.Reanchor(ctx, cq, pq, userID, now); err != nil {
		return false, err
	}
	return true, nil
}

// addBleedingDay is the forward path for a flow day d (see the package comment of this file).
func (s *Service) addBleedingDay(ctx context.Context, userID uint64, d, flowFrom civildate.Date, now time.Time) (bool, error) {
	ts := sql.NullTime{Time: now, Valid: true}
	before, hasBefore, err := s.historyRow(s.q.GetCycleHistoryOnOrBefore(ctx, store.GetCycleHistoryOnOrBeforeParams{UserID: userID, OnDate: d}))
	if err != nil {
		return false, fmt.Errorf("healthlog: period before: %w", err)
	}
	if hasBefore {
		if before.PeriodStartDate == d {
			return false, nil // already recorded
		}
		if covered, end := periodCovers(before, d); covered {
			if before.Source != sourceBleedingLog || !end.Valid || !d.After(end.Date) {
				return false, nil // inside the period, or a button row (never edited)
			}
			return true, s.extendPeriod(ctx, userID, before, d, ts)
		}
	}

	if yesterday := d.AddDays(-1); flowFrom.IsZero() || !yesterday.Before(flowFrom) {
		row, err := s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: yesterday})
		switch {
		case err == nil:
			if row.BleedingIntensity.Valid && row.BleedingIntensity.String != "" {
				return false, nil // not a new period start (Laravel's yesterday rule)
			}
		case !errors.Is(err, sql.ErrNoRows):
			return false, fmt.Errorf("healthlog: yesterday's log: %w", err)
		}
	}

	next, hasNext, err := s.historyRow(s.q.GetCycleHistoryAfter(ctx, store.GetCycleHistoryAfterParams{UserID: userID, AfterDate: d}))
	if err != nil {
		return false, fmt.Errorf("healthlog: period after: %w", err)
	}
	if hasNext && d.DiffDays(next.PeriodStartDate) <= adjacentDays {
		if next.Source != sourceBleedingLog {
			return false, nil // right before a button row: the row is not moved
		}
		bleeding := next.BleedingLength
		if bleeding.Valid {
			bleeding.Int32 += int32(d.DiffDays(next.PeriodStartDate)) //nolint:gosec // G115: day counts fit int32
		}
		return true, s.writePeriod(ctx, userID, next, d, next.PeriodEndDate, bleeding, ts)
	}

	params := store.InsertCycleHistoryParams{UserID: userID, PeriodStartDate: d, Now: ts}
	if hasNext { // back-dated between two periods: a closed one-day period, later days extend it
		params.PeriodEndDate = civildate.NullDate{Date: d, Valid: true}
		params.BleedingLength = sql.NullInt32{Int32: 1, Valid: true}
	} else if hasBefore && before.Source == sourceBleedingLog && !before.PeriodEndDate.Valid && !before.BleedingLength.Valid {
		if err := s.closeAtLastBleeding(ctx, userID, before, d, ts); err != nil {
			return false, err
		}
	}
	id, err := s.q.InsertCycleHistory(ctx, params)
	if err != nil {
		return false, fmt.Errorf("healthlog: create period: %w", err)
	}
	// A new confirmed bleeding_log row is a real period start (the prediction log, T-M9-39b). Moving a start back,
	// extending, merging or a split on removal is an edit of existing periods, not a new start.
	s.markStart(uint64(id)) //nolint:gosec // G115: AUTO_INCREMENT ids are positive
	return true, nil
}

// extendPeriod moves bleeding_log period h's end forward to d, then merges the next period into it when that one is
// a bleeding_log period starting at most adjacentDays after d (the rebuild would join them too).
func (s *Service) extendPeriod(ctx context.Context, userID uint64, h store.CycleHistory, d civildate.Date, ts sql.NullTime) error {
	end := civildate.NullDate{Date: d, Valid: true}
	bleeding := sql.NullInt32{Int32: int32(h.PeriodStartDate.DiffDays(d) + 1), Valid: true} //nolint:gosec // G115: day counts fit int32
	next, hasNext, err := s.historyRow(s.q.GetCycleHistoryAfter(ctx, store.GetCycleHistoryAfterParams{UserID: userID, AfterDate: h.PeriodStartDate}))
	if err != nil {
		return fmt.Errorf("healthlog: period after: %w", err)
	}
	if hasNext && next.Source == sourceBleedingLog && d.DiffDays(next.PeriodStartDate) <= adjacentDays {
		end, bleeding = next.PeriodEndDate, sql.NullInt32{}
		if nextEnd, ok := periodEnd(next); ok {
			end = civildate.NullDate{Date: nextEnd, Valid: true}
			bleeding = sql.NullInt32{Int32: int32(h.PeriodStartDate.DiffDays(nextEnd) + 1), Valid: true} //nolint:gosec // G115
		}
		if err := s.q.DeleteBleedingLogPeriod(ctx, store.DeleteBleedingLogPeriodParams{ID: next.ID, UserID: userID}); err != nil {
			return fmt.Errorf("healthlog: merge periods: %w", err)
		}
	}
	return s.writePeriod(ctx, userID, h, h.PeriodStartDate, end, bleeding, ts)
}

// removeBleedingDay rebuilds the bleeding_log period around d after d lost its flow (see the file comment).
func (s *Service) removeBleedingDay(ctx context.Context, userID uint64, d civildate.Date, now time.Time) (bool, error) {
	ts := sql.NullTime{Time: now, Valid: true}
	p, ok, err := s.historyRow(s.q.GetCycleHistoryOnOrBefore(ctx, store.GetCycleHistoryOnOrBeforeParams{UserID: userID, OnDate: d}))
	if err != nil {
		return false, fmt.Errorf("healthlog: period before: %w", err)
	}
	if !ok || p.Source != sourceBleedingLog {
		return false, nil
	}
	end, closed := periodEnd(p)
	if !closed { // an open period spans its hard cap, up to the next period
		end = p.PeriodStartDate.AddDays(openPeriodMaxDays - 1)
		next, hasNext, err := s.historyRow(s.q.GetCycleHistoryAfter(ctx, store.GetCycleHistoryAfterParams{UserID: userID, AfterDate: p.PeriodStartDate}))
		if err != nil {
			return false, fmt.Errorf("healthlog: period after: %w", err)
		}
		if hasNext && !next.PeriodStartDate.After(end) {
			end = next.PeriodStartDate.AddDays(-1)
		}
	}
	if d.After(end) {
		return false, nil
	}
	days, err := s.q.ListBleedingDaysBetween(ctx, store.ListBleedingDaysBetweenParams{UserID: userID, FromDate: p.PeriodStartDate, ToDate: end})
	if err != nil {
		return false, fmt.Errorf("healthlog: bleeding days: %w", err)
	}
	if len(days) == 0 {
		if err := s.q.DeleteBleedingLogPeriod(ctx, store.DeleteBleedingLogPeriodParams{ID: p.ID, UserID: userID}); err != nil {
			return false, fmt.Errorf("healthlog: delete period: %w", err)
		}
		return true, nil
	}
	if !closed { // the open period keeps no end; only its start can move
		if days[0] == p.PeriodStartDate {
			return false, nil
		}
		return true, s.writePeriod(ctx, userID, p, days[0], civildate.NullDate{}, sql.NullInt32{}, ts)
	}
	segments := splitDays(days)
	changed := false
	for i, seg := range segments {
		first, last := seg[0], seg[len(seg)-1]
		segEnd := civildate.NullDate{Date: last, Valid: true}
		bleeding := sql.NullInt32{Int32: int32(first.DiffDays(last) + 1), Valid: true} //nolint:gosec // G115: day counts fit int32
		if i == 0 {
			if first == p.PeriodStartDate && p.PeriodEndDate == segEnd && p.BleedingLength == bleeding {
				continue
			}
			if err := s.writePeriod(ctx, userID, p, first, segEnd, bleeding, ts); err != nil {
				return false, err
			}
			changed = true
			continue
		}
		if _, err := s.q.InsertCycleHistory(ctx, store.InsertCycleHistoryParams{
			UserID: userID, PeriodStartDate: first, PeriodEndDate: segEnd, BleedingLength: bleeding, Now: ts,
		}); err != nil {
			return false, fmt.Errorf("healthlog: split period: %w", err)
		}
		changed = true
	}
	return changed, nil
}

// splitDays groups sorted bleeding days into periods: a gap of more than adjacentDays between two days splits them.
func splitDays(days []civildate.Date) [][]civildate.Date {
	var out [][]civildate.Date
	for i, d := range days {
		if i == 0 || days[i-1].DiffDays(d) > adjacentDays {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], d)
	}
	return out
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

// periodEnd is h's end: logged, else from its bleeding length; ok false for an open period.
func periodEnd(h store.CycleHistory) (civildate.Date, bool) {
	if h.PeriodEndDate.Valid {
		return h.PeriodEndDate.Date, true
	}
	if h.BleedingLength.Valid && h.BleedingLength.Int32 > 0 {
		return h.PeriodStartDate.AddDays(int(h.BleedingLength.Int32) - 1), true
	}
	return civildate.Date{}, false
}

// periodCovers reports whether a bleeding day on d (after h's start) belongs to period h, and h's end
// (invalid for an open period).
func periodCovers(h store.CycleHistory, d civildate.Date) (bool, civildate.NullDate) {
	if end, ok := periodEnd(h); ok {
		return end.DiffDays(d) <= adjacentDays, civildate.NullDate{Date: end, Valid: true}
	}
	return h.PeriodStartDate.DiffDays(d) < openPeriodMaxDays, civildate.NullDate{}
}

// writePeriod rewrites bleeding_log period h's range (the query refuses any other source).
func (s *Service) writePeriod(ctx context.Context, userID uint64, h store.CycleHistory, start civildate.Date, end civildate.NullDate, bleeding sql.NullInt32, ts sql.NullTime) error {
	if err := s.q.UpdateBleedingLogPeriod(ctx, store.UpdateBleedingLogPeriodParams{
		PeriodStartDate: start, PeriodEndDate: end, BleedingLength: bleeding, Now: ts, ID: h.ID, UserID: userID,
	}); err != nil {
		return fmt.Errorf("healthlog: update period: %w", err)
	}
	return nil
}

// closeAtLastBleeding closes the open bleeding_log period previous at its last bleeding day before newStart
// (bleeding_length = days + 1); nothing when it has no bleeding day.
func (s *Service) closeAtLastBleeding(ctx context.Context, userID uint64, previous store.CycleHistory, newStart civildate.Date, ts sql.NullTime) error {
	last, err := s.q.LastBleedingDateBetween(ctx, store.LastBleedingDateBetweenParams{
		UserID: userID, FromDate: previous.PeriodStartDate, BeforeDate: newStart,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("healthlog: last bleeding day: %w", err)
	}
	return s.writePeriod(ctx, userID, previous, previous.PeriodStartDate, civildate.NullDate{Date: last, Valid: true},
		sql.NullInt32{Int32: int32(previous.PeriodStartDate.DiffDays(last) + 1), Valid: true}, ts) //nolint:gosec // G115: day counts fit int32
}

// raisePregnancyFlowAlert turns a new or heavier flow logged in pregnancy into the day's bleeding symptom
// (pregnancy_symptom_logs: the day's row gets has_bleeding + the severity, its other symptoms kept; a heavier severity
// the user logged herself is kept) and has the pregnancy alert rules evaluated after the commit (T-M10-08b, D-165):
// bleeding is a critical symptom, so preg.critical_symptom opens (or re-raises) its urgent inbox row. The evaluation
// waits for the commit because the write holds the per-user lock the inbox insert's user check would wait on.
// Recording the symptom keeps the alert standing: every later evaluation (the feed, the pregnancy Today and alerts
// reads, the next save) sees the bleeding too — also when this Service runs on a caller's transaction and the
// evaluation is left to them.
func (s *Service) raisePregnancyFlowAlert(ctx context.Context, userID uint64, date civildate.Date, intensity, locale string,
	now time.Time,
) error {
	sev := pregalerts.FlowSeverity(intensity)
	if sev == "" {
		return nil
	}
	pq := pregstore.New(s.raw)
	cur, err := pq.GetSymptomLog(ctx, pregstore.GetSymptomLogParams{UserID: userID, LogDate: date})
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return fmt.Errorf("healthlog: pregnancy symptoms: %w", err)
	}
	logged := cur.HasBleeding.Valid && cur.HasBleeding.Bool
	if !logged || pregalerts.SeverityRank(cur.BleedingSeverity.String) < pregalerts.SeverityRank(sev) {
		if err := pq.RecordFlowBleedingSymptom(ctx, pregstore.RecordFlowBleedingSymptomParams{UserID: userID, LogDate: date,
			BleedingSeverity: sql.NullString{String: sev, Valid: true}, Now: sql.NullTime{Time: now, Valid: true}}); err != nil {
			return fmt.Errorf("healthlog: pregnancy bleeding symptom: %w", err)
		}
	}
	if s.pregAlerts != nil {
		s.pregAlerts.pending, s.pregAlerts.locale, s.pregAlerts.now = true, locale, now
	}
	return nil
}

// pregnancySpotting turns spotting newly logged in pregnancy (the log sheet's «لکه‌بینی», bleeding.spotting) into the
// day's spotting symptom and has the pregnancy alert rules evaluated after the commit, as raisePregnancyFlowAlert does
// for a flow: spotting is a critical symptom up to week 12 (preg.critical_symptom), so the urgent alert opens with
// the save instead of at the next Today read. Clearing the flag changes nothing (a symptom once logged stays logged,
// like a lighter flow).
func (s *Service) pregnancySpotting(ctx context.Context, userID uint64, date civildate.Date, before, after bool, locale string,
	now time.Time,
) error {
	if before || !after {
		return nil
	}
	mode, err := s.LifeMode(ctx, userID)
	if err != nil {
		return err
	}
	if mode != taxonomy.ModePregnancy {
		return nil
	}
	if err := pregstore.New(s.raw).RecordSpottingSymptom(ctx, pregstore.RecordSpottingSymptomParams{UserID: userID,
		LogDate: date, SpottingSeverity: sql.NullString{String: "mild", Valid: true},
		Now: sql.NullTime{Time: now, Valid: true}}); err != nil {
		return fmt.Errorf("healthlog: pregnancy spotting symptom: %w", err)
	}
	if s.pregAlerts != nil {
		s.pregAlerts.pending, s.pregAlerts.locale, s.pregAlerts.now = true, locale, now
	}
	return nil
}

// pregnancyEval is a pending evaluation of the pregnancy alert rules (after the write's commit).
type pregnancyEval struct {
	pending bool
	locale  string
	now     time.Time
}

// evaluatePregnancyAlerts evaluates the user's pregnancy alert rules on the pool (after a commit). The log is committed,
// so a failure is logged, not returned (as the v1 pregnancy saves do, D-21): the recorded symptom raises the alert at
// the next evaluation.
func (s *Service) evaluatePregnancyAlerts(ctx context.Context, userID uint64, locale string, now time.Time) {
	pq := pregstore.New(s.raw)
	_, err := pregnancyalerts.New(pq).EvaluateOn(ctx, pq, userID, now, pregv2.Lang{Locale: locale, Default: locale})
	if err != nil && !errors.Is(err, pregnancyalerts.ErrNotActive) {
		slog.WarnContext(ctx, "healthlog: pregnancy alert evaluation failed after a flow log",
			slog.Uint64("user_id", userID), slog.String("error", err.Error()))
	}
}
