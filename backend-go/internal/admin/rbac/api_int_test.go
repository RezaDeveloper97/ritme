package rbac_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/admins"
	adminauth "github.com/ritme/backend-go/internal/admin/auth"
	"github.com/ritme/backend-go/internal/admin/content"
	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/admin/rbac"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/admin/support"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// Seeded admin ids besides admintest.SuperID (1) and EditorID (2).
const (
	super2ID    uint64 = 3
	clinicalID  uint64 = 4
	clinical2ID uint64 = 5
	marketingID uint64 = 6
	doctorID    uint64 = 7
	supportID   uint64 = 8
	analystID   uint64 = 9
)

func newEnv(t *testing.T) *admintest.Env {
	t.Helper()
	e := admintest.New(t)
	e.Kit.UseAuditChain(e.DB) // T-M10-33: every audit row of these tests is a chained append
	r := e.Route()
	rbac.NewHandlers(e.DB, e.Kit, admintest.Quiet).Routes(r, e.Kit)
	content.New(content.Deps{DB: e.DB, Disk: media.NewDisk(e.Storage), Logger: admintest.Quiet}).Routes(r, e.Kit)
	support.NewHandlers(e.DB, e.Storage, admintest.Quiet).Routes(r, e.Kit)
	usersH := users.NewHandlers(e.DB, admintest.Quiet)
	adminsH := admins.NewHandlers(e.DB, e.Kit.Sessions(), admintest.Quiet)
	authH := adminauth.NewHandlers(e.Kit, store.New(e.DB), nil)
	get, post := fiber.MethodGet, fiber.MethodPost
	r(get, "/auth/me", e.Kit.Admin(authH.Me))
	r(get, "/users", e.Kit.Admin(usersH.List))
	r(get, "/users/directory/export", e.Kit.Admin(usersH.ExportDirectory))
	r(get, "/users/:id", e.Kit.Admin(usersH.Show))
	r(post, "/users/:id/block", e.Kit.Admin(usersH.Block))
	r(fiber.MethodPut, "/users/:id", e.Kit.Admin(usersH.Update))
	r(fiber.MethodDelete, "/users/:id", e.Kit.Admin(usersH.Destroy))
	r(post, "/admins", e.Kit.Super(adminsH.Store))
	r(fiber.MethodPut, "/admins/:id", e.Kit.Super(adminsH.Update))
	r(post, "/auth/login", e.Kit.Public(authH.Login))
	r(get, "/something-new", e.Kit.Admin(func(c fiber.Ctx) error { return httpadmin.OK(c, nil) })) // unmapped

	const pw = "$2y$12$lo8Z3h/WEQp7b8AwCvwiTObVkv7/Xa68c/VnNTSszAOrR4kLj1HBi" //nolint:gosec // G101: test fixture hash
	e.Exec(`INSERT INTO admins (id, name, email, password, role, is_active, created_at, updated_at) VALUES
		(3, 'Root Two', 'root2@ritme.test', ?, 'super', 1, NOW(), NOW()),
		(4, 'Maryam', 'clin@ritme.test', ?, 'clinical_support', 1, NOW(), NOW()),
		(5, 'Clin Two', 'clin2@ritme.test', ?, 'clinical_support', 1, NOW(), NOW()),
		(6, 'Mark', 'mkt@ritme.test', ?, 'marketing', 1, NOW(), NOW()),
		(7, 'Dr', 'dr@ritme.test', ?, 'doctor', 1, NOW(), NOW()),
		(8, 'Sup', 'sup@ritme.test', ?, 'support', 1, NOW(), NOW()),
		(9, 'Ana', 'ana@ritme.test', ?, 'analyst', 1, NOW(), NOW())`, pw, pw, pw, pw, pw, pw, pw)
	e.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES
		(10, 'Sara', '09120000010', NOW(), NOW()), (11, 'Mina', '09120000011', NOW(), NOW())`)
	e.Exec(`INSERT INTO user_profiles (user_id, subscription_type, user_goal, cycle_duration, last_period_start,
		created_at, updated_at) VALUES (10, 'free', 'ttc', 29, '2026-09-20', NOW(), NOW())`)
	return e
}

func TestPermissions_PerRole(t *testing.T) {
	e := newEnv(t)
	cases := []struct {
		admin  uint64
		method string
		path   string
		want   int
	}{
		// marketing (§5.8: product/marketing messages, challenges, segments): no content, users, support, audit,
		// roles or health.
		{marketingID, fiber.MethodGet, "/articles", 403},
		{marketingID, fiber.MethodGet, "/support-reports", 403},
		{marketingID, fiber.MethodGet, "/audit-logs", 403},
		{marketingID, fiber.MethodGet, "/roles", 403},
		{marketingID, fiber.MethodPost, "/health-access", 403},
		{marketingID, fiber.MethodPost, "/users/10/block", 403},
		{marketingID, fiber.MethodGet, "/users", 403},
		// doctor: only the doctor inbox (+ dashboard).
		{doctorID, fiber.MethodGet, "/articles", 403},
		{doctorID, fiber.MethodGet, "/users", 403},
		// support (§5.8: masked users, subscriptions, tickets): users view, support inbox; not content, no block.
		{supportID, fiber.MethodGet, "/support-reports", 200},
		{supportID, fiber.MethodGet, "/users", 200},
		{supportID, fiber.MethodPost, "/users/10/block", 403},
		{supportID, fiber.MethodGet, "/articles", 403},
		{supportID, fiber.MethodGet, "/users/directory/export", 403},
		// clinical support: health requests, support, users; not content.
		{clinicalID, fiber.MethodGet, "/health-access", 200},
		{clinicalID, fiber.MethodGet, "/articles", 403},
		// editor keeps content, but no health data, no audit log (B-N9-02b), no super or unmapped routes.
		{admintest.EditorID, fiber.MethodGet, "/articles", 200},
		{admintest.EditorID, fiber.MethodGet, "/audit-logs", 403},
		{admintest.EditorID, fiber.MethodGet, "/something-new", 403},
		{admintest.SuperID, fiber.MethodGet, "/something-new", 200},
		// analyst (§5.8: KPI and anonymous export only): 403 on the user list, the anonymised export works.
		{analystID, fiber.MethodGet, "/users", 403},
		{analystID, fiber.MethodGet, "/users/10", 403},
		{analystID, fiber.MethodDelete, "/users/11", 403},
		{analystID, fiber.MethodPost, "/users/11/block", 403},
		{analystID, fiber.MethodGet, "/users/directory/export", 200},
		{analystID, fiber.MethodGet, "/audit-logs/export", 403},
		{admintest.EditorID, fiber.MethodPost, "/health-access", 403},
		{admintest.EditorID, fiber.MethodGet, "/roles", 403},
		// audit chain verification: super only.
		{admintest.EditorID, fiber.MethodGet, "/audit/verify", 403},
		{analystID, fiber.MethodGet, "/audit/verify", 403},
		{admintest.SuperID, fiber.MethodGet, "/audit/verify", 200},
		// super: everything.
		{admintest.SuperID, fiber.MethodGet, "/roles", 200},
		{admintest.SuperID, fiber.MethodGet, "/audit-logs", 200},
	}
	for _, tc := range cases {
		var r admintest.Resp
		if tc.method == fiber.MethodGet {
			r = e.As(tc.admin).Get(tc.path)
		} else {
			r = e.As(tc.admin).JSON(tc.method, tc.path, map[string]any{})
		}
		assert.Equal(t, tc.want, r.Status, "admin %d %s %s", tc.admin, tc.method, tc.path)
		if tc.want == 403 {
			assert.Equal(t, httpadmin.CodeForbidden, r.Code(), "admin %d %s %s", tc.admin, tc.method, tc.path)
		}
	}
	// 401 stays the session answer only.
	assert.Equal(t, 401, e.Anonymous().Get("/audit-logs").Status)

	// /auth/me carries the matrix.
	me := e.As(marketingID).Get("/auth/me")
	require.Equal(t, 200, me.Status)
	assert.Equal(t, "manage", me.Obj("permissions")["messages"])
	assert.Equal(t, "manage", me.Obj("permissions")["segments"])
	assert.NotContains(t, me.Obj("permissions"), "content")
	assert.NotContains(t, me.Obj("permissions"), "health")
}

func TestRoles_EditAndReset(t *testing.T) {
	e := newEnv(t)
	super := e.As(admintest.SuperID)
	list := super.Get("/roles")
	require.Equal(t, 200, list.Status)
	assert.Len(t, list.Data()["roles"], len(httpadmin.Roles))

	// Locked and unknown roles.
	assert.Equal(t, rbac.CodeRoleLocked, super.JSON(fiber.MethodPut, "/roles/super", map[string]any{"permissions": map[string]any{}}).Code())
	assert.Equal(t, 404, super.JSON(fiber.MethodPut, "/roles/nope", map[string]any{"permissions": map[string]any{}}).Status)
	assert.Equal(t, 422, super.JSON(fiber.MethodPut, "/roles/marketing", map[string]any{"permissions": map[string]any{"bogus": "view"}}).Status)
	assert.Equal(t, 422, super.JSON(fiber.MethodPut, "/roles/marketing", map[string]any{"permissions": map[string]any{"health": "view"}}).Status)
	// An editor cannot edit roles.
	assert.Equal(t, 403, e.As(admintest.EditorID).JSON(fiber.MethodPut, "/roles/marketing", map[string]any{"permissions": map[string]any{}}).Status)

	// Give marketing the support inbox (view) and content.
	r := super.JSON(fiber.MethodPut, "/roles/marketing", map[string]any{"permissions": map[string]any{
		"dashboard": "view", "support": "view", "content": "manage"}})
	require.Equal(t, 200, r.Status, string(r.Raw))
	mkt := e.As(marketingID)
	assert.Equal(t, 200, mkt.Get("/support-reports").Status)
	assert.Equal(t, 200, mkt.Get("/articles").Status)

	// Reset: back to the §5.8 defaults (no content, no support inbox).
	require.Equal(t, 200, super.JSON(fiber.MethodPost, "/roles/marketing/reset", nil).Status)
	assert.Equal(t, 403, mkt.Get("/articles").Status)
	assert.Equal(t, 403, mkt.Get("/support-reports").Status)
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action IN ('role.update','role.reset') AND admin_id = 1`))
}

