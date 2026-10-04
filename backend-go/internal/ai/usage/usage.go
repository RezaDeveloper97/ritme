// Package usage is the durable side of the AI platform (B-N6-05): the per-call usage + cost log
// (`ai_usage_logs`, Recorder), the global daily cost cap (Budget, the ai.Limiter) and the admin aggregates
// (GET /api/admin/v1/ai/usage). Rows hold counts, sizes, the cost estimate, latency and outcome — never a prompt,
// an answer, a file or any other content.
package usage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
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
	UsageByDay(ctx context.Context, arg store.UsageByDayParams) ([]store.UsageByDayRow, error)
	UsageByFeature(ctx context.Context, arg store.UsageByFeatureParams) ([]store.UsageByFeatureRow, error)
}

// Recorder writes one ai_usage_logs row per call (ai.Recorder). A failed insert is logged (without the usage
// payload's user) and never fails the call.
type Recorder struct {
	q      Store
	clock  clock.Clock
	logger *slog.Logger
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
	q         Store
	capMicros uint64
	clock     clock.Clock
	logger    *slog.Logger
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
	return &Budget{q: q, capMicros: USDToMicros(capUSD), clock: c, logger: logger}
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
