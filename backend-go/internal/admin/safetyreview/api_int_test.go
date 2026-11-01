package safetyreview_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/safetyreview"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	advisor1  uint64 = 3
	advisor2  uint64 = 4
	marketing uint64 = 7
)

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	safetyreview.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Dr Ahmadi', 'med1@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW()),
		(4, 'Dr Niko', 'med2@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW()),
		(7, 'Mark', 'mkt@ritme.test', ?, 'marketing', 1, NOW(), NOW())`, pw, pw, pw)
	// Start from an empty queue: the migrations seed catalog / assistant rows flagged for review.
	e.Exec(`UPDATE catalog_items SET needs_review = 0`)
	e.Exec(`UPDATE assistant_departments SET needs_review = 0`)
	e.Exec(`UPDATE message_contents SET is_approved = 1`)
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES
		(50, 'U1', '09120000050', NOW(), NOW()), (51, 'U2', '09120000051', NOW(), NOW())`)
	return e
}

func ts(d time.Duration) string {
	return time.Now().In(civildate.Tehran).Add(d).Format(time.DateTime)
}

func day(n int) string { return civildate.InTehran(time.Now()).AddDays(n).String() }

func seedSignals(e *admintest.Env) {
	h := -time.Hour
	e.Exec(`INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, glucose_mg_dl, created_at, updated_at) VALUES
		(50, 'bp', ?, 185, 90, NULL, NOW(), NOW()),
		(51, 'bp', ?, 150, 125, NULL, NOW(), NOW()),
		(50, 'bp', ?, 140, 90, NULL, NOW(), NOW()),
		(50, 'bp', ?, 190, 100, NULL, NOW(), NOW()),
		(50, 'glucose', ?, NULL, NULL, 50.0, NOW(), NOW()),
		(51, 'glucose', ?, NULL, NULL, 60.0, NOW(), NOW())`,
		ts(h), ts(48*h), ts(h), ts(240*h), ts(2*h), ts(2*h))
	e.Exec(`INSERT INTO epds_checks (user_id, kind, taken_on, answers, total, self_harm, urgent, created_at, updated_at) VALUES
		(50, 'full', ?, '{}', 12, 1, 1, NOW(), NOW()),
		(51, 'full', ?, '{}', 3, 0, 0, NOW(), NOW()),
		(51, 'full', ?, '{}', 9, 2, 1, NOW(), NOW())`, day(0), day(-1), day(-9))
	e.Exec(`INSERT INTO pregnancy_kick_sessions (user_id, started_at, ended_at, kicks, created_at, updated_at) VALUES
		(50, ?, ?, 5, NOW(), NOW()), (51, ?, ?, 10, NOW(), NOW()), (51, ?, NULL, 2, NOW(), NOW())`,
		ts(-3*time.Hour), ts(-50*time.Minute), ts(-2*time.Hour), ts(-100*time.Minute), ts(-10*time.Minute))
	e.Exec(`INSERT INTO pregnancy_contraction_sessions (user_id, started_at, alert_at, created_at, updated_at) VALUES
		(50, ?, ?, NOW(), NOW()), (51, ?, NULL, NOW(), NOW())`, ts(-3*time.Hour), ts(-2*time.Hour), ts(-time.Hour))
	e.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, created_at, updated_at) VALUES
		(50, ?, 'bleeding', 'lochia_amount', '', 'heavy', NOW(), NOW()),
		(50, ?, 'bleeding', 'lochia_amount', '', 'heavy', NOW(), NOW()),
		(51, ?, 'bleeding', 'lochia_amount', '', 'medium', NOW(), NOW())`, day(0), day(-1), day(0))
}

func seedQueue(e *admintest.Env) {
	e.Exec(`INSERT INTO message_rules (id, title, content, conditions, exceptions, status, created_by, created_at, updated_at) VALUES
		(9011, 'PMS likely tomorrow (B)', '{}', '[]', '[]', 'pending_approval', 3, ?, ?),
		(9012, 'Draft rule', '{}', '[]', '[]', 'draft', 3, NOW(), NOW())`, ts(-100*time.Hour), ts(-100*time.Hour))
	e.Exec(`INSERT INTO clinical_config_versions (id, config_key, version, value, effective_from, status, proposed_by, created_at, updated_at) VALUES
		(21, 'glucose.post_meal', 2, '{"mgdl":170}', ?, 'pending', 4, ?, ?),
		(22, 'bp.crisis', 2, '{"systolic":180,"diastolic":120}', ?, 'approved', 4, NOW(), NOW())`,
		day(3), ts(-time.Hour), ts(-time.Hour), day(-3))
	e.Exec(`INSERT INTO clinical_config_approvals (version_id, admin_id, created_at) VALUES (21, 3, NOW())`)
	e.Exec(`INSERT INTO learning_instructors (id, user_id, display_name, title, status, created_at, updated_at)
		VALUES (1, 50, 'Leila M', 'ماما', 'approved', NOW(), NOW())`)
	e.Exec(`INSERT INTO learning_courses (id, instructor_id, kind, title, status, published_at, created_at, updated_at)
		VALUES (1, 1, 'course', 'Birth', 'published', NOW(), NOW(), NOW())`)
	e.Exec(`INSERT INTO learning_lessons (id, course_id, kind, title, status, review_status, reviewed_at, published_at, created_at, updated_at) VALUES
		(31, 1, 'video', 'Breathing in labour', 'published', 'pending', NULL, NOW(), NOW(), NOW()),
		(32, 1, 'video', 'Reviewed lesson', 'published', 'approved', ?, NOW(), NOW(), ?),
		(33, 1, 'video', 'Edited lesson', 'published', 'approved', ?, NOW(), NOW(), ?),
		(34, 1, 'video', 'Draft lesson', 'draft', 'pending', NULL, NULL, NOW(), NOW())`,
		ts(0), ts(-time.Hour), ts(-2*time.Hour), ts(-time.Minute))
	e.Exec(`UPDATE assistant_departments SET needs_review = 1 ORDER BY id LIMIT 1`)
	e.Exec(`UPDATE catalog_items SET needs_review = 1 WHERE id IN (SELECT id FROM (SELECT id FROM catalog_items WHERE ` +
		"`group`" + ` = (SELECT MIN(` + "`group`" + `) FROM catalog_items) ORDER BY id LIMIT 2) x)`)
}

func items(r admintest.Resp) []map[string]any {
	q, _ := r.Data()["queue"].(map[string]any)
	raw, _ := q["items"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		out = append(out, m)
	}
	return out
}

func find(list []map[string]any, source string, id float64) map[string]any {
	for _, it := range list {
		if it["source"] == source && it["id"] == id {
			return it
		}
	}
	return nil
}

func TestSafety_PermissionsAndAuth(t *testing.T) {
	e := newEnv(t)
	assert.Equal(t, 401, e.Anonymous().Get("/safety").Status)
	assert.Equal(t, 403, e.As(marketing).Get("/safety").Status)
	assert.Equal(t, 403, e.As(marketing).JSON(fiber.MethodPost, "/safety/queue/message/9011/claim", nil).Status)
	assert.Equal(t, 200, e.As(advisor1).Get("/safety").Status)
	assert.Equal(t, 200, e.As(1).Get("/safety").Status)
}

func TestSafety_SignalsAggregateWithoutHealthFields(t *testing.T) {
	e := newEnv(t)
	seedSignals(e)
	r := e.As(advisor1).Get("/safety")
	require.Equal(t, 200, r.Status, r.Body)
	assert.EqualValues(t, 7, r.Data()["window_days"])
	want := map[string][2]float64{
		"bp_crisis": {2, 1}, "glucose_urgent_low": {1, 0}, "epds_q10": {1, 1}, "reduced_kicks": {1, 0},
		"contractions_511": {1, 0}, "postpartum_heavy_bleeding": {2, 0},
	}
	sigs, _ := r.Data()["signals"].([]any)
	require.Len(t, sigs, len(want))
	for _, s := range sigs {
		m, _ := s.(map[string]any)
		w, ok := want[m["key"].(string)]
		require.True(t, ok, m)
		assert.Equal(t, w[0], m["count"], m["key"])
		assert.Equal(t, w[1], m["previous"], m["key"])
		assert.Equal(t, w[0]-w[1], m["delta"], m["key"])
		assert.ElementsMatch(t, []string{"key", "count", "previous", "delta"}, keys(m))
	}
	raw, err := json.Marshal(r.Body)
	require.NoError(t, err)
	for _, banned := range []string{"user_id", "mobile", "systolic", "diastolic", "glucose_mg_dl", "answers", "self_harm",
		"value_code", "09120000050", "U1"} {
		assert.NotContains(t, string(raw), banned)
	}
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestSafety_QueueAggregatesEverySource(t *testing.T) {
	e := newEnv(t)
	seedQueue(e)
	r := e.As(advisor1).Get("/safety")
	require.Equal(t, 200, r.Status, r.Body)
	q, _ := r.Data()["queue"].(map[string]any)
	by, _ := q["by_source"].(map[string]any)
	assert.Equal(t, map[string]any{"message": 1.0, "threshold": 1.0, "lesson": 2.0, "assistant": 1.0, "catalog": 2.0}, by)
	assert.EqualValues(t, 6, q["pending"])
	assert.EqualValues(t, 72, q["sla_hours"])
	assert.EqualValues(t, 1, q["overdue"])

	list := items(r)
	msg := find(list, "message", 9011)
	require.NotNil(t, msg)
	assert.Equal(t, "PMS likely tomorrow (B)", msg["title"])
	assert.Equal(t, "Dr Ahmadi", msg["submitted_by"])
	assert.Equal(t, true, msg["overdue"])
	assert.GreaterOrEqual(t, msg["age_hours"], 99.0)
	assert.Nil(t, find(list, "message", 9012))

	th := find(list, "threshold", 21)
	require.NotNil(t, th)
	assert.Equal(t, "partially_approved", th["status"])
	assert.EqualValues(t, 1, th["approvals"])
	assert.EqualValues(t, 2, th["required_approvals"])
	assert.Equal(t, "glucose.post_meal", th["ref"])
	assert.Nil(t, find(list, "threshold", 22))

	assert.Equal(t, "pending", find(list, "lesson", 31)["status"])
	assert.Equal(t, "changed", find(list, "lesson", 33)["status"])
	assert.Equal(t, "Leila M", find(list, "lesson", 31)["submitted_by"])
	assert.Nil(t, find(list, "lesson", 32))
	assert.Nil(t, find(list, "lesson", 34))

	checks, _ := r.Data()["checks"].(map[string]any)
	parity, _ := checks["text_parity"].(map[string]any)
	assert.Equal(t, true, parity["ok"])
	e.Exec("UPDATE message_contents SET is_approved = 0 WHERE `group` = 'vitals_alert' AND is_active = 1 LIMIT 1")
	checks, _ = e.As(advisor1).Get("/safety").Data()["checks"].(map[string]any)
	parity, _ = checks["text_parity"].(map[string]any)
	assert.Equal(t, false, parity["ok"])
	assert.EqualValues(t, 1, parity["unapproved"])
	assert.Nil(t, checks["modal_latency_p95_ms"])
}

func TestSafety_ClaimAndRelease(t *testing.T) {
	e := newEnv(t)
	seedQueue(e)
	assert.Equal(t, 404, e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/nope/9011/claim", nil).Status)
	assert.Equal(t, 404, e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/message/9012/claim", nil).Status)
	assert.Equal(t, 404, e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/message/x/claim", nil).Status)

	r := e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/message/9011/claim", nil)
	require.Equal(t, 200, r.Status, r.Body)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'safety.claim' AND target_type = 'safety.message' AND target_id = 9011"))
	// Claiming again is idempotent (no second audit row); another admin gets 409.
	assert.Equal(t, 200, e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/message/9011/claim", nil).Status)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'safety.claim'"))
	c := e.As(advisor2).JSON(fiber.MethodPost, "/safety/queue/message/9011/claim", nil)
	assert.Equal(t, 409, c.Status)
	assert.Equal(t, safetyreview.CodeAlreadyClaimed, c.Code())

	claimed, _ := find(items(e.As(advisor2).Get("/safety")), "message", 9011)["claimed_by"].(map[string]any)
	assert.Equal(t, "Dr Ahmadi", claimed["name"])

	rel := e.As(advisor2).JSON(fiber.MethodPost, "/safety/queue/message/9011/release", nil)
	assert.Equal(t, 403, rel.Status)
	assert.Equal(t, safetyreview.CodeNotClaimer, rel.Code())
	assert.Equal(t, 200, e.As(advisor1).JSON(fiber.MethodPost, "/safety/queue/message/9011/release", nil).Status)
	assert.Nil(t, find(items(e.As(advisor2).Get("/safety")), "message", 9011)["claimed_by"])
	assert.Equal(t, 200, e.As(advisor2).JSON(fiber.MethodPost, "/safety/queue/message/9011/claim", nil).Status)
	// A super admin may release anyone's claim.
	assert.Equal(t, 200, e.As(1).JSON(fiber.MethodPost, "/safety/queue/message/9011/release", nil).Status)
}
