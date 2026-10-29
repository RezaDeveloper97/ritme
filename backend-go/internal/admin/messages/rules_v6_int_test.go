package messages_test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/messages/rules"
	mstore "github.com/ritme/backend-go/internal/messages/store"
)

// servedBody is what users are served for rule id: the body of variant a after the approved-version overlay
// (rules.ServedRows, the loader of rules.ActiveRows) — "" when the rule is not served.
func servedBody(t *testing.T, e *admintest.Env, id uint64) (string, int) {
	t.Helper()
	ctx := context.Background()
	q := mstore.New(e.DB)
	rows, err := q.ListActiveMessageRules(ctx, mstore.ListActiveMessageRulesParams{})
	require.NoError(t, err)
	rows, err = rules.ServedRows(ctx, q, rows)
	require.NoError(t, err)
	for i := range rows {
		if r := rules.FromRow(&rows[i]); r.ID == id && r.Served() {
			return r.Content.A.Body, r.Priority
		}
	}
	return "", 0
}

func idOf(r admintest.Resp) uint64 { return uint64(r.Obj("rule")["id"].(float64)) }

// T-M10-02 (M-02, M-03): an approved clinical v1 stays live while v2 waits; approving v2 serves it; a priority-only
// edit keeps the approval; rejecting a draft keeps the approved version live.
func TestRulesV6_VersionedApproval(t *testing.T) {
	e := rulesEnv(t)
	med := e.As(medicalID)

	r := med.JSON(fiber.MethodPost, "/messages/rules", clinicalRule())
	require.Equal(t, 201, r.Status, r.Body)
	id := idOf(r)
	path := "/messages/rules/" + itoa(int(id))
	require.Equal(t, 200, med.JSON(fiber.MethodPost, path+"/submit", nil).Status)
	require.Equal(t, 200, med.JSON(fiber.MethodPost, path+"/approve", nil).Status)
	require.Equal(t, 200, med.JSON(fiber.MethodPost, path+"/toggle", nil).Status)
	body, _ := servedBody(t, e, id)
	assert.Equal(t, "b", body, "v1 is live")

	// Text edit → v2 pending, users still get v1.
	edit := clinicalRule()
	edit["content"].(map[string]any)["a"].(map[string]any)["body"] = "v2 body"
	up := med.JSON(fiber.MethodPut, path, edit)
	require.Equal(t, 200, up.Status, up.Body)
	assert.Equal(t, "pending_approval", up.Obj("rule")["status"])
	assert.EqualValues(t, 2, up.Obj("rule")["version"])
	assert.EqualValues(t, 1, up.Obj("rule")["approved_version"])
	body, _ = servedBody(t, e, id)
	assert.Equal(t, "b", body, "the approved v1 keeps serving")
	show := med.Get(path)
	require.Equal(t, 200, show.Status)
	assert.Equal(t, "pending_approval", show.Obj("rule")["status"])
	assert.Equal(t, "v2 body", show.Obj("rule")["content"].(map[string]any)["a"].(map[string]any)["body"])

	// Approve → v2 served.
	ok := med.JSON(fiber.MethodPost, path+"/approve", nil)
	require.Equal(t, 200, ok.Status, ok.Body)
	assert.EqualValues(t, 2, ok.Obj("rule")["approved_version"])
	body, _ = servedBody(t, e, id)
	assert.Equal(t, "v2 body", body)

	// Priority-only edit → v3, approval kept (approved_version follows the head).
	ops := clinicalRule()
	ops["content"].(map[string]any)["a"].(map[string]any)["body"] = "v2 body"
	ops["priority"] = 170
	up = med.JSON(fiber.MethodPut, path, ops)
	require.Equal(t, 200, up.Status, up.Body)
	assert.Equal(t, "approved", up.Obj("rule")["status"])
	assert.EqualValues(t, 3, up.Obj("rule")["version"])
	assert.EqualValues(t, 3, up.Obj("rule")["approved_version"])
	assert.EqualValues(t, medicalID, up.Obj("rule")["approved_by"])
	body, prio := servedBody(t, e, id)
	assert.Equal(t, "v2 body", body)
	assert.Equal(t, 170, prio)

	// Text edit → v4 pending; an ops edit on top keeps it pending and still serves v3 with the new priority; reject →
	// draft, v3 still live.
	edit["priority"] = 170
	edit["content"].(map[string]any)["a"].(map[string]any)["body"] = "v4 body"
	require.Equal(t, 200, med.JSON(fiber.MethodPut, path, edit).Status)
	edit["priority"] = 180
	up = med.JSON(fiber.MethodPut, path, edit)
	require.Equal(t, 200, up.Status, up.Body)
	assert.Equal(t, "pending_approval", up.Obj("rule")["status"])
	assert.EqualValues(t, 3, up.Obj("rule")["approved_version"])
	body, prio = servedBody(t, e, id)
	assert.Equal(t, "v2 body", body)
	assert.Equal(t, 180, prio, "ops fields are the head's")
	rej := med.JSON(fiber.MethodPost, path+"/reject", nil)
	require.Equal(t, 200, rej.Status, rej.Body)
	assert.Equal(t, "draft", rej.Obj("rule")["status"])
	body, _ = servedBody(t, e, id)
	assert.Equal(t, "v2 body", body, "reject keeps the approved version live")
}