func TestAudit_RowsFromModulesAndFilters(t *testing.T) {
	e := newEnv(t)
	super := e.As(admintest.SuperID)
	ed := e.As(admintest.EditorID)
	// users module
	require.Equal(t, 200, ed.JSON(fiber.MethodPost, "/users/11/block", nil).Status)
	// content module
	r := ed.JSON(fiber.MethodPost, "/affirmations", map[string]any{"text": map[string]any{"fa": "متن تأکیدی"}, "is_active": true})
	require.Equal(t, 201, r.Status, string(r.Raw))
	// admins module (super)
	r = super.JSON(fiber.MethodPost, "/admins", map[string]any{"name": "New", "email": "new@ritme.test",
		"password": "Secret-pass-123", "password_confirmation": "Secret-pass-123", "role": "analyst"})
	require.Equal(t, 201, r.Status, string(r.Raw))

	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'user.block' AND target_type = 'user' AND target_id = 11 AND admin_id = 2`))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action LIKE 'affirmation.%' AND admin_id = 2`))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'admin.create' AND admin_id = 1`))
	assert.Equal(t, "analyst", e.String(`SELECT JSON_VALUE(detail, '$.role') FROM admin_audit_logs WHERE action = 'admin.create'`))

	list := super.Get("/audit-logs?admin_id=2&action=user.")
	require.Equal(t, 200, list.Status)
	require.Len(t, list.Items(), 1)
	item := list.Items()[0].(map[string]any)
	assert.Equal(t, "user.block", item["action"])
	assert.Equal(t, "Editor", item["admin"].(map[string]any)["name"])
	assert.Equal(t, 422, super.Get("/audit-logs?from=bad").Status)

	// Export: CSV for super; editor has export too (legacy full set); the export itself is audited.
	req := httptest.NewRequest(fiber.MethodGet, httpadmin.Prefix+"/audit-logs/export", nil)
	req.Host = admintest.Host
	req.AddCookie(&http.Cookie{Name: e.Kit.SessionCookieName(), Value: sessionID(t, e, admintest.SuperID)})
	res, err := e.App.Test(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(res.Body)
	assert.Equal(t, 200, res.StatusCode)
	assert.Contains(t, res.Header.Get("Content-Type"), "text/csv")
	assert.Contains(t, string(body), "user.block")
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'audit_log.export'`))
	// product manager-like roles without export: support has neither audit nor export.
	assert.Equal(t, 403, e.As(supportID).Get("/audit-logs/export").Status)
}

