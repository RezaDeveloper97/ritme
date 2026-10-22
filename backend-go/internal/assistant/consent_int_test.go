package assistant_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-M10-23: the per-item switches («دستیار چه چیزهایی را ببیند») leave consent-ledger evidence (assistant_profile_items,
// source feature, scope {items}); an unchanged choice adds none, turning everything off is a revoke of what was shared.
func TestProfileSwitches_ConsentLedger(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000301", false)
	for _, shared := range [][]string{{"cycle", "age"}, {"age", "cycle"}, {}} {
		r := e.do(t, "PUT", "/api/v1/assistant/profile", tok, map[string]any{"shared": shared})
		require.Equal(t, 200, r.status, r.raw)
	}
	rows, err := e.db.Query(`SELECT action, scope FROM consent_events WHERE user_id = ? AND consent = 'assistant_profile_items' AND source = 'feature' ORDER BY id`, uid)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var actions []string
	var items [][]any
	for rows.Next() {
		var a, s string
		require.NoError(t, rows.Scan(&a, &s))
		var m map[string][]any
		require.NoError(t, json.Unmarshal([]byte(s), &m))
		actions, items = append(actions, a), append(items, m["items"])
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"grant", "revoke"}, actions)
	assert.ElementsMatch(t, []any{"cycle", "age"}, items[0])
	assert.ElementsMatch(t, []any{"cycle", "age"}, items[1], "the revoke names what was no longer shared")
}
