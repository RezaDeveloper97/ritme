package metrics

import (
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler serves /metrics (the registry) and /debug/pprof/*. Mount it only on the private listener.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}))
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

// NewServer is the private listener on addr (METRICS_ADDR). There is no write timeout: a CPU profile or a trace
// streams for its ?seconds= duration.
func NewServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
	}
}
