package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CB-TEEN-01: the teen routes and the parent link on bloom's companion routes, through the production registry.

func (e *companionEnv) teenMode(name string) {
	e.t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, 'female', 'teen', NOW(), NOW())`, e.ids[name])
	require.NoError(e.t, err)
}

func TestTeenRoutes_ProfileKitNoteAndIsolation(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120003001")
	sara := e.user("Sara", "09120003002")
	e.teenMode("Nila")
	e.teenMode("Sara")

	for _, rt := range [][2]string{{fiber.MethodGet, "/api/v1/teen/profile"}, {fiber.MethodPut, "/api/v1/teen/profile"},
		{fiber.MethodGet, "/api/v1/teen/today"}, {fiber.MethodPut, "/api/v1/teen/kit/pads"},
		{fiber.MethodPut, "/api/v1/teen/parent-note"}, {fiber.MethodGet, "/api/v1/teen/linked"}} {
		r := e.do(rt[0], rt[1], "", map[string]any{})
		assert.Equal(t, fiber.StatusUnauthorized, r.Status, rt[1])
	}

	r := e.do(fiber.MethodGet, "/api/v1/teen/profile", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Nil(t, r.data()["profile"])
	assert.Equal(t, true, r.data()["needs_onboarding"])
	assert.Equal(t, true, r.data()["is_teen_mode"])
	assert.Equal(t, map[string]any{"shop": false, "banners": false, "ads": false, "plus_upsell": false, "commercial_recommendations": false}, r.data()["allows"])

	r = e.do(fiber.MethodPut, "/api/v1/teen/profile", nila, map[string]any{"age_band": "18_99", "menarche": "yes"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Raw, `"age_band"`)
	assert.Contains(t, r.Raw, `"menarche"`)

	r = e.do(fiber.MethodPut, "/api/v1/teen/parent-note", nila, map[string]any{"note": "hi"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Equal(t, "teen_profile_required", r.Body["error_code"])

	r = e.do(fiber.MethodPut, "/api/v1/teen/profile", nila, map[string]any{"age_band": "13_15", "menarche": "not_yet"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/teen/parent-note", nila, map[string]any{"note": fmt.Sprintf("%0281d", 0)})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "note max 280")
	r = e.do(fiber.MethodPut, "/api/v1/teen/parent-note", nila, map[string]any{"note": "Need pads"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)

	r = e.do(fiber.MethodPut, "/api/v1/teen/kit/lipstick", nila, map[string]any{"checked": true})
	assert.Equal(t, fiber.StatusNotFound, r.Status)
	r = e.do(fiber.MethodPut, "/api/v1/teen/kit/pads", nila, map[string]any{})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	r = e.do(fiber.MethodPut, "/api/v1/teen/kit/pads", nila, map[string]any{"checked": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.EqualValues(t, 1, r.data()["checked_count"])

	r = e.do(fiber.MethodGet, "/api/v1/teen/today", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	readiness, _ := r.data()["readiness"].(map[string]any)
	assert.Equal(t, "estimate_coming_months", readiness["code"])
	preview, _ := r.data()["parent_preview"].(map[string]any)
	assert.Equal(t, "unknown", preview["next_period_week"], "no first period yet")
	assert.Equal(t, "Need pads", preview["note"])

	// Sara (teen B) sees none of Nila's rows.
	r = e.do(fiber.MethodGet, "/api/v1/teen/today", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Nil(t, r.data()["profile"])
	kit, _ := r.data()["kit"].(map[string]any)
	assert.EqualValues(t, 0, kit["checked_count"])
	preview, _ = r.data()["parent_preview"].(map[string]any)
	assert.Nil(t, preview["note"])
}

func TestTeenRoutes_ParentLinkRules(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120003101")   // teen A
	sara := e.user("Sara", "09120003102")   // teen B
	mom := e.user("Mom", "09120003103")     // Nila's mother
	adult := e.user("Adult", "09120003104") // an adult owner (not teen)
	stranger := e.user("Stranger", "09120003105")
	e.teenMode("Nila")
	e.teenMode("Sara")

	// Type rules.
	r := e.do(fiber.MethodPost, "/api/v1/companions", nila, map[string]any{"type": "parent"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Raw, `"phone"`, "a parent invite needs the parent's number")
	r = e.do(fiber.MethodPost, "/api/v1/companions", adult, map[string]any{"type": "parent", "phone": "09120003103"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Equal(t, "parent_needs_teen", r.Body["error_code"])
	r = e.do(fiber.MethodPost, "/api/v1/companions", nila, map[string]any{"type": "partner"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Equal(t, "teen_parent_only", r.Body["error_code"])
	r = e.do(fiber.MethodPost, "/api/v1/companions", nila, map[string]any{"type": "parent", "phone": "09120003103", "grants": map[string]any{"cycle": "view"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	assert.Contains(t, r.Raw, `"grants.cycle"`)
	r = e.do(fiber.MethodPost, "/api/v1/companions", nila, map[string]any{"type": "parent", "phone": "09120003103", "grants": map[string]any{"teen_notes": "edit"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "a parent never writes")
	r = e.do(fiber.MethodPost, "/api/v1/companions", adult, map[string]any{"type": "partner", "grants": map[string]any{"teen_kit": "view"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "teen sections are parent-only")

	// Nila invites her mother sharing only the period week; mom accepts.
	id, code := e.invite(nila, map[string]any{"type": "parent", "display_name": "Mom", "phone": "09120003103", "grants": map[string]any{"teen_period_week": "view"}})
	r = e.do(fiber.MethodPost, "/api/v1/companions/accept", mom, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, map[string]any{"teen_period_week": "view", "teen_kit": "none", "teen_notes": "none"}, r.data()["grants"])
	assert.Equal(t, []any{}, r.data()["can_record_for"])

	// The parent's only read is the card; no section reads, no care writes for Nila.
	for _, s := range []string{"cycle", "symptoms", "meds", "appointments", "pregnancy", "teen_period_week", "teen_kit", "teen_notes"} {
		r = e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/links/%d/sections/%s", id, s), mom, nil)
		assert.Equal(t, fiber.StatusNotFound, r.Status, s)
	}
	r = e.do(fiber.MethodPost, "/api/v1/care/medications", mom, medBody(map[string]any{"for_user_id": e.ids["Nila"]}))
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	r = e.do(fiber.MethodPost, "/api/v1/care/medications", nila, medBody(nil))
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	medID := idOf(r.data()["id"])
	r = e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d?for_user_id=%d", medID, e.ids["Nila"]), mom, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	r = e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/care/medications/%d", medID), mom, nil)
	assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)

	r = e.do(fiber.MethodGet, "/api/v1/teen/linked", mom, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	require.Len(t, r.list(), 1)
	card, _ := r.list()[0].(map[string]any)
	assert.EqualValues(t, id, card["link_id"])
	assert.Equal(t, "unknown", card["next_period_week"])
	assert.Nil(t, card["kit_ready"])
	assert.Nil(t, card["note"])
	assert.Equal(t, true, card["read_only"])

	// Nobody else gets a card: a stranger, the other teen, the teen herself.
	for _, tok := range []string{stranger, sara, nila} {
		r = e.do(fiber.MethodGet, "/api/v1/teen/linked", tok, nil)
		require.Equal(t, fiber.StatusOK, r.Status)
		assert.Empty(t, r.list())
	}

	// Grants: a parent link keeps to the teen sections; the teen widens and the card follows at once.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", id), nila, map[string]any{"grants": map[string]any{"symptoms": "view"}})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status)
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", id), sara, map[string]any{"grants": map[string]any{"teen_notes": "view"}})
	assert.Equal(t, fiber.StatusNotFound, r.Status, "teen B cannot change teen A's link")
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", id), mom, map[string]any{"grants": map[string]any{"teen_notes": "view"}})
	assert.Equal(t, fiber.StatusNotFound, r.Status, "the parent cannot widen her own access")
	r = e.do(fiber.MethodPut, "/api/v1/teen/profile", nila, map[string]any{"age_band": "13_15", "menarche": "over_1y"})
	require.Equal(t, fiber.StatusOK, r.Status)
	r = e.do(fiber.MethodPut, "/api/v1/teen/parent-note", nila, map[string]any{"note": "Cramps today"})
	require.Equal(t, fiber.StatusOK, r.Status)
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", id), nila, map[string]any{"grants": map[string]any{"teen_notes": "view", "teen_kit": "view"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodGet, "/api/v1/teen/linked", mom, nil)
	card, _ = r.list()[0].(map[string]any)
	assert.Nil(t, card["next_period_week"], "un-granted at once")
	assert.Equal(t, false, card["kit_ready"])
	assert.Equal(t, "Cramps today", card["note"])

	// The audit trail shows the parent's reads (no payload).
	assert.Positive(t, e.count("SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND actor_id = ? AND action = 'read' AND section LIKE 'teen_%'", e.ids["Nila"], e.ids["Mom"]))

	// The teen ends it: nothing left.
	r = e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", id), nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	r = e.do(fiber.MethodGet, "/api/v1/teen/linked", mom, nil)
	assert.Empty(t, r.list())
}

func TestTeenRoutes_NoBannersOrPlusUpsellForTeen(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120003201")
	adult := e.user("Adult", "09120003202")
	e.teenMode("Nila")
	_, err := e.db.Exec(`INSERT INTO banners (title, image_path, position, is_active, sort_order, created_at, updated_at)
		VALUES ('{"fa":"تخفیف","en":"Sale"}', 'banners/a.png', 'home_top', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)

	r := e.do(fiber.MethodGet, "/api/v1/banners", adult, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	pos, _ := r.data()["positions"].(map[string]any)
	assert.Len(t, pos["home_top"], 1)

	r = e.do(fiber.MethodGet, "/api/v1/banners", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	pos, _ = r.data()["positions"].(map[string]any)
	for slot, items := range pos {
		assert.Empty(t, items, "no banners for a teen (%s)", slot)
	}

	r = e.do(fiber.MethodGet, "/api/v1/home/cycle-overview", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Nil(t, r.data()["plus_trial_offer"])
}

// L1 over HTTP: a partner invite of an owner who switched to teen mode afterwards is a clear 422 on accept.
func TestTeenRoutes_AcceptAfterModeChange(t *testing.T) {
	e := newCompanionEnv(t)
	owner := e.user("Owner", "09120003301")
	ali := e.user("Ali", "09120003302")
	_, code := e.invite(owner, map[string]any{"type": "partner", "grants": map[string]any{"cycle": "view"}})
	e.teenMode("Owner")
	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Equal(t, "invite_not_allowed", r.Body["error_code"])
	assert.Zero(t, e.count("SELECT COUNT(*) FROM companions WHERE status = 'active'"))
}

// Plus (M2): no trial offer, trial or checkout for a teen-mode account.
func TestTeenRoutes_NoPlusForTeen(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120003401")
	adult := e.user("Adult", "09120003402")

	_, err := e.db.Exec(`INSERT INTO plus_plans (code, title, badge, duration_months, price_rials, monthly_display_rials,
		is_highlighted, is_active, sort_order, created_at, updated_at)
		VALUES ('yearly', '{"fa":"سالانه","en":"Yearly"}', NULL, 12, 1000000, 83333, 1, 1, 1, NOW(), NOW())`)
	require.NoError(t, err) // an active plan, so the trial offer (default percent) runs
	// Both start the trial as adults (the offer runs with it); then Nila switches to teen mode.
	for _, tok := range []string{nila, adult} {
		r := e.do(fiber.MethodPost, "/api/v1/plus/trial/start", tok, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	}
	// This env has no Plus config (0 trial days): give both trials a week so the offer runs.
	_, err = e.db.Exec(`UPDATE plus_trials SET ends_at = '2026-09-30 10:00:00'`)
	require.NoError(t, err)
	e.teenMode("Nila")

	for _, path := range []string{"/api/v1/plus/trial/start", "/api/v1/plus/checkout"} {
		r := e.do(fiber.MethodPost, path, nila, map[string]any{"plan_id": 1, "preview": true})
		assert.Equal(t, fiber.StatusForbidden, r.Status, path)
		assert.Equal(t, "teen_commercial_blocked", r.Body["error_code"], path)
	}
	for path, key := range map[string]string{"/api/v1/plus/status": "trial_offer", "/api/v1/plus/trial": "offer"} {
		r := e.do(fiber.MethodGet, path, adult, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		assert.NotNil(t, r.data()[key], "adult %s", path)
		r = e.do(fiber.MethodGet, path, nila, nil)
		require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
		require.Contains(t, r.data(), key)
		assert.Nil(t, r.data()[key], "teen %s", path)
	}
	r := e.do(fiber.MethodGet, "/api/v1/plus/trial", nila, nil)
	plans, _ := r.data()["plans"].([]any)
	require.NotEmpty(t, plans)
	assert.Nil(t, plans[0].(map[string]any)["offer_price"], "no offer prices for a teen")
}
