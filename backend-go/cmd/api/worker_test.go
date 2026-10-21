package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/config"
)

func TestHealthTarget(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":80")
	t.Setenv("METRICS_ADDR", "127.0.0.1:9100")
	for _, tc := range []struct {
		args       []string
		addr, path string
	}{
		{nil, ":80", "/up"},
		{[]string{"ready"}, ":80", "/ready"},
		{[]string{"worker"}, "127.0.0.1:9100", "/up"},
		{[]string{"worker", "ready"}, "127.0.0.1:9100", "/ready"},
	} {
		addr, path := healthTarget(tc.args)
		assert.Equal(t, tc.addr, addr, tc.args)
		assert.Equal(t, tc.path, path, tc.args)
	}
}

func TestMetricsAddrFromEnv_DefaultWhenUnset(t *testing.T) {
	t.Setenv("METRICS_ADDR", "x") // registers the restore; then unset it for the test
	require.NoError(t, os.Unsetenv("METRICS_ADDR"))
	assert.Equal(t, config.DefaultMetricsAddr, metricsAddrFromEnv())
}

// The worker role: background hooks start without a listener (even under API_RUN_WORKERS=0, which the worker
// container inherits), /up and /ready answer on the private listener, and a cancelled context stops the hooks.
func TestWork_StartsHooksAndServesProbes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	reg := apihttp.NewRegistry()
	var starts, stops atomic.Int32
	reg.Register("poller", func(r fiber.Router, _ *apihttp.Deps) {
		auth.OnLifecycle(r, func() error { starts.Add(1); return nil }, func() { stops.Add(1) })
	})
	app := newApp(reg, testDeps())
	auth.SetRunWorkers(app, false)

	var started atomic.Bool
	addr, stopMetrics, err := serveMetrics(t.Context(), logger, "127.0.0.1:0", workerHandler(&started, nil, nil))
	require.NoError(t, err)
	t.Cleanup(stopMetrics)

	assert.Equal(t, nethttp.StatusServiceUnavailable, probe(t, addr, "/up").code, "not started yet")

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- work(ctx, logger, app, &started) }()
	require.Eventually(t, started.Load, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), starts.Load())

	assert.Equal(t, nethttp.StatusOK, probe(t, addr, "/up").code)
	ready := probe(t, addr, "/ready")
	assert.Equal(t, nethttp.StatusServiceUnavailable, ready.code, "no DB / Redis in this test")
	assert.ElementsMatch(t, []any{"db", "redis"}, ready.body["failed"])
	// The metrics stay on the same listener.
	assert.Equal(t, nethttp.StatusOK, probe(t, addr, "/metrics").code)

	cancel()
	require.NoError(t, <-done)
	assert.Equal(t, int32(1), stops.Load())
	assert.False(t, started.Load())
}

// GET /ready is 200 once the hooks run and both dependencies answer.
func TestWorkerHandler_ReadyWhenStartedAndDepsAnswer(t *testing.T) {
	var started atomic.Bool
	started.Store(true)
	h := workerHandler(&started, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/ready", nil))
	assert.Equal(t, nethttp.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, `{"status":"unavailable","failed":["db","redis"]}`, rec.Body.String())

	started.Store(false)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(nethttp.MethodGet, "/ready", nil))
	assert.JSONEq(t, `{"status":"unavailable","failed":["worker","db","redis"]}`, rec.Body.String())
}

type probeResult struct {
	code int
	body map[string]any
}

func probe(t *testing.T, addr, path string) probeResult {
	t.Helper()
	resp, err := nethttp.Get("http://" + addr + path) //nolint:noctx // local test listener
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	out := probeResult{code: resp.StatusCode}
	_ = json.Unmarshal(raw, &out.body)
	return out
}
