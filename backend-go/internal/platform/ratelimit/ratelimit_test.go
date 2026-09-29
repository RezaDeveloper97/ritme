package ratelimit_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

func newLimiter(t *testing.T) (*ratelimit.Limiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return ratelimit.New(cache.NewFromClient(rdb, "ritme-go:"), clock.Real{}), mr
}

func TestAttempt_FixedWindow(t *testing.T) {
	l, mr := newLimiter(t)
	ctx := context.Background()
	t0 := time.Unix(1_790_000_000, 0)

	for i := 1; i <= 5; i++ {
		res, err := l.Attempt(ctx, "k", 5, time.Minute, t0.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
		assert.True(t, res.Allowed, "hit %d", i)
		assert.Equal(t, i, res.Hits)
		assert.Equal(t, 5-i, res.Remaining)
		assert.Equal(t, t0.Unix()+1+60, res.ResetAt, "timer set on the first hit only")
	}
	// 6th: rejected, not counted; retry after = window end - now.
	now := t0.Add(20 * time.Second)
	res, err := l.Attempt(ctx, "k", 5, time.Minute, now)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, 5, res.Hits)
	assert.Equal(t, 0, res.Remaining)
	assert.Equal(t, 41, res.RetryAfter(now))

	// Keys live under the Go prefix and expire with the window.
	assert.True(t, mr.Exists("ritme-go:throttle:k"))
	assert.True(t, mr.Exists("ritme-go:throttle:k:timer"))
	assert.Equal(t, time.Minute, mr.TTL("ritme-go:throttle:k:timer"))

	// Window over → a fresh window starts.
	mr.FastForward(61 * time.Second)
	res, err = l.Attempt(ctx, "k", 5, time.Minute, t0.Add(62*time.Second))
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, 1, res.Hits)
	assert.Equal(t, t0.Unix()+62+60, res.ResetAt)
}

func TestAttempt_ExpiredTimerResetsCounter(t *testing.T) {
	// tooManyAttempts(): attempts >= max but the timer is gone → resetAttempts, then hit.
	l, mr := newLimiter(t)
	ctx := context.Background()
	require.NoError(t, mr.Set("ritme-go:throttle:k", "9"))
	res, err := l.Attempt(ctx, "k", 5, time.Minute, time.Unix(100, 0))
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, 1, res.Hits)
}

func TestAttempt_SharedKeyDifferentLimits(t *testing.T) {
	// Laravel's unnamed throttles share one counter per IP: 5 hits exhaust a 5/min route
	// while a 10/min route still passes.
	l, _ := newLimiter(t)
	ctx := context.Background()
	now := time.Unix(1_790_000_000, 0)
	for range 5 {
		_, err := l.Attempt(ctx, "ip", 5, time.Minute, now)
		require.NoError(t, err)
	}
	r5, _ := l.Attempt(ctx, "ip", 5, time.Minute, now)
	r10, _ := l.Attempt(ctx, "ip", 10, time.Minute, now)
	assert.False(t, r5.Allowed)
	assert.True(t, r10.Allowed)
	assert.Equal(t, 6, r10.Hits)
	assert.Equal(t, 4, r10.Remaining)
}

func newApp(l *ratelimit.Limiter, handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Post("/x", l.Middleware(3, time.Minute, nil), handler)
	return app
}

func TestMiddleware_HeadersAnd429(t *testing.T) {
	l, _ := newLimiter(t)
	app := newApp(l, func(c fiber.Ctx) error { return c.SendString("ok") })
	send := func() (int, map[string]string, string) {
		r := httptest.NewRequest(fiber.MethodPost, "/x", nil)
		r.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
		resp, err := app.Test(r)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		h := map[string]string{}
		for _, k := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After", "X-RateLimit-Reset"} {
			h[k] = resp.Header.Get(k)
		}
		return resp.StatusCode, h, string(body)
	}
	for i, want := range []string{"2", "1", "0"} {
		status, h, _ := send()
		assert.Equal(t, 200, status, "request %d", i)
		assert.Equal(t, "3", h["X-RateLimit-Limit"])
		assert.Equal(t, want, h["X-RateLimit-Remaining"])
		assert.Empty(t, h["Retry-After"])
	}
	status, h, body := send()
	assert.Equal(t, 429, status)
	assert.Equal(t, "{\n    \"message\": \"Too Many Attempts.\"\n}", body)
	assert.Equal(t, "3", h["X-RateLimit-Limit"])
	assert.Equal(t, "0", h["X-RateLimit-Remaining"])
	assert.Equal(t, "60", h["Retry-After"]) // frozen request clock: the whole window is left
	// 2026-09-23T10:00:00+03:30 = 1790145000; reset = now + retry-after.
	assert.Equal(t, "1790145060", h["X-RateLimit-Reset"])
}

