package assistant_test

// T-M12-32 (P-08): a new message reads only the newest Options.HistoryTurns user / assistant turns of the
// conversation — the older ones are never sent to the model — while GET /conversations/{id} still lists every turn.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/ai"
	"github.com/ritme/backend-go/internal/assistant"
	"github.com/ritme/backend-go/internal/consent"
)

// seedTurns stores n alternating turns t01 (user), t02 (assistant), … as legacy plaintext bodies, then a card row and
// an empty assistant reply (a failed stream) — the two kinds that never go back to the model.
func (e *env) seedTurns(t *testing.T, conv uint64, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		role := assistant.RoleUser
		if i%2 == 0 {
			role = assistant.RoleAssistant
		}
		_, err := e.db.Exec(`INSERT INTO assistant_messages (conversation_id, role, body, status, created_at, updated_at)
			VALUES (?, ?, ?, 'complete', NOW(), NOW())`, conv, role, fmt.Sprintf("t%02d", i))
		require.NoError(t, err)
	}
	_, err := e.db.Exec(`INSERT INTO assistant_messages (conversation_id, role, body, card, status, created_at, updated_at)
		VALUES (?, 'card', '', 'emergency', 'complete', NOW(), NOW())`, conv)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO assistant_messages (conversation_id, role, body, status, created_at, updated_at)
		VALUES (?, 'assistant', '', 'failed', NOW(), NOW())`, conv)
	require.NoError(t, err)
}

func texts(msgs []ai.ChatMessage) []string {
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Text)
	}
	return out
}

func TestAssistant_HistoryWindowFromConfig(t *testing.T) {
	e := setupWith(t, 0, nil, func(o *assistant.Options) { o.HistoryTurns = 4 })
	uid, tok := e.user(t, "09120000071", false)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "nutrition")
	e.seedTurns(t, conv, 26)

	r := e.send(t, tok, conv, map[string]any{"text": "new question"})
	require.Equal(t, 200, r.status, r.raw)
	req := e.spy.last()
	// The newest 4 chat turns, oldest first, then the new message. The card and the empty reply stored after t26
	// take no slot (else only t25, t26 would be left); t01…t22 are never sent.
	assert.Equal(t, []string{"t23", "t24", "t25", "t26", "new question"}, texts(req.Messages))
	assert.Equal(t, ai.RoleUser, req.Messages[0].Role, "the window starts on a user turn")

	// The conversation still renders in full: 26 turns + card + failed reply + the new exchange.
	show := e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil)
	require.Equal(t, 200, show.status, show.raw)
	msgs, _ := show.data()["messages"].([]any)
	stored := e.count(t, `SELECT COUNT(*) FROM assistant_messages WHERE conversation_id = ?`, conv)
	assert.GreaterOrEqual(t, stored, 30)
	assert.Len(t, msgs, stored)
	first, _ := msgs[0].(map[string]any)
	assert.Equal(t, "t01", first["text"], "the oldest turn is still listed")
}

func TestAssistant_HistoryWindowDefaultAndOddBoundary(t *testing.T) {
	// Default window (20): with 26 stored turns the model gets t07…t26. A window of 3 starting on an assistant turn
	// (t24) drops it, so the model always opens with a user turn.
	e := setup(t)
	uid, tok := e.user(t, "09120000072", false)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "nutrition")
	e.seedTurns(t, conv, 26)
	r := e.send(t, tok, conv, map[string]any{"text": "q"})
	require.Equal(t, 200, r.status, r.raw)
	got := texts(e.spy.last().Messages)
	require.Len(t, got, assistant.HistoryTurns+1)
	assert.Equal(t, "t07", got[0])
	assert.NotContains(t, got, "t06")

	e3 := setupWith(t, 0, nil, func(o *assistant.Options) { o.HistoryTurns = 3 })
	uid3, tok3 := e3.user(t, "09120000073", false)
	e3.accept(t, uid3, consent.AIAssistant)
	conv3 := e3.newConv(t, tok3, "nutrition")
	e3.seedTurns(t, conv3, 26)
	r = e3.send(t, tok3, conv3, map[string]any{"text": "q"})
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, []string{"t25", "t26", "q"}, texts(e3.spy.last().Messages))
}
