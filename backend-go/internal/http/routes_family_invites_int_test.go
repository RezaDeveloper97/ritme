package http

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/family/invites"
)

// familyInvite creates a v6 invite as token through POST /family/members and returns (link id, code).
func (e *companionEnv) familyInvite(token string, body map[string]any) (uint64, string) {
	e.t.Helper()
	if _, ok := body["consent_version"]; !ok {
		body["consent_version"] = invites.ConsentVersion
	}
	r := e.do(fiber.MethodPost, "/api/v1/family/members", token, body)
	require.Equal(e.t, fiber.StatusCreated, r.Status, r.Raw)
	link, _ := r.data()["link"].(map[string]any)
	inv, _ := r.data()["invite"].(map[string]any)
	code, _ := inv["code"].(string)
	require.Regexp(e.t, codeRe, code)
	return idOf(link["id"]), code
}

// TestFamilyInviteRoutes_Lifecycle: create → preview → accept with a mutual answer, the owner's 201 shape, no-store,
// validation 422 and the 401 body.
func TestFamilyInviteRoutes_Lifecycle(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009001")
	ali := e.user("Ali", "09120009002")

	r := e.do(fiber.MethodPost, "/api/v1/family/members", "", map[string]any{})
	assert.Equal(t, fiber.StatusUnauthorized, r.Status)
	assert.Equal(t, "unauthenticated", r.Body["error_code"])

	r = e.do(fiber.MethodPost, "/api/v1/family/members", sara, map[string]any{"role": "boss", "template": "custom"})
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	errs, _ := r.Body["errors"].(map[string]any)
	assert.Contains(t, errs, "role")
	assert.Contains(t, errs, "consent_version")

	linkID, code := e.familyInvite(sara, map[string]any{"role": "partner", "template": "meds_appointments",
		"grants":         map[string]any{"appointments": "view"}, // no write level: a code accept activates at once
		"mutual_request": map[string]any{"domains": []string{"appointments"}}, "display_name": "Ali"})

	r = e.do(fiber.MethodPost, "/api/v1/family/invites/preview", ali, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Sara", r.data()["owner"].(map[string]any)["first_name"])
	assert.Equal(t, "partner", r.data()["role"])
	assert.Equal(t, false, r.data()["owner_confirmation"])

	r = e.do(fiber.MethodPost, "/api/v1/family/invites/accept", ali, map[string]any{"code": code, "mutual": map[string]any{"appointments": "view"}})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	link, _ := r.data()["link"].(map[string]any)
	assert.Equal(t, float64(linkID), link["id"])
	assert.Equal(t, "active", link["status"])
	mutual, _ := r.data()["mutual_link"].(map[string]any)
	require.NotNil(t, mutual)
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM companions WHERE owner_id = ? AND companion_user_id = ? AND status = 'active'`, e.ids["Ali"], e.ids["Sara"]))
	// The owner is told.
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND type = 'family'`, e.ids["Sara"]))
}

// TestInvite_PreviewIsThrottledAndGeneric: preview answers every refusal with the same generic 422, draws on the
// /companions/accept per-account budget (one limiter for preview, accept and decline), counts toward the global
// failed-code breaker and stops when the breaker is open (FAM-L3).
func TestInvite_PreviewIsThrottledAndGeneric(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009101")
	ali := e.user("Ali", "09120009102")
	_, code := e.familyInvite(sara, map[string]any{"role": "partner", "template": "cycle_buddy"})
	_, phoneCode := e.familyInvite(sara, map[string]any{"role": "friend", "template": "cycle_buddy", "phone": "09120009199"})

	preview := func(tok, c string) reply {
		return e.do(fiber.MethodPost, "/api/v1/family/invites/preview", tok, map[string]any{"code": c})
	}
	unknown := preview(ali, "ZZZZZZ")
	require.Equal(t, fiber.StatusUnprocessableEntity, unknown.Status, unknown.Raw)
	assert.Equal(t, "invite_invalid", unknown.Body["error_code"])
	for name, r := range map[string]reply{
		"own code":       preview(sara, code),
		"another number": preview(ali, phoneCode),
		"malformed":      preview(ali, "RT0"),
	} {
		assert.Equal(t, unknown.Status, r.Status, name)
		assert.Equal(t, unknown.Raw, r.Raw, "%s: the same body as an unknown code", name)
	}
	hour, err := e.mr.Get("ritme-go-b4n2:companion-accept-breaker:hour")
	require.NoError(t, err)
	assert.Equal(t, "4", hour, "every preview refusal counts toward the global breaker")

	// Ali used 3 previews; the rest of the shared budget goes on /companions/accept, then preview is 429 too.
	for i := 3; i < CompanionAcceptPerUser; i++ {
		r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": "ZZZZZZ"})
		require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, "attempt %d", i+1)
	}
	r := preview(ali, code)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, invites.ErrorCodeTooManyAttempts, r.Body["error_code"])
	assert.Equal(t, fiber.StatusTooManyRequests, e.do(fiber.MethodPost, "/api/v1/family/invites/accept", ali, map[string]any{"code": code}).Status)
	assert.Equal(t, fiber.StatusTooManyRequests, e.do(fiber.MethodPost, "/api/v1/family/invites/decline", ali, map[string]any{"code": code}).Status)

	// An open breaker stops preview for everyone, valid code or not.
	e.clearThrottles("companion-accept")
	require.NoError(t, e.mr.Set("ritme-go-b4n2:companion-accept-breaker:open", "1"))
	e.mr.SetTTL("ritme-go-b4n2:companion-accept-breaker:open", CompanionAcceptBreakerCooldown)
	fresh := e.user("Mina", "09120009103")
	r = preview(fresh, code)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, "too_many_requests", r.Body["error_code"])
}

