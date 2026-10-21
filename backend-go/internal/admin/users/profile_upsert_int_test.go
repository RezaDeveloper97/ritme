package users_test

import (
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/users"
)

// T-M9-23b: user_profiles.user_id is UNIQUE, so concurrent admin first-profile writes for one
// user must end in one row and no 500 (the upsert hands the loser the existing id).
func TestUpdate_ConcurrentFirstProfileWrites(t *testing.T) {
	e := admintest.New(t)
	h := users.NewHandlers(e.DB, admintest.Quiet)
	e.Route()(fiber.MethodPut, "/users/:id", e.Kit.Admin(h.Update))
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (30, 'Nila', '09125550030', NOW(), NOW())`)

	const n = 8
	clients := make([]*admintest.Client, n)
	for i := range clients {
		clients[i] = e.As(admintest.SuperID) // log in before the race starts
	}
	statuses := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			statuses[i] = clients[i].JSON(fiber.MethodPut, "/users/30",
				map[string]any{"subscription_type": "free", "user_goal": "ttc"}).Status
		}()
	}
	close(start)
	wg.Wait()

	for i, s := range statuses {
		assert.Equal(t, 200, s, "request %d", i)
	}
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM user_profiles WHERE user_id = 30`))
	assert.Equal(t, "ttc", e.String(`SELECT user_goal FROM user_profiles WHERE user_id = 30`))
	require.Equal(t, 200, clients[0].JSON(fiber.MethodPut, "/users/30",
		map[string]any{"subscription_type": "premium"}).Status)
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM user_profiles WHERE user_id = 30`))
}
