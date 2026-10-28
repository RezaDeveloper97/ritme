package http

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/companion"
)

// T-M9-02 (D-133, FAM-H3) through the production router: routes_pregnancy.go wires pregnancy.Handlers.SetFamilyGrants,
// so POST /pregnancy/deactivate and /pregnancy/activate reset every pregnancy grant of the owner's links to none, with
// a grants_changed audit row and an owner notice — the next pregnancy is never shared from its first day.
func TestPregnancyRoutes_ResetPregnancyGrants(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009301")
	ali := e.user("Ali", "09120009302")
	saraID := e.ids["Sara"]
	e.step(sara, "gender", map[string]any{"gender": "female"})
	e.step(ali, "gender", map[string]any{"gender": "male"})
	r := e.do(fiber.MethodPost, "/api/v1/pregnancy/onboarding", sara, map[string]any{"age_source": "lmp", "lmp_date": "2026-08-02"})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)

	// A grant given during this pregnancy works.
	_, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"pregnancy": "view", "cycle": "view"}})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code}).Status)
	link := idOfLink(t, e, ali)
	pregSection := "/api/v1/companions/links/" + strconv.FormatUint(link, 10) + "/sections/pregnancy"
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, pregSection, ali, nil).Status)

	grants := func() int {
		return e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'pregnancy'`, link)
	}
	resets := func(reason companion.ResetReason) int {
		return e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'companion'
			AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = ? AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.reason')) = ?`,
			saraID, companion.EventPregnancySharingReset, string(reason))
	}
	audits := func() int {
		return e.count(`SELECT COUNT(*) FROM companion_audit_logs WHERE owner_id = ? AND companion_id = ? AND section = 'pregnancy'
			AND action = 'grants_changed' AND actor_id = ?`, saraID, link, saraID)
	}
	audit0 := audits()

	// Deactivate → none, audited, owner notified; the cycle grant stays.
	r = e.do(fiber.MethodPost, "/api/v1/pregnancy/deactivate", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Zero(t, grants(), "deactivate resets the pregnancy grant")
	// The cycle grant row stays (stored with its v6 code since T-M11-05)…
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companion_grants WHERE companion_id = ? AND section = 'cycle_phase'`, link))
	assert.Equal(t, audit0+1, audits())
	assert.Equal(t, 1, resets(companion.ResetPregnancyDeactivated))
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, pregSection, ali, nil).Status)
	// …but the mode change (pregnancy → cycle) paused it until she reviews (F-23): denied before, live after she
	// re-approves through the real API.
	cycleSection := "/api/v1/companions/links/" + strconv.FormatUint(link, 10) + "/sections/cycle"
	r = e.do(fiber.MethodGet, cycleSection, ali, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Nil(t, r.Body["data"])
	r = e.do(fiber.MethodPut, "/api/v1/companions/"+strconv.FormatUint(link, 10)+"/grants", sara,
		map[string]any{"grants": map[string]any{"cycle": "view"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, cycleSection, ali, nil).Status, "re-approved")
	assert.Equal(t, audit0+1, audits(), "the re-approval writes no pregnancy-section row")

	// A stale grant from before the next pregnancy (here put back by hand) is reset when she activates again.
	e.exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'pregnancy', 'view', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, link)
	r = e.do(fiber.MethodPost, "/api/v1/pregnancy/activate", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Zero(t, grants(), "activate resets the pregnancy grant")
	assert.Equal(t, audit0+2, audits())
	assert.Equal(t, 1, resets(companion.ResetPregnancyActivated))
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodGet, pregSection, ali, nil).Status,
		"the new pregnancy is not shared from its first day")
}