// A seeded rule inserted already approved (no snapshot row) keeps serving its text while an edit waits.
func TestRulesV6_SeededApprovedRowKeepsServing(t *testing.T) {
	e := rulesEnv(t)
	e.Exec(`INSERT INTO message_rules (id, rule_key, title, level, kind, priority, placement, content, conditions, exceptions,
		status, is_active, version, approved_version, created_at, updated_at)
		VALUES (70, 'seed.tip', 'seed', 'info', 'clinical', 300, 'smart_tip', '{"a":{"title":"t","body":"seeded"}}', '[]', '[]',
		'approved', 1, 1, 1, NOW(), NOW())`)
	edit := clinicalRule()
	edit["level"], edit["priority"], edit["rule_key"] = "info", 300, "seed.tip"
	edit["content"] = map[string]any{"a": map[string]any{"title": "t", "body": "edited"}}
	up := e.As(medicalID).JSON(fiber.MethodPut, "/messages/rules/70", edit)
	require.Equal(t, 200, up.Status, up.Body)
	assert.Equal(t, "pending_approval", up.Obj("rule")["status"])
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM message_rule_versions WHERE rule_id = 70"), "v1 snapshot backfilled + v2")
	body, _ := servedBody(t, e, 70)
	assert.Equal(t, "seeded", body)

	// The key is unique.
	other := clinicalRule()
	other["rule_key"] = "seed.tip"
	dup := e.As(medicalID).JSON(fiber.MethodPost, "/messages/rules", other)
	assert.Equal(t, 422, dup.Status)
	assert.Contains(t, dup.Errors(), "rule_key")
}

func safetyRule() map[string]any {
	return map[string]any{
		"title": "BP", "level": "urgent", "type": "safety", "priority": 1000, "slot": "urgent_modal",
		"content": map[string]any{"a": map[string]any{"title": "t", "body": "b"}, "seen": "s", "confidence": "c",
			"contact": "۱۱۵"},
		"conditions": []any{}, "exceptions": []any{},
	}
}

