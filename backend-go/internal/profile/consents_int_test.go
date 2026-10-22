package profile_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-23: a Privacy toggle writes a consent-ledger event (source settings) with the projection; an unchanged toggle
// and a «no» to what was never granted write none; account deletion (DELETE /account) still works with events present
// and removes them.
func TestConsents_PrivacyTogglesWriteTheLedger(t *testing.T) {
	e, _ := privacyEnv(t)
	id, tok := e.user(t, "09120000201")
	for _, body := range []string{
		`{"consents":{"ai_lab_analysis":true,"anonymous_stats":false},"versions":{"ai_lab_analysis":1}}`,
		`{"consents":{"ai_lab_analysis":true},"versions":{"ai_lab_analysis":1}}`,
		`{"consents":{"ai_lab_analysis":false,"anonymous_stats":true}}`,
	} {
		r := e.do(t, http.MethodPut, "/api/v1/profile/consents", tok, body)
		require.Equal(t, 200, r.status, r.raw)
	}
	rows, err := e.db.Query(`SELECT CONCAT(consent, ':', version, ':', action, ':', source) FROM consent_events WHERE user_id = ? ORDER BY id`, id)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var got []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		got = append(got, s)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{
		"ai_lab_analysis:1:grant:settings",
		"ai_lab_analysis:1:revoke:settings",
		"anonymous_stats:1:grant:settings",
	}, got)

	r := e.do(t, http.MethodDelete, "/api/v1/account", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM consent_events WHERE user_id = ?`, id).Scan(&n))
	assert.Zero(t, n, "the ledger goes with the account (FK cascade past the append-only triggers)")
}
