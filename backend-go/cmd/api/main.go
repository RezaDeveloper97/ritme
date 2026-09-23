// Command api is the Ritme Go HTTP service: config → MariaDB/Redis → router → Fiber.
//
// `api healthcheck` probes GET /up on HTTP_ADDR and exits 0/1; the Docker HEALTHCHECK
// uses it because the runtime image has no shell or curl.
package main

import (
	"context"
	"database/sql"
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
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
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
