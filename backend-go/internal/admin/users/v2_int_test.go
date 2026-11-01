package users_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/users"
)

const (
	supportID  uint64 = 8
	analystID  uint64 = 9
	clinicalID uint64 = 4
	doctorID   uint64 = 7
)

func newV2Env(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	r := e.Route()
	h := users.NewHandlers(e.DB, admintest.Quiet)
	get := fiber.MethodGet
	r(get, "/users", e.Kit.Admin(h.List))
	r(get, "/users/directory", e.Kit.Admin(h.Directory))
	r(get, "/users/directory/export", e.Kit.Admin(h.ExportDirectory))
	r(get, "/users/:id", e.Kit.Admin(h.Show))
	r(get, "/users/:id/overview", e.Kit.Admin(h.Overview))
	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(4, 'Maryam', 'clin@ritme.test', ?, 'clinical_support', 1, NOW(), NOW()),
		(7, 'Dr', 'dr@ritme.test', ?, 'doctor', 1, NOW(), NOW()),
		(8, 'Sup', 'sup@ritme.test', ?, 'support', 1, NOW(), NOW()),
		(9, 'Ana', 'ana@ritme.test', ?, 'analyst', 1, NOW(), NOW())`, pw, pw, pw, pw)
	e.Exec(`INSERT INTO users (id, name, mobile, email, created_at, updated_at) VALUES
		(10, 'Sara', '09121234567', 'sara@example.com', NOW(), NOW()), (11, 'Mina', '09350000011', NULL, NOW(), NOW())`)
	e.Exec(`INSERT INTO user_profiles (user_id, subscription_type, user_goal, cycle_duration, last_period_start,
		created_at, updated_at) VALUES (10, 'free', 'ttc', 29, '2026-09-20', NOW(), NOW())`)
	e.Exec(`INSERT INTO user_consents (user_id, consent, granted, version, granted_at, created_at, updated_at)
		VALUES (10, 'analytics', 1, 2, NOW(), NOW(), NOW())`)
	e.Exec(`INSERT INTO support_reports (user_id, message, status, created_at, updated_at)
		VALUES (10, 'my period is late secret-note', 'open', NOW(), NOW())`)
	return e
}

func TestDirectory_MaskedFiltersAndNoHealth(t *testing.T) {
	e := newV2Env(t)
	sup := e.As(supportID)
	r := sup.Get("/users/directory")
	require.Equal(t, 200, r.Status, string(r.Raw))
	require.Len(t, r.Items(), 2)
	items, _ := json.Marshal(r.Items()) // the options block lists every mode, ttc included
	raw := string(r.Raw)
	for _, leak := range []string{"09121234567", "sara@example.com", "Sara", "user_goal", "2026-09-20", "RT-000A"} {
		assert.NotContains(t, raw, leak)
	}
	for _, leak := range []string{"ttc", "cycle_duration"} {
		assert.NotContains(t, string(items), leak)
	}
	assert.Contains(t, raw, `0912\u2022\u2022\u20224567`)

	// Projection filters read admin_user_activity (nightly); before the first run nobody matches them.
	assert.Len(t, sup.Get("/users/directory?plan=free").Items(), 0)
	require.NoError(t, users.NewActivity(e.DB, admintest.Quiet).Rebuild(t.Context(), time.Now()))
	assert.Len(t, sup.Get("/users/directory?plan=plus").Items(), 0)
	assert.Len(t, sup.Get("/users/directory?plan=free&churn=new").Items(), 2)
	assert.Len(t, sup.Get("/users/directory?cohort="+time.Now().Format("2006-01")).Items(), 2)
	assert.Len(t, sup.Get("/users/directory?q=Mina").Items(), 1)
	assert.Equal(t, 422, sup.Get("/users/directory?plan=gold").Status)
	assert.Equal(t, 422, sup.Get("/users/directory?cohort=2026-13").Status)

	// Role gates: a doctor has no users capability.
	assert.Equal(t, 403, e.As(doctorID).Get("/users/directory").Status)
	assert.Equal(t, 403, e.As(doctorID).Get("/users/10/overview").Status)
	assert.Equal(t, 401, e.Anonymous().Get("/users/directory").Status)
}

func TestDirectoryExport_AnonymisedAndNeedsExportCap(t *testing.T) {
	e := newV2Env(t)
	assert.Equal(t, 403, e.As(supportID).Get("/users/directory/export").Status) // users view, no export cap
	r := e.As(admintest.SuperID).Get("/users/directory/export")
	require.Equal(t, 200, r.Status)
	body := string(r.Raw)
	assert.True(t, strings.HasPrefix(body, "ref,mode,plan,signup_month,logged_days_30,data_quality,churn_risk,blocked"))
	for _, leak := range []string{"0912", "Sara", "example.com"} {
		assert.NotContains(t, body, leak)
	}
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'users.export'`))
}

func TestOverview_HealthHiddenWithoutGrant(t *testing.T) {
	e := newV2Env(t)
	clin := e.As(clinicalID)
	r := clin.Get("/users/10/overview")
	require.Equal(t, 200, r.Status, string(r.Raw))
	assert.Nil(t, r.Data()["health"])
	assert.Equal(t, false, r.Obj("health_access")["granted"])
	raw := string(r.Raw)
	for _, leak := range []string{"ttc", "secret-note", "my period", "2026-09-20"} {
		assert.NotContains(t, raw, leak)
	}
	assert.Len(t, r.Data()["consents"], 1)
	assert.Len(t, r.Data()["support_reports"], 1)
	assert.Equal(t, 404, clin.Get("/users/9999/overview").Status)

	// An approved, live grant for this admin and user unlocks only the gated block.
	e.Exec(`INSERT INTO admin_health_access_requests (admin_id, user_id, reason, ticket, status, decided_by, decided_at,
		expires_at, created_at, updated_at) VALUES (4, 10, 'cycle prediction complaint', '#1', 'approved', 1, NOW(),
		DATE_ADD(NOW(), INTERVAL 6 HOUR), NOW(), NOW())`)
	r = clin.Get("/users/10/overview")
	require.Equal(t, 200, r.Status)
	assert.Equal(t, true, r.Obj("health_access")["granted"])
	assert.Equal(t, "ttc", r.Obj("health")["user_goal"])
	assert.NotContains(t, string(r.Raw), "secret-note") // support text is never returned
	// Another user stays hidden (IDOR).
	assert.Nil(t, clin.Get("/users/11/overview").Data()["health"])
	// A role without the health capability never gets it, even with a row.
	e.Exec(`UPDATE admin_health_access_requests SET admin_id = 8`)
	assert.Nil(t, e.As(supportID).Get("/users/10/overview").Data()["health"])
}
