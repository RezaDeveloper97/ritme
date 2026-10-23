package users_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/users"
)

// T-M9-30: an admin edit of a user's plan / goal writes user_profiles.user_goal (an engine input), so it moves the
// user's engine_rev by exactly 1 — the first edit (profile insert + update) and a later one alike; a refused edit
// does not.
func TestBump_AdminUserEdit(t *testing.T) {
	e := admintest.New(t)
	h := users.NewHandlers(e.DB, admintest.Quiet)
	e.Route()(fiber.MethodPut, "/users/:id", e.Kit.Admin(h.Update))
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (31, 'Nila', '09125550031', NOW(), NOW())`)
	rev := func() int {
		return e.Int(`SELECT COALESCE((SELECT engine_rev FROM user_revisions WHERE user_id = 31), 0)`)
	}
	c := e.As(admintest.SuperID)

	require.Equal(t, 200, c.JSON(fiber.MethodPut, "/users/31", map[string]any{"subscription_type": "free", "user_goal": "ttc"}).Status)
	assert.Equal(t, 1, rev())
	require.Equal(t, 200, c.JSON(fiber.MethodPut, "/users/31", map[string]any{"subscription_type": "premium"}).Status)
	assert.Equal(t, 2, rev())

	r := c.JSON(fiber.MethodPut, "/users/31", map[string]any{"subscription_type": "platinum"})
	require.Equal(t, 422, r.Status)
	assert.Equal(t, 2, rev())
}