// TestFamilyShareRequestRoutes: 201, the 7-day 429, 404 for a stranger and the owner-only approval.
func TestFamilyShareRequestRoutes(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009201")
	ali := e.user("Ali", "09120009202")
	mina := e.user("Mina", "09120009203")
	_, code := e.familyInvite(sara, map[string]any{"role": "partner", "template": "cycle_buddy"})
	require.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, "/api/v1/family/invites/accept", ali, map[string]any{"code": code}).Status)

	body := map[string]any{"to_user_id": e.ids["Sara"], "domains": map[string]any{"meds": "view"}}
	r := e.do(fiber.MethodPost, "/api/v1/family/share-requests", ali, body)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	id := idOf(r.data()["id"])
	assert.Equal(t, "pending", r.data()["status"])
	r = e.do(fiber.MethodPost, "/api/v1/family/share-requests", ali, body)
	assert.Equal(t, fiber.StatusTooManyRequests, r.Status, r.Raw)
	assert.Equal(t, "too_many_requests", r.Body["error_code"])
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, "/api/v1/family/share-requests", mina, body).Status)

	path := fmt.Sprintf("/api/v1/family/share-requests/%d", id)
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, path+"/approve", ali, nil).Status, "the requester cannot approve")
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodPost, path+"/approve", mina, nil).Status)
	in := e.do(fiber.MethodGet, "/api/v1/family/share-requests?box=in", sara, nil)
	require.Equal(t, fiber.StatusOK, in.Status)
	assert.Len(t, in.list(), 1)
	r = e.do(fiber.MethodPost, path+"/approve", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "accepted", r.data()["status"])
}

// Review M2 / L1: a /companions code-only invite with a write grant is accepted as awaiting_owner — the owner gets the
// "confirm it" notice (not "accepted"), and the member sees neither the owner's id nor her full name.
func TestCompanionAccept_AwaitingOwnerHidesOwner(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara Ahmadi", "09120009301")
	ali := e.user("Ali", "09120009302")
	linkID, code := e.invite(sara, map[string]any{"type": "partner", "grants": map[string]any{"meds": "edit"}})

	r := e.do(fiber.MethodPost, "/api/v1/companions/accept", ali, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "awaiting_owner", r.data()["status"])
	owner, _ := r.data()["owner"].(map[string]any)
	assert.Nil(t, owner["id"])
	assert.Equal(t, "Sara", owner["name"])
	assert.Empty(t, r.data()["grants"])
	assert.Equal(t, 1, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = 'awaiting_owner'`, e.ids["Sara Ahmadi"]))
	assert.Equal(t, 0, e.count(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ? AND JSON_UNQUOTE(JSON_EXTRACT(data, '$.event')) = 'accepted'`, e.ids["Sara Ahmadi"]))

	r = e.do(fiber.MethodPost, fmt.Sprintf("/api/v1/family/members/%d/confirm", linkID), sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "active", r.data()["status"])
}

// Review L2: an owner whose mode no longer allows the link answers the same generic 422 as an unknown code.
func TestInvite_ModeRefusalIsGeneric(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009401")
	ali := e.user("Ali", "09120009402")
	_, code := e.familyInvite(sara, map[string]any{"role": "partner", "template": "cycle_buddy"})
	_, err := e.db.Exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at)
		VALUES (?, 'female', 'teen', NOW(), NOW()) ON DUPLICATE KEY UPDATE life_mode = 'teen'`, e.ids["Sara"])
	require.NoError(t, err)
	unknown := e.do(fiber.MethodPost, "/api/v1/family/invites/preview", ali, map[string]any{"code": "ZZZZZZ"})
	for _, path := range []string{"preview", "accept"} { // decline of a valid code is a 200 whatever her mode
		r := e.do(fiber.MethodPost, "/api/v1/family/invites/"+path, ali, map[string]any{"code": code})
		assert.Equal(t, fiber.StatusUnprocessableEntity, r.Status, path)
		assert.Equal(t, unknown.Raw, r.Raw, path)
	}
}
