package httpadmin

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/configcache"
)

// Publishes wraps an admin write handler: once it has succeeded (its transaction committed), the config caches of
// tables are invalidated in this process and the change goes out on ritme-go:config-changed to the others (T-M9-13).
func Publishes(h fiber.Handler, tables ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := h(c); err != nil {
			return err
		}
		if c.Response().StatusCode() < fiber.StatusBadRequest {
			configcache.Changed(c.Context(), tables...)
		}
		return nil
	}
}
