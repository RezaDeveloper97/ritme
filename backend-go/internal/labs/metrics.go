package labs

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/platform/metrics"
)

// aiFiltered counts AI lab summaries discarded for the rules summary (T-M10-20): labs_ai_filtered_total{reason},
// reason = invalid | category | diagnostic | dose (guard.go). Served by the private metrics listener only.
var aiFiltered = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "labs_ai_filtered_total",
	Help: "AI lab summaries replaced by the rules summary, by filter reason.",
}, []string{"reason"})

func init() { metrics.Registry().MustRegister(aiFiltered) }

func countFiltered(reason string) { aiFiltered.WithLabelValues(reason).Inc() }
