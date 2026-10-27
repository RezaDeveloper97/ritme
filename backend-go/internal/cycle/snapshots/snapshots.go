// Package snapshots keeps the engine's output over time for the SPEC v6 §5.6 KPIs and §5.2 user monitoring (T-M9-39,
// D-137; docs/engine-v6/DECISIONS.md A-05, A-06; gap 06 §4.1, §4.2). Go-only analytics tables (goose 00109):
//
//   - engine_daily_snapshots: the nightly `engine:snapshot_nightly` job resolves every active user (A-06: any session,
//     log or event in the last 30 days) on her own local day (stored zone, Tehran without one), in batches of
//     inputs.BatchSize through inputs.LoadMany + engine.Prepare + Resolve, and upserts one row of codes per user and
//     day. `engine:snapshot_purge` keeps 400 days.
//   - engine_prediction_log: after a real period start commits, periods.Service calls the hook (Hook), which queues
//     `engine:prediction_log` with the revisions around the write. The job records the `last` prediction of the
//     previous cycle (the state without the new start, resolved the day before it), fills actual_start / error_days on
//     that cycle's rows, and records the `first` prediction of the new cycle (resolved on its start day). Kept forever.
//
// Only codes and the engine's own next-start prediction are stored (§2.40: never a list of future cycles). Nothing in
// the engine reads these tables. Logs carry no health data: no dates, phases or user-level values.
package snapshots

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/platform/usertz"
)

// Jobs (docs/engine-v6/jobs.md). Cron specs are on the Tehran wall clock.
const (
	JobNightly    = "engine:snapshot_nightly"
	NightlyCron   = "30 3 * * *"
	JobPurge      = "engine:snapshot_purge"
	PurgeCron     = "50 3 * * *"
	JobPrediction = "engine:prediction_log"
)

const (
	// ActiveDays is the A-06 activity window.
	ActiveDays = 30
	// RetentionDays is how long a daily snapshot is kept.
	RetentionDays = 400
	// purgeChunk is PurgeEngineDailySnapshots' LIMIT.
	purgeChunk = 5000
)

// Service writes the snapshot and prediction-log tables.
type Service struct {
	db  *sql.DB
	q   *store.Queries
	log *slog.Logger
}

// New returns the service over db (logger nil = slog.Default()).
func New(db *sql.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, q: store.New(db), log: logger}
}

// RegisterJobs puts the nightly snapshot, the purge and the prediction-log job on q; now is the Tehran clock.
func (s *Service) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	if err := q.RegisterPeriodic(queue.Job{
		Type: JobNightly, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { return s.Nightly(ctx, now()) }, NightlyCron, queue.Unique(23*time.Hour)); err != nil {
		return err
	}
	if err := q.RegisterPeriodic(queue.Job{
		Type: JobPurge, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 10 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { return s.Purge(ctx, now()) }, PurgeCron, queue.Unique(23*time.Hour)); err != nil {
		return err
	}
	q.Register(queue.Job{
		Type: JobPrediction, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{time.Minute, 5 * time.Minute},
		Timeout: time.Minute,
	}, s.PredictionJob(now))
	return nil
}

// Nightly writes today's snapshot of every active user. Idempotent: one row per (user, user-local day), upserted.
func (s *Service) Nightly(ctx context.Context, now time.Time) error {
	tehranNow := now.In(civildate.Tehran)
	since := tehranNow.AddDate(0, 0, -ActiveDays)
	var after uint64
	for {
		page, err := s.q.ListActiveSnapshotUsers(ctx, store.ListActiveSnapshotUsersParams{
			AfterID: after, Since: sql.NullTime{Time: since, Valid: true}, SinceDay: civildate.FromTime(since),
			Limit: inputs.BatchSize,
		})
		if err != nil {
			return fmt.Errorf("snapshots: active users: %w", err)
		}
		if len(page) == 0 {
			return nil
		}
		if err := s.snapshotBatch(ctx, page, tehranNow); err != nil {
			return err
		}
		after = page[len(page)-1].UserID
		if len(page) < inputs.BatchSize {
			return nil
		}
	}
}

// snapshotBatch resolves one page of users (grouped by local day: LoadMany shares one today) and upserts their rows
// in one transaction.
func (s *Service) snapshotBatch(ctx context.Context, page []store.ListActiveSnapshotUsersRow, now time.Time) error {
	ids := make([]uint64, 0, len(page))
	for _, u := range page {
		ids = append(ids, u.UserID)
	}
	revRows, err := s.q.ListUserEngineRevs(ctx, ids)
	if err != nil {
		return fmt.Errorf("snapshots: revisions: %w", err)
	}
	revs := make(map[uint64]uint64, len(revRows))
	for _, r := range revRows {
		revs[r.UserID] = r.EngineRev
	}

	created := sql.NullTime{Time: now.Truncate(time.Second), Valid: true}
	var rows []store.UpsertEngineDailySnapshotParams
	for today, group := range groupByLocalDay(page, now) {
		ins, err := inputs.LoadMany(ctx, s.q, group, today)
		if err != nil {
			return fmt.Errorf("snapshots: %w", err)
		}
		p := engineparams.For(today)
		d := engine.DayOf(today)
		for _, uid := range group {
			st := engine.Prepare(ins[uid], p)
			row := SnapshotRow(st, st.Resolve(d, d))
			row.UserID, row.Day, row.EngineRev, row.CreatedAt = uid, today, revs[uid], created
			rows = append(rows, row)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("snapshots: begin: %w", err)
	}
	q := s.q.WithTx(tx)
	for _, r := range rows {
		if err := q.UpsertEngineDailySnapshot(ctx, r); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("snapshots: upsert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("snapshots: commit: %w", err)
	}
	return nil
}

// groupByLocalDay maps each user's local day (stored zone, Tehran when absent or invalid) to her ids.
func groupByLocalDay(page []store.ListActiveSnapshotUsersRow, now time.Time) map[civildate.Date][]uint64 {
	out := map[civildate.Date][]uint64{}
	for _, u := range page {
		loc := civildate.Tehran
		if u.Timezone.Valid {
			if l, ok := usertz.Parse(u.Timezone.String); ok {
				loc = l
			}
		}
		day := civildate.FromTime(now.In(loc))
		out[day] = append(out[day], u.UserID)
	}
	return out
}

// SnapshotRow maps a resolved status to the code columns of a snapshot row (user, day, revision and created_at are
// the caller's).
func SnapshotRow(st *engine.State, s engine.Status) store.UpsertEngineDailySnapshotParams {
	row := store.UpsertEngineDailySnapshotParams{
		MainPhase:    s.MainPhase.String(),
		Subphase:     s.Subphase.String(),
		Confidence:   s.Confidence.String(),
		DataQuality:  s.DataQuality.String(),
		WarningsMask: uint32(s.Warnings),
		Mode:         st.Mode().String(),
	}
	if late, ok := s.DaysLate.Get(); ok {
		row.DaysLate = sql.NullInt16{Int16: int16(max(math.MinInt16, min(math.MaxInt16, late))), Valid: true}
	}
	return row
}

// Purge deletes the snapshots older than RetentionDays (Tehran day), chunk by chunk. Idempotent.
func (s *Service) Purge(ctx context.Context, now time.Time) error {
	before := civildate.InTehran(now).AddDays(-RetentionDays)
	for {
		n, err := s.q.PurgeEngineDailySnapshots(ctx, before)
		if err != nil {
			return fmt.Errorf("snapshots: purge: %w", err)
		}
		if n < purgeChunk {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(errors.New("snapshots: purge interrupted"), err)
		}
	}
}
