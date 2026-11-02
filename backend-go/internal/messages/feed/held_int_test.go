package feed_test

// T-M10-08c: an open preg.* row of a locked rule whose message_rules row is off (here: preg.bp_severe deactivated) is
// still fanned out as the feed's urgent modal, through the code-held rule (feed.InstallHeldRules, registered by the
// pregnancy package), over the real routes.

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const heldAt = "2026-10-21T10:00:00+03:30"

func TestFeedInt_UrgentModalForInactiveLockedRule(t *testing.T) {
	if !testdb.Enabled() {
		t.Skip("TEST_DB_DSN not set")
	}
	db := testdb.New(t)
	ctx := context.Background()
	exec := func(q string, args ...any) {
		t.Helper()
		_, err := db.Exec(q, args...)
		require.NoError(t, err)
	}
	keysPath, err := filepath.Abs("../../../contract/fixtures/keys")
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffe08', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`)
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mina', '09120008381', NOW(), NOW())`)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	uid := uint64(id) //nolint:gosec // test id
	exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 'lmp', '2026-05-01', 1, NOW(), NOW())`, uid)
	exec(`INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'pregnancy', NOW(), NOW())`, uid)
	tok, err := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365).Issue(ctx, uid, time.Now())
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App: config.App{Env: "testing", URL: "http://localhost"}, SMS: config.SMS{Provider: "log"},
			Passport: config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30}, StoragePath: keysPath,
			CycleEngine: config.CycleEngineV6,
			EngineCache: config.EngineCache{Enabled: true, L1Entries: 1000},
		},
		DB: db, Cache: cache.NewFromClient(rdb, "ritme-go-held:"), Logger: quiet,
	})
	t.Cleanup(func() { _ = app.ShutdownWithTimeout(5 * time.Second) })
	call := func(method, path, locale, body string) (int, map[string]any, string) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", locale)
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		req.Header.Set(clock.Header, heldAt)
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out, string(raw)
	}

	exec(`UPDATE message_rules SET is_active = 0 WHERE rule_key = 'preg.bp_severe'`)
	configcache.Changed(ctx, rules.Table)

	st, _, raw := call(nethttp.MethodPost, "/api/v1/pregnancy/weekly", "en",
		`{"pregnancy_week":25,"log_date":"2026-10-21","systolic_pressure":168,"diastolic_pressure":112}`)
	require.Equal(t, nethttp.StatusCreated, st, raw)
	var rowID, ruleID uint64
	var state string
	require.NoError(t, db.QueryRow(`SELECT id, rule_id, state FROM message_inbox WHERE user_id = ? AND rule_key = 'preg.bp_severe'`, uid).
		Scan(&rowID, &ruleID, &state))
	assert.Equal(t, "active", state, "the code-held rule raised the alert")

	for _, loc := range []string{"en", "fa"} {
		st, body, raw := call(nethttp.MethodGet, "/api/v1/messages/feed?slots=urgent_modal", loc, "")
		require.Equal(t, nethttp.StatusOK, st, raw)
		items := body["data"].(map[string]any)["slots"].(map[string]any)["urgent_modal"].([]any)
		require.Len(t, items, 1, "%s: %s", loc, raw)
		item := items[0].(map[string]any)
		assert.EqualValues(t, rowID, item["alert_id"], loc)
		assert.NotEmpty(t, item["title"], loc)
		for _, k := range []string{"title", "body"} {
			assert.NotContains(t, item[k], "{", "%s %s: rendered", loc, k)
		}
		assert.Equal(t, "safety", item["kind"], loc)
		assert.Equal(t, "preg.bp_severe", item["rule_key"], loc)
		if loc == "en" {
			assert.Equal(t, "Your blood pressure is in the severe range for pregnancy", item["title"])
			assert.Contains(t, item["body"], "115")
		} else {
			assert.Contains(t, item["body"], "۱۱۵", "the fa copy, not the fallback")
		}
	}
	require.NoError(t, db.QueryRow(`SELECT state FROM message_inbox WHERE id = ?`, rowID).Scan(&state))
	assert.Equal(t, "active", state, "the feed leaves the row open")
	assert.NotZero(t, ruleID)
}
