package assistant_test

// B-N7-06b security fixes end to end: bodies / summaries sealed at rest, the AI gate's in-flight slot held for the
// whole stream, summary throttles and no free regeneration, a model-rated emergency → 115 card, links stripped from
// the live stream, photos (reuse, orphan sweep, delete-all).

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// testLimiter is a ratelimit.Limiter on the test Redis under a unique prefix (the test skips without TEST_REDIS_ADDR).
func testLimiter(t *testing.T) *ratelimit.Limiter {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	c := cache.NewFromClient(redis.NewClient(&redis.Options{Addr: addr}), "ritme-go-test-"+hex.EncodeToString(b)+":")
	t.Cleanup(func() { _ = c.Close() })
	return ratelimit.New(c, clock.Real{})
}

func r0summary(t *testing.T, e *env, conv uint64) string {
	t.Helper()
	_, tok := e.ownerToken(t, conv)
	r := e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	s, _ := r.data()["summary"].(map[string]any)["text"].(string)
	require.NotEmpty(t, s)
	return s
}

func (e *env) ownerToken(t *testing.T, conv uint64) (uint64, string) {
	t.Helper()
	var uid uint64
	require.NoError(t, e.db.QueryRow(`SELECT user_id FROM assistant_conversations WHERE id = ?`, conv).Scan(&uid))
	tok, err := e.iss.Issue(t.Context(), uid, time.Now())
	require.NoError(t, err)
	return uid, tok.AccessToken
}

func deltas(raw string) string {
	var b strings.Builder
	for _, ev := range events(raw) {
		if ev.name == "delta" {
			b.WriteString(ev.data["text"].(string))
		}
	}
	return b.String()
}

func TestAssistant_SealedAtRest(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", false)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "general")
	const secret = "RITME-FAKE:triage سردرد پشت چشم دارم"
	r := e.send(t, tok, conv, map[string]any{"text": secret})
	require.Equal(t, 200, r.status, r.raw)
	r = e.do(t, "POST", fmt.Sprintf("/api/v1/assistant/conversations/%d/summary", conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)

	rows, err := e.db.Query(`SELECT body FROM assistant_messages WHERE conversation_id = ?`, conv)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	n := 0
	for rows.Next() {
		var body string
		require.NoError(t, rows.Scan(&body))
		assert.True(t, strings.HasPrefix(body, "v1:"), "body sealed: %q", body)
		assert.NotContains(t, body, "سردرد")
		n++
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, 2, n)
	var summary string
	var title *string
	require.NoError(t, e.db.QueryRow(`SELECT summary, title FROM assistant_conversations WHERE id = ?`, conv).Scan(&summary, &title))
	assert.True(t, strings.HasPrefix(summary, "v1:"), "summary sealed")
	assert.Nil(t, title, "no plaintext title at rest")

	// The API reads plaintext: the title comes from the sealed first turn.
	r = e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, secret, r.data()["messages"].([]any)[0].(map[string]any)["text"])
	assert.Equal(t, secret, r.data()["conversation"].(map[string]any)["title"])
	r = e.do(t, "GET", "/api/v1/assistant/conversations", tok, nil)
	assert.Equal(t, secret, r.data()["conversations"].([]any)[0].(map[string]any)["title"])

	// A ciphertext moved to another row does not open (bound to the row id); a legacy plaintext row reads as is.
	_, err = e.db.Exec(`INSERT INTO assistant_messages (conversation_id, role, body, status, created_at, updated_at)
		SELECT conversation_id, 'assistant', body, 'complete', NOW(), NOW() FROM assistant_messages WHERE conversation_id = ? ORDER BY id LIMIT 1`, conv)
	require.NoError(t, err)
	_, err = e.db.Exec(`INSERT INTO assistant_messages (conversation_id, role, body, status, created_at, updated_at)
		VALUES (?, 'assistant', 'legacy plaintext', 'complete', NOW(), NOW())`, conv)
	require.NoError(t, err)
	r = e.do(t, "GET", fmt.Sprintf("/api/v1/assistant/conversations/%d", conv), tok, nil)
	msgs := r.data()["messages"].([]any)
	require.Len(t, msgs, 4)
	assert.Empty(t, msgs[2].(map[string]any)["text"])
	assert.Equal(t, "legacy plaintext", msgs[3].(map[string]any)["text"])
}

func TestAssistant_InFlightSlotHeldUntilStreamEnds(t *testing.T) {
	e := setupWith(t, 1, nil)
	uid, tok := e.user(t, "09120000001", true)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "general")
	block, started := make(chan struct{}), make(chan struct{}, 4)
	e.spy.set(func(s *spyChat) { s.block, s.started = block, started })

	first := make(chan resp, 1)
	go func() { first <- e.send(t, tok, conv, map[string]any{"text": "first"}) }()
	<-started
	time.Sleep(200 * time.Millisecond) //nolint:sleep // the handler returned; the stream writer then waits on the provider (no hook to observe it)
	r := e.send(t, tok, conv, map[string]any{"text": "second"})
	assert.Equal(t, 429, r.status, r.raw)
	assert.Equal(t, "ai_busy", r.body["error_code"])

	close(block)
	r = <-first
	require.Equal(t, 200, r.status, r.raw)
	require.NotNil(t, find(events(r.raw), "done"), r.raw)
	e.spy.set(func(s *spyChat) { s.block, s.started = nil, nil })
	r = e.send(t, tok, conv, map[string]any{"text": "third"})
	assert.Equal(t, 200, r.status, "the slot is free once the stream ended")
}

