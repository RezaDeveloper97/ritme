package rbac_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/rbac"
)

// B-N9-02b security follow-up.

func requestAccess(t *testing.T, c *admintest.Client) string {
	t.Helper()
	r := c.JSON(fiber.MethodPost, "/health-access", map[string]any{"user_id": 10,
		"reason": "User reports wrong cycle predictions", "ticket": "#77"})
	require.Equal(t, 201, r.Status, string(r.Raw))
	return "/health-access/" + itoa(uint64(r.Obj("request")["id"].(float64)))
}

func TestHealthAccess_ApproverIndependence(t *testing.T) {
	e := newEnv(t)
	super := e.As(admintest.SuperID)

	// The requester (super 1) creates an approver; even once the account is old enough, the audit row disqualifies it.
	r := super.JSON(fiber.MethodPost, "/admins", map[string]any{"name": "Puppet", "email": "puppet@ritme.test",
		"password": "Secret-pass-123", "password_confirmation": "Secret-pass-123", "role": "super"})
	require.Equal(t, 201, r.Status, string(r.Raw))
	puppet := uint64(r.Obj("admin")["id"].(float64))
	e.Exec(`UPDATE admins SET created_at = NOW() - INTERVAL 3 DAY WHERE id = ?`, puppet)
	path := requestAccess(t, super)
	res := e.As(puppet).JSON(fiber.MethodPost, path+"/approve", nil)
	assert.Equal(t, 422, res.Status)
	assert.Equal(t, rbac.CodeApproverNotIndependent, res.Code())

	// A fresh account (< 24 h, super2 was created NOW()) is refused too.
	assert.Equal(t, rbac.CodeApproverNotIndependent, e.As(super2ID).JSON(fiber.MethodPost, path+"/approve", nil).Code())

	// An approver whose password the requester changed (admin.update) is refused.
	e.Exec(`UPDATE admins SET created_at = NOW() - INTERVAL 3 DAY WHERE id = ?`, super2ID)
	upd := super.JSON(fiber.MethodPut, "/admins/"+itoa(super2ID), map[string]any{"name": "Root Two",
		"email": "root2@ritme.test", "role": "super", "password": "Other-pass-456", "password_confirmation": "Other-pass-456"})
	require.Equal(t, 200, upd.Status, string(upd.Raw))
	assert.Equal(t, rbac.CodeApproverNotIndependent, e.As(super2ID).JSON(fiber.MethodPost, path+"/approve", nil).Code())

	// Older than 30 days the change no longer counts; the approval also raises the alert line (log only).
	// admin_audit_logs is append-only (00105 trigger): lift the guard to backdate the rows, then put it back.
	e.Exec(`DROP TRIGGER admin_audit_logs_no_update`)
	e.Exec(`UPDATE admin_audit_logs SET created_at = NOW() - INTERVAL 31 DAY WHERE target_type = 'admin' AND target_id = ?`, super2ID)
	e.Exec("CREATE TRIGGER `admin_audit_logs_no_update` BEFORE UPDATE ON `admin_audit_logs` FOR EACH ROW\n  SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'admin_audit_logs is append-only'")
	ok := e.As(super2ID).JSON(fiber.MethodPost, path+"/approve", nil)
	require.Equal(t, 200, ok.Status, string(ok.Raw))
	assert.Equal(t, rbac.StatusApproved, ok.Obj("request")["status"])
}

func TestHealthAccess_PendingExpiresAndRequesterRecheck(t *testing.T) {
	e := newEnv(t)
	clin := e.As(clinicalID)
	path := requestAccess(t, clin)
	e.Exec(`UPDATE admin_health_access_requests SET created_at = NOW() - INTERVAL 25 HOUR`)
	r := e.As(admintest.SuperID).JSON(fiber.MethodPost, path+"/approve", nil)
	assert.Equal(t, 409, r.Status)
	assert.Equal(t, rbac.CodeNotPending, r.Code())
	assert.Equal(t, 409, e.As(admintest.SuperID).JSON(fiber.MethodPost, path+"/reject", nil).Status)
	mine := clin.Get("/health-access").Items()
	require.Len(t, mine, 1)
	assert.Equal(t, rbac.StatusExpired, mine[0].(map[string]any)["status"])

	// The requester's role lost the health capability between request and approval.
	path = requestAccess(t, e.As(clinical2ID))
	require.Equal(t, 200, e.As(admintest.SuperID).JSON(fiber.MethodPut, "/roles/clinical_support",
		map[string]any{"permissions": map[string]any{"users": "view"}}).Status)
	r = e.As(admintest.SuperID).JSON(fiber.MethodPost, path+"/approve", nil)
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, rbac.CodeRequesterNotPermitted, r.Code())
}

