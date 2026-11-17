package http

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// «ریتمی همراه» v2: create with preferences, the partial PUT /preferences (owner only, 404 for a foreign id), the
// member's settings and the home day signals (null unless shared; a coarse boolean otherwise), and gender unspecified.
func TestCompanionV2_PreferencesAndSignals(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120007001") // owner
	ali := e.user("Ali", "09120007002")   // male companion
	nina := e.user("Nina", "09120007003") // another owner
	saraID := e.ids["Sara"]

	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(sara, "goal", map[string]any{"goal": "cycle"})
	e.step(sara, "cycle", map[string]any{"last_period_start": "2026-09-05", "period_duration": 5, "cycle_duration": 28})
	e.step(ali, "gender", map[string]any{"gender": "male"})

	// Gender unspecified is accepted and is the woman path (goal still required to finish).
	e.step(nina, "gender", map[string]any{"gender": "unspecified"})
	r := e.do(fiber.MethodPost, "/api/v1/onboarding/complete", nina, nil)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Raw, `"goal"`)
	r = e.do(fiber.MethodPut, "/api/v1/onboarding/steps/gender", nina, map[string]any{"gender": "other"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)

	// 422 on bad preference values (Laravel-style, on the field).
	for body, field := range map[string]map[string]any{
		"goal":           {"type": "partner", "goal": "romance"},
		"goals.1":        {"type": "partner", "goals": []string{"cycle", "romance"}},
		"signals.pain":   {"type": "partner", "signals": map[string]any{"pain": "maybe"}},
		"signals.weight": {"type": "partner", "signals": map[string]any{"weight": true}},
		"action_prefs.0": {"type": "partner", "action_prefs": []string{"flowers"}},
	} {
		r := e.do(fiber.MethodPost, "/api/v1/companions", sara, field)
		assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, body+" "+r.Raw)
		assert.Contains(t, r.Raw, `"`+body+`"`, r.Raw)
	}

	// Create without preferences → defaults.
	e.plus(sara)
	r = e.do(fiber.MethodPost, "/api/v1/companions", sara, map[string]any{"type": "partner"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	comp := r.data()["companion"].(map[string]any)
	assert.Equal(t, "unknown", comp["goal"])
	assert.Equal(t, []any{}, comp["goals"])
	assert.Equal(t, map[string]any{"pain": false, "mood": false, "energy": false, "sleep": false}, comp["signals"])
	assert.Equal(t, []any{}, comp["action_prefs"])
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodDelete, fmt.Sprintf("/api/v1/companions/%d", idOf(comp["id"])), sara, nil).Status)

	// Create with preferences: deduped actions in the chosen order.
	link, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"cycle": "view"},
		"goals": []string{"support", "quality_time", "support"}, "signals": map[string]any{"pain": true, "sleep": true},
		"action_prefs": []string{"calm", "food", "calm"}})
	r = e.do(fiber.MethodGet, fmt.Sprintf("/api/v1/companions/%d", link), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "support", r.data()["goal"], "the first goal, for clients older than the multi-select")
	assert.Equal(t, []any{"support", "quality_time"}, r.data()["goals"])
	assert.Equal(t, map[string]any{"pain": true, "mood": false, "energy": false, "sleep": true}, r.data()["signals"])
	assert.Equal(t, []any{"calm", "food"}, r.data()["action_prefs"])

	r = e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	settings := map[string]any{"goal": "support", "goals": []any{"support", "quality_time"}, "action_prefs": []any{"calm", "food"}}
	assert.Equal(t, settings, r.data()["settings"], r.Raw)
	assert.NotContains(t, r.data(), "signals", "the member never sees the consents themselves")

	// Home without any log today: shared signals are false, unshared null.
	r = e.home(ali)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	p := partnersOf(r)[0].(map[string]any)
	assert.Equal(t, settings, p["settings"])
	assert.Equal(t, map[string]any{"pain": false, "energy": nil, "sleep": false}, p["signals"], r.Raw)

	// Today's log: pain + a low mood + short sleep. The pain score never leaves; mood never does (SPEC §1.7).
	now, err := time.Parse(time.RFC3339, companionNow) // the request clock of every e.do
	require.NoError(t, err)
	today := civildate.InTehran(now).String()
	_, err = e.db.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, created_at, updated_at) VALUES
		(?, ?, 'pain', 'location', 'abdomen', 'severe', 8, NOW(), NOW()),
		(?, ?, 'mood', 'moods', 'sad', 'yes', NULL, NOW(), NOW()),
		(?, ?, 'sleep', 'duration', '', '3_6', NULL, NOW(), NOW()),
		(?, '2020-01-01', 'appetite_energy', 'energy', '', 'low', NULL, NOW(), NOW())`,
		saraID, today, saraID, today, saraID, today, saraID)
	require.NoError(t, err)
	r = e.home(ali)
	p = partnersOf(r)[0].(map[string]any)
	assert.Equal(t, map[string]any{"pain": true, "energy": nil, "sleep": true}, p["signals"], r.Raw)
	for _, hidden := range []string{"abdomen", "severe", `"mood"`, "sad"} {
		assert.NotContains(t, r.Raw, hidden)
	}

	// Partial PUT: only the sent keys change; audited without values.
	before := e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND action = 'grants_changed'`, saraID)
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara,
		map[string]any{"signals": map[string]any{"mood": true, "pain": false}, "action_prefs": []string{"help"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"support", "quality_time"}, r.data()["goals"], "goals not sent → unchanged")
	assert.Equal(t, map[string]any{"pain": false, "mood": true, "energy": false, "sleep": true}, r.data()["signals"])
	assert.Equal(t, []any{"help"}, r.data()["action_prefs"])
	assert.Equal(t, before+1, e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND action = 'grants_changed'`, saraID))
	r = e.home(ali)
	p = partnersOf(r)[0].(map[string]any)
	assert.Equal(t, map[string]any{"pain": nil, "energy": nil, "sleep": true}, p["signals"], "share_mood on: still no mood")
	assert.NotContains(t, r.Raw, `"mood"`)

	// The single goal of an older client replaces the list; an empty list clears it.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara, map[string]any{"goal": "low_energy"})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, []any{"low_energy"}, r.data()["goals"])
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara, map[string]any{"goals": []string{}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "unknown", r.data()["goal"])
	assert.Equal(t, []any{}, r.data()["goals"])

	// 422 on PUT, 404 for anyone else (the member, another owner), an unknown id.
	r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), sara, map[string]any{"goal": "x"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	for _, tok := range []string{ali, nina} {
		r = e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/preferences", link), tok, map[string]any{"goal": "info"})
		assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)
	}
	r = e.do(fiber.MethodPut, "/api/v1/companions/999999/preferences", sara, map[string]any{"goal": "info"})
	assert.Equal(t, fiber.StatusNotFound, r.Status, r.Raw)

	// The cycle section decided away → every signal null (the signals ride on the cycle read).
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPut, fmt.Sprintf("/api/v1/companions/%d/grants", link), sara,
		map[string]any{"grants": map[string]any{}}).Status)
	p = partnersOf(e.home(ali))[0].(map[string]any)
	assert.Equal(t, map[string]any{"pain": nil, "energy": nil, "sleep": nil}, p["signals"])
}
