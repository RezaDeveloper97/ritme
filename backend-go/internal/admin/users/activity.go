package users

// The user-directory projection (T-M10-32, D-181; SPEC v6 §5.2, A-08; gap 06 §2.5): the nightly `admin.user_activity`
// job rebuilds admin_user_activity from user_activity_days (T-M10-30), the Plus tier, the stored life mode and the
// newest engine_daily_snapshots row (T-M9-39), so the directory filters on indexed columns and never aggregates logs
// per request. Codes and counts of days only — never what was logged.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// Projection job (docs/engine-v6/jobs.md): after engine:snapshot_nightly (03:30).
const (
	JobUserActivity  = "admin.user_activity"
	UserActivityCron = "30 4 * * *"
	activityPage     = 1000
)

// Churn windows (A-08): logged days in the recent 14 days vs the 14 before; 7 days without any activity is high; an
// account younger than 14 days is `new`.
const (
	churnWindowDays   = 14
	churnInactiveDays = 7
)

// ChurnInput is what the churn rule reads.
type ChurnInput struct {
	Recent, Prior  int  // logged days in the recent / prior 14 days
	DaysInactive   int  // days since the last activity day (0 = active today)
	NeverActive    bool // no activity day at all
	AccountAgeDays int  // days since signup
}

// Churn is the SPEC §5.2 / A-08 rule: new (account < 14 days) → high (recent ≤ 50 % of prior, or ≥ 7 days without
// activity) → medium (recent ≤ 75 % of prior) → low. With no logged day in the prior window there is no baseline,
// so the ratio rule does not apply (only inactivity can make it high).
func Churn(in ChurnInput) string {
	switch {
	case in.AccountAgeDays < churnWindowDays:
		return segments.ChurnNew
	case in.NeverActive || in.DaysInactive >= churnInactiveDays:
		return segments.ChurnHigh
	case in.Prior > 0 && in.Recent*100 <= in.Prior*50:
		return segments.ChurnHigh
	case in.Prior > 0 && in.Recent*100 <= in.Prior*75:
		return segments.ChurnMedium
	default:
		return segments.ChurnLow
	}
}

// MainWarning is the engine's main warning of a snapshot mask: the first warning that requires user input (§2.33) in
// spec order, else the first warning in spec order; "" when none.
func MainWarning(mask uint32) string {
	set := engine.WarningSet(mask)
	first := ""
	for w := range set.All() {
		if w.RequiresUserInput() {
			return w.String()
		}
		if first == "" {
			first = w.String()
		}
	}
	return first
}

// Activity rebuilds the projection.
type Activity struct {
	db  *sql.DB
	q   *store.Queries
	log *slog.Logger
}

// NewActivity returns the projection writer over db.
func NewActivity(db *sql.DB, logger *slog.Logger) *Activity {
	if logger == nil {
		logger = slog.Default()
	}
	return &Activity{db: db, q: store.New(db), log: logger}
}

// RegisterJobs puts the nightly job on q; now is the Tehran clock.
func (a *Activity) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobUserActivity, Priority: queue.Low, MaxTries: 3, Backoff: []time.Duration{10 * time.Minute, 30 * time.Minute},
		Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error { return a.Rebuild(ctx, now()) }, UserActivityCron, queue.Unique(23*time.Hour))
}

func uint8Of(n int64) uint8 { return uint8(max(0, min(255, n))) } //nolint:gosec // G115: clamped

