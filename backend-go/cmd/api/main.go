// Command api is the Ritme Go HTTP service: config → MariaDB/Redis → router → Fiber.
//
// `api healthcheck` probes GET /up on HTTP_ADDR and exits 0/1; the Docker HEALTHCHECK
// uses it because the runtime image has no shell or curl.
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

	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/config"
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
		os.Exit(healthcheck())
	}

	if err := run(logger); err != nil {
		logger.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DB, cfg.App.Location)
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()

	if cfg.RunMigrations {
		if err := migrateOnStart(ctx, logger, pool); err != nil {
			return err
		}
	}

	rdb := cache.New(cfg.Redis)
	defer func() { _ = rdb.Close() }()
	if err := rdb.Ping(ctx); err != nil {
		return err
	}
	// cache_requests_total{cache="cycle_engine"}: the cycle-engine cache reads through Redis() directly.
	rdb.TrackGets("cycle-engine:", "cycle_engine")
	// Config caches (T-M9-13): admin writes in any process invalidate every process through Redis pub/sub;
	// without Redis the 60 s TTL still applies.
	configcache.Start(ctx, rdb, logger)

	// A busy metrics port must not take the API down (a second local server, the contract harness): it is logged
	// as an error and the API serves without the private listener.
	if _, stopMetrics, err := startMetrics(ctx, logger, cfg.HTTP.MetricsAddr); err != nil {
		logger.Error("metrics listener not started", slog.String("error", err.Error()))
	} else {
		defer stopMetrics()
	}

	app := newApp(apihttp.Default(), &apihttp.Deps{Config: cfg, DB: pool, Cache: rdb, Logger: logger})

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

// startMetrics serves /metrics and /debug/pprof/* on the private listener addr (METRICS_ADDR; "" = off). It binds
// synchronously so a busy port is reported as an error. The returned func shuts it down; bound is the actual address
// ("" when off).
func startMetrics(ctx context.Context, logger *slog.Logger, addr string) (bound string, stop func(), err error) {
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

// healthcheck returns the process exit code for `api healthcheck`.
func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8020"
	}
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
	// The target is our own listener from HTTP_ADDR, pinned to loopback above, not user input.
	url := "http://" + net.JoinHostPort(host, port) + "/up"
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
