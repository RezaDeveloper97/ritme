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

// TestMessagesV2Routes covers GET /messages/selected and POST /messages/events (B-N9-05, D-86): 401, validation,
// safety-first selection, an unapproved medical message that is never served, the cooldown after an impression, the
// A/B variant staying the same for a user, and the event endpoint's rate limit.
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
	ins := func(title, level, kind string, prio int, status string, active int, extra string) uint64 {
		r, err := db.Exec(`INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
			conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'smart_tip', ?, 1, 50, '[]', '[]', NULL, NULL, `+extra+`, 100, ?, ?, 1, NOW(), NOW())`,
			title, level, kind, prio, content, status, active)
		require.NoError(t, err)
		id, _ := r.LastInsertId()
		return uint64(id) //nolint:gosec // test ids
	}
	product := ins("product", "suggestion", "product", 50, "draft", 1, "1")
	unapproved := ins("clinical pending", "worth_followup", "clinical", 700, "pending_approval", 1, "0")

	st, body = call(fiber.MethodGet, sel, at, nil)
	require.Equal(t, 200, st, body)
	msg := body["data"].(map[string]any)["message"].(map[string]any)
	assert.EqualValues(t, product, msg["rule_id"], "an unapproved medical message is never served")
	assert.Equal(t, "Go", msg["cta"].(map[string]any)["label"])
	assert.Contains(t, msg["title"], "Sara", "{name} is filled")
	variant := msg["variant"]
	assert.NotContains(t, msg, "seen", "an info/suggestion message carries no medical fields")

	exec("UPDATE message_rules SET status = 'approved' WHERE id = ?", unapproved)
	safety := ins("safety", "follow_soon", "safety", 1000, "approved", 1, "0")
	exec(`UPDATE message_rules SET content = '{"a":{"title":"Call","body":"b"},"seen":"s","confidence":"c","contact":"tel:115"}', variants_enabled = 0 WHERE id = ?`, safety)
	_, body = call(fiber.MethodGet, sel, at, nil)
	msg = body["data"].(map[string]any)["message"].(map[string]any)
	assert.EqualValues(t, safety, msg["rule_id"], "safety first")
	assert.Equal(t, "tel:115", msg["contact"])
	assert.Equal(t, false, msg["dismissible"])
	exec("DELETE FROM message_rules WHERE id = ?", safety)
	_, body = call(fiber.MethodGet, sel, at, nil)
	assert.EqualValues(t, unapproved, body["data"].(map[string]any)["message"].(map[string]any)["rule_id"], "approved clinical beats product")
	exec("UPDATE message_rules SET is_active = 0 WHERE id = ?", unapproved)

	// Stable A/B: the same user keeps the variant.
	for range 3 {
		_, body = call(fiber.MethodGet, sel, at, nil)
		assert.Equal(t, variant, body["data"].(map[string]any)["message"].(map[string]any)["variant"])
	}

	// Events: validation, unknown rule, then an impression starts the 1-day cooldown.
	const ev = "/api/v1/messages/events"
	st, _ = call(fiber.MethodPost, ev, "anon", map[string]any{"rule_id": product, "event": "impression"})
	assert.Equal(t, 401, st)
	st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": product, "event": "bogus"})
	assert.Equal(t, 422, st)
	st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": 99999, "event": "click"})
	assert.Equal(t, 404, st)
	st, body = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": product, "variant": variant, "event": "impression"})
	require.Equal(t, 201, st, body)
	var n int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM message_events WHERE user_id = ? AND rule_id = ? AND event = 'impression'", uid, product).Scan(&n))
	assert.Equal(t, 1, n)
	_, body = call(fiber.MethodGet, sel, at, nil)
	assert.Nil(t, body["data"].(map[string]any)["message"], "inside the 1-day cooldown")
	_, body = call(fiber.MethodGet, sel, "2026-10-13T12:00:00+03:30", nil)
	assert.NotNil(t, body["data"].(map[string]any)["message"], "cooldown elapsed")

	// Rate limit: 120 / minute / user.
	limited := false
	for range 125 {
		if st, _ = call(fiber.MethodPost, ev, at, map[string]any{"rule_id": product, "event": "click"}); st == 429 {
			limited = true
			break
		}
	}
	assert.True(t, limited, "the events endpoint is rate limited")
}
