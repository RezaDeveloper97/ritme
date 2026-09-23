package http

import "github.com/gofiber/fiber/v3"

func init() {
	Register("health", func(r fiber.Router, _ *Deps) {
		// Liveness only, like Laravel's /up: it must not depend on MariaDB or Redis,
		// so a DB hiccup never makes Docker restart the container.
		r.Get("/up", func(c fiber.Ctx) error {
			return c.SendString("OK")
		})
	})
}
