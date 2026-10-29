package messages_test

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/messages"
)

const (
	marketingID uint64 = 3
	medicalID   uint64 = 4
)

func rulesEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	messages.New(e.DB, admintest.Quiet).Routes(e.Route(), e.Kit)
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Mark', 'mkt@ritme.test', ?, 'marketing', 1, NOW(), NOW()),
		(4, 'Dr', 'med@ritme.test', ?, 'medical_advisor', 1, NOW(), NOW())`, pw, pw)
	e.Exec("DELETE FROM message_rules")
	e.Exec("DELETE FROM message_events")
	return e
}

func clinicalRule() map[string]any {
	return map[string]any{
		"title": "PMS", "level": "worth_followup", "kind": "clinical", "priority": 60 + 100, "placement": "smart_tip",
		"content": map[string]any{"a": map[string]any{"title": "t {name}", "body": "b"}, "seen": "s", "confidence": "c"},
		"conditions": []any{map[string]any{"op": "and", "items": []any{
			map[string]any{"fact": "mode", "op": "in", "value": []any{"cycle"}}}}},
		"exceptions": []any{}, "cooldown_days": 7, "audience_percent": 100, "goal": "open symptoms",
	}
}

func productRule() map[string]any {
	return map[string]any{
		"title": "Log reminder", "level": "suggestion", "kind": "product", "priority": 30, "placement": "push",
		"content":    map[string]any{"a": map[string]any{"title": "t", "body": "b", "cta_label": "Log", "cta_target": "ritme://log"}},
		"conditions": []any{}, "exceptions": []any{}, "cap_count": 1, "cap_period": "day",
	}
}

func TestRules_CreateValidateList(t *testing.T) {
	e := rulesEnv(t)
	mk := e.As(marketingID)

	bad := productRule()
	bad["priority"] = 500
	r := mk.JSON(fiber.MethodPost, "/messages/rules", bad)
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "priority")

	r = mk.JSON(fiber.MethodPost, "/messages/rules", productRule())
	require.Equal(t, 201, r.Status, r.Body)
	rule := r.Obj("rule")
	assert.Equal(t, "draft", rule["status"])
	assert.Equal(t, false, rule["needs_medical_approval"])
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM message_rule_versions"))

	list := mk.Get("/messages/rules?kind=product")
	require.Equal(t, 200, list.Status)
	assert.Len(t, list.Items(), 1)
	assert.Contains(t, list.Data(), "enums")
	assert.Equal(t, 401, e.Anonymous().Get("/messages/rules").Status)
	// Static route is not shadowed by /messages/:id.
	assert.Equal(t, 200, e.As(admintest.EditorID).Get("/messages/rules").Status)
}

func TestRules_MedicalApprovalWorkflowAndInvalidation(t *testing.T) {
	e := rulesEnv(t)
	mk, med := e.As(marketingID), e.As(medicalID)

	r := med.JSON(fiber.MethodPost, "/messages/rules", clinicalRule()) // T-M10-02: clinical authoring needs the clinical capability
	require.Equal(t, 201, r.Status, r.Body)
	id := int(r.Obj("rule")["id"].(float64))
	path := "/messages/rules/" + itoa(id)
	assert.Equal(t, true, r.Obj("rule")["needs_medical_approval"])

	// Not approved: cannot be published; approving needs a pending message and the clinical capability.
	assert.Equal(t, "approval_required", mk.JSON(fiber.MethodPost, path+"/toggle", nil).Code())
	assert.Equal(t, "not_pending", med.JSON(fiber.MethodPost, path+"/approve", nil).Code())
	assert.Equal(t, 200, mk.JSON(fiber.MethodPost, path+"/submit", nil).Status)
	assert.Equal(t, 403, mk.JSON(fiber.MethodPost, path+"/approve", nil).Status, "marketing has no clinical capability")
	ok := med.JSON(fiber.MethodPost, path+"/approve", nil)
	require.Equal(t, 200, ok.Status, ok.Body)
	assert.Equal(t, "approved", ok.Obj("rule")["status"])
	assert.EqualValues(t, medicalID, ok.Obj("rule")["approved_by"])

	pub := mk.JSON(fiber.MethodPost, path+"/toggle", nil)
	require.Equal(t, 200, pub.Status)
	assert.Equal(t, true, pub.Obj("rule")["is_active"])

	// A text edit withdraws the approval of the head (pending again), bumps the version and snapshots it; the approved
	// version keeps serving (T-M10-02, M-02/M-03).
	edit := clinicalRule()
	edit["content"].(map[string]any)["a"].(map[string]any)["body"] = "changed"
	up := med.JSON(fiber.MethodPut, path, edit)
	require.Equal(t, 200, up.Status, up.Body)
	assert.Equal(t, "pending_approval", up.Obj("rule")["status"])
	assert.Nil(t, up.Obj("rule")["approved_by"])
	assert.EqualValues(t, 2, up.Obj("rule")["version"])
	assert.EqualValues(t, 1, up.Obj("rule")["approved_version"])
	assert.EqualValues(t, 1, up.Obj("rule")["served_version"])
	assert.Equal(t, 2, e.Int("SELECT COUNT(*) FROM message_rule_versions WHERE rule_id = ?", id))

	// Reject returns a pending message to draft; the approved version stays.
	rej := med.JSON(fiber.MethodPost, path+"/reject", nil)
	assert.Equal(t, "draft", rej.Obj("rule")["status"])
	assert.EqualValues(t, 1, rej.Obj("rule")["approved_version"])

	// A non-medical message is not submitted at all.
	pr := mk.JSON(fiber.MethodPost, "/messages/rules", productRule())
	pid := itoa(int(pr.Obj("rule")["id"].(float64)))
	assert.Equal(t, "approval_not_needed", mk.JSON(fiber.MethodPost, "/messages/rules/"+pid+"/submit", nil).Code())
	assert.Equal(t, 200, mk.JSON(fiber.MethodPost, "/messages/rules/"+pid+"/toggle", nil).Status)
}

func TestRules_ConflictCheckAndPreview(t *testing.T) {
	e := rulesEnv(t)
	mk := e.As(marketingID)
	other := clinicalRule()
	other["title"] = "headache"
	other["priority"] = 155
	r := e.As(medicalID).JSON(fiber.MethodPost, "/messages/rules", other)
	oid := itoa(int(r.Obj("rule")["id"].(float64)))
	e.Exec("UPDATE message_rules SET is_active = 1, status = 'approved'")

	cf := mk.JSON(fiber.MethodPost, "/messages/rules/conflicts", map[string]any{"rule": clinicalRule()})
	require.Equal(t, 200, cf.Status, cf.Body)
	list := cf.Data()["conflicts"].([]any)
	require.Len(t, list, 1)
	c := list[0].(map[string]any)
	assert.Equal(t, "headache", c["title"])
	assert.Equal(t, "this", c["winner"], "priority 160 beats 155")
	_ = oid

	// Preview with explicit facts: matches, and says why not when the condition fails.
	pv := mk.JSON(fiber.MethodPost, "/messages/rules/preview", map[string]any{
		"rule": clinicalRule(), "facts": map[string]any{"mode": "cycle", "name": "Sara"}})
	require.Equal(t, 200, pv.Status, pv.Body)
	assert.Equal(t, true, pv.Data()["eligible"])
	assert.Equal(t, "t Sara", pv.Data()["title"])
	pv = mk.JSON(fiber.MethodPost, "/messages/rules/preview", map[string]any{
		"rule": clinicalRule(), "facts": map[string]any{"mode": "pregnancy", "name": "Sara"}})
	assert.Equal(t, false, pv.Data()["eligible"])
	assert.Contains(t, pv.Data()["reasons"], "conditions")

	// A real user id: audited, and 404 for an unknown one.
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (50, 'Sara K', '09120000050', NOW(), NOW())`)
	pv = mk.JSON(fiber.MethodPost, "/messages/rules/preview", map[string]any{"rule": clinicalRule(), "user_id": 50})
	require.Equal(t, 200, pv.Status, pv.Body)
	assert.Equal(t, 404, mk.JSON(fiber.MethodPost, "/messages/rules/preview", map[string]any{"rule": clinicalRule(), "user_id": 9999}).Status)
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'message_rule.preview'"))
}

