// Command api is the Ritme Go HTTP service: config → MariaDB/Redis → router → Fiber.
//
// `api healthcheck` probes GET /up on HTTP_ADDR and exits 0/1; the Docker HEALTHCHECK
// uses it because the runtime image has no shell or curl. `api healthcheck ready` probes
// GET /ready instead (MariaDB + Redis reachable, T-M9-10): the deploy scripts wait on it.
//
// `api engine-shadow-report [-n N]` prints the CYCLE_ENGINE=shadow difference counts and samples (T-M9-32,
// engine_shadow.go).
//
// `api worker` is the background role (T-M9-12, worker.go): job worker, scheduler and pollers, no HTTP listener but
// the private metrics one. `api healthcheck worker [ready]` probes its /up or /ready there (METRICS_ADDR).
// API_RUN_WORKERS=0 keeps all of that out of the API process (stage and beta, which run a worker container).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // Asia/Tehran must resolve even in an image without zoneinfo.

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

const shutdownTimeout = 10 * time.Second

func main() {
	// Every record logged with a request context carries its X-Request-ID (T-M9-07).
	logger := slog.New(httpx.LogHandler(slog.NewJSONHandler(os.Stderr, nil)))
	slog.SetDefault(logger)

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck(healthTarget(os.Args[2:])))
	}
	if len(os.Args) > 1 && os.Args[1] == "engine-shadow-report" {
		os.Exit(engineShadowReport(os.Args[2:], os.Stdout))
	}

	serve := run
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		serve = runWorker
	}
	if err := serve(logger); err != nil {
		logger.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	runWorkers, err := auth.RunWorkersFromEnv()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, rdb, closeDeps, err := openDeps(ctx, logger, cfg, cfg.RunMigrations)
	if err != nil {
		return err
	}
	defer closeDeps()

	// A busy metrics port must not take the API down (a second local server, the contract harness): it is logged
	// as an error and the API serves without the private listener.
	if _, stopMetrics, err := startMetrics(ctx, logger, cfg.HTTP.MetricsAddr); err != nil {
		logger.Error("metrics listener not started", slog.String("error", err.Error()))
	} else {
		defer stopMetrics()
	}

	app := newApp(apihttp.Default(), &apihttp.Deps{Config: cfg, DB: pool, Cache: rdb, Logger: logger})
	auth.SetRunWorkers(app, runWorkers)
	if !runWorkers {
		// The `api worker` container runs the job worker, the scheduler and the pollers (safety.nightly included,
		// T-M10-24); this process only dispatches.
		logger.Info("background work off (API_RUN_WORKERS=0): jobs are dispatched for the worker",
			slog.Int("hooks", auth.BackgroundHooks(app)))
	}

	logger.Info("listening",
		slog.String("addr", cfg.HTTP.Addr),
		slog.String("env", cfg.App.Env),
		slog.Any("domains", apihttp.Domains()),
	)
	err = app.Listen(cfg.HTTP.Addr, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       shutdownTimeout,
	})
	logger.Info("stopped")
	return err
}

// openDeps opens MariaDB (running the goose migrations first when migrate is set) and Redis, and starts the config
// cache invalidation. The returned func closes both.
func openDeps(ctx context.Context, logger *slog.Logger, cfg *config.Config, migrate bool) (*sql.DB, *cache.Client, func(), error) {
	pool, err := db.Open(ctx, cfg.DB, cfg.App.Location)
	if err != nil {
		return nil, nil, nil, err
	}
	if migrate {
		if err := migrateOnStart(ctx, logger, pool); err != nil {
			_ = pool.Close()
			return nil, nil, nil, err
		}
	}

	rdb := cache.New(cfg.Redis)
	closeAll := func() {
		_ = rdb.Close()
		_ = pool.Close()
	}
	if err := rdb.Ping(ctx); err != nil {
		closeAll()
		return nil, nil, nil, err
	}
	// cache_requests_total{cache="cycle_engine"}: the cycle-engine cache reads through Redis() directly.
	rdb.TrackGets("cycle-engine:", "cycle_engine")
	// Config caches (T-M9-13): admin writes in any process invalidate every process through Redis pub/sub;
	// without Redis the 60 s TTL still applies.
	configcache.Start(ctx, rdb, logger)
	return pool, rdb, closeAll, nil
}

// startMetrics serves /metrics and /debug/pprof/* on the private listener addr (METRICS_ADDR; "" = off). It binds
// synchronously so a busy port is reported as an error. The returned func shuts it down; bound is the actual address
// ("" when off).
func startMetrics(ctx context.Context, logger *slog.Logger, addr string) (bound string, stop func(), err error) {
	return serveMetrics(ctx, logger, addr, nil)
}

// serveMetrics is startMetrics with handler in place of metrics.Handler() (nil = metrics.Handler()); the worker adds
// its /up and /ready to it.
func serveMetrics(ctx context.Context, logger *slog.Logger, addr string, handler nethttp.Handler) (bound string, stop func(), err error) {
	if addr == "" {
		logger.Info("metrics listener off")
		return "", func() {}, nil
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return "", nil, fmt.Errorf("metrics: listen %s: %w", addr, err)
	}
	srv := metrics.NewServer(addr)
	if handler != nil {
		srv.Handler = handler
	}
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			logger.Error("metrics listener", slog.String("error", err.Error()))
		}
	}()
	bound = ln.Addr().String()
	logger.Info("metrics listening", slog.String("addr", bound))
	return bound, func() {
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}, nil
}

// migrateOnStart runs db.MigrateOnStart (RUN_MIGRATIONS=true: stage, where goose owns the schema
// since T-M2-28) and logs what it decided. A failure stops the process: serving on a schema that
// is behind the code would be worse than not serving.
func migrateOnStart(ctx context.Context, logger *slog.Logger, pool *sql.DB) error {
	res, err := db.MigrateOnStart(ctx, pool)
	if err != nil {
		return err
	}
	logger.Info("migrations",
		slog.String("action", string(res.Action)),
		slog.Any("applied", res.Applied),
		slog.Int64("version", res.Version),
	)
	return nil
}

// healthTarget maps the `api healthcheck` arguments to the probed listener and path: none → /up (liveness) on
// HTTP_ADDR, "ready" → /ready there; "worker" → the worker's /up on METRICS_ADDR, "worker ready" → its /ready.
func healthTarget(args []string) (addr, path string) {
	addr = os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8020"
	}
	if len(args) > 0 && args[0] == "worker" {
		args = args[1:]
		addr = metricsAddrFromEnv()
	}
	if len(args) > 0 && args[0] == "ready" {
		return addr, "/ready"
	}
	return addr, "/up"
}

// metricsAddrFromEnv is METRICS_ADDR as config.Load reads it (unset = config.DefaultMetricsAddr).
func metricsAddrFromEnv() string {
	if v, ok := os.LookupEnv("METRICS_ADDR"); ok {
		return v
	}
	return config.DefaultMetricsAddr
}

// healthcheck returns the process exit code for `api healthcheck`: 0 when GET path on addr answers 200.
func healthcheck(addr, path string) int {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// The target is our own listener from HTTP_ADDR / METRICS_ADDR, pinned to loopback above, not user input.
	url := "http://" + net.JoinHostPort(host, port) + path
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, url, nil) //nolint:gosec // G704: see above
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	resp, err := nethttp.DefaultClient.Do(req) //nolint:gosec // G704: see above
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}
