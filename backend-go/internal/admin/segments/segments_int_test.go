package segments_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/segments"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

const (
	marketingID uint64 = 5
	supportID   uint64 = 6
)

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	segments.New(e.DB, admintest.Quiet).Routes(segments.Route(e.Route()), e.Kit)
	uh := users.NewHandlers(e.DB, admintest.Quiet)
	e.Route()(fiber.MethodGet, "/users/directory", e.Kit.Admin(uh.Directory))
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(5, 'Mk', 'mk@ritme.test', ?, 'marketing', 1, NOW(), NOW()),
		(6, 'Sup', 'sup@ritme.test', ?, 'support', 1, NOW(), NOW())`, pw, pw)
	// 12 Plus users with an old account (churn computed), 3 free users.
	for i := range 15 {
		e.Exec(`INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, ?, '2026-01-05 10:00:00', NOW())`,
			100+i, fmt.Sprintf("0912%07d", 100+i))
		if i < 12 {
			e.Exec(`INSERT INTO plus_subscriptions (user_id, status, source, starts_at, ends_at, created_at, updated_at)
				VALUES (?, 'active', 'purchase', '2026-01-01 00:00:00', '2027-01-01 00:00:00', NOW(), NOW())`, 100+i)
		}
	}
	require.NoError(t, users.NewActivity(e.DB, admintest.Quiet).Rebuild(t.Context(), time.Now()))
	return e
}

func TestSegments_CRUDRolesAndApply(t *testing.T) {
	e := newEnv(t)
	mk := e.As(marketingID)
	assert.Equal(t, 403, e.As(supportID).Get("/segments").Status, "support has no segments capability")

	r := mk.JSON(fiber.MethodPost, "/segments", map[string]any{"name": "Plus lapsing",
		"filters": map[string]any{"plan": "plus", "churn": "high"}})
	require.Equal(t, 201, r.Status, string(r.Raw))
	id := uint64(r.Data()["id"].(float64))
	assert.Equal(t, "plus", r.Obj("filters")["plan"])

	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/segments", map[string]any{"name": "Plus lapsing",
		"filters": map[string]any{"plan": "plus"}}).Status, "name taken")
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/segments", map[string]any{"name": "Search",
		"filters": map[string]any{"q": "0912"}}).Status, "no free text in a segment")
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/segments", map[string]any{"name": "Bad",
		"filters": map[string]any{"mode": "x"}}).Status)
	assert.Equal(t, 422, mk.JSON(fiber.MethodPost, "/segments", map[string]any{"name": "Bad",
		"filters": map[string]any{"symptom": "cramps"}}).Status)

	// Size: 12 plus users with high churn (no activity) ≥ k = 10 → shown; free users (3) → suppressed.
	show := mk.Get(fmt.Sprintf("/segments/%d", id))
	require.Equal(t, 200, show.Status)
	assert.Equal(t, float64(12), show.Data()["size"])
	r = mk.JSON(fiber.MethodPut, fmt.Sprintf("/segments/%d", id), map[string]any{"name": "Free lapsing",
		"filters": map[string]any{"plan": "free", "churn": "high"}})
	require.Equal(t, 200, r.Status, string(r.Raw))
	show = mk.Get(fmt.Sprintf("/segments/%d", id))
	assert.Nil(t, show.Data()["size"])
	assert.Equal(t, true, show.Data()["size_suppressed"])
	assert.Len(t, mk.Get("/segments").Items(), 1)

	// Apply in the directory (super: users view); explicit params override the segment.
	su := e.As(admintest.SuperID)
	assert.Len(t, su.Get(fmt.Sprintf("/users/directory?segment=%d", id)).Items(), 3)
	assert.Len(t, su.Get(fmt.Sprintf("/users/directory?segment=%d&plan=plus", id)).Items(), 12)
	assert.Equal(t, 422, su.Get("/users/directory?segment=999").Status)

	// The `segment` message fact is true (contains the id) for a matching user only.
	res := segments.NewResolver(e.DB)
	ids, err := res.UserSegments(t.Context(), 112, time.Now())
	require.NoError(t, err)
	assert.Equal(t, []string{fmt.Sprint(id)}, ids)
	ids, err = res.UserSegments(t.Context(), 100, time.Now())
	require.NoError(t, err)
	assert.Empty(t, ids)
	f := facts.New(facts.Sources{Account: accountSource{res: res, uid: 112}}, civildate.InTehran(time.Now()), "fa")
	v, ok, err := f.Get(t.Context(), "segment")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Contains(t, v, fmt.Sprint(id))

	require.Equal(t, 200, mk.JSON(fiber.MethodDelete, fmt.Sprintf("/segments/%d", id), nil).Status)
	assert.Equal(t, 404, mk.Get(fmt.Sprintf("/segments/%d", id)).Status)
	assert.Equal(t, 3, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action LIKE 'segment.%'`))
}

// accountSource is the messages StoreSource shape: an AccountSource that is also a SegmentSource.
type accountSource struct {
	res *segments.Resolver
	uid uint64
}

func (accountSource) Account(context.Context) (*facts.Account, error) { return nil, nil }

func (a accountSource) Segments(ctx context.Context) ([]string, error) {
	return a.res.UserSegments(ctx, a.uid, time.Now())
}