func itoa(i int) string { return strconv.Itoa(i) }

func TestRules_EngineViewAndSimulate(t *testing.T) {
	e := rulesEnv(t)
	mk := e.As(marketingID)
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at)
		SELECT 5, 'Sup', 'sup@ritme.test', password, 'support', 1, NOW(), NOW() FROM admins WHERE id = 3`)
	hi := clinicalRule()
	hi["title"] = "hi"
	lo := clinicalRule()
	lo["title"] = "lo"
	lo["priority"] = 120
	for _, b := range []map[string]any{hi, lo} {
		require.Equal(t, 201, e.As(medicalID).JSON(fiber.MethodPost, "/messages/rules", b).Status)
	}
	e.Exec("UPDATE message_rules SET is_active = 1, status = 'approved'")

	v := mk.Get("/messages/rules/engine")
	require.Equal(t, 200, v.Status, v.Body)
	assert.Contains(t, v.Data(), "placements")

	sim := func(mode string) map[string]any {
		r := mk.JSON(fiber.MethodPost, "/messages/rules/simulate", map[string]any{
			"facts": map[string]any{"mode": mode, "name": "Sara"}, "placements": []string{"smart_tip"}})
		require.Equal(t, 200, r.Status, r.Body)
		return r.Data()["placements"].([]any)[0].(map[string]any)
	}
	p := sim("cycle")
	w := p["winner"].(map[string]any)
	assert.Equal(t, "hi", w["title"])
	assert.Equal(t, "t Sara", w["message_title"])
	assert.Len(t, p["trace"], 2)
	p = sim("pregnancy")
	assert.Nil(t, p["winner"])
	assert.Equal(t, "conditions", p["trace"].([]any)[0].(map[string]any)["reason"])
	assert.Equal(t, 0, e.Int("SELECT COUNT(*) FROM message_events"))

	assert.Equal(t, 403, e.As(5).JSON(fiber.MethodPost, "/messages/rules/simulate", map[string]any{"facts": map[string]any{}}).Status)
	assert.Equal(t, 403, e.As(5).Get("/messages/rules/engine").Status)
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/messages/rules/simulate", map[string]any{"placements": []string{"nope"}}).Status)
	assert.Equal(t, 401, e.Anonymous().JSON(fiber.MethodPost, "/messages/rules/simulate", map[string]any{}).Status)
}
