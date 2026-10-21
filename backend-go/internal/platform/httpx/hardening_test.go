package httpx_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andybalholm/brotli"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// largeJSON is a /cycle/month-like body: repetitive Persian JSON, ~60 KB.
func largeJSON() any {
	days := make([]any, 0, 300)
	for i := range 300 {
		days = append(days, jsonx.Obj("day", i, "phase", "فولیکولار", "note", "روز چرخه، پیش\u200cبینی بر اساس میانه سه دوره", "ok", true))
	}
	return jsonx.Obj("success", true, "data", jsonx.Obj("days", days))
}

func compressApp() *fiber.App {
	app := fiber.New()
	app.Use(httpx.Compress())
	app.Get("/large", func(c fiber.Ctx) error { return httpx.OK(c, largeJSON()) })
	app.Get("/small", func(c fiber.Ctx) error { return httpx.OK(c, jsonx.Obj("a", 1)) })
	app.Get("/text", func(c fiber.Ctx) error { return c.SendString(strings.Repeat("x", 4096)) })
	app.Get("/stream", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/event-stream; charset=utf-8")
		return c.SendStream(strings.NewReader(strings.Repeat("data: x\n\n", 1000)))
	})
	app.Get("/no-transform", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "no-transform")
		return httpx.OK(c, largeJSON())
	})
	return app
}

func get(t *testing.T, app *fiber.App, path, enc string) (int, map[string]string, []byte) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	if enc != "" {
		req.Header.Set(fiber.HeaderAcceptEncoding, enc)
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	h := map[string]string{}
	for k := range resp.Header {
		h[k] = resp.Header.Get(k)
	}
	return resp.StatusCode, h, body
}

// T-M9-10 / D-132: large JSON is gzip/brotli-compressed to < 30 %; small, non-JSON and streamed bodies are not.
func TestCompress(t *testing.T) {
	app := compressApp()

	_, plainH, plain := get(t, app, "/large", "")
	require.Greater(t, len(plain), 30_000)
	assert.Empty(t, plainH["Content-Encoding"])

	_, h, gz := get(t, app, "/large", "gzip, deflate")
	assert.Equal(t, "gzip", h["Content-Encoding"])
	assert.Contains(t, h["Vary"], "Accept-Encoding")
	assert.Less(t, float64(len(gz)), 0.30*float64(len(plain)), "gzip %d of %d", len(gz), len(plain))
	zr, err := gzip.NewReader(bytes.NewReader(gz))
	require.NoError(t, err)
	unzipped, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Equal(t, plain, unzipped)

	_, h, br := get(t, app, "/large", "gzip, br")
	assert.Equal(t, "br", h["Content-Encoding"])
	assert.Less(t, float64(len(br)), 0.30*float64(len(plain)))
	unbr, err := io.ReadAll(brotli.NewReader(bytes.NewReader(br)))
	require.NoError(t, err)
	assert.Equal(t, plain, unbr)

	for _, path := range []string{"/small", "/text", "/stream", "/no-transform"} {
		_, h, _ := get(t, app, path, "gzip, br")
		assert.Empty(t, h["Content-Encoding"], path)
	}

	_, h, _ = get(t, app, "/large", "identity")
	assert.Empty(t, h["Content-Encoding"])
}

// T-M9-10: the request context carries the deadline unless the route is allow-listed.
func TestRequestDeadline(t *testing.T) {
	app := fiber.New()
	app.Use(httpx.RequestDeadline(5*time.Second, func(c fiber.Ctx) bool { return c.Path() == "/long" }))
	remaining := func(c fiber.Ctx) error {
		dl, ok := c.Context().Deadline()
		if !ok {
			return c.SendString("none")
		}
		return c.SendString(time.Until(dl).Round(time.Second).String())
	}
	app.Get("/short", remaining)
	app.Get("/long", remaining)
	// A handler passing the fiber.Ctx itself as the context: the bounded context is still reachable.
	app.Get("/via-ctx", func(c fiber.Ctx) error {
		bounded, ok := httpx.DeadlineFrom(c)
		if !ok {
			return c.SendString("none")
		}
		dl, _ := bounded.Deadline()
		return c.SendString(time.Until(dl).Round(time.Second).String())
	})

	_, _, body := get(t, app, "/short", "")
	assert.Equal(t, "5s", string(body))
	_, _, body = get(t, app, "/long", "")
	assert.Equal(t, "none", string(body))
	_, _, body = get(t, app, "/via-ctx", "")
	assert.Equal(t, "5s", string(body))
	_, ok := httpx.DeadlineFrom(context.Background())
	assert.False(t, ok)

	off := fiber.New()
	off.Use(httpx.RequestDeadline(0, nil))
	off.Get("/short", remaining)
	_, _, body = get(t, off, "/short", "")
	assert.Equal(t, "none", string(body))
}

