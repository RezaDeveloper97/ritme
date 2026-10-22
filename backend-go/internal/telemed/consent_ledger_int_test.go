package telemed_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-23: every telemed share change is a consent-ledger event (telemed_share, source telemed) whose scope names the
// booking, its doctor and the scopes that changed — at booking time, on a revoke, on a re-grant and on a full revoke.
// An unchanged scope writes nothing. Evidence only: AuthorizeShare is unchanged.
func TestBooking_ConsentLedgerEvents(t *testing.T) {
	be := setupBooking(t, true)
	uid, tok := be.user(t, "Nava", "09120000001")
	bid := be.pay(t, tok, "16:00", map[string]any{"share": []any{"cycle_summary", "bbt_lh"}})
	p := fmt.Sprintf("/api/v1/telemed/bookings/%d/consent", bid)

	r := be.do(t, http.MethodPut, p, tok, "fa", map[string]any{"bbt_lh": false, "cycle_summary": true})
	require.Equal(t, http.StatusOK, r.status, r.body)
	r = be.do(t, http.MethodPut, p, tok, "fa", map[string]any{"bbt_lh": true})
	require.Equal(t, http.StatusOK, r.status, r.body)
	r = be.do(t, http.MethodDelete, p, tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	r = be.do(t, http.MethodDelete, p, tok, "fa", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)

	scope := func(items string) string {
		return fmt.Sprintf(`{"booking_id": %d, "doctor_id": %d, "items": [%s]}`, bid, be.a, items)
	}
	type ev struct{ action, scope string }
	rows, err := be.DB.Query(`SELECT action, JSON_EXTRACT(scope, '$') FROM consent_events
		WHERE user_id = ? AND consent = 'telemed_share' AND source = 'telemed' AND version = 1 ORDER BY id`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var got []ev
	for rows.Next() {
		var e ev
		require.NoError(t, rows.Scan(&e.action, &e.scope))
		got = append(got, e)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 4)
	assert.Equal(t, "grant", got[0].action)
	assert.JSONEq(t, scope(`"cycle_summary", "bbt_lh"`), got[0].scope)
	assert.Equal(t, "revoke", got[1].action)
	assert.JSONEq(t, scope(`"bbt_lh"`), got[1].scope, "cycle_summary was already shared: no event for it")
	assert.Equal(t, "grant", got[2].action)
	assert.JSONEq(t, scope(`"bbt_lh"`), got[2].scope)
	assert.Equal(t, "revoke", got[3].action)
	assert.ElementsMatch(t, []any{"cycle_summary", "bbt_lh"}, decodeItems(t, got[3].scope), "the second full revoke changed nothing")
}

func decodeItems(t *testing.T, scope string) []any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(scope), &m))
	items, _ := m["items"].([]any)
	return items
}
