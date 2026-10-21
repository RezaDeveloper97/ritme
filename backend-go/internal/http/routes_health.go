package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/httpx"
)

// ReadyTimeout bounds each /ready dependency check.
const ReadyTimeout = time.Second

func init() {
	Register("health", func(r fiber.Router, d *Deps) {
		// Liveness only, like Laravel's /up: it must not depend on MariaDB or Redis,
		// so a DB hiccup never makes Docker restart the container.
		r.Get("/up", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})
		// Readiness (T-M9-10): MariaDB and Redis each answer a ping within ReadyTimeout, else 503 naming the failed
		// check. The deploy scripts wait on it (`api healthcheck ready`); nginx does not route it and the Docker
		// HEALTHCHECK stays on /up.
		r.Get("/ready", httpx.Ready(ReadyTimeout, readyChecks(d)...))
	})
}

func readyChecks(d *Deps) []httpx.ReadyCheck {
	var dbPing, redisPing func(context.Context) error
	if d != nil && d.DB != nil {
		dbPing = d.DB.PingContext
	}
	if d != nil && d.Cache != nil {
		redisPing = d.Cache.Ping
	}
	return []httpx.ReadyCheck{{Name: "db", Ping: dbPing}, {Name: "redis", Ping: redisPing}}
}