func TestMiddleware_HeadersOnHandlerError(t *testing.T) {
	l, _ := newLimiter(t)
	app := newApp(l, func(fiber.Ctx) error { return httpx.Fail(422, "Validation failed") })
	resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/x", nil))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 422, resp.StatusCode)
	assert.Equal(t, "3", resp.Header.Get("X-RateLimit-Limit"))
	assert.Equal(t, "2", resp.Header.Get("X-RateLimit-Remaining"))
}

func TestSignature(t *testing.T) {
	app := fiber.New()
	var guest, user string
	app.Get("/", func(c fiber.Ctx) error {
		guest = ratelimit.Signature(c, func(fiber.Ctx) string { return "" })
		user = ratelimit.Signature(c, func(fiber.Ctx) string { return "1004" })
		return nil
	})
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/", nil))
	require.NoError(t, err)
	_ = resp.Body.Close()
	// sha1("1004"); the guest key is sha1("|" + client IP).
	assert.Equal(t, "70b8dcb93382715a55ce5f2a8356ef5636a2d2da", user)
	assert.Len(t, guest, 40)
	assert.NotEqual(t, guest, user)
}

func TestNamed_OwnCounter(t *testing.T) {
	// A named limiter never shares the unnamed per-user counter (auth throttles).
	l, mr := newLimiter(t)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	user := func(fiber.Ctx) string { return "7" }
	app.Post("/w", l.Named("writes", 2, time.Minute, user), func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Post("/u", l.Middleware(2, time.Minute, user), func(c fiber.Ctx) error { return c.SendString("ok") })
	status := func(path string) int {
		resp, err := app.Test(httptest.NewRequest(fiber.MethodPost, path, nil))
		require.NoError(t, err)
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	assert.Equal(t, 200, status("/w"))
	assert.Equal(t, 200, status("/w"))
	assert.Equal(t, 429, status("/w"))
	// The unnamed counter for the same user is untouched.
	assert.Equal(t, 200, status("/u"))
	// sha1("7") under the name.
	assert.True(t, mr.Exists("ritme-go:throttle:writes:902ba3cda1883801594b6e1b452790cc53948fda"))
	assert.Panics(t, func() { l.Named("", 1, time.Minute, user) })
}

func TestNamedWith_CustomRejection(t *testing.T) {
	// A Go-only route may answer its own 429; the counter, headers and window are unchanged.
	l, _ := newLimiter(t)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(nil)})
	app.Use(clock.Middleware(clock.Real{}, true))
	var got ratelimit.Rejection
	reject := func(_ fiber.Ctx, r ratelimit.Rejection) error {
		got = r
		return httpx.Fail(429, "wait", "error_code", "too_many_requests").WithHeader("Retry-After", r.Headers()["Retry-After"])
	}
	app.Post("/w", l.NamedWith("writes", 1, time.Minute, func(fiber.Ctx) string { return "7" }, reject),
		func(c fiber.Ctx) error { return c.SendString("ok") })
	send := func() (int, string, string) {
		r := httptest.NewRequest(fiber.MethodPost, "/w", nil)
		r.Header.Set(clock.Header, "2026-09-23T10:00:00+03:30")
		resp, err := app.Test(r)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Retry-After"), string(body)
	}
	status, _, _ := send()
	assert.Equal(t, 200, status)
	status, retry, body := send()
	assert.Equal(t, 429, status)
	assert.Equal(t, "60", retry)
	assert.JSONEq(t, `{"success":false,"message":"wait","error_code":"too_many_requests"}`, body)
	assert.Equal(t, ratelimit.Rejection{RetryAfter: 60, Limit: 1, Reset: 1790145060}, got)
	assert.Equal(t, map[string]string{
		"Retry-After": "60", "X-RateLimit-Limit": "1", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1790145060",
	}, got.Headers())
}
