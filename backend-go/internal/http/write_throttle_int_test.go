package http

import (
	"context"
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
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// Drops this package's test database afterwards (integration tests only).
func TestMain(m *testing.M) { testdb.Main(m) }

// TestWriteThrottle_PerUser mounts the real registry (routes_care/checkups/fertility.go) on a
// test database and an in-memory Redis: one user's writes across those domains share one
// WriteThrottleMax/min budget (security audit M3-M7 #5), reads are not counted, and another
// user is unaffected.
func TestWriteThrottle_PerUser(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES ('0199c0de-0000-7000-8000-00000c0ffee9', 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`)
	require.NoError(t, err)
	iss := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)
	user := func(mobile string) string {
		res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		tok, err := iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test ids
		require.NoError(t, err)
		return tok.AccessToken
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
		Cache:  cache.NewFromClient(rdb, "ritme-go-t19:"),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})

	send := func(method, path, token string) *nethttp.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Language", "en")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
		resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
		require.NoError(t, err)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return resp
	}

	alice, bob := user("09120001901"), user("09120001902")

	// Writes to three domains draw on one budget: 422/404 answers count too (the handler ran).
	writes := []struct{ method, path string }{
		{fiber.MethodPost, "/api/v1/care/medications"},
		{fiber.MethodPut, "/api/v1/checkups/999999/settings"},
		{fiber.MethodPut, "/api/v1/fertility/days/2026-09-23"},
	}
	for i := range WriteThrottleMax {
		w := writes[i%len(writes)]
		resp := send(w.method, w.path, alice)
		require.NotEqual(t, fiber.StatusTooManyRequests, resp.StatusCode, "write %d", i+1)
		assert.Equal(t, "60", resp.Header.Get("X-RateLimit-Limit"))
	}
	resp := send(fiber.MethodPost, "/api/v1/checkups/custom", alice)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Retry-After"))

	// Reads are not throttled; another user has her own budget.
	assert.Equal(t, fiber.StatusOK, send(fiber.MethodGet, "/api/v1/care/medications", alice).StatusCode)
	assert.NotEqual(t, fiber.StatusTooManyRequests, send(fiber.MethodPost, "/api/v1/care/medications", bob).StatusCode)

	// The auth throttles' unnamed counter is untouched: refresh-session still answers.
	assert.NotEqual(t, fiber.StatusTooManyRequests, send(fiber.MethodPost, "/api/v1/auth/refresh-session", alice).StatusCode)
}