func TestAssistant_SummaryThrottleAndNoFreeRegeneration(t *testing.T) {
	e := setupWith(t, 0, testLimiter(t))
	uid, tok := e.user(t, "09120000001", false)
	e.accept(t, uid, consent.AIAssistant)
	path := func(c uint64) string { return fmt.Sprintf("/api/v1/assistant/conversations/%d/summary", c) }

	// Emergency turns only: no model answer → nothing to summarise, no model call.
	only := e.newConv(t, tok, "general")
	require.Equal(t, 200, e.send(t, tok, only, map[string]any{"text": "chest pain"}).status)
	r := e.do(t, "POST", path(only), tok, nil)
	assert.Equal(t, 422, r.status, r.raw)
	assert.Equal(t, 0, e.spy.calls())

	conv := e.newConv(t, tok, "general")
	require.Equal(t, 200, e.send(t, tok, conv, map[string]any{"text": "سلام"}).status)
	require.Equal(t, 200, e.do(t, "POST", path(conv), tok, nil).status)
	calls := e.spy.calls()
	// New emergency turns do not buy a regenerated summary.
	require.Equal(t, 200, e.send(t, tok, conv, map[string]any{"text": "heavy bleeding"}).status)
	r = e.do(t, "POST", path(conv), tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, calls, e.spy.calls(), "no model call without a new answer")

	// The feature's burst throttle (10 / minute, shared with the chat) applies to summaries.
	status := 0
	for range 12 {
		if status = e.do(t, "POST", path(conv), tok, nil).status; status == 429 {
			break
		}
	}
	assert.Equal(t, 429, status)
}

func TestAssistant_ModelEmergencyCardAndLinksStripped(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "general")

	e.spy.set(func(s *spyChat) {
		s.reply = "See https://evil.example/x?y=1 and www.bad.example today. Call now.\n[[triage:emergency]]"
	})
	r := e.send(t, tok, conv, map[string]any{"text": "دلم درد می\u200cکند"})
	require.Equal(t, 200, r.status, r.raw)
	streamed := deltas(r.raw)
	assert.NotContains(t, streamed, "evil")
	assert.NotContains(t, streamed, "www.")
	assert.NotContains(t, streamed, "[[")
	assert.Contains(t, streamed, "Call now.")
	evs := events(r.raw)
	card := find(evs, "card")
	require.NotNil(t, card, r.raw)
	assert.Equal(t, "115", card["message"].(map[string]any)["card"].(map[string]any)["phone"])
	done := find(evs, "done")
	assert.Equal(t, true, done["emergency"])
	assert.NotContains(t, done["message"].(map[string]any)["text"], "evil")
	assert.Equal(t, 1, e.count(t, `SELECT emergency FROM assistant_conversations WHERE id = ?`, conv))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM assistant_messages WHERE conversation_id = ? AND card = 'emergency'`, conv))
}

func TestAssistant_PhotoReuseSweepAndDeleteAll(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	e.accept(t, uid, consent.AIAssistant)
	conv := e.newConv(t, tok, "skin_beauty")
	photoID := func(r resp) any {
		require.Equal(t, 201, r.status, r.raw)
		return r.data()["photo"].(map[string]any)["id"]
	}

	used := photoID(e.upload(t, tok, pngBytes(t)))
	require.Equal(t, 200, e.send(t, tok, conv, map[string]any{"text": "look", "photo_id": used}).status)
	r := e.send(t, tok, conv, map[string]any{"text": "again", "photo_id": used})
	assert.Equal(t, 422, r.status, "a photo is attached once")

	stale := photoID(e.upload(t, tok, pngBytes(t)))
	_, err := e.db.Exec(`UPDATE files SET created_at = '2026-09-01 09:00:00' WHERE id = ?`, stale)
	require.NoError(t, err)
	_, err = e.db.Exec(`UPDATE files SET created_at = '2026-09-01 09:00:00' WHERE id = ?`, used)
	require.NoError(t, err)
	orphan := photoID(e.upload(t, tok, pngBytes(t)))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM files WHERE id = ?`, stale), "an unattached photo older than 24 h is swept")
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM files WHERE id = ?`, used), "an attached one stays")

	r = e.do(t, "DELETE", "/api/v1/assistant/conversations", tok, nil)
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM files WHERE user_id = ? AND purpose = 'assistant_photo'`, uid),
		"delete-all removes attached and unattached photos (orphan %v)", orphan)
}
