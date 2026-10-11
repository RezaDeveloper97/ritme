package http

// CB-SHOP-04 over the production wiring (routes_shop_checklist.go): the real cycle prediction drives the supply
// reminder, the real pregnancy due date shows on the list, the companion pregnancy grant is the family share, and a
// teen-mode account gets no shop.

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShopChecklist_ProductionWiring(t *testing.T) {
	e := newCompanionEnv(t)
	sara, ali, _ := securityPair(t, e) // Sara: cycle mode, last period 2026-09-05 / 28 days; Ali: partner without pregnancy
	saraID := e.ids["Sara"]

	for _, p := range []string{"/api/v1/shop/checklist", "/api/v1/shop/supply-reminder"} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, p, "", nil).Status, p)
	}

	// Supply reminder: off by default, no care reminder; on → one, 3 days before the predicted 2026-10-03.
	r := e.do(fiber.MethodGet, "/api/v1/shop/supply-reminder", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, false, r.data()["enabled"])
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, saraID))
	r = e.do(fiber.MethodPut, "/api/v1/shop/supply-reminder", sara, map[string]any{"enabled": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "2026-09-30", r.data()["remind_on"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'custom'`, saraID))
	r = e.do(fiber.MethodPut, "/api/v1/shop/supply-reminder", sara, map[string]any{"enabled": false})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Zero(t, e.count(`SELECT COUNT(*) FROM reminders WHERE user_id = ?`, saraID))

	// Family share needs the pregnancy grant: Ali (cycle / symptoms / meds / appointments only) is refused.
	path := fmt.Sprintf("/api/v1/shop/checklist?for_user_id=%d", saraID)
	r = e.do(fiber.MethodGet, path, ali, nil)
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "companion_forbidden", r.Body["error_code"])

	// Pregnant: due date on the list; the predictor is off for the pregnancy mode (no supply reminder).
	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', '2026-09-05', NOW(), NOW())`, saraID)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at)
		SELECT id, 'pregnancy', 'view', NOW(), NOW() FROM companions WHERE owner_id = ?`, saraID)
	require.NoError(t, err)
	r = e.do(fiber.MethodGet, "/api/v1/shop/checklist", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "2027-06-12", r.data()["due_date"])
	r = e.do(fiber.MethodGet, path, ali, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["read_only"])

	// Teen mode: no shop.
	nila := e.user("Nila", "09120008009")
	e.teenMode("Nila")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, "/api/v1/shop/checklist", nila, nil).Status)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPut, "/api/v1/shop/supply-reminder", nila, map[string]any{"enabled": true}).Status)
}
