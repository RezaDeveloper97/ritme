package sharelinks_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-23: creating a share link appends a grant event (share_link, source sharelink) whose scope is {link_id,
// expires_at, items}; the first revoke appends a revoke with {link_id}; a second revoke and a foreign revoke add
// nothing. Evidence only — the token stays the gate.
func TestShareLinks_ConsentLedgerEvents(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	e.seed(t, uid)
	_, otherTok := e.user(t, "09120000002", true)

	r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "6m", "sections": []string{"basics", "labs"}})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := uint64(r.data()["id"].(float64))
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, "/api/v1/health-record/share-links/"+jsonNum(id), otherTok, nil).status)
	r = e.do(t, http.MethodDelete, "/api/v1/health-record/share-links/"+jsonNum(id), tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	_ = e.do(t, http.MethodDelete, "/api/v1/health-record/share-links/"+jsonNum(id), tok, nil) // 200 or 404 (same clock): no event either way

	rows, err := e.db.Query(`SELECT action, scope FROM consent_events WHERE user_id = ? AND consent = 'share_link' AND source = 'sharelink' ORDER BY id`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var actions []string
	var scopes []map[string]any
	for rows.Next() {
		var a, s string
		require.NoError(t, rows.Scan(&a, &s))
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(s), &m))
		actions, scopes = append(actions, a), append(scopes, m)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"grant", "revoke"}, actions)
	assert.InDelta(t, float64(id), scopes[0]["link_id"], 0)
	assert.Equal(t, "2026-10-13T09:00:00+03:30", scopes[0]["expires_at"])
	assert.Equal(t, []any{"basics", "labs"}, scopes[0]["items"])
	assert.Equal(t, map[string]any{"link_id": float64(id)}, scopes[1])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM consent_events WHERE user_id <> ?`, uid).Scan(&n))
	assert.Zero(t, n, "the foreign revoke attempt wrote nothing")
}