// Rebuild writes every user's projection row for the Tehran day of now (pages of 1 000 users, one read and one
// transaction per page), then deletes rows it did not touch. Idempotent: upsert keyed by user.
func (a *Activity) Rebuild(ctx context.Context, now time.Time) error {
	now = now.In(civildate.Tehran)
	day := civildate.FromTime(now)
	var after uint64
	written := 0
	for {
		page, err := a.q.ListAdminActivitySources(ctx, store.ListAdminActivitySourcesParams{
			Now: now.Truncate(time.Second), ToDay: day, From30: day.AddDays(-29),
			RecentFrom: day.AddDays(-(churnWindowDays - 1)), PriorFrom: day.AddDays(-(2*churnWindowDays - 1)),
			PriorTo: day.AddDays(-churnWindowDays), SnapshotSince: day.AddDays(-1), AfterID: after, Limit: activityPage,
		})
		if err != nil {
			return fmt.Errorf("users: activity sources: %w", err)
		}
		if len(page) == 0 {
			break
		}
		if err := a.writePage(ctx, page, day); err != nil {
			return err
		}
		written += len(page)
		after = page[len(page)-1].UserID
		if len(page) < activityPage {
			break
		}
	}
	if _, err := a.q.DeleteStaleAdminUserActivity(ctx, day); err != nil {
		return fmt.Errorf("users: activity stale: %w", err)
	}
	a.log.InfoContext(ctx, "admin user activity rebuilt", slog.Int("users", written))
	return nil
}

func (a *Activity) writePage(ctx context.Context, page []store.ListAdminActivitySourcesRow, day civildate.Date) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("users: activity begin: %w", err)
	}
	q := a.q.WithTx(tx)
	for _, r := range page {
		if err := q.UpsertAdminUserActivity(ctx, ProjectionRow(r, day)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("users: activity upsert: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("users: activity commit: %w", err)
	}
	return nil
}

// lastDayOf reads the MAX(day) subquery (DATE → time.Time with parseTime, or bytes/string).
func lastDayOf(v any) (civildate.Date, bool) {
	switch x := v.(type) {
	case time.Time:
		return civildate.FromTime(x), true
	case []byte:
		d, err := civildate.Parse(string(x))
		return d, err == nil
	case string:
		d, err := civildate.Parse(x)
		return d, err == nil
	}
	return civildate.Date{}, false
}

// ProjectionRow maps one source row to the projection on day.
func ProjectionRow(r store.ListAdminActivitySourcesRow, day civildate.Date) store.UpsertAdminUserActivityParams {
	in := ChurnInput{Recent: int(r.LogDaysRecent14), Prior: int(r.LogDaysPrior14), AccountAgeDays: 1 << 20}
	if r.CreatedAt.Valid {
		in.AccountAgeDays = civildate.InTehran(r.CreatedAt.Time).DiffDays(day)
	}
	row := store.UpsertAdminUserActivityParams{
		UserID: r.UserID, LogDays30: uint8Of(r.LogDays30), LogDaysRecent14: uint8Of(r.LogDaysRecent14),
		LogDaysPrior14: uint8Of(r.LogDaysPrior14), Plan: "free", UpdatedOn: day,
	}
	if last, ok := lastDayOf(r.LastDay); ok {
		row.LastActivityAt = sql.NullTime{Time: last.TehranMidnight(), Valid: true}
		in.DaysInactive = last.DiffDays(day)
	} else {
		in.NeverActive = true
	}
	row.Churn = Churn(in)
	switch {
	case r.PlusActive != 0:
		row.Plan = "plus"
	case r.TrialRunning != 0:
		row.Plan = "trial"
	}
	// Mode: the engine's mode of the newest snapshot; without one the stored life mode; else cycle (the engine's zero).
	switch {
	case r.SnapshotMode.Valid:
		row.Mode = r.SnapshotMode.String
	case r.LifeMode.Valid && r.LifeMode.String != "":
		row.Mode = r.LifeMode.String
	default:
		row.Mode = engine.ModeCycle.String()
	}
	if len(row.Mode) > 16 {
		row.Mode = row.Mode[:16]
	}
	if r.SnapshotDataQuality.Valid {
		row.DataQuality = sql.NullString{String: r.SnapshotDataQuality.String, Valid: true}
		if w := MainWarning(uint32(r.SnapshotWarningsMask.Int32)); w != "" { //nolint:gosec // G115: a bit mask
			row.MainWarning = sql.NullString{String: w, Valid: true}
		}
	}
	return row
}
