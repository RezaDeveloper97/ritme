package snapshots

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// Hook is the periods.StartHook that queues the prediction-log job after a real period start commits. The task id
// is the user and the revision after the write, so a repeated dispatch of one write is dropped. A failed dispatch is
// logged (no data) and never fails the period write.
func Hook(q *queue.Queue, logger *slog.Logger) periods.StartHook {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, ev periods.StartEvent) {
		id := JobPrediction + ":" + strconv.FormatUint(ev.UserID, 10) + ":" + strconv.FormatUint(ev.RevAfter, 10)
		err := q.DispatchWith(ctx, JobPrediction, ev, queue.TaskID(id))
		if err != nil && !errors.Is(err, queue.ErrDuplicate) {
			logger.WarnContext(ctx, "snapshots: prediction log dispatch failed", slog.String("job", JobPrediction))
		}
	}
}

// PredictionJob is the engine:prediction_log handler; payload = periods.StartEvent.
func (s *Service) PredictionJob(now func() time.Time) queue.Handler {
	return func(ctx context.Context, payload []byte) error {
		var ev periods.StartEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			return fmt.Errorf("snapshots: prediction payload: %w", err)
		}
		return s.RecordPrediction(ctx, ev, now())
	}
}

// Prediction is one engine_prediction_log row (without user and created_at).
type Prediction struct {
	CycleStart      civildate.Date
	PredictedNext   civildate.Date
	BasisCycles     uint8
	OvulationSource string
	Confidence      string
}

// Outcome is what one real start writes: the last prediction of the previous cycle (then resolved with Actual) and
// the first prediction of the new cycle. Nil = nothing to write.
type Outcome struct {
	Last   *Prediction
	Actual civildate.Date // the new start
	First  *Prediction
}

// RecordPrediction writes the rows of the real start ev.PeriodID. Idempotent: `first` is insert-once, `last` is
// upserted with the same values, actual/error are recomputed equal. Nothing is written when the period is gone, is not
// a real period (estimate, onboarding seed) or is not the newest real start (an edit or a backfill of an old period).
// The revisions in ev only identify the write: the state before it is the current input without the new start (the
// pre-write snapshot is not kept), which equals the pre-write input for every period write.
func (s *Service) RecordPrediction(ctx context.Context, ev periods.StartEvent, now time.Time) error {
	row, err := s.q.GetUserPeriod(ctx, store.GetUserPeriodParams{UserID: ev.UserID, ID: ev.PeriodID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("snapshots: period: %w", err)
	}
	if row.IsEstimated || row.Source == string(enums.DataSourceUserProfileConfirmed) {
		return nil
	}
	in, err := inputs.Load(ctx, s.q, ev.UserID, row.PeriodStartDate)
	if err != nil {
		return fmt.Errorf("snapshots: %w", err)
	}
	out := PredictAround(in, row.PeriodStartDate)
	if out.Last == nil && out.First == nil {
		return nil
	}

	created := sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("snapshots: begin: %w", err)
	}
	q := s.q.WithTx(tx)
	if err := writeOutcome(ctx, q, ev.UserID, out, created); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("snapshots: commit: %w", err)
	}
	return nil
}

func writeOutcome(ctx context.Context, q *store.Queries, uid uint64, out Outcome, created sql.NullTime) error {
	if p := out.Last; p != nil {
		if err := q.UpsertLastPrediction(ctx, store.UpsertLastPredictionParams{
			UserID: uid, CycleStart: p.CycleStart, PredictedNext: p.PredictedNext, BasisCycles: p.BasisCycles,
			OvulationSource: p.OvulationSource, Confidence: p.Confidence, CreatedAt: created,
		}); err != nil {
			return fmt.Errorf("snapshots: last prediction: %w", err)
		}
		if _, err := q.ResolvePredictionCycle(ctx, store.ResolvePredictionCycleParams{
			ActualStart: civildate.NullDate{Date: out.Actual, Valid: true}, UserID: uid, CycleStart: p.CycleStart,
		}); err != nil {
			return fmt.Errorf("snapshots: resolve cycle: %w", err)
		}
	}
	if p := out.First; p != nil {
		if err := q.InsertFirstPrediction(ctx, store.InsertFirstPredictionParams{
			UserID: uid, CycleStart: p.CycleStart, PredictedNext: p.PredictedNext, BasisCycles: p.BasisCycles,
			OvulationSource: p.OvulationSource, Confidence: p.Confidence, CreatedAt: created,
		}); err != nil {
			return fmt.Errorf("snapshots: first prediction: %w", err)
		}
	}
	return nil
}

// PredictAround is the pure part of RecordPrediction. in is the user's input after the write (loaded with today =
// start); start must be its newest real period start, else the outcome is empty.
//
//   - Last: the input without the new start, resolved on start − 1 (A-05: the last prediction before the next start),
//     when that state is anchored on a real logged start (the declared onboarding anchor and roll-forward references
//     are never a cycle_start) and predicts a next start.
//   - First: the full input resolved on start, when it is anchored on start and predicts a next start.
//
// A skipped predicted start is not a period, so it never resolves a cycle.
func PredictAround(in engine.Input, start civildate.Date) Outcome {
	d := engine.DayOf(start)
	if d == engine.NoDay || len(in.Periods) == 0 {
		return Outcome{}
	}
	newest := engine.NoDay
	found := false
	for _, p := range in.Periods {
		newest = max(newest, p.Start)
		found = found || p.Start == d
	}
	if !found || newest != d {
		return Outcome{}
	}
	out := Outcome{Actual: start}

	before := withoutStart(in, d)
	bp := engineparams.For(start.AddDays(-1))
	if p, ok := prediction(engine.Prepare(before, bp), d-1); ok && p.CycleStart.Before(start) {
		out.Last = &p
	}
	if p, ok := prediction(engine.Prepare(in, engineparams.For(start)), d); ok && p.CycleStart == start {
		out.First = &p
	}
	return out
}

// prediction resolves st on day and returns the next-start prediction of a cycle anchored on a real logged start.
func prediction(st *engine.State, day engine.Day) (Prediction, bool) {
	s := st.Resolve(day, day)
	a := s.Anchors
	if a.CurrentPeriodStartSource != engine.StartUserLogged || !a.CurrentPeriodStart.IsSet() ||
		!a.PredictedNextPeriodStart.IsSet() || s.Limits.EngineSuspended {
		return Prediction{}, false
	}
	return Prediction{
		CycleStart:      a.CurrentPeriodStart.Civil(),
		PredictedNext:   a.PredictedNextPeriodStart.Civil(),
		BasisCycles:     uint8(max(0, min(st.ValidCycleCount(), 255))), //nolint:gosec // clamped to 0..255
		OvulationSource: a.OvulationSource.String(),
		Confidence:      s.Confidence.String(),
	}, true
}

// withoutStart is in without the period starting on d and without markers after d (the day before d sees at most d,
// the loader's today + 1 rule).
func withoutStart(in engine.Input, d engine.Day) engine.Input {
	out := in
	out.Periods = nil
	for _, p := range in.Periods {
		if p.Start != d {
			out.Periods = append(out.Periods, p)
		}
	}
	out.LH = nil
	for _, m := range in.LH {
		if m.Date <= d {
			out.LH = append(out.LH, m)
		}
	}
	out.BBT = nil
	for _, m := range in.BBT {
		if m.Date <= d {
			out.BBT = append(out.BBT, m)
		}
	}
	out.PregnancyTests = nil
	for _, m := range in.PregnancyTests {
		if m.Date <= d {
			out.PregnancyTests = append(out.PregnancyTests, m)
		}
	}
	return out
}
