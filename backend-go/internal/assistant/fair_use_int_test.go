package assistant_test

// T-M10-29 (A-04): a Plus user past the assistant's monthly fair-use cap gets a polite in-chat answer (200, SSE,
// guardrail fair_use_cap) — never the 429/402 of the AI gate and never a model call; an emergency still gets the
// 115 card.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/plus"
)

func TestAssistant_PlusFairUseCapPoliteReply(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000071", true) // trial = Plus allowances, running at testNow
	now, err := time.Parse(time.RFC3339, testNow)
	require.NoError(t, err)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "nutrition")

	// This month's uses already at the default cap (300).
	_, err = e.db.Exec(`INSERT INTO plus_usage_counters (user_id, feature, period_start, used, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())`,
		uid, string(plus.AssistantUnlimited), plus.PeriodStart(now), plus.DefaultAssistantFairUse)
	require.NoError(t, err)
	calls := e.spy.calls()

	r := e.send(t, tok, conv, map[string]any{"text": "پیام ۳۰۱"})
	require.Equal(t, 200, r.status, r.raw)
	done := find(events(r.raw), "done")
	require.NotNil(t, done, r.raw)
	assert.Equal(t, true, done["fair_use_cap"])
	msg, _ := done["message"].(map[string]any)
	require.NotNil(t, msg)
	assert.Contains(t, msg["text"], "سقف استفادهٔ منصفانه")
	assert.Equal(t, calls, e.spy.calls(), "no model call at the cap")

	r = e.send(t, tok, conv, map[string]any{"text": "می\u200cخواهم خودکشی کنم"})
	require.Equal(t, 200, r.status, r.raw)
	assert.NotNil(t, find(events(r.raw), "card"), "an emergency always gets the 115 card")
}
