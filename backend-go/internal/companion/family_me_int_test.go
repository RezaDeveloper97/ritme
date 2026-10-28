package companion_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
)

// GET /api/v1/family/me (T-M11-05, D-204): the owner sees every grant with its state, the member only the effective
// domains; a mode change pauses the owner-data grants (owner: paused, member: gone); expired links vanish for the
// member; nobody sees another account's links; never cached.
func TestFamilyMe(t *testing.T) {
	e := newPrivacyEnv(t)
	owner, otok := e.account("09120095301", "female", "cycle")
	member, mtok := e.account("09120095302", "male", "")
	_, stranger := e.account("09120095303", "male", "")
	link := e.link(owner, member, "2026-09-20 09:00:00", companion.Grants{
		companion.SectionCycle: companion.LevelView, companion.SectionMeds: companion.LevelEdit, companion.SectionPregnancy: companion.LevelView})

	r := e.do(fiber.MethodGet, "/api/v1/family/me", "", nil)
	assert.Equal(t, fiber.StatusUnauthorized, r.Status)

	r = e.do(fiber.MethodGet, "/api/v1/family/me", otok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "private, no-store", r.Header)
	owned := r.data()["as_owner"].([]any)
	require.Len(t, owned, 1)
	l := owned[0].(map[string]any)
	assert.EqualValues(t, link, l["id"])
	assert.Equal(t, "partner", l["role"])
	assert.Equal(t, "active", l["status"])
	states := map[string]string{}
	for _, g := range l["grants"].([]any) {
		m := g.(map[string]any)
		states[m["domain"].(string)+":"+m["level"].(string)] = m["state"].(string)
	}
	assert.Equal(t, map[string]string{"cycle_phase:view": "active", "meds:write": "active", "pregnancy:view": "inactive"}, states)
	assert.Empty(t, r.data()["as_member"])

	r = e.do(fiber.MethodGet, "/api/v1/family/me", mtok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	mine := r.data()["as_member"].([]any)
	require.Len(t, mine, 1)
	m := mine[0].(map[string]any)
	assert.EqualValues(t, owner, m["owner"].(map[string]any)["id"])
	assert.Equal(t, []any{
		map[string]any{"domain": "cycle_phase", "level": "view"},
		map[string]any{"domain": "meds", "level": "write"},
	}, m["domains"], "effective only: no pregnancy outside the pregnancy mode")
	assert.NotContains(t, r.Raw, "paused")
	assert.NotContains(t, r.Raw, "inactive")

	r = e.do(fiber.MethodGet, "/api/v1/family/me", stranger, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["as_owner"])
	assert.Empty(t, r.data()["as_member"])

	// The owner moves to TTC: every owner-data grant pauses until she reviews (F-23).
	e.exec(`UPDATE user_life_profiles SET life_mode = NULL WHERE user_id = ?`, owner)
	e.exec(`INSERT INTO user_profiles (user_id, user_goal, created_at, updated_at) VALUES (?, 'ttc', NOW(), NOW())`, owner)
	r = e.do(fiber.MethodGet, "/api/v1/family/me", mtok, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["as_member"].([]any)[0].(map[string]any)["domains"])
	r = e.do(fiber.MethodGet, "/api/v1/family/me", otok, nil)
	l = r.data()["as_owner"].([]any)[0].(map[string]any)
	assert.Equal(t, true, l["review_pending"])
	for _, g := range l["grants"].([]any) {
		gm := g.(map[string]any)
		if gm["domain"] != "pregnancy" {
			assert.Equal(t, "paused", gm["state"], gm["domain"])
		}
	}
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, section(link, "meds"), mtok, nil).Status)

	// An expired link is not listed for the member any more (no expiry sweep needed).
	e.exec(`UPDATE companions SET expires_at = '2026-09-01 09:00:00' WHERE id = ?`, link)
	r = e.do(fiber.MethodGet, "/api/v1/family/me", mtok, nil)
	assert.Empty(t, r.data()["as_member"])
	r = e.do(fiber.MethodGet, "/api/v1/family/me", otok, nil)
	assert.Equal(t, "expired", r.data()["as_owner"].([]any)[0].(map[string]any)["status"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, section(link, "cycle"), mtok, nil).Status, "expired → 404 (F-21)")
}
