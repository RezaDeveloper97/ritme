package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// WriteThrottleMax is how many writes (POST/PUT/PATCH/DELETE) one user may send per minute to
// the Go-only health domains that use writeThrottle (security audit M3-M7 #5). Far above what
// tapping through the app produces; a script creating rows gets 429 "Too Many Attempts.".
const WriteThrottleMax = 60

// writeThrottleName is the counter shared by every route using writeThrottle: one budget per
// user across care, checkups and fertility, apart from the auth throttles' counter.
const writeThrottleName = "writes"

// writeThrottle is the per-user write limit. Attach it after the auth guard (the key is the
// user id). Without Redis (unit tests) it lets everything through.
func writeThrottle(d *Deps) fiber.Handler {
	if d.Cache == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return ratelimit.New(d.Cache, clock.Real{}).Named(writeThrottleName, WriteThrottleMax, time.Minute, auth.ThrottleIdentity)
}