// SPEC §5.8: a messages-only admin cannot author safety / clinical / content rules or toggle / delete safety ones;
// the clinical capability can; a locked rule cannot be switched off or deleted.
func TestRulesV6_AuthorityAndLocked(t *testing.T) {
	e := rulesEnv(t)
	mk, med := e.As(marketingID), e.As(medicalID)

	assert.Equal(t, 403, mk.JSON(fiber.MethodPost, "/messages/rules", safetyRule()).Status)
	assert.Equal(t, 403, mk.JSON(fiber.MethodPost, "/messages/rules", clinicalRule()).Status)
	s := med.JSON(fiber.MethodPost, "/messages/rules", safetyRule())
	require.Equal(t, 201, s.Status, s.Body)
	rule := s.Obj("rule")
	assert.Equal(t, "safety", rule["kind"])
	assert.Equal(t, "safety", rule["type"], "spec alias returned")
	assert.Equal(t, "urgent_modal", rule["slot"])
	assert.Equal(t, "BP", rule["name"])
	assert.Equal(t, "follow_soon", rule["level"], "spec level urgent stored as follow_soon")
	assert.Equal(t, "urgent", rule["level_name"])
	sp := "/messages/rules/" + itoa(int(idOf(s)))
	e.Exec("UPDATE message_rules SET status = 'approved', approved_version = version WHERE id = ?", idOf(s))

	edit := safetyRule()
	edit["title"] = "BP 2"
	assert.Equal(t, 403, mk.JSON(fiber.MethodPut, sp, edit).Status)
	assert.Equal(t, 403, mk.JSON(fiber.MethodPost, sp+"/toggle", nil).Status)
	assert.Equal(t, 403, mk.JSON(fiber.MethodDelete, sp, nil).Status)
	assert.Equal(t, 200, med.JSON(fiber.MethodPut, sp, edit).Status)
	assert.Equal(t, 200, med.JSON(fiber.MethodPost, sp+"/toggle", nil).Status)

	// product → clinical needs the clinical capability.
	p := mk.JSON(fiber.MethodPost, "/messages/rules", productRule())
	require.Equal(t, 201, p.Status, p.Body)
	pp := "/messages/rules/" + itoa(int(idOf(p)))
	toClinical := clinicalRule()
	assert.Equal(t, 403, mk.JSON(fiber.MethodPut, pp, toClinical).Status)
	assert.Equal(t, 200, med.JSON(fiber.MethodPut, pp, toClinical).Status)
	assert.Equal(t, 403, mk.JSON(fiber.MethodPut, pp, productRule()).Status, "editing a clinical rule")

	// Locked: switching off and deleting are 409; switching on is allowed.
	e.Exec("UPDATE message_rules SET locked = 1 WHERE id = ?", idOf(s))
	assert.Equal(t, "locked", med.JSON(fiber.MethodPost, sp+"/toggle", nil).Code())
	assert.Equal(t, "locked", med.JSON(fiber.MethodDelete, sp, nil).Code())
	e.Exec("UPDATE message_rules SET is_active = 0 WHERE id = ?", idOf(s))
	assert.Equal(t, 200, med.JSON(fiber.MethodPost, sp+"/toggle", nil).Status)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_rules WHERE id = ?", idOf(s)))

	// Every action keeps writing the admin audit row.
	assert.Positive(t, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'message_rule.toggle'"))
	assert.Positive(t, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'message_rule.update'"))
}

// The v6 fields round-trip and are validated per field.
func TestRulesV6_NewFieldsRoundTrip(t *testing.T) {
	e := rulesEnv(t)
	mk := e.As(marketingID)
	in := productRule()
	delete(in, "cap_count")
	delete(in, "cap_period")
	in["placement"] = "daily_challenge"
	in["caps"] = []any{map[string]any{"count": 1, "period": "day"}, map[string]any{"count": 3, "period": "week"}}
	in["variants"] = []any{map[string]any{"key": "a", "weight": 60}, map[string]any{"key": "c", "weight": 40}}
	in["content"] = map[string]any{"a": map[string]any{"title": "t", "body": "b", "cta_label": "Go", "cta_target": "ritme://x"},
		"variants": map[string]any{"c": map[string]any{"title": "t3", "body": "b3", "cta_label": "Go", "cta_target": "ritme://x"}}}
	in["starts_at"], in["ends_at"] = "2026-11-01 08:00:00", "2026-12-01 00:00:00"
	in["hour_windows"] = []any{[]any{8, 12}, []any{18, 22}}
	in["goal_event"] = "log_saved"
	in["topics"] = []any{}
	r := mk.JSON(fiber.MethodPost, "/messages/rules", in)
	require.Equal(t, 201, r.Status, r.Body)
	rule := r.Obj("rule")
	assert.Len(t, rule["caps"], 2)
	assert.Len(t, rule["variants"], 2)
	assert.Equal(t, "2026-11-01 08:00:00", rule["starts_at"])
	assert.Equal(t, "log_saved", rule["goal_event"])
	assert.Equal(t, false, rule["locked"])

	bad := productRule()
	bad["goal_event"] = "nope"
	bad["hour_windows"] = []any{[]any{8, 8}}
	bad["caps"] = []any{map[string]any{"count": 0, "period": "day"}}
	res := mk.JSON(fiber.MethodPost, "/messages/rules", bad)
	assert.Equal(t, 422, res.Status)
	for _, f := range []string{"goal_event", "hour_windows", "caps"} {
		assert.Contains(t, res.Errors(), f)
	}
}
