package events

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// Job types and schedules (docs/engine-v6/jobs.md).
const (
	JobRollup   = "messages.rollup"
	JobPrune    = "messages.prune"
	RollupCron  = "20 0 * * *"
	PruneCron   = "45 3 * * *"
	Retention   = 180 // days of raw rows (M-13)
	PruneBatch  = 5000
	pruneRounds = 2000 // ≤ 10 M rows per run; the next night continues
)

// JobQuerier is what the jobs run (store.Queries).
type JobQuerier interface {
	RollupMessageEvents(ctx context.Context, arg store.RollupMessageEventsParams) error
	BackfillMessageEventDaily(ctx context.Context, beforeAt sql.NullTime) error
	DeleteMessageEventsBefore(ctx context.Context, arg store.DeleteMessageEventsBeforeParams) (int64, error)
}

// Jobs are the rollup and retention jobs of message_events (M-13).
type Jobs struct{ q JobQuerier }

// NewJobs wires the jobs.
func NewJobs(q JobQuerier) *Jobs { return &Jobs{q: q} }

// Rollup counts yesterday's raw rows (Tehran day before now) into message_event_daily. Idempotent: the upsert
// overwrites each count with the same number.
func (j *Jobs) Rollup(ctx context.Context, now time.Time) error {
	today := civildate.InTehran(now)
	return j.RollupDay(ctx, today.AddDays(-1))
}

// RollupDay counts the raw rows of one Tehran day.
func (j *Jobs) RollupDay(ctx context.Context, day civildate.Date) error {
	err := j.q.RollupMessageEvents(ctx, store.RollupMessageEventsParams{
		FromAt: sql.NullTime{Time: day.TehranMidnight(), Valid: true},
		ToAt:   sql.NullTime{Time: day.AddDays(1).TehranMidnight(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("messages: rollup %s: %w", day, err)
	}
	return nil
}

// Prune deletes raw rows older than Retention days (Tehran midnight) in batches of PruneBatch, after adding to the
// rollup every key of those rows it does not hold yet, so the rollup survives (rows already rolled up stay as they
// are). Idempotent: a second run finds nothing to delete. It returns the rows deleted.
func (j *Jobs) Prune(ctx context.Context, now time.Time) (int64, error) {
	cutoff := sql.NullTime{Time: civildate.InTehran(now).AddDays(-Retention).TehranMidnight(), Valid: true}
	if err := j.q.BackfillMessageEventDaily(ctx, cutoff); err != nil {
		return 0, fmt.Errorf("messages: prune backfill: %w", err)
	}
	var total int64
	for range pruneRounds {
		n, err := j.q.DeleteMessageEventsBefore(ctx, store.DeleteMessageEventsBeforeParams{BeforeAt: cutoff, Limit: PruneBatch})
		if err != nil {
			return total, fmt.Errorf("messages: prune: %w", err)
		}
		total += n
		if n < PruneBatch {
			break
		}
	}
	return total, nil
}

// RegisterJobs schedules both jobs on queue low, unique per day (T-M9-11).
func (j *Jobs) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	if err := q.RegisterPeriodic(queue.Job{
		Type: JobRollup, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{5 * time.Minute, 30 * time.Minute},
		Timeout: 10 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { return j.Rollup(ctx, now()) }, RollupCron, queue.Unique(23*time.Hour)); err != nil {
		return err
	}
	return q.RegisterPeriodic(queue.Job{
		Type: JobPrune, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { _, err := j.Prune(ctx, now()); return err }, PruneCron,
		queue.Unique(23*time.Hour))
}