// T-M9-10b: DriverContext bounds the fiber.Ctx itself and leaves every other context alone.
func TestDriverContext(t *testing.T) {
	app := fiber.New()
	app.Use(httpx.RequestDeadline(5*time.Second, func(c fiber.Ctx) bool { return c.Path() == "/long" }))
	report := func(c fiber.Ctx) error {
		has := func(ctx context.Context) string {
			if _, ok := httpx.DriverContext(ctx).Deadline(); ok {
				return "1"
			}
			return "0"
		}
		type key struct{}
		return c.SendString(has(c) + has(context.WithoutCancel(c)) + has(context.WithValue(c, key{}, 1)) + has(c.Context()))
	}
	app.Get("/short", report)
	app.Get("/long", report)

	// fiber.Ctx → bounded; WithoutCancel(c) / WithValue(c) → untouched; c.Context() already has the deadline.
	_, _, body := get(t, app, "/short", "")
	assert.Equal(t, "1001", string(body))
	// Long-running routes: nothing is bounded.
	_, _, body = get(t, app, "/long", "")
	assert.Equal(t, "0000", string(body))

	bg := context.Background()
	assert.Equal(t, bg, httpx.DriverContext(bg))
}

// pingDriver is a database/sql driver whose connections only answer Ping (with err).
type pingDriver struct{ err error }

func (d pingDriver) Open(string) (driver.Conn, error) { return pingConn(d), nil }

type pingConn struct{ err error }

func (c pingConn) Ping(context.Context) error        { return c.err }
func (pingConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (pingConn) Close() error                        { return nil }
func (pingConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }

var registerPingDrivers sync.Once

func pingDB(t *testing.T, up bool) *sql.DB {
	t.Helper()
	registerPingDrivers.Do(func() {
		sql.Register("ready-up", pingDriver{})
		sql.Register("ready-down", pingDriver{err: errors.New("db down")})
	})
	name := "ready-up"
	if !up {
		name = "ready-down"
	}
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func redisPing(t *testing.T, up bool) func(context.Context) error {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = rdb.Close() })
	if !up {
		mr.Close()
	}
	return func(ctx context.Context) error { return rdb.Ping(ctx).Err() }
}

// T-M9-10: /ready → 200 when MariaDB and Redis answer, 503 naming the failed check otherwise.
func TestReady(t *testing.T) {
	cases := []struct {
		name        string
		dbUp, redUp bool
		status      int
		body        string
	}{
		{"both up", true, true, fiber.StatusOK, `{"status":"ok"}`},
		{"redis down", true, false, fiber.StatusServiceUnavailable, `{"status":"unavailable","failed":["redis"]}`},
		{"db down", false, true, fiber.StatusServiceUnavailable, `{"status":"unavailable","failed":["db"]}`},
		{"both down", false, false, fiber.StatusServiceUnavailable, `{"status":"unavailable","failed":["db","redis"]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/ready", httpx.Ready(time.Second,
				httpx.ReadyCheck{Name: "db", Ping: pingDB(t, tc.dbUp).PingContext},
				httpx.ReadyCheck{Name: "redis", Ping: redisPing(t, tc.redUp)},
			))
			status, h, body := get(t, app, "/ready", "")
			assert.Equal(t, tc.status, status)
			assert.JSONEq(t, tc.body, string(body))
			assert.Equal(t, "no-store", h["Cache-Control"])
		})
	}

	app := fiber.New()
	app.Get("/ready", httpx.Ready(time.Second, httpx.ReadyCheck{Name: "db"}))
	status, _, body := get(t, app, "/ready", "")
	assert.Equal(t, fiber.StatusServiceUnavailable, status, "an unconfigured dependency is not ready")
	assert.JSONEq(t, `{"status":"unavailable","failed":["db"]}`, string(body))
}