func sessionID(t *testing.T, e *admintest.Env, adminID uint64) string {
	t.Helper()
	sess, err := e.Kit.Sessions().Create(t.Context(), adminID, false, time.Now())
	require.NoError(t, err)
	return sess.ID()
}

func itoa(n uint64) string { return strconv.FormatUint(n, 10) }

func TestHealthAccess_RequestApproveReadExpire(t *testing.T) {
	e := newEnv(t)
	clin := e.As(clinicalID)

	// Without a grant: 403 health_access_required; /users/:id hides the health fields.
	r := clin.Get("/health-access/users/10")
	assert.Equal(t, 403, r.Status)
	assert.Equal(t, httpadmin.CodeHealthAccess, r.Code())
	u := clin.Get("/users/10")
	require.Equal(t, 200, u.Status)
	assert.Nil(t, u.Obj("profile")["cycle_duration"])
	assert.Equal(t, false, u.Obj("health_access")["granted"])

	// Validation.
	r = clin.JSON(fiber.MethodPost, "/health-access", map[string]any{"user_id": 10, "reason": "short"})
	assert.Equal(t, 422, r.Status)
	assert.Contains(t, r.Errors(), "reason")
	assert.Contains(t, r.Errors(), "ticket")
	assert.Equal(t, 422, clin.JSON(fiber.MethodPost, "/health-access", map[string]any{"user_id": 999, "reason": "Ticket about cycle predictions", "ticket": "4821"}).Status)

	r = clin.JSON(fiber.MethodPost, "/health-access", map[string]any{"user_id": 10,
		"reason": "User reports wrong cycle predictions", "ticket": "#4821"})
	require.Equal(t, 201, r.Status, string(r.Raw))
	req := r.Obj("request")
	assert.Equal(t, rbac.StatusPending, req["status"])
	id := uint64(req["id"].(float64))
	path := func(action string) string { return "/health-access/" + itoa(id) + "/" + action }

	// Pending is not access.
	assert.Equal(t, 403, clin.Get("/health-access/users/10").Status)
	// Only a super approves; editors and the requester's role cannot.
	assert.Equal(t, 403, clin.JSON(fiber.MethodPost, path("approve"), nil).Status)
	assert.Equal(t, 403, e.As(admintest.EditorID).JSON(fiber.MethodPost, path("approve"), nil).Status)

	a := e.As(admintest.SuperID).JSON(fiber.MethodPost, path("approve"), nil)
	require.Equal(t, 200, a.Status, string(a.Raw))
	assert.Equal(t, rbac.StatusApproved, a.Obj("request")["status"])
	assert.Equal(t, 409, e.As(super2ID).JSON(fiber.MethodPost, path("approve"), nil).Status) // already decided

	// Read allowed and audited — on both endpoints.
	h := clin.Get("/health-access/users/10")
	require.Equal(t, 200, h.Status, string(h.Raw))
	assert.EqualValues(t, 29, h.Obj("profile")["cycle_duration"])
	u = clin.Get("/users/10")
	assert.EqualValues(t, 29, u.Obj("profile")["cycle_duration"])
	assert.Equal(t, true, u.Obj("health_access")["granted"])
	assert.Equal(t, 2, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'health_access.read' AND admin_id = 4 AND target_id = 10`))
	// The audit row never carries the health payload.
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE detail LIKE '%2026-09-20%' OR detail LIKE '%29%'`))

	// IDOR: the grant covers user 10 only, and only for its requester.
	assert.Equal(t, 403, clin.Get("/health-access/users/11").Status)
	assert.Equal(t, 403, e.As(clinical2ID).Get("/health-access/users/10").Status)
	// Another clinical admin neither sees nor revokes the request.
	other := e.As(clinical2ID).Get("/health-access")
	assert.Empty(t, other.Items())
	assert.Equal(t, 404, e.As(clinical2ID).JSON(fiber.MethodPost, path("revoke"), nil).Status)
	// Super sees the whole queue.
	assert.Len(t, e.As(admintest.SuperID).Get("/health-access").Items(), 1)

	// Expire (as 30 minutes passing): 403 again, status reads "expired".
	e.Exec(`UPDATE admin_health_access_requests SET expires_at = NOW() - INTERVAL 1 MINUTE WHERE id = ?`, id)
	assert.Equal(t, 403, clin.Get("/health-access/users/10").Status)
	assert.Nil(t, clin.Get("/users/10").Obj("profile")["cycle_duration"])
	mine := clin.Get("/health-access").Items()
	require.Len(t, mine, 1)
	assert.Equal(t, rbac.StatusExpired, mine[0].(map[string]any)["status"])
}

