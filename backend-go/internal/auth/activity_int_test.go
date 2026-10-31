package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/activity"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// T-M10-30: RequireUser marks the session day through the activity writer; the request path stays at one query (the
// auth JOIN, T-M9-09) on the first request of the day and on every later one, and a log write marks logged = 1.
func TestRequireUser_MarksActivityDays(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234598", t0)
	var uid uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM users WHERE mobile = '09121234598'").Scan(&uid))

	// An instrumented pool so the per-request counter sees every query of the guard.
	counted, err := db.OpenDSN(t.Context(), testdb.DSN(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = counted.Close() })
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	m := activity.Install(app, e.db, cache.NewFromClient(rdb, "ritme-go:"), quiet)
	require.NotNil(t, m)
	t.Cleanup(m.Close)
	guard := auth.NewGuardWith(&testKey.PublicKey, store.New(counted), clock.Real{}, quiet)
	app.Use(metrics.HTTP())
	app.Use(clock.Middleware(clock.Real{}, true))
	read := func(c fiber.Ctx) error {
		c.Set("X-Queries", itoa(metrics.QueryCounterFrom(c.Context()).Load()))
		return c.SendStatus(204)
	}
	app.Get("/r", guard.RequireUser, read)
	app.Post("/w", guard.RequireUser, func(c fiber.Ctx) error {
		activity.MarkLogged(c.Context(), uid) // what revision.Bump does after its UPDATE
		return read(c)
	})
	call := func(method, path, now, zone string) string {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set(clock.Header, now)
		if zone != "" {
			req.Header.Set(activity.TimezoneHeader, zone)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		require.Equal(t, 204, resp.StatusCode)
		return resp.Header.Get("X-Queries")
	}
	days := func() []string {
		m.Flush()
		rows, err := e.db.Query("SELECT CONCAT(DATE_FORMAT(`day`, '%Y-%m-%d'), ':', `session`, `logged`) FROM user_activity_days WHERE user_id = ? ORDER BY `day`", uid)
		require.NoError(t, err)
		defer func() { _ = rows.Close() }()
		var out []string
		for rows.Next() {
			var s string
			require.NoError(t, rows.Scan(&s))
			out = append(out, s)
		}
		require.NoError(t, rows.Err())
		return out
	}

	// Warm the guard's OAuth client cache (its first request also lists the clients) on an earlier day.
	call(http.MethodGet, "/r", "2026-10-19T09:00:00+03:30", "")
	m.Flush()
	_, err = e.db.Exec("DELETE FROM user_activity_days")
	require.NoError(t, err)

	// Two requests the same local day → one row, session = 1; each request is the auth query only.
	assert.Equal(t, "1", call(http.MethodGet, "/r", "2026-10-20T09:00:00+03:30", ""))
	assert.Equal(t, "1", call(http.MethodGet, "/r", "2026-10-20T21:00:00+03:30", ""))
	assert.Equal(t, []string{"2026-10-20:10"}, days())

	// A log write → logged = 1 on the same row.
	assert.Equal(t, "1", call(http.MethodPost, "/w", "2026-10-20T22:00:00+03:30", ""))
	assert.Equal(t, []string{"2026-10-20:11"}, days())

	// A Berlin user's day boundary: 23:30 Berlin on 10-21 is already 10-22 in Tehran; 00:10 Berlin is the next day.
	call(http.MethodGet, "/r", "2026-10-21T23:30:00+02:00", "Europe/Berlin")
	assert.Equal(t, []string{"2026-10-20:11", "2026-10-21:10"}, days())
	call(http.MethodGet, "/r", "2026-10-22T00:10:00+02:00", "Europe/Berlin")
	assert.Equal(t, []string{"2026-10-20:11", "2026-10-21:10", "2026-10-22:10"}, days())

	// The stored zone is used without the header (read by the writer, not on the request).
	_, err = e.db.Exec("INSERT INTO user_profiles (user_id, timezone, created_at, updated_at) VALUES (?, 'Europe/Berlin', NOW(), NOW())", uid)
	require.NoError(t, err)
	assert.Equal(t, "1", call(http.MethodGet, "/r", "2026-10-23T23:30:00+02:00", ""))
	assert.Contains(t, days(), "2026-10-23:10")

	// Redis down: the writer still upserts (idempotent).
	mr.Close()
	call(http.MethodGet, "/r", "2026-10-25T10:00:00+03:30", "")
	assert.Contains(t, days(), "2026-10-25:10")
}

// Without an installed marker the guard behaves exactly as before.
func TestRequireUser_NoMarkerNoActivity(t *testing.T) {
	e := setup(t, clock.Real{})
	tok := e.login(t, "09121234597", t0)
	r := e.do(t, http.MethodGet, "/api/v1/auth/user", tok, "", "")
	require.Equal(t, 200, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow("SELECT COUNT(*) FROM user_activity_days").Scan(&n))
	assert.Zero(t, n)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
