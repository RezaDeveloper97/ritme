package sharelinks_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// User decisions 2026-10-12 (docs/security/canvas-record-review.md M2 / L5) on the 7-day report link: the loss opt-in
// and the teen sharing policy.

// tracked «ended» rows of a report / snapshot body (the loss-derived pregnancies).
func endedRows(t *testing.T, data map[string]any) int {
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
			m := it.(map[string]any)
			if m["source"] == "tracked" && m["outcome"] == "ended" {
				n++
				assert.Nil(t, m["date"], "a loss never carries its date")
			}
		}
	}
	return n
}

func (e *env) seedLoss(t *testing.T, uid uint64) {
	t.Helper()
	e.exec(t, `INSERT INTO pregnancy_losses (user_id, loss_type, occurred_on, private_note, next_step, created_at, updated_at)
		VALUES (?, 'ectopic', '2025-11-20', 'Q0lQSEVSVEVYVA==', 'ttc', '2025-11-21 09:00:00', '2025-11-21 09:00:00')`, uid)
}

func TestShareLinks_LossOptIn(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	e.seed(t, uid)
	e.seedLoss(t, uid)
	const p = "/api/v1/health-record/share-links"
	sections := []string{"basics", "pregnancies"}

	// Preview: off by default, on only with include_losses=1; a non-boolean is a 422.
	r := e.do(t, http.MethodGet, "/api/v1/health-record/report?range=1y&sections=basics,pregnancies", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, endedRows(t, r.data()))
	r = e.do(t, http.MethodGet, "/api/v1/health-record/report?range=1y&sections=basics,pregnancies&include_losses=1", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 1, endedRows(t, r.data()))
	assert.NotContains(t, r.raw, "ectopic")
	assert.NotContains(t, r.raw, "2025-11-20")
	r = e.do(t, http.MethodGet, "/api/v1/health-record/report?range=1y&sections=basics&include_losses=maybe", tok, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)

	// A link without the switch: stored false, the public read carries no loss.
	r = e.do(t, http.MethodPost, p, tok, map[string]any{"range": "1y", "sections": sections})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, false, r.data()["include_losses"])
	off := r.data()["token"].(string)
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+off, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, endedRows(t, r.data()))

	// With the switch: stored true, the read carries one neutral «ended» row.
	r = e.do(t, http.MethodPost, p, tok, map[string]any{"range": "1y", "sections": sections, "include_losses": true})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, true, r.data()["include_losses"])
	on := r.data()["token"].(string)
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+on, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 1, endedRows(t, r.data()))
	assert.NotContains(t, r.raw, "ectopic")

	// The read path enforces the stored choice: a link flagged off never shows a loss, even if its snapshot has one.
	e.exec(t, `UPDATE health_share_links SET include_losses = 0 WHERE user_id = ?`, uid)
	r = e.do(t, http.MethodGet, "/api/v1/shared-reports/"+on, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, endedRows(t, r.data()))

	r = e.do(t, http.MethodGet, p, tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	items := r.data()["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, false, items[0].(map[string]any)["include_losses"])
}

func TestShareLinks_TeenNeedsParentLink(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true) // a running Plus trial: the teen gate answers before the paywall anyway
	e.exec(t, `INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at) VALUES (?, 'female', 'teen', NOW(), NOW())`, uid)
	_, freeTeenTok := e.user(t, "09120000003", false)
	e.exec(t, `INSERT INTO user_life_profiles (user_id, gender, life_mode, created_at, updated_at)
		SELECT id, 'female', 'teen', NOW(), NOW() FROM users WHERE mobile = '09120000003'`)
	const p = "/api/v1/health-record/share-links"
	body := map[string]any{"range": "3m", "sections": []string{"basics"}}

	r := e.do(t, http.MethodGet, p, tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"teen": true, "parent_linked": false, "doctor_sharing": false, "emergency_card": true},
		r.data()["policy"])

	for _, tk := range []string{tok, freeTeenTok} {
		r = e.do(t, http.MethodPost, p, tk, body)
		assert.Equal(t, http.StatusForbidden, r.status, r.raw)
		assert.Equal(t, "teen_share_needs_parent", r.body["error_code"], "never a Plus paywall for a teen")
		assert.Contains(t, r.body["message"], "parent link")
	}
	var n int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM health_share_links WHERE user_id = ?", uid).Scan(&n))
	assert.Zero(t, n)

	// An invited (not yet accepted) parent link does not count; an active one does.
	e.exec(t, `INSERT INTO companions (owner_id, type, status, created_at, updated_at) VALUES (?, 'parent', 'invited', NOW(), NOW())`, uid)
	assert.Equal(t, http.StatusForbidden, e.do(t, http.MethodPost, p, tok, body).status)
	e.exec(t, `UPDATE companions SET status = 'active' WHERE owner_id = ?`, uid)
	r = e.do(t, http.MethodGet, p, tok, nil)
	assert.Equal(t, true, r.data()["policy"].(map[string]any)["doctor_sharing"])
	r = e.do(t, http.MethodPost, p, tok, body)
	assert.Equal(t, http.StatusCreated, r.status, r.raw)

	// An adult is never gated.
	_, adultTok := e.user(t, "09120000002", true)
	r = e.do(t, http.MethodGet, p, adultTok, nil)
	assert.Equal(t, map[string]any{"teen": false, "parent_linked": false, "doctor_sharing": true, "emergency_card": true},
		r.data()["policy"])
	assert.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, p, adultTok, body).status)
}