func TestHealthAccess_SelfApprovalRejectRevoke(t *testing.T) {
	e := newEnv(t)
	super := e.As(admintest.SuperID)
	body := map[string]any{"user_id": 10, "reason": "Escalated clinical ticket review", "ticket": "T-1"}
	r := super.JSON(fiber.MethodPost, "/health-access", body)
	require.Equal(t, 201, r.Status)
	id := itoa(uint64(r.Obj("request")["id"].(float64)))
	assert.Equal(t, rbac.CodeSelfApproval, super.JSON(fiber.MethodPost, "/health-access/"+id+"/approve", nil).Code())
	require.Equal(t, 200, e.As(super2ID).JSON(fiber.MethodPost, "/health-access/"+id+"/reject", nil).Status)
	assert.Equal(t, 403, super.Get("/health-access/users/10").Status) // super is not exempt

	r = e.As(clinicalID).JSON(fiber.MethodPost, "/health-access", body)
	id = itoa(uint64(r.Obj("request")["id"].(float64)))
	require.Equal(t, 200, super.JSON(fiber.MethodPost, "/health-access/"+id+"/approve", nil).Status)
	require.Equal(t, 200, e.As(clinicalID).Get("/health-access/users/10").Status)
	require.Equal(t, 200, e.As(clinicalID).JSON(fiber.MethodPost, "/health-access/"+id+"/revoke", nil).Status)
	assert.Equal(t, 403, e.As(clinicalID).Get("/health-access/users/10").Status)
	assert.Equal(t, 409, e.As(clinicalID).JSON(fiber.MethodPost, "/health-access/"+id+"/revoke", nil).Status)
}
