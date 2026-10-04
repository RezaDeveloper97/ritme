// Package usage is the durable side of the AI platform (B-N6-05): the per-call usage + cost log
// (`ai_usage_logs`, Recorder), the global and per-user daily cost caps (Budget: ai.Limiter + ai.UserLimiter,
// B-N6-05b) and the admin aggregates (GET /api/admin/v1/ai/usage). Rows hold counts, sizes, the cost estimate,
// latency and outcome — never a prompt, an answer, a file or any other content.
//
// Retention (B-N6-05b, L5): a row keeps its user_id for RetentionDays, then the user is nulled (the cost history
// stays for the caps and the admin totals). The clean-up runs on write — Recorder.Record runs Anonymize at most
// once per AnonymizeEvery per process, in the background — so it needs no scheduler and runs exactly where rows
// are produced; with no AI traffic there is nothing new to anonymize, and the first call after a quiet spell
// catches up. The user's linked rows are part of GET /profile/export (internal/profile).
package usage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Store is the persistence of the package (internal/ai/store).
type Store interface {
	InsertUsage(ctx context.Context, arg store.InsertUsageParams) error
	SumCostSince(ctx context.Context, createdAt sql.NullTime) (int64, error)
	SumUserCostSince(ctx context.Context, arg store.SumUserCostSinceParams) (int64, error)
	AnonymizeUsageBefore(ctx context.Context, before sql.NullTime) (int64, error)
	UsageByDay(ctx context.Context, arg store.UsageByDayParams) ([]store.UsageByDayRow, error)
	UsageByFeature(ctx context.Context, arg store.UsageByFeatureParams) ([]store.UsageByFeatureRow, error)
}

// Retention of the user link (B-N6-05b, L5).
const (
	RetentionDays  = 90
	AnonymizeEvery = time.Hour
	// anonymizeBatches bounds one clean-up run (each batch is ≤ 5000 rows, AnonymizeUsageBefore).
	anonymizeBatches = 20
)

// Recorder writes one ai_usage_logs row per call (ai.Recorder). A failed insert is logged (without the usage
// payload's user) and never fails the call.
type Recorder struct {
	q      Store
	clock  clock.Clock
	logger *slog.Logger

	mu          sync.Mutex
	lastCleanup time.Time
	running     bool
}

// NewRecorder wires the recorder on db.
func NewRecorder(db *sql.DB, c clock.Clock, logger *slog.Logger) *Recorder {
	return NewRecorderWith(store.New(db), c, logger)
}