func TestUsers_UserGoalIsHealthData(t *testing.T) {
	e := newEnv(t)
	clin := e.As(clinicalID)
	list := clin.Get("/users")
	require.Equal(t, 200, list.Status)
	for _, it := range list.Items() {
		assert.Nil(t, it.(map[string]any)["user_goal"])
	}
	u := clin.Get("/users/10")
	require.Equal(t, 200, u.Status)
	assert.Nil(t, u.Obj("user")["user_goal"])
	assert.Nil(t, u.Obj("profile")["user_goal"])

	// Update without user_goal keeps the stored goal; the audit row never carries it.
	// T-M10-33: clinical support views users only (§5.8 "masked users"); the update is a super admin's.
	assert.Equal(t, 403, clin.JSON(fiber.MethodPut, "/users/10", map[string]any{"subscription_type": "free"}).Status)
	upd := e.As(admintest.SuperID).JSON(fiber.MethodPut, "/users/10", map[string]any{"subscription_type": "free"})
	require.Equal(t, 200, upd.Status, string(upd.Raw))
	assert.Equal(t, "ttc", e.String(`SELECT user_goal FROM user_profiles WHERE user_id = 10`))
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'user.update' AND detail LIKE '%ttc%'`))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'user.update'`))

	// With a grant the goal shows.
	path := requestAccess(t, clin)
	require.Equal(t, 200, e.As(admintest.SuperID).JSON(fiber.MethodPost, path+"/approve", nil).Status)
	u = clin.Get("/users/10")
	assert.Equal(t, "ttc", u.Obj("user")["user_goal"])
	assert.Equal(t, "ttc", u.Obj("profile")["user_goal"])
}

func TestAdminAPI_NoStoreAndFailedLoginAudited(t *testing.T) {
	e := newEnv(t)
	req := httptest.NewRequest(fiber.MethodGet, httpadmin.Prefix+"/users", nil)
	req.Host = admintest.Host
	req.AddCookie(&http.Cookie{Name: e.Kit.SessionCookieName(), Value: sessionID(t, e, admintest.SuperID)})
	res, err := e.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	// Unauthenticated answers too.
	req = httptest.NewRequest(fiber.MethodGet, httpadmin.Prefix+"/users", nil)
	req.Host = admintest.Host
	res, err = e.App.Test(req)
	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header.Get("Cache-Control"))

	anon := e.Anonymous()
	r := anon.JSON(fiber.MethodPost, "/auth/login", map[string]any{"email": "root@ritme.test", "password": "wrong-pass"})
	assert.Equal(t, 422, r.Status, string(r.Raw))
	r = anon.JSON(fiber.MethodPost, "/auth/login", map[string]any{"email": "ghost@ritme.test", "password": "wrong-pass"})
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'auth.login_failed' AND target_id = 1 AND admin_id IS NULL`))
	assert.Equal(t, 1, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE action = 'auth.login_failed' AND target_id = 0`))
	assert.Equal(t, 0, e.Int(`SELECT COUNT(*) FROM admin_audit_logs WHERE detail LIKE '%@ritme.test%'`))
}

func TestAudit_IPHiddenForNonSuperAndEditorEditable(t *testing.T) {
	e := newEnv(t)
	super := e.As(admintest.SuperID)
	require.Equal(t, 200, super.JSON(fiber.MethodPost, "/users/11/block", nil).Status)
	// Grant the editor the audit log; the editor sees rows without the ip, super with it.
	require.Equal(t, 200, super.JSON(fiber.MethodPut, "/roles/editor",
		map[string]any{"permissions": map[string]any{"audit": "view", "content": "manage"}}).Status)
	ed := e.As(admintest.EditorID).Get("/audit-logs?action=user.block")
	require.Equal(t, 200, ed.Status, string(ed.Raw))
	require.NotEmpty(t, ed.Items())
	assert.Nil(t, ed.Items()[0].(map[string]any)["ip"])
	sp := super.Get("/audit-logs?action=user.block")
	assert.NotNil(t, sp.Items()[0].(map[string]any)["ip"])
}
