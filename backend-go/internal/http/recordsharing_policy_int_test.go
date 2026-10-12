package http

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// User decisions 2026-10-12 (docs/security/canvas-record-review.md M2 / L5) through the production registry: the loss
// opt-in of a 24h doctor code, and the teen sharing policy (doctor code / report link only with an active parent link,
// the emergency card always).

func lossRows(t *testing.T, data map[string]any) int {
	t.Helper()
	rec, _ := data["record"].(map[string]any)
	secs, _ := rec["sections"].([]any)
	n := 0
	for _, s := range secs {
		sec := s.(map[string]any)
		if sec["key"] != "pregnancies" {
			continue
		}
		items, _ := sec["data"].(map[string]any)["items"].([]any)
		for _, it := range items {
			if m := it.(map[string]any); m["source"] == "tracked" && m["outcome"] == "ended" {
				n++
			}
		}
	}
	return n
}

func TestRecordSharing_CodeLossOptIn(t *testing.T) {
	e := newCompanionEnv(t)
	sara := e.user("Sara", "09120009401")
	_, err := e.db.Exec(`INSERT INTO pregnancy_losses (user_id, loss_type, occurred_on, private_note, next_step, created_at, updated_at)
		VALUES (?, 'miscarriage', '2026-03-10', 'Q0lQSEVSVEVYVA==', 'rest', '2026-03-11 09:00:00', '2026-03-11 09:00:00')`, e.ids["Sara"])
	require.NoError(t, err)
	const codes = "/api/v1/health-record/share-codes"

	r := e.do(fiber.MethodPost, codes, sara, map[string]any{"include_losses": "yes"})
	require.Equal(t, fiber.StatusUnprocessableEntity, r.Status, r.Raw)
	assert.Contains(t, r.Body["errors"], "include_losses")

	// Default: excluded, stored false.
	r = e.do(fiber.MethodPost, codes, sara, map[string]any{})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, false, r.data()["include_losses"])
	open := e.do(fiber.MethodGet, "/api/v1/shared-reports/"+r.data()["token"].(string), "", nil)
	require.Equal(t, fiber.StatusOK, open.Status, open.Raw)
	assert.Equal(t, 0, lossRows(t, open.data()))
	byCode := e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": r.data()["code"]})
	require.Equal(t, fiber.StatusOK, byCode.Status, byCode.Raw)
	assert.Equal(t, 0, lossRows(t, byCode.data()))

	// Opt-in: one neutral «ended» row, never the type or the date.
	r = e.do(fiber.MethodPost, codes, sara, map[string]any{"include_losses": true})
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	assert.Equal(t, true, r.data()["include_losses"])
	open = e.do(fiber.MethodGet, "/api/v1/shared-reports/"+r.data()["token"].(string), "", nil)
	require.Equal(t, fiber.StatusOK, open.Status, open.Raw)
	assert.Equal(t, 1, lossRows(t, open.data()))
	assert.NotContains(t, open.Raw, "miscarriage")
	assert.NotContains(t, open.Raw, "2026-03-10")
	byCode = e.do(fiber.MethodPost, "/api/v1/shared-reports/code", "", map[string]any{"code": r.data()["code"]})
	require.Equal(t, fiber.StatusOK, byCode.Status, byCode.Raw)
	assert.Equal(t, 1, lossRows(t, byCode.data()))

	// The owner's overview shows which code carries the losses.
	r = e.do(fiber.MethodGet, "/api/v1/health-record/sharing", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	items := r.data()["doctor"].(map[string]any)["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, true, items[0].(map[string]any)["include_losses"])
	assert.Equal(t, false, items[1].(map[string]any)["include_losses"])

	// The emergency card never carries a loss.
	r = e.do(fiber.MethodGet, "/api/v1/health-record/emergency-card", sara, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.NotContains(t, r.Raw, "ended")
	assert.NotContains(t, r.Raw, "miscarriage")
}

func TestRecordSharing_TeenPolicy(t *testing.T) {
	e := newCompanionEnv(t)
	nila := e.user("Nila", "09120009501")
	mom := e.user("Mom", "09120009502")
	adult := e.user("Adult", "09120009503")
	e.teenMode("Nila")
	const codes = "/api/v1/health-record/share-codes"

	// Without a parent link: no doctor code, no report link (403 before any Plus paywall), the emergency card works.
	r := e.do(fiber.MethodGet, "/api/v1/health-record/sharing", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, map[string]any{"teen": true, "parent_linked": false, "doctor_sharing": false, "emergency_card": true},
		r.data()["policy"])
	r = e.do(fiber.MethodPost, codes, nila, map[string]any{})
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "teen_share_needs_parent", r.Body["error_code"])
	assert.NotEmpty(t, r.Body["message"])
	r = e.do(fiber.MethodPost, "/api/v1/health-record/share-links", nila, map[string]any{"range": "3m", "sections": []string{"basics"}})
	assert.Equal(t, fiber.StatusForbidden, r.Status, r.Raw)
	assert.Equal(t, "teen_share_needs_parent", r.Body["error_code"])
	assert.Zero(t, e.count("SELECT COUNT(*) FROM health_share_links WHERE user_id = ?", e.ids["Nila"]))

	r = e.do(fiber.MethodGet, "/api/v1/health-record/emergency-card", nila, nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodPut, "/api/v1/health-record/emergency-card", nila, map[string]any{"show_on_lock_screen": true})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodPost, "/api/v1/health-record/emergency-card/public-link", nila, nil)
	require.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
	r = e.do(fiber.MethodGet, "/api/v1/emergency-cards/"+r.data()["token"].(string), "", nil)
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	assert.Equal(t, "Nila", r.data()["name"])

	// A pending invite is not enough; once her mother accepts, the doctor code works.
	_, code := e.invite(nila, map[string]any{"type": "parent", "display_name": "Mom", "phone": "09120009502",
		"grants": map[string]any{"teen_notes": "view"}})
	assert.Equal(t, fiber.StatusForbidden, e.do(fiber.MethodPost, codes, nila, map[string]any{}).Status)
	r = e.do(fiber.MethodPost, "/api/v1/companions/accept", mom, map[string]any{"code": code})
	require.Equal(t, fiber.StatusOK, r.Status, r.Raw)
	r = e.do(fiber.MethodGet, "/api/v1/health-record/sharing", nila, nil)
	assert.Equal(t, true, r.data()["policy"].(map[string]any)["parent_linked"])
	assert.Equal(t, true, r.data()["policy"].(map[string]any)["doctor_sharing"])
	r = e.do(fiber.MethodPost, codes, nila, map[string]any{"sections": []string{"basics"}})
	assert.Equal(t, fiber.StatusCreated, r.Status, r.Raw)

	// An adult is never gated.
	r = e.do(fiber.MethodPost, codes, adult, map[string]any{})
	assert.Equal(t, fiber.StatusCreated, r.Status, r.Raw)
}
