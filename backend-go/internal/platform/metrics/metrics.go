// Package metrics is the process-wide Prometheus registry of the Go API (T-M9-07, docs/engine-v6/README.md §6).
//
// The registry is served only by the private listener (Server: METRICS_ADDR, default 127.0.0.1:9090), never by the
// public Fiber app. Metric names are the README §6 names, verbatim:
//
//	http_request_duration_seconds{route,method,status}  route = Fiber route template, never the raw path
//	db_query_duration_seconds{query}                     sqlc query name (`-- name:`), "raw" otherwise
//	db_queries_per_request{route}                        DB calls one request made (QueryCounter)
//	cache_requests_total{cache,result}                   result = hit | miss | error
//
// plus the Go runtime and process collectors. The registry is a package-level singleton on purpose: the DB driver
// wrapper, the cache client and the HTTP middleware are built in different places (testdb opens pools too) and
// Prometheus collectors are global by nature; tests read it through Gather.
package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Cache results for cache_requests_total.
const (
	CacheHit   = "hit"
	CacheMiss  = "miss"
	CacheError = "error"
)

// durationBuckets span 1 ms … 5 s (HTTP and DB latencies).
var durationBuckets = []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5}

// queryCountBuckets fit the README §8 budgets (warm 1, cold ≤ 5, /home ≤ 8) and catch N+1 loops.
var queryCountBuckets = []float64{0, 1, 2, 3, 4, 5, 6, 8, 10, 15, 20, 30, 50, 100}

var (
	registry = prometheus.NewRegistry()

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency by Fiber route template, method and status.",
		Buckets: durationBuckets,
	}, []string{"route", "method", "status"})

	dbQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "MariaDB query latency by sqlc query name (raw for hand-written SQL).",
		Buckets: durationBuckets,
	}, []string{"query"})

	dbQueriesPerRequest = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_queries_per_request",
		Help:    "Number of DB queries one HTTP request made, by route template.",
		Buckets: queryCountBuckets,
	}, []string{"route"})

	cacheRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_requests_total",
		Help: "Cache lookups by cache name and result (hit, miss, error).",
	}, []string{"cache", "result"})
)

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpDuration, dbQueryDuration, dbQueriesPerRequest, cacheRequests,
	)
}

// Registry is the process registry. Later tasks register their own collectors on it (engine_*, asynq gauges).
func Registry() *prometheus.Registry { return registry }

// ObserveHTTP records one request.
func ObserveHTTP(route, method, status string, d time.Duration) {
	httpDuration.WithLabelValues(route, method, status).Observe(d.Seconds())
}

// ObserveQuery records one DB query by its name.
func ObserveQuery(name string, d time.Duration) {
	dbQueryDuration.WithLabelValues(name).Observe(d.Seconds())
}

// ObserveQueriesPerRequest records how many queries one request on route made.
func ObserveQueriesPerRequest(route string, n int64) {
	dbQueriesPerRequest.WithLabelValues(route).Observe(float64(n))
}

// IncCache counts one cache lookup; result is CacheHit, CacheMiss or CacheError.
func IncCache(cache, result string) {
	cacheRequests.WithLabelValues(cache, result).Inc()
}

// QueryCounter counts the DB queries of one request. It is safe for concurrent use (a handler may fan out).
type QueryCounter struct{ n atomic.Int64 }

// Load returns the count so far.
func (q *QueryCounter) Load() int64 {
	if q == nil {
		return 0
	}
	return q.n.Load()
}

type queryCounterKey struct{}

// QueryCounterKey is the context / fiber.Locals key of the request's *QueryCounter. The HTTP middleware stores the
// counter under it in both, so a handler that passes c.Context() and one that passes the fiber.Ctx itself (whose
// Value reads Locals) are both counted.
var QueryCounterKey any = queryCounterKey{}

// WithQueryCounter returns ctx carrying a fresh counter.
func WithQueryCounter(ctx context.Context) (context.Context, *QueryCounter) {
	q := &QueryCounter{}
	return context.WithValue(ctx, QueryCounterKey, q), q
}

// QueryCounterFrom returns the request's counter, or nil outside a request (jobs, migrations).
func QueryCounterFrom(ctx context.Context) *QueryCounter {
	if ctx == nil {
		return nil
	}
	q, _ := ctx.Value(QueryCounterKey).(*QueryCounter)
	return q
}

// CountQuery adds one query to the request's counter, if any.
func CountQuery(ctx context.Context) {
	if q := QueryCounterFrom(ctx); q != nil {
		q.n.Add(1)
	}
}
