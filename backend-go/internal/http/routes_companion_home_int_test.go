package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// step answers one onboarding v2 step as token.
func (e *companionEnv) step(token, name string, body map[string]any) {
	e.t.Helper()
	r := e.do(fiber.MethodPut, "/api/v1/onboarding/steps/"+name, token, body)
	require.Equal(e.t, fiber.StatusOK, r.Status, r.Raw)
}

func (e *companionEnv) home(token string) reply {
	e.t.Helper()
	return e.do(fiber.MethodGet, "/api/v1/companion/home", token, nil)
}

func partnersOf(r reply) []any {
	p, _ := r.data()["partners"].([]any)
	return p
}

// Bloom B-N4-03: the male account path (409 on the women's cycle endpoints, companion life mode) and the companion
// home for every access combination — no link, granted / ungranted sections, admin tips, revoked link, women's
// accounts refused.
func TestCompanionHome_AccessCombinations(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120003001") // owner, cycling woman
	ali := e.user("Ali", "09120003002")   // male companion
	nina := e.user("Nina", "09120003003") // woman, companion of Sara
	legacy := e.user("Legacy", "09120003004")
	reza := e.user("Reza", "09120003005") // another male companion, linked to nobody
	saraID := e.ids["Sara"]

	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(sara, "goal", map[string]any{"goal": "cycle"})
	e.step(sara, "cycle", map[string]any{"last_period_start": "2026-09-05", "period_duration": 5, "cycle_duration": 28})
	e.step(ali, "gender", map[string]any{"gender": "male"})
	r := e.do(fiber.MethodPost, "/api/v1/onboarding/complete", ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	e.step(nina, "gender", map[string]any{"gender": "female"})
	e.step(reza, "gender", map[string]any{"gender": "male"})

	// 401 without a token.
	assert.Equal(t, fiber.StatusUnauthorized, e.home("").Status)

	// Women's (and never-asked) accounts: 403 not_companion_account.
	for _, tok := range []string{sara, nina, legacy} {
		r := e.home(tok)
		assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
		assert.Equal(t, "not_companion_account", r.Body["error_code"])
	}

	// The male path: companion mode, no cycle engine on the women's endpoints.
	r = e.do(fiber.MethodGet, "/api/v1/profile/life-stage", ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "companion", r.data()["mode"])
	r = e.do(fiber.MethodPut, "/api/v1/profile/life-stage", ali, map[string]any{"mode": "pregnancy"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Raw, `"mode"`)
	for _, path := range []string{"/api/v1/home", "/api/v1/home/cycle-overview", "/api/v1/home/sections/articles", "/api/v1/messages/daily", "/api/v1/messages/mode"} {
		r := e.do(fiber.MethodGet, path, ali, nil)
		assert.Equal(t, fiber.StatusConflict, r.Status, path+" "+r.Raw)
		assert.Equal(t, "companion_account", r.Body["error_code"], path)
		assert.Equal(t, "/companion", r.Body["home"], path)
	}
	r = e.do(fiber.MethodGet, "/api/v1/auth/user", ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["profile_completed"], "a finished male onboarding is complete")
	// A woman is untouched.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, "/api/v1/home/cycle-overview", sara, nil).Status)
	assert.Equal(t, "cycle", e.do(fiber.MethodGet, "/api/v1/profile/life-stage", sara, nil).data()["mode"])

	// No link yet: the empty state pointing to the code entry.
	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "companion", r.data()["mode"])
	assert.Equal(t, false, r.data()["has_partners"])
	assert.Empty(t, partnersOf(r))
	empty, _ := r.data()["empty_state"].(map[string]any)
	require.NotNil(t, empty)
	assert.Equal(t, "enter_code", empty["action"])
	assert.Equal(t, "Ali", r.data()["viewer"].(map[string]any)["name"])

	// A pending invite shows nothing.
	link, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"cycle": "view", "meds": "edit"}})
	assert.Empty(t, partnersOf(e.home(ali)))

	// Accepted with cycle (view) and meds (edit): those two views, the rest null; each read audited.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)
	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["has_partners"])
	assert.Nil(t, r.data()["empty_state"])
	ps := partnersOf(r)
	require.Len(t, ps, 1)
	p := ps[0].(map[string]any)
	assert.Equal(t, "Sara", p["partner_name"])
	assert.Equal(t, float64(saraID), p["link"].(map[string]any)["owner"].(map[string]any)["id"])
	cyc, _ := p["cycle"].(map[string]any)
	require.NotNil(t, cyc, r.Raw)
	assert.Equal(t, true, cyc["has_data"])
	assert.NotNil(t, cyc["predicted_next_period_start"])
	assert.NotNil(t, p["meds"], "granted meds is a list")
	assert.Nil(t, p["appointments"])
	assert.Nil(t, p["symptoms"])
	assert.Nil(t, p["pregnancy"])
	assert.Nil(t, p["child"])
	assert.Contains(t, []any{"menstrual", "follicular", "fertile", "luteal"}, p["phase"])
	assert.Len(t, p["tips"], 3)
	assert.Contains(t, p["note"], "Sara", "the note names the partner")
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND action = 'read' AND section = 'cycle'`, saraID, e.ids["Ali"]))
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND action = 'read' AND section = 'meds'`, saraID, e.ids["Ali"]))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND action = 'read' AND section IN ('symptoms','appointments','pregnancy')`, saraID))

	// Another male account sees none of it (IDOR).
	r = e.home(reza)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, partnersOf(r))
	assert.NotContains(t, r.Raw, "Sara")

	// Admin tips: a live row replaces the embedded copy; an empty title hides the slot.
	_, err := e.db.Exec(`INSERT INTO message_contents (` + "`group`" + `, item_key, locale, payload, is_active, is_approved, sort_order, created_at, updated_at) VALUES
		('companion_tip', 'general_tip_1', 'en', '{"title":"Text {name} a kind word","body":"Admin body"}', 1, 1, 0, NOW(), NOW()),
		('companion_tip', 'general_tip_2', 'en', '{"title":"","body":"hidden"}', 1, 1, 0, NOW(), NOW())`)
	require.NoError(t, err)

	// No grant at all: no data, general phase, no new audit rows.
	before := e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read'`)
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", link), sara, map[string]any{"grants": map[string]any{}}).Status)
	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	p = partnersOf(r)[0].(map[string]any)
	for _, k := range []string{"cycle", "pregnancy", "symptoms", "meds", "appointments"} {
		assert.Nil(t, p[k], k)
	}
	assert.Equal(t, "general", p["phase"])
	tips := p["tips"].([]any)
	require.Len(t, tips, 2, "general_tip_2 hidden by its empty admin title")
	assert.Equal(t, "Text Sara a kind word", tips[0].(map[string]any)["title"])
	assert.Equal(t, "Admin body", tips[0].(map[string]any)["body"])
	assert.Equal(t, before, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE action = 'read'`))
	assert.NotContains(t, r.Raw, "2026-09-05", "no cycle data leaks without a grant")

	// Revoked by the owner: the link and its data disappear, the empty state is back.
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", link), sara, nil).Status)
	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, partnersOf(r))
	assert.Equal(t, false, r.data()["has_partners"])
	assert.NotNil(t, r.data()["empty_state"])
}
