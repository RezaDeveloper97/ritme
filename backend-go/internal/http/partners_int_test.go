package http

import (
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func partnerForm(serviceID any) map[string]any {
	return map[string]any{"service_id": serviceID, "full_name": "Sara", "phone": "09120000000", "city": "Tehran",
		"address": "Valiasr 12", "preferred_date": "2026-09-25", "preferred_window": "morning", "consent_share": true, "consent_import": true}
}

func TestPartnerLab_RoutesGuardsAndIDOR(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009001")
	ali := e.user("Ali", "09120009002")
	const p = "/api/v1/partners/lab-requests"

	for _, path := range []string{"/api/v1/partners/lab-services", p} {
		assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodGet, path, "", nil).Status, path)
	}
	assert.Equal(t, fiber.StatusUnauthorized, e.do(fiber.MethodPost, p, "", partnerForm(1)).Status)

	// No active service → empty catalog (no Today card), and a booking is refused.
	r := e.do(fiber.MethodGet, "/api/v1/partners/lab-services", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Empty(t, r.data()["services"])
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p, sara, partnerForm(1)).Status)

	res, err := e.db.Exec(`INSERT INTO partner_lab_services (code, kind, title, partner_name, partner_code, is_active, sort_order, created_at, updated_at)
		VALUES ('home_blood_draw', 'home_draw', '{"fa":"آزمایش در منزل","en":"Home blood draw"}', 'Lab X', 'fake', 1, 1, NOW(), NOW())`)
	require.NoError(t, err)
	sid, _ := res.LastInsertId()
	r = e.do(fiber.MethodGet, "/api/v1/partners/lab-services", sara, nil)
	require.Len(t, r.data()["services"], 1)

	// Validation: consent_share is required and must be true; unknown body keys are ignored.
	bad := partnerForm(sid)
	bad["consent_share"] = false
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p, sara, bad).Status)
	bad = partnerForm(sid)
	bad["preferred_date"] = "2030-01-01"
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p, sara, bad).Status)

	ok := partnerForm(sid)
	ok["user_id"] = 1
	r = e.do(fiber.MethodPost, p, sara, ok)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	id := strconv.Itoa(int(r.data()["id"].(float64)))

	// Owner-only: another user sees nothing and gets the same 404 for read, cancel and import.
	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodGet, p+"/"+id, sara, nil).Status)
	assert.Empty(t, e.do(fiber.MethodGet, p, ali, nil).data()["requests"])
	for _, c := range [][2]string{{fiber.MethodGet, p + "/" + id}, {fiber.MethodPost, p + "/" + id + "/cancel"},
		{fiber.MethodPost, p + "/" + id + "/import-result"}} {
		got := e.do(c[0], c[1], ali, map[string]any{"consent": true})
		assert.Equal(t, fiber.StatusNotFound, got.Status, c[1]+" "+got.Raw)
		assert.Equal(t, "lab_request_not_found", got.Body["error_code"])
	}
	assert.Equal(t, fiber.StatusNotFound, e.do(fiber.MethodGet, p+"/abc", sara, nil).Status)

	// A result that is not ready cannot be imported; consent is validated first.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.do(fiber.MethodPost, p+"/"+id+"/import-result", sara, map[string]any{}).Status)
	got := e.do(fiber.MethodPost, p+"/"+id+"/import-result", sara, map[string]any{"consent": true})
	assert.Equal(t, fiber.StatusConflict, got.Status, got.Raw)
	assert.Equal(t, "lab_result_not_ready", got.Body["error_code"])

	// The list carries no personal details; the detail does.
	list := e.do(fiber.MethodGet, p, sara, nil)
	assert.NotContains(t, list.Raw, "Valiasr")
	assert.Contains(t, e.do(fiber.MethodGet, p+"/"+id, sara, nil).Raw, "Valiasr")

	assert.Equal(t, fiber.StatusOK, e.do(fiber.MethodPost, p+"/"+id+"/cancel", sara, nil).Status)
	assert.Equal(t, fiber.StatusConflict, e.do(fiber.MethodPost, p+"/"+id+"/cancel", sara, nil).Status)
}
