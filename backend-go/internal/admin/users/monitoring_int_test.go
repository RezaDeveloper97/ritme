package users_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	platformmetrics "github.com/ritme/backend-go/internal/platform/metrics"
)

var projNow = time.Date(2026, 10, 31, 4, 30, 0, 0, civildate.Tehran)

func day(n int) string { return civildate.FromTime(projNow).AddDays(n).String() }

// logged inserts logged activity days [from, to] (offsets from today).
func logged(e *admintest.Env, uid uint64, from, to int) {
	for d := from; d <= to; d++ {
		e.Exec(`INSERT INTO user_activity_days (user_id, day, session, logged) VALUES (?, ?, 1, 1)`, uid, day(d))
	}
}

// The nightly job: churn per A-08, plan, mode and data_quality / main warning from the snapshot.
func TestUserActivity_RebuildProjection(t *testing.T) {
	e := admintest.New(t)
	e.Exec(`INSERT INTO users (id, mobile, created_at, updated_at) VALUES
		(20, '09120000020', '2026-01-01 10:00:00', NOW()), (21, '09120000021', '2026-01-01 10:00:00', NOW()),
		(22, '09120000022', '2026-01-01 10:00:00', NOW()), (23, '09120000023', '2026-10-21 10:00:00', NOW()),
		(24, '09120000024', '2026-01-01 10:00:00', NOW())`)
	// 20: 10 logged days in the prior window, 5 recent → high. 21: 10 / 8 → low. 22: last activity 7 days ago → high.
	// 23: account 10 days old → new. 24: 10 / 7 → medium, with a snapshot and a Plus subscription.
	logged(e, 20, -27, -18)
	logged(e, 20, -4, 0)
	logged(e, 21, -27, -18)
	logged(e, 21, -7, 0)
	logged(e, 22, -27, -7)
	logged(e, 24, -27, -18)
	logged(e, 24, -6, 0)
	e.Exec(`INSERT INTO engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality, warnings_mask,
		days_late, mode, engine_rev, created_at) VALUES (24, ?, 'luteal', 'late_luteal', 'medium', 'partial', ?, NULL, 'ttc', 3, NOW())`,
		day(0), (1<<3)|(1<<13))
	e.Exec(`INSERT INTO plus_subscriptions (user_id, status, source, starts_at, ends_at, created_at, updated_at)
		VALUES (24, 'active', 'purchase', '2026-10-01 00:00:00', '2026-12-01 00:00:00', NOW(), NOW())`)

	a := users.NewActivity(e.DB, admintest.Quiet)
	require.NoError(t, a.Rebuild(t.Context(), projNow))
	require.NoError(t, a.Rebuild(t.Context(), projNow), "idempotent")
	churn := func(id int) string { return e.String(`SELECT churn FROM admin_user_activity WHERE user_id = ?`, id) }
	assert.Equal(t, "high", churn(20))
	assert.Equal(t, "low", churn(21))
	assert.Equal(t, "high", churn(22))
	assert.Equal(t, "new", churn(23))
	assert.Equal(t, "medium", churn(24))
	assert.Equal(t, 5, e.Int(`SELECT COUNT(*) FROM admin_user_activity`))
	assert.Equal(t, 15, e.Int(`SELECT log_days_30 FROM admin_user_activity WHERE user_id = 20`))
	assert.Equal(t, "plus|ttc|partial|stale_onboarding_anchor", e.String(
		`SELECT CONCAT_WS('|', plan, mode, data_quality, main_warning) FROM admin_user_activity WHERE user_id = 24`))
	assert.Equal(t, "free|cycle", e.String(`SELECT CONCAT_WS('|', plan, mode) FROM admin_user_activity WHERE user_id = 20`))
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM admin_user_activity WHERE user_id = 20 AND data_quality IS NOT NULL`))
}

// Directory with 10 000 users: one count and one page query, also with a saved segment applied (+1).
func TestDirectory_10kUsersAtMostThreeQueries(t *testing.T) {
	e := admintest.New(t)
	e.Exec(`INSERT INTO users (id, mobile, created_at, updated_at)
		SELECT 1000 + a.n * 1000 + b.n * 100 + c.n * 10 + d.n, CONCAT('0912', LPAD(a.n * 1000 + b.n * 100 + c.n * 10 + d.n, 7, '0')),
		       '2026-01-01 10:00:00', NOW()
		FROM (SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
		      UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) a,
		     (SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
		      UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) b,
		     (SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
		      UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) c,
		     (SELECT 0 n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5
		      UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9) d`)
	require.Equal(t, 10000, e.Int(`SELECT COUNT(*) FROM users`))
	require.NoError(t, users.NewActivity(e.DB, admintest.Quiet).Rebuild(t.Context(), projNow))
	require.Equal(t, 10000, e.Int(`SELECT COUNT(*) FROM admin_user_activity`))

	counted, err := db.OpenDSN(t.Context(), testdb.DSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = counted.Close() })
	h := users.NewHandlers(counted, admintest.Quiet)
	ctx, q := platformmetrics.WithQueryCounter(context.Background())
	rows, total, err := h.DirectoryPage(ctx, segments.Filters{Plan: "free", Churn: "high"}, projNow, 50, 100)
	require.NoError(t, err)
	assert.Len(t, rows, 50)
	assert.Equal(t, int64(10000), total)
	assert.LessOrEqual(t, q.Load(), int64(2))
}

// A-09: the code is set once, searchable, and a collision extends it to 7 characters.
func TestRefCode_AssignCollisionAndSearch(t *testing.T) {
	e := newV2Env(t)
	rc, err := users.NewRefCoder(e.DB, "testing", "test-secret", admintest.Quiet)
	require.NoError(t, err)
	e.Exec(`INSERT INTO users (id, mobile, ref_code, created_at, updated_at) VALUES (12, '09350000012', ?, NOW(), NOW())`,
		rc.Code(11, 6))
	require.NoError(t, rc.Backfill(t.Context()))
	assert.Equal(t, rc.Code(10, 6), e.String(`SELECT ref_code FROM users WHERE id = 10`))
	assert.Equal(t, rc.Code(11, 7), e.String(`SELECT ref_code FROM users WHERE id = 11`), "collision → 7 characters")
	require.NoError(t, rc.Assign(t.Context(), 10), "idempotent")
	assert.Equal(t, rc.Code(10, 6), e.String(`SELECT ref_code FROM users WHERE id = 10`))
	require.NoError(t, rc.SignupHook()(t.Context(), 10, "", time.Now()))

	sup := e.As(supportID)
	r := sup.Get("/users/directory?q=" + rc.Code(10, 6))
	require.Equal(t, 200, r.Status, string(r.Raw))
	require.Len(t, r.Items(), 1)
	item := r.Items()[0].(map[string]any)
	assert.Equal(t, rc.Code(10, 6), item["ref"])
	assert.Equal(t, float64(10), item["id"])
	assert.Equal(t, rc.Code(10, 6), r.Obj("filters")["q"])
	assert.Equal(t, rc.Code(10, 6), sup.Get("/users/10/overview").Data()["ref"])
}

// The detail without a grant: engine card (codes) and cycle lengths only — no goal, symptom, note or dates.
func TestOverview_EngineCardWithoutHealth(t *testing.T) {
	e := newV2Env(t)
	today := civildate.InTehran(time.Now())
	e.Exec(`INSERT INTO engine_daily_snapshots (user_id, day, main_phase, subphase, confidence, data_quality, warnings_mask,
		days_late, mode, engine_rev, created_at) VALUES (10, ?, 'follicular', 'mid_follicular', 'high', 'good', 8, NULL, 'ttc', 1, NOW())`,
		today.String())
	for _, d := range []int{0, -29, -57, -88} {
		e.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, is_confirmed, created_at, updated_at)
			VALUES (10, ?, 1, NOW(), NOW())`, today.AddDays(d-3).String())
	}
	e.Exec(`INSERT INTO daily_health_logs (user_id, log_date, headache_intensity, notes, created_at, updated_at)
		VALUES (10, ?, 'migraine-secret', 'note-secret', NOW(), NOW())`, today.String())
	e.Exec(`INSERT INTO message_rules (id, title, kind, placement, content, conditions, exceptions, created_at, updated_at) VALUES
		(9001, 'PCOS clinical tip', 'clinical', 'smart_tip', '{}', '[]', '[]', NOW(), NOW()),
		(9002, 'Autumn promo', 'marketing', 'smart_tip', '{}', '[]', '[]', NOW(), NOW())`)
	e.Exec(`INSERT INTO message_events (rule_id, user_id, variant, event, version, slot, created_at) VALUES
		(9001, 10, 'a', 'shown', 2, 'smart_tip', NOW()), (9002, 10, 'a', 'shown', 1, 'smart_tip', NOW())`)

	r := e.As(supportID).Get("/users/10/overview")
	require.Equal(t, 200, r.Status, string(r.Raw))
	card := r.Obj("engine")
	require.NotNil(t, card)
	assert.Equal(t, "follicular", card["main_phase"])
	assert.Equal(t, "low_history", card["main_warning"])
	assert.Equal(t, []any{float64(29), float64(28), float64(31)}, r.Data()["cycle_lengths"])
	assert.Nil(t, r.Data()["health"])
	raw := string(r.Raw)
	for _, leak := range []string{"migraine-secret", "note-secret", "secret-note", "PCOS", "user_goal\":\"ttc",
		today.AddDays(-3).String(), "09121234567"} {
		assert.NotContains(t, raw, leak)
	}
	assert.Contains(t, raw, "Autumn promo")
	msgs := r.Data()["messages"].([]any)
	require.Len(t, msgs, 2)
	rule := msgs[1].(map[string]any)["rule"].(map[string]any)
	assert.Equal(t, float64(9001), rule["id"])
	assert.Equal(t, float64(2), rule["version"])
	assert.Nil(t, rule["title"], "clinical title needs a grant")
	assert.Zero(t, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'health_access.read'`))
}

// Legacy /users: the full mobile only for users at manage (T-M10-33 note).
func TestLegacyUsers_MobileMaskedForViewers(t *testing.T) {
	e := newV2Env(t)
	r := e.As(supportID).Get("/users")
	require.Equal(t, 200, r.Status)
	assert.NotContains(t, string(r.Raw), "09121234567")
	assert.NotContains(t, string(e.As(supportID).Get("/users/10").Raw), "09121234567")
	assert.Len(t, e.As(supportID).Get("/users?q=09121234567").Items(), 1, "search still matches the full number")
	assert.Contains(t, string(e.As(admintest.SuperID).Get("/users/10").Raw), "09121234567")
	assert.Contains(t, string(e.As(admintest.EditorID).Get("/users").Raw), "09121234567")
}
