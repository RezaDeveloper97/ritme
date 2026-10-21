package main

import (
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// /metrics and /debug/pprof are never served by the public app (T-M9-07): only the private listener has them.
func TestApp_NoMetricsOnThePublicApp(t *testing.T) {
	app := newApp(apihttp.NewRegistry(), testDeps())
	for _, path := range []string{"/metrics", "/debug/pprof/", "/debug/pprof/heap"} {
		resp := do(t, app, httptest.NewRequest(nethttp.MethodGet, path, nil))
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode, path)
	}
}

func TestStartMetrics_PrivateListenerServesMetricsAndPprof(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	addr, stop, err := startMetrics(t.Context(), logger, "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(stop)

	for path, want := range map[string]string{
		"/metrics":      "http_request_duration_seconds",
		"/debug/pprof/": "goroutine",
	} {
		resp, err := nethttp.Get("http://" + addr + path) //nolint:noctx // local test listener
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		assert.Equal(t, nethttp.StatusOK, resp.StatusCode, path)
		assert.Contains(t, string(body), want, path)
	}

	off, stopOff, err := startMetrics(t.Context(), logger, "")
	require.NoError(t, err)
	stopOff()
	assert.Empty(t, off, `METRICS_ADDR="" turns the listener off`)
}

// accessLine returns the "request" access-log record of logs.
func accessLine(t *testing.T, logs string) map[string]any {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		if rec["msg"] == "request" {
			return rec
		}
	}
	t.Fatalf("no access line in %q", logs)
	return nil
}

// X-Request-ID is echoed when valid, generated otherwise, and is on the access-log line together with the
// request's query count; a log call inside the handler carries it too (httpx.LogHandler).
func TestApp_RequestIDEchoedAndLogged(t *testing.T) {
	var logs strings.Builder
	deps := testDeps()
	deps.Logger = slog.New(httpx.LogHandler(slog.NewJSONHandler(&logs, nil)))
	reg := apihttp.NewRegistry()
	reg.Register("probe", func(r fiber.Router, d *apihttp.Deps) {
		r.Get("/api/v1/probe", func(c fiber.Ctx) error {
			metrics.CountQuery(c.Context()) // what the DB driver wrapper does per query
			metrics.CountQuery(c.Context())
			d.Logger.InfoContext(c.Context(), "inside handler")
			return c.SendString("ok")
		})
	})
	app := newApp(reg, deps)

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/probe", nil)
	req.Header.Set(httpx.HeaderRequestID, "client-abc-123")
	resp := do(t, app, req)
	assert.Equal(t, "client-abc-123", resp.Header.Get(httpx.HeaderRequestID))
	rec := accessLine(t, logs.String())
	assert.Equal(t, "client-abc-123", rec["request_id"])
	assert.InDelta(t, 2, rec["queries"], 0)
	assert.Contains(t, logs.String(), `"msg":"inside handler","request_id":"client-abc-123"`)
	assert.Equal(t, 1, strings.Count(logs.String(), `"msg":"request"`))
	assert.Equal(t, 2, strings.Count(logs.String(), `"request_id"`), "access line not duplicated by LogHandler")

	logs.Reset()
	req = httptest.NewRequest(nethttp.MethodGet, "/api/v1/probe", nil)
	req.Header.Set(httpx.HeaderRequestID, "not valid!")
	resp = do(t, app, req)
	got := resp.Header.Get(httpx.HeaderRequestID)
	assert.NotEqual(t, "not valid!", got)
	assert.True(t, httpx.ValidRequestID(got))
	assert.Equal(t, got, accessLine(t, logs.String())["request_id"])
}

// A request above 500 ms logs a "slow request" warning with its route template, status and query count.
func TestApp_SlowRequestWarns(t *testing.T) {
	var logs strings.Builder
	deps := testDeps()
	deps.Logger = slog.New(slog.NewJSONHandler(&logs, nil))
	reg := apihttp.NewRegistry()
	reg.Register("slow", func(r fiber.Router, _ *apihttp.Deps) {
		r.Get("/api/v1/slow/:id", func(c fiber.Ctx) error {
			metrics.CountQuery(c.Context())
			time.Sleep(slowRequest + 20*time.Millisecond)
			return c.SendString("ok")
		})
	})
	resp := do(t, newApp(reg, deps), httptest.NewRequest(nethttp.MethodGet, "/api/v1/slow/7", nil))
	require.Equal(t, fiber.StatusOK, resp.StatusCode)
	out := logs.String()
	assert.Contains(t, out, `"level":"WARN","msg":"slow request"`)
	assert.Contains(t, out, `"route":"/api/v1/slow/:id","status":200,"queries":1`)
}
