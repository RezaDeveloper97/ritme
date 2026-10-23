package state

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Engine metrics (README §6); the L1/L2 lookups are cache_requests_total{cache="engine_l1|engine_l2"}.
var (
	buildSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "engine_build_seconds",
		Help:    "Time to prepare one engine state on a cache miss, by input source (db, l2, shared).",
		Buckets: []float64{.00001, .000025, .00005, .0001, .00025, .0005, .001, .0025, .005, .01, .05},
	}, []string{"source"})

	verifyMismatch = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "engine_cache_verify_mismatch_total",
		Help: "Cache hits whose state differed from a rebuild from the database (ENGINE_CACHE_VERIFY).",
	})
)

func init() {
	metrics.Registry().MustRegister(buildSeconds, verifyMismatch)
}
