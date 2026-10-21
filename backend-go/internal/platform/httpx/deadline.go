package httpx

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

type deadlineKey struct{}

// RequestDeadline gives the request context (c.Context()) a deadline of d (T-M9-10), so a stuck query made with it
// is cancelled instead of holding a pool connection; the handler then fails like any other DB error (500). skip
// exempts the long-running routes (AI, uploads, exports); d <= 0 turns it off.
//
// fiber.Ctx used directly as a context.Context never carries a deadline (its Deadline/Done are fixed). The bounded
// context is therefore also stored in Locals, where Value — and so DeadlineFrom — finds it through the fiber.Ctx too.
func RequestDeadline(d time.Duration, skip func(fiber.Ctx) bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		if d <= 0 || (skip != nil && skip(c)) {
			return c.Next()
		}
		ctx, cancel := context.WithTimeout(c.Context(), d)
		defer cancel()
		c.SetContext(ctx)
		c.Locals(deadlineKey{}, ctx)
		return c.Next()
	}
}

// DeadlineFrom returns the request's deadline-bounded context when ctx is a request context or the fiber.Ctx itself
// (nil, false outside RequestDeadline: jobs, migrations, exempt routes).
func DeadlineFrom(ctx context.Context) (context.Context, bool) {
	if ctx == nil {
		return nil, false
	}
	if bounded, ok := ctx.Value(deadlineKey{}).(context.Context); ok && bounded != nil {
		return bounded, true
	}
	return nil, false
}