// NewRecorderWith wires the recorder on a store (tests).
func NewRecorderWith(q Store, c clock.Clock, logger *slog.Logger) *Recorder {
	if c == nil {
		c = clock.Real{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Recorder{q: q, clock: c, logger: logger}
}

func u32(n int) uint32 {
	if n <= 0 {
		return 0
	}
	if n > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(n)
}

// Record implements ai.Recorder.
func (r *Recorder) Record(ctx context.Context, u ai.Usage) {
	now := r.clock.Now().In(civildate.Tehran).Truncate(time.Second)
	var user sql.NullInt64
	if u.UserID > 0 && u.UserID <= math.MaxInt64 {
		user = sql.NullInt64{Int64: int64(u.UserID), Valid: true}
	}
	err := r.q.InsertUsage(ctx, store.InsertUsageParams{
		UserID: user, Feature: trunc(string(u.Feature), 32), Op: trunc(u.Op, 32), Provider: trunc(u.Provider, 16),
		Model: trunc(u.Model, 64), InputTokens: u32(u.InputTokens), OutputTokens: u32(u.OutputTokens),
		AudioBytes: u32(u.AudioBytes), ImageBytes: u32(u.ImageBytes), CostMicros: u.CostMicros,
		LatencyMs: u32(int(min(u.Latency.Milliseconds(), math.MaxUint32))), Ok: u.OK,
		CreatedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		r.logger.LogAttrs(ctx, slog.LevelError, "ai: usage log insert failed",
			slog.String("feature", string(u.Feature)), slog.String("op", u.Op), slog.String("error", err.Error()))
	}
	r.maybeAnonymize(ctx)
}

// maybeAnonymize starts a background clean-up when the last one is older than AnonymizeEvery.
func (r *Recorder) maybeAnonymize(ctx context.Context) {
	now := r.clock.Now()
	r.mu.Lock()
	if r.running || (!r.lastCleanup.IsZero() && now.Sub(r.lastCleanup) < AnonymizeEvery) {
		r.mu.Unlock()
		return
	}
	r.running, r.lastCleanup = true, now
	r.mu.Unlock()
	go func() {
		defer func() {
			r.mu.Lock()
			r.running = false
			r.mu.Unlock()
		}()
		if _, err := r.Anonymize(context.WithoutCancel(ctx)); err != nil {
			r.logger.LogAttrs(ctx, slog.LevelError, "ai: usage log anonymize failed", slog.String("error", err.Error()))
		}
	}()
}

// Anonymize nulls the user of every row older than RetentionDays (in bounded batches) and returns how many rows
// it changed.
func (r *Recorder) Anonymize(ctx context.Context) (int64, error) {
	before := r.clock.Now().In(civildate.Tehran).AddDate(0, 0, -RetentionDays).Truncate(time.Second)
	var total int64
	for range anonymizeBatches {
		n, err := r.q.AnonymizeUsageBefore(ctx, sql.NullTime{Time: before, Valid: true})
		if err != nil {
			return total, fmt.Errorf("ai usage: anonymize: %w", err)
		}
		total += n
		if n < 5000 {
			break
		}
	}
	return total, nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Budget is the global daily cost cap (ai.Limiter): the estimated cost of every call since the start of the
// Tehran day must stay below CapMicros. A read error refuses too (fail closed). Concurrent calls may overshoot
// the cap by the calls already in flight when it is reached (bounded by the request throttles).
type Budget struct {
	q             Store
	capMicros     uint64
	userCapMicros uint64 // per user and day (B-N6-05b); 0 refuses every call (fail closed)
	clock         clock.Clock
	logger        *slog.Logger
}

// USDToMicros converts a USD amount to micro-USD (rounded down; negative → 0).
func USDToMicros(usd float64) uint64 {
	if usd <= 0 || math.IsNaN(usd) {
		return 0
	}
	return uint64(math.Floor(usd * 1e6))
}

// NewBudget wires the cap (AI_DAILY_COST_CAP_USD) on db.
func NewBudget(db *sql.DB, capUSD float64, c clock.Clock, logger *slog.Logger) *Budget {
	return NewBudgetWith(store.New(db), capUSD, c, logger)
}

// NewBudgetWith wires the cap on a store (tests).
func NewBudgetWith(q Store, capUSD float64, c clock.Clock, logger *slog.Logger) *Budget {
	if c == nil {
		c = clock.Real{}
	}
	if logger == nil {
		logger = slog.Default()
	}
	// The per-user cap defaults to the global one until WithUserCap sets it (tests, tools).
	return &Budget{q: q, capMicros: USDToMicros(capUSD), userCapMicros: USDToMicros(capUSD), clock: c, logger: logger}
}

// WithUserCap sets the per-user daily cap (AI_USER_DAILY_COST_CAP_USD).
func (b *Budget) WithUserCap(usd float64) *Budget {
	b.userCapMicros = USDToMicros(usd)
	return b
}

// UserCapMicros is the per-user daily cap in micro-USD.
func (b *Budget) UserCapMicros() uint64 { return b.userCapMicros }

// SpentBy is the estimated cost of the user's calls today (micro-USD).
func (b *Budget) SpentBy(ctx context.Context, userID uint64) (uint64, error) {
	if userID > math.MaxInt64 {
		return 0, fmt.Errorf("ai usage: user id out of range")
	}
	n, err := b.q.SumUserCostSince(ctx, store.SumUserCostSinceParams{
		UserID: sql.NullInt64{Int64: int64(userID), Valid: true}, Since: sql.NullTime{Time: b.dayStart(b.clock.Now()), Valid: true},
	})
	if err != nil {
		return 0, fmt.Errorf("ai usage: spent today by user: %w", err)
	}
	return uint64(max(n, 0)), nil
}

// AllowUser implements ai.UserLimiter: the user's own daily cap (fail closed on a read error).
func (b *Budget) AllowUser(ctx context.Context, userID uint64) error {
	spent, err := b.SpentBy(ctx, userID)
	if err != nil {
		b.logger.LogAttrs(ctx, slog.LevelError, "ai: user budget unreadable, refusing the call", slog.String("error", err.Error()))
		return ai.ErrUserBudgetExceeded
	}
	if spent >= b.userCapMicros {
		// No user id in the line: the cap reached is enough to see abuse in the admin aggregates.
		b.logger.LogAttrs(ctx, slog.LevelWarn, "ai: user daily cost cap reached", slog.Uint64("cap_micros", b.userCapMicros))
		return ai.ErrUserBudgetExceeded
	}
	return nil
}

// CapMicros is the daily cap in micro-USD.
func (b *Budget) CapMicros() uint64 { return b.capMicros }

// dayStart is the start of today in Tehran.
func (b *Budget) dayStart(now time.Time) time.Time {
	return civildate.InTehran(now).TehranMidnight()
}

// Spent is the estimated cost of today's calls (micro-USD).
func (b *Budget) Spent(ctx context.Context) (uint64, error) {
	n, err := b.q.SumCostSince(ctx, sql.NullTime{Time: b.dayStart(b.clock.Now()), Valid: true})
	if err != nil {
		return 0, fmt.Errorf("ai usage: spent today: %w", err)
	}
	return uint64(max(n, 0)), nil
}

// Allow implements ai.Limiter.
func (b *Budget) Allow(ctx context.Context) error {
	spent, err := b.Spent(ctx)
	if err != nil {
		b.logger.LogAttrs(ctx, slog.LevelError, "ai: budget unreadable, refusing the call", slog.String("error", err.Error()))
		return ai.ErrBudgetExceeded
	}
	if spent >= b.capMicros {
		b.logger.LogAttrs(ctx, slog.LevelWarn, "ai: daily cost cap reached",
			slog.Uint64("spent_micros", spent), slog.Uint64("cap_micros", b.capMicros))
		return ai.ErrBudgetExceeded
	}
	return nil
}
