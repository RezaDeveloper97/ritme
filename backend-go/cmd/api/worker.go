package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	nethttp "net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// workerReadyTimeout bounds each /ready dependency check of the worker (the API's is the same 1 s).
const workerReadyTimeout = apihttp.ReadyTimeout

// runWorker is `api worker` (T-M9-12, docs/engine-v6/jobs.md): the background role of the same binary. It mounts the
// same routes as the API, so every domain registers its jobs, scheduler entries and pollers exactly as it does there,
// but it never listens for HTTP: it starts the background hooks (asynq server + scheduler, the labs, health-record
// extract and learning dispatch pollers) and serves only the private metrics listener (METRICS_ADDR), with /up and
// /ready added for `api healthcheck worker [ready]`. API_RUN_WORKERS does not apply here.
//
// It never runs migrations (the API owns them; compose starts the worker after a healthy API).
func runWorker(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, rdb, closeDeps, err := openDeps(ctx, logger, cfg, false)
	if err != nil {
		return err
	}
	defer closeDeps()

	app := newApp(apihttp.Default(), &apihttp.Deps{Config: cfg, DB: pool, Cache: rdb, Logger: logger})

	var started atomic.Bool
	// No listener but this one: a busy port is logged (the probes then fail) and the worker keeps working.
	if _, stopMetrics, err := serveMetrics(ctx, logger, cfg.HTTP.MetricsAddr, workerHandler(&started, pool, rdb)); err != nil {
		logger.Error("metrics listener not started", slog.String("error", err.Error()))
	} else {
		defer stopMetrics()
	}

	return work(ctx, logger, app, &started)
}

// work starts app's background hooks, marks started, waits for ctx and stops them (running jobs drain).
func work(ctx context.Context, logger *slog.Logger, app *fiber.App, started *atomic.Bool) error {
	if err := auth.StartBackground(app); err != nil {
		return err
	}
	started.Store(true)
	logger.Info("worker started", slog.Int("hooks", auth.BackgroundHooks(app)), slog.Any("domains", apihttp.Domains()))

	<-ctx.Done()
	started.Store(false)
	logger.Info("worker stopping")
	done := make(chan struct{})
	go func() {
		auth.StopBackground(app)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownTimeout + 5*time.Second):
		logger.Error("worker: background hooks did not stop in time")
	}
	logger.Info("stopped")
	return nil
}

// workerHandler is the worker's private listener: metrics.Handler() plus GET /up (liveness: the background hooks
// run) and GET /ready (they run and MariaDB + Redis each answer a ping within workerReadyTimeout; else 503 naming
// the failed checks, like the API's /ready).
func workerHandler(started *atomic.Bool, pool *sql.DB, rdb *cache.Client) nethttp.Handler {
	mux := nethttp.NewServeMux()
	mux.Handle("/", metrics.Handler())
	mux.HandleFunc("GET /up", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		if !started.Load() {
			writeStatus(w, nethttp.StatusServiceUnavailable, map[string]any{"status": "starting"})
			return
		}
		writeStatus(w, nethttp.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		failed := []string{}
		if !started.Load() {
			failed = append(failed, "worker")
		}
		checks := []struct {
			name string
			ping func(context.Context) error
		}{{"db", pingOf(pool)}, {"redis", redisPingOf(rdb)}}
		for _, c := range checks {
			ctx, cancel := context.WithTimeout(r.Context(), workerReadyTimeout)
			err := c.ping(ctx)
			cancel()
			if err != nil {
				failed = append(failed, c.name)
			}
		}
		if len(failed) > 0 {
			writeStatus(w, nethttp.StatusServiceUnavailable, map[string]any{"status": "unavailable", "failed": failed})
			return
		}
		writeStatus(w, nethttp.StatusOK, map[string]any{"status": "ok"})
	})
	return mux
}

func pingOf(pool *sql.DB) func(context.Context) error {
	if pool == nil {
		return func(context.Context) error { return errNoDep }
	}
	return pool.PingContext
}

func redisPingOf(rdb *cache.Client) func(context.Context) error {
	if rdb == nil {
		return func(context.Context) error { return errNoDep }
	}
	return rdb.Ping
}

type depError string

func (e depError) Error() string { return string(e) }

const errNoDep = depError("not configured")

func writeStatus(w nethttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
