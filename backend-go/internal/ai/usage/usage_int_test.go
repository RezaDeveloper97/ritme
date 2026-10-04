package usage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// fixed is a clock pinned at 2026-10-03 12:00 Tehran.
var fixed = clock.Fixed(time.Date(2026, 10, 3, 12, 0, 0, 0, time.FixedZone("IRST", 3*3600+1800)))

func TestRecorderBudgetAndAdminUsage(t *testing.T) {
	e := admintest.New(t)
	budget := usage.NewBudget(e.DB, 0.05, fixed, admintest.Quiet) // 50 000 micro-USD
	usage.NewAdminHandlers(e.DB, budget, fixed).Routes(e.Route(), e.Kit)
	rec := usage.NewRecorder(e.DB, fixed, admintest.Quiet)
	uid, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES ('09120000001', 'Sara', NOW(), NOW())").LastInsertId()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, budget.Allow(ctx))
	rec.Record(ctx, ai.Usage{Provider: "gemini", Model: "gemini-flash-latest", Feature: ai.FeatureAssistant, Op: ai.OpChat,
		InputTokens: 1000, OutputTokens: 200, ImageBytes: 3000, CostMicros: 30000, Latency: 1500 * time.Millisecond, OK: true, UserID: uint64(uid)}) //nolint:gosec // test id
	rec.Record(ctx, ai.Usage{Provider: "gemini", Model: "gemini-flash-latest", Feature: ai.FeatureLabAnalysis, Op: ai.OpExtract,
		InputTokens: 2000, CostMicros: 25000, OK: false})
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM ai_usage_logs"))
	assert.Equal(t, "2026-10-03 12:00:00", e.String("SELECT DATE_FORMAT(MIN(created_at), '%Y-%m-%d %H:%i:%s') FROM ai_usage_logs"), "Tehran wall-clock")
	require.ErrorIs(t, budget.Allow(ctx), ai.ErrBudgetExceeded, "55 000 ≥ 50 000")

	// an older day outside the default window
	e.Exec(`INSERT INTO ai_usage_logs (feature, op, provider, model, cost_micros, ok, created_at) VALUES ('voice_log', 'transcribe', 'gemini', 'm', 7, 1, '2026-06-01 10:00:00')`)

	assert.Equal(t, 401, e.Anonymous().Get("/ai/usage").Status)
	r := e.As(admintest.EditorID).Get("/ai/usage")
	require.Equal(t, 200, r.Status, r.Body)
	d := r.Data()
	assert.Equal(t, "2026-09-04", d["from"])
	assert.Equal(t, "2026-10-03", d["to"])
	today := r.Obj("today")
	assert.InDelta(t, 55000, today["spent_micros"], 0)
	assert.InDelta(t, 50000, today["cap_micros"], 0)
	assert.Equal(t, true, today["exhausted"])
	totals := r.Obj("totals")
	assert.InDelta(t, 2, totals["calls"], 0)
	assert.InDelta(t, 1, totals["ok_calls"], 0)
	assert.InDelta(t, 0.055, totals["cost_usd"], 1e-9)
	days := d["by_day"].([]any)
	require.Len(t, days, 1)
	assert.Equal(t, "2026-10-03", days[0].(map[string]any)["day"])
	feats := d["by_feature"].([]any)
	require.Len(t, feats, 2)
	top := feats[0].(map[string]any)
	assert.Equal(t, "assistant", top["feature"])
	assert.InDelta(t, 1, top["users"], 0)
	assert.InDelta(t, 1500, top["avg_latency_ms"], 0)
	assert.NotContains(t, r.Body, "user_id")

	r = e.As(admintest.EditorID).Get("/ai/usage?from=2026-06-01&to=2026-06-30")
	require.Equal(t, 200, r.Status)
	assert.InDelta(t, 7, r.Obj("totals")["cost_micros"], 0)

	for _, q := range []string{"?from=nope", "?to=2026-13-01", "?from=2026-10-03&to=2026-10-01", "?from=2026-01-01&to=2026-10-01"} {
		r = e.As(admintest.EditorID).Get("/ai/usage" + q)
		assert.Equal(t, 422, r.Status, q)
		assert.Equal(t, "validation_failed", r.Code(), q)
	}
}

