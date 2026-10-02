package main

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

// bodyLimit matches the upload ceiling of the PHP stack (25 MB).
const bodyLimit = 25 * 1024 * 1024

// newApp builds the Fiber app: proxy/IP handling, access log, CORS, then every
// registered domain from reg.
func newApp(reg *apihttp.Registry, deps *apihttp.Deps) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "ritme-backend-go",
		BodyLimit: bodyLimit,
		// Laravel trusts every proxy (bootstrap/app.php trustProxies(at: '*')): the
		// container port is only reachable through nginx. c.IP() returns the first
		// valid address of X-Forwarded-For, like Symfony's getClientIp() does when
		// all hops are trusted.
		ProxyHeader:        fiber.HeaderXForwardedFor,
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0/0", "::/0"}},
		EnableIPValidation: true,
		ErrorHandler:       httpx.ErrorHandler(deps.Logger),
	})

	app.Use(accessLog(deps.Logger))
	// Lets the strangler smoke tests see which stack answered (deploy/switch-go-route.sh).
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Backend", "go")
		return c.Next()
	})
	// Per-request frozen clock (X-Test-Now), same gate as Laravel's TestClock.
	app.Use(clock.Middleware(clock.Real{}, clock.TestClockEnabledFromEnv()))
	app.Use(corsMiddleware(deps.Config.CORS.AllowedOrigins))

	reg.Mount(app, deps)
	return app
}

// accessLog writes one JSON line per request.
func accessLog(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			status = httpx.StatusOf(err)
		}
		level := slog.LevelInfo
		if status >= fiber.StatusInternalServerError {
			level = slog.LevelError
		} else if c.Path() == "/up" {
			return err // Docker probes /up every 10s; don't flood the log.
		}
		attrs := []slog.Attr{
			slog.String("method", c.Method()),
			slog.String("path", httpx.LogPath(c.Path())),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("ip", c.IP()),
		}
		if err != nil && status >= fiber.StatusInternalServerError {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
		logger.LogAttrs(c.Context(), level, "request", attrs...)
		return err
	}
}
