package httpx

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// ReadyCheck is one readiness dependency: Ping nil counts as failed (the dependency is not configured).
type ReadyCheck struct {
	Name string
	Ping func(context.Context) error
}

// Ready is the readiness handler (T-M9-10): every check must answer within timeout → 200 {"status":"ok"}, else 503
// {"status":"unavailable","failed":[names…]} in check order. Never cached.
func Ready(timeout time.Duration, checks ...ReadyCheck) fiber.Handler {
	return func(c fiber.Ctx) error {
		failed := []string{}
		for _, chk := range checks {
			if !pingWithin(c.Context(), timeout, chk.Ping) {
				failed = append(failed, chk.Name)
			}
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		if len(failed) > 0 {
			return JSON(c, fiber.StatusServiceUnavailable, jsonx.Obj("status", "unavailable", "failed", failed))
		}
		return JSON(c, fiber.StatusOK, jsonx.Obj("status", "ok"))
	}
}

func pingWithin(parent context.Context, timeout time.Duration, ping func(context.Context) error) bool {
	if ping == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return ping(ctx) == nil
}
