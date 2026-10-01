package http

import (
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

// TestMessagesNudgesRoute mounts the real registry (routes_messages.go, CB-COND-06b / D-41): GET
// /api/v1/messages/nudges answers 401 without a token, the nudges for a qualifying user and `nudges: []` for
// another user (her log is not read). The rules themselves are covered by internal/messages/conditionnudges.
func TestMessagesNudgesRoute(t *testing.T) {
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
		VALUES ('0199c0de-0000-7000-8000-00000c0ffeea', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	iss := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)
	user := func(mobile string) (uint64, string) {
		res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		tok, err := iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
		require.NoError(t, err)
		return uint64(id), tok.AccessToken //nolint:gosec // test ids
	}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	Mount(app, &Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB:     db,
		Cache:  cache.NewFromClient(rdb, "ritme-go-nudges:"),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})

	get := func(token, lang string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(fiber.MethodGet, "/api/v1/messages/nudges", nethttp.NoBody)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Language", lang)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		var body map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		return resp.StatusCode, body
	}
	keysOf := func(body map[string]any) []string {
		out := []string{}
		for _, n := range body["data"].(map[string]any)["nudges"].([]any) {
			out = append(out, n.(map[string]any)["key"].(string))
		}
		return out
	}

	status, body := get("", "en")
	assert.Equal(t, fiber.StatusUnauthorized, status)
	assert.Equal(t, "unauthenticated", body["error_code"])

	alice, aliceTok := user("09120002001")
	bob, bobTok := user("09120002002")
	for _, uid := range []uint64{alice, bob} {
		exec(`INSERT INTO cycle_histories (user_id, period_start_date, bleeding_length, is_confirmed, created_at, updated_at)
			VALUES (?, '2026-09-14', 5, 1, '2026-09-14 09:00:00', '2026-09-14 09:00:00')`, uid)
	}
	for _, day := range []string{"2026-09-15", "2026-09-16"} {
		exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, source, created_at, updated_at)
			VALUES (?, ?, 'pain', 'location', 'abdomen', 'severe', '8', 'manual', NOW(), NOW())`, alice, day)
		exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
			VALUES (?, ?, 'bleeding', 'flow', '', 'heavy', 'manual', NOW(), NOW())`, alice, day)
	}

	status, body = get(aliceTok, "en")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, "2026-09-14", body["data"].(map[string]any)["from"])
	assert.Equal(t, []string{"heavy_pain", "heavy_bleeding"}, keysOf(body))

	status, body = get(bobTok, "en")
	require.Equal(t, fiber.StatusOK, status, body)
	assert.Equal(t, []any{}, body["data"].(map[string]any)["nudges"], "another user's log raises nothing")

	// Localized: the Persian copy for a Persian request.
	_, body = get(aliceTok, "fa")
	first := body["data"].(map[string]any)["nudges"].([]any)[0].(map[string]any)
	assert.Contains(t, first["title"], "درد")
	assert.Contains(t, first["body"], "۲ روز", "{days} in Persian digits for fa")
}
