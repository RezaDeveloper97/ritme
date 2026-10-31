package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// TestMessagesV2Routes covers GET /messages/selected and POST /messages/events (B-N9-05, D-86; since T-M10-06
// /selected is the feed of one slot under the SPEC §5.4 slot × kind table): 401, validation, the A/B variant staying
// the same for a user, the server-side shown (once per message, slot and day), the cooldown it starts, an unapproved
// medical message that is never served, the safety slot, and the event endpoint's aliases and rate limit.
func TestMessagesV2Routes(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	exec := func(query string, args ...any) {
		t.Helper()
		_, err := db.Exec(query, args...)
		require.NoError(t, err)
	}
	exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffeeb', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`)
	iss := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara K', '09120003001', NOW(), NOW())`)
	require.NoError(t, err)
	uid, _ := res.LastInsertId()
	tok, err := iss.Issue(context.Background(), uint64(uid), time.Now()) //nolint:gosec // test ids
	require.NoError(t, err)
	token := tok.AccessToken
	exec(`INSERT INTO user_profiles (user_id, subscription_type, user_goal, cycle_duration, last_period_start, created_at, updated_at)
		VALUES (?, 'free', 'non_ttc', 28, '2026-09-20', NOW(), NOW())`, uid)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	Mount(app, &Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"}, SMS: config.SMS{Provider: "log"},
			Passport: config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30}, StoragePath: keysPath,
		},
		DB: db, Cache: cache.NewFromClient(rdb, "ritme-go-msgv2:"), Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})

	call := func(method, path, at string, body any) (int, map[string]any) {
		t.Helper()
		var rd io.Reader = nethttp.NoBody
		if body != nil {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, rd)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", "en")
		if token != "" && at != "anon" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if at != "" && at != "anon" {
			req.Header.Set(clock.Header, at)
		}
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	const at = "2026-10-11T12:00:00+03:30"
	const sel = "/api/v1/messages/selected?placement=smart_tip"

	st, body := call(fiber.MethodGet, sel, "anon", nil)
	assert.Equal(t, 401, st)
	assert.Equal(t, "unauthenticated", body["error_code"])
	st, _ = call(fiber.MethodGet, "/api/v1/messages/selected?placement=bogus", at, nil)
	assert.Equal(t, 422, st)

	st, body = call(fiber.MethodGet, sel, at, nil)
	require.Equal(t, 200, st, body)
	assert.Nil(t, body["data"].(map[string]any)["message"], "no rules yet")

	content := `{"a":{"title":"Hi {name}","body":"body","cta_label":"Go","cta_target":"ritme://x"},"b":{"title":"Yo {name}","body":"body","cta_label":"Go","cta_target":"ritme://x"}}`
	ins := func(title, level, kind, placement string, prio int, status string, cooldown int) uint64 {
		t.Helper()
		approved := "NULL"
		if status == "approved" {
			approved = "1"
		}
		r, err := db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
			conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
			approved_version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, 50, '[]', '[]', NULL, NULL, ?, 100, ?, 1, 1, `+approved+`, NOW(), NOW())`,
			title, level, kind, prio, placement, content, cooldown, status)
		require.NoError(t, err)
		id, _ := r.LastInsertId()
		return uint64(id) //nolint:gosec // test ids
	}
	count := func(q string, args ...any) int {
		t.Helper()
		var n int
		require.NoError(t, db.QueryRow(q, args...).Scan(&n))
		return n
	}
	// Event rows go through the buffered writer (≤ 1 s, T-M10-06): poll.
	eventually := func(want int, q string, args ...any) {
		t.Helper()
		require.Eventually(t, func() bool { return count(q, args...) == want }, 5*time.Second, 50*time.Millisecond,
			"%s: want %d, have %d", q, want, count(q, args...))
	}
	msgOf := func(body map[string]any) map[string]any {
		t.Helper()
		m, _ := body["data"].(map[string]any)["message"].(map[string]any)
		return m
	}
	const challenge = "/api/v1/messages/selected?placement=daily_challenge"
	const shownOf = "SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'impression'"

	// A product message in its slot: {name} filled, no medical fields, a stable A/B variant, one server-side shown.
	product := ins("product", "suggestion", "product", "daily_challenge", 50, "draft", 0)
	st, body = call(fiber.MethodGet, challenge, at, nil)
	require.Equal(t, 200, st, body)
	msg := msgOf(body)
	require.NotNil(t, msg, body)
	assert.EqualValues(t, product, msg["rule_id"])
	assert.Equal(t, "daily_challenge", msg["placement"])
	assert.Equal(t, "Go", msg["cta"].(map[string]any)["label"])
	assert.Contains(t, msg["title"], "Sara", "{name} is filled")
	assert.NotContains(t, msg, "seen", "an info/suggestion message carries no medical fields")
	variant := msg["variant"]
	for range 3 {
		_, body = call(fiber.MethodGet, challenge, at, nil)
		require.NotNil(t, msgOf(body))
		assert.Equal(t, variant, msgOf(body)["variant"], "the same user keeps the variant")
	}
	eventually(1, shownOf, uid, product)
	time.Sleep(1500 * time.Millisecond) // past one flush: the refetches logged nothing more
	assert.Equal(t, 1, count(shownOf, uid, product), "shown once per message, slot and day")

	// The kind table: a product message is not a smart_tip (kind_not_allowed).
	misplaced := ins("misplaced", "suggestion", "product", "smart_tip", 900, "approved", 0)
	_, body = call(fiber.MethodGet, sel, at, nil)
	assert.Nil(t, msgOf(body), "a product rule in smart_tip is never served")
	exec("UPDATE message_rules SET is_active = 0 WHERE id = ?", misplaced)

	// The server-side shown starts the cooldown.
	exec("UPDATE message_rules SET is_active = 0 WHERE id = ?", product)
	cool := ins("cooldown", "suggestion", "product", "daily_challenge", 60, "draft", 1)
	_, body = call(fiber.MethodGet, challenge, at, nil)
	require.NotNil(t, msgOf(body))
	assert.EqualValues(t, cool, msgOf(body)["rule_id"])
	eventually(1, shownOf, uid, cool)
	_, body = call(fiber.MethodGet, challenge, at, nil)
	assert.Nil(t, msgOf(body), "inside the 1-day cooldown")
	_, body = call(fiber.MethodGet, challenge, "2026-10-13T12:00:00+03:30", nil)
	assert.NotNil(t, msgOf(body), "cooldown elapsed")

	// Medical approval: a pending clinical message is never served; an approved one is.
	unapproved := ins("clinical pending", "worth_followup", "clinical", "smart_tip", 700, "pending_approval", 0)
	approved := ins("clinical", "info", "clinical", "smart_tip", 500, "approved", 0)
	_, body = call(fiber.MethodGet, sel, at, nil)
	require.NotNil(t, msgOf(body), body)
	assert.EqualValues(t, approved, msgOf(body)["rule_id"], "an unapproved medical message is never served")
	assert.NotEqual(t, unapproved, msgOf(body)["rule_id"])

	// The safety slot: never dismissible, with its contact.
	safety := ins("safety", "follow_soon", "safety", "urgent_modal", 1000, "approved", 0)
	exec(`UPDATE message_rules SET content = '{"a":{"title":"Call","body":"b"},"seen":"s","confidence":"c","contact":"tel:115"}', variants_enabled = 0 WHERE id = ?`, safety)
	_, body = call(fiber.MethodGet, "/api/v1/messages/selected?placement=urgent_modal", at, nil)
	msg = msgOf(body)
	require.NotNil(t, msg, body)
	assert.EqualValues(t, safety, msg["rule_id"])
	assert.Equal(t, "tel:115", msg["contact"])
	assert.Equal(t, false, msg["dismissible"])

	// Events: validation, unknown rule, a client impression the server already logged, a click.
	const ev = "/api/v1/messages/events"
	st, _ = call(fiber.MethodPost, ev, "anon", map[string]any{"rule_id": product, "event": "impression"})
	assert.Equal(t, 401, st)
	st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": product, "event": "bogus"})
	assert.Equal(t, 422, st)
	st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": 99999, "event": "click"})
	assert.Equal(t, 404, st)
	st, body = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": approved, "variant": "a", "event": "impression"})
	require.Equal(t, 201, st, body)
	assert.Equal(t, false, body["data"].(map[string]any)["recorded"], "the server logged today's shown already")
	st, body = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": approved, "variant": "a", "event": "clicked"})
	require.Equal(t, 201, st, body)
	eventually(1, "SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'click' AND slot = 'smart_tip'", uid, approved)

	// Rate limit: 120 / minute / user.
	limited := false
	for range 125 {
		if st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": approved, "event": "click"}); st == 429 {
			limited = true
			break
		}
	}
	assert.True(t, limited, "the events endpoint is rate limited")
}
