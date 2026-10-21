package metrics_test

import (
	"context"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/metrics"
)

// family returns the gathered metric family name, or nil.
func family(t *testing.T, name string) *dto.MetricFamily {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

func labels(m *dto.Metric) map[string]string {
	out := map[string]string{}
	for _, lp := range m.GetLabel() {
		out[lp.GetName()] = lp.GetValue()
	}
	return out
}

// histogram returns the sample count and sum of the series of name whose labels include want.
func histogram(t *testing.T, name string, want map[string]string) (count uint64, sum float64) {
	t.Helper()
	mf := family(t, name)
	if mf == nil {
		return 0, 0
	}
next:
	for _, m := range mf.GetMetric() {
		got := labels(m)
		for k, v := range want {
			if got[k] != v {
				continue next
			}
		}
		count += m.GetHistogram().GetSampleCount()
		sum += m.GetHistogram().GetSampleSum()
	}
	return count, sum
}

func routeValues(t *testing.T, name string) []string {
	t.Helper()
	var out []string
	if mf := family(t, name); mf != nil {
		for _, m := range mf.GetMetric() {
			out = append(out, labels(m)["route"])
		}
	}
	return out
}

func do(t *testing.T, app *fiber.App, method, path string) *nethttp.Response {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest(method, path, nil))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// The route label is the Fiber template, never the raw path (README §6), and the per-request query counter reaches
// the handler through both c.Context() and the fiber.Ctx itself.
func TestHTTP_RouteLabelIsTheTemplate(t *testing.T) {
	app := fiber.New()
	app.Use(metrics.HTTP())
	app.Get("/api/v1/cycle/date/:date", func(c fiber.Ctx) error {
		metrics.CountQuery(c.Context())
		metrics.CountQuery(c) // fiber.Ctx is a context.Context whose Value reads Locals
		return c.SendString("ok")
	})
	auth := app.Group("/api/v1/private", func(c fiber.Ctx) error { return fiber.ErrUnauthorized })
	auth.Get("/thing", func(c fiber.Ctx) error { return c.SendString("never") })

	require.Equal(t, fiber.StatusOK, do(t, app, fiber.MethodGet, "/api/v1/cycle/date/2026-10-01").StatusCode)
	require.Equal(t, fiber.StatusOK, do(t, app, fiber.MethodGet, "/api/v1/cycle/date/2026-10-02").StatusCode)
	require.Equal(t, fiber.StatusNotFound, do(t, app, fiber.MethodGet, "/random/probe/123").StatusCode)
	require.Equal(t, fiber.StatusUnauthorized, do(t, app, fiber.MethodGet, "/api/v1/private/thing").StatusCode)

	n, _ := histogram(t, "http_request_duration_seconds",
		map[string]string{"route": "/api/v1/cycle/date/:date", "method": "GET", "status": "200"})
	assert.Equal(t, uint64(2), n)
	n, sum := histogram(t, "db_queries_per_request", map[string]string{"route": "/api/v1/cycle/date/:date"})
	assert.Equal(t, uint64(2), n)
	assert.InDelta(t, 4, sum, 0.001, "two counted queries per request")

	n, _ = histogram(t, "http_request_duration_seconds", map[string]string{"route": metrics.RouteUnmatched, "status": "404"})
	assert.Equal(t, uint64(1), n)
	n, _ = histogram(t, "http_request_duration_seconds", map[string]string{"route": "/api/v1/private", "status": "401"})
	assert.Equal(t, uint64(1), n, "a group middleware's rejection is labelled with its prefix")

	for _, r := range routeValues(t, "http_request_duration_seconds") {
		assert.NotContains(t, r, "2026-10", "raw path leaked into the route label")
		assert.NotContains(t, r, "/random", "unmatched path leaked into the route label")
	}
}

func TestQueryCounter(t *testing.T) {
	metrics.CountQuery(context.Background()) // no counter: no-op, no panic
	assert.Nil(t, metrics.QueryCounterFrom(context.Background()))
	var nilCounter *metrics.QueryCounter
	assert.Zero(t, nilCounter.Load())

	ctx, q := metrics.WithQueryCounter(context.Background())
	metrics.CountQuery(ctx)
	metrics.CountQuery(ctx)
	assert.Equal(t, int64(2), q.Load())
	assert.Same(t, q, metrics.QueryCounterFrom(ctx))
}

func TestHandler_ServesMetricsAndPprof(t *testing.T) {
	metrics.IncCache("test_cache", metrics.CacheHit)
	metrics.ObserveQuery("GetSomething", 0)
	srv := httptest.NewServer(metrics.Handler())
	t.Cleanup(srv.Close)

	get := func(path string) (int, string) {
		resp, err := nethttp.Get(srv.URL + path) //nolint:noctx // test server
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	status, body := get("/metrics")
	require.Equal(t, nethttp.StatusOK, status)
	for _, name := range []string{
		"http_request_duration_seconds", "db_query_duration_seconds", "db_queries_per_request",
		`cache_requests_total{cache="test_cache",result="hit"} 1`, "go_goroutines",
	} {
		assert.Contains(t, body, name)
	}

	status, body = get("/debug/pprof/")
	require.Equal(t, nethttp.StatusOK, status)
	assert.Contains(t, body, "goroutine")
}