// B-N6-05b (M2): the per-user daily cap counts only that user's calls of the Tehran day and fails closed.
func TestBudget_PerUserCap(t *testing.T) {
	e := admintest.New(t)
	budget := usage.NewBudget(e.DB, 5, fixed, admintest.Quiet).WithUserCap(0.01) // 10 000 micro-USD per user
	rec := usage.NewRecorder(e.DB, fixed, admintest.Quiet)
	a, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES ('09120000011', 'A', NOW(), NOW())").LastInsertId()
	require.NoError(t, err)
	b, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES ('09120000012', 'B', NOW(), NOW())").LastInsertId()
	require.NoError(t, err)
	ua, ub := uint64(a), uint64(b) //nolint:gosec // test ids
	ctx := context.Background()
	assert.Equal(t, uint64(10000), budget.UserCapMicros())
	require.NoError(t, budget.AllowUser(ctx, ua))
	rec.Record(ctx, ai.Usage{Provider: "gemini", Model: "m", Feature: ai.FeatureVoiceLog, Op: ai.OpTranscribe, CostMicros: 6000, UserID: ua})
	require.NoError(t, budget.AllowUser(ctx, ua))
	rec.Record(ctx, ai.Usage{Provider: "gemini", Model: "m", Feature: ai.FeatureVoiceLog, Op: ai.OpParseLog, CostMicros: 4000, UserID: ua})
	require.ErrorIs(t, budget.AllowUser(ctx, ua), ai.ErrUserBudgetExceeded, "10 000 ≥ 10 000")
	require.NoError(t, budget.AllowUser(ctx, ub), "another user is not affected")
	require.NoError(t, budget.Allow(ctx), "the global cap is far away")
	// yesterday's spend does not count
	e.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, cost_micros, ok, created_at) VALUES (?, 'voice_log', 'transcribe', 'gemini', 'm', 99999, 1, '2026-10-02 23:59:00')`, ub)
	require.NoError(t, budget.AllowUser(ctx, ub))

	zero := usage.NewBudget(e.DB, 5, fixed, admintest.Quiet).WithUserCap(0)
	require.ErrorIs(t, zero.AllowUser(ctx, ub), ai.ErrUserBudgetExceeded, "0 refuses (fail closed)")
}

// B-N6-05b (L5): rows older than 90 days lose their user; newer rows and the cost history stay.
func TestRecorder_AnonymizesAfterRetention(t *testing.T) {
	e := admintest.New(t)
	uid, err := e.Exec("INSERT INTO users (mobile, name, created_at, updated_at) VALUES ('09120000021', 'A', NOW(), NOW())").LastInsertId()
	require.NoError(t, err)
	for _, at := range []string{"2026-07-04 11:59:00", "2026-07-05 12:30:00", "2026-10-01 10:00:00"} {
		e.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, cost_micros, ok, created_at) VALUES (?, 'assistant', 'chat', 'gemini', 'm', 10, 1, ?)`, uid, at)
	}
	rec := usage.NewRecorder(e.DB, fixed, admintest.Quiet)
	n, err := rec.Anonymize(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "only the row before 2026-07-05 12:00 (90 days before the clock)")
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM ai_usage_logs WHERE user_id IS NOT NULL"))
	assert.Equal(t, 30, e.Int("SELECT CAST(SUM(cost_micros) AS SIGNED) FROM ai_usage_logs"), "the cost history stays")

	// Record runs the clean-up in the background (at most once per hour)
	e.Exec(`INSERT INTO ai_usage_logs (user_id, feature, op, provider, model, cost_micros, ok, created_at) VALUES (?, 'assistant', 'chat', 'gemini', 'm', 10, 1, '2026-01-01 00:00:00')`, uid)
	rec2 := usage.NewRecorder(e.DB, fixed, admintest.Quiet)
	rec2.Record(context.Background(), ai.Usage{Provider: "gemini", Model: "m", Feature: ai.FeatureAssistant, Op: ai.OpChat, UserID: uint64(uid)}) //nolint:gosec // test id
	require.Eventually(t, func() bool {
		return e.Int("SELECT COUNT(*) FROM ai_usage_logs WHERE user_id IS NOT NULL AND created_at < '2026-07-05'") == 0
	}, 5*time.Second, 20*time.Millisecond)
}
