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

// DriverContext is the context a DB driver should run a query with (T-M9-10b). Most service calls pass the fiber.Ctx
// itself as their context.Context, and it never carries a deadline; for exactly that case DriverContext returns the
// request's bounded context from RequestDeadline, so the query is cancelled at the request deadline too. Any other ctx
// is returned unchanged: one with its own deadline or cancellation keeps it, and a context derived from the fiber.Ctx
// (context.WithoutCancel(c) for work that must outlive the request, a WithValue wrapper) is left alone on purpose.
// Long-running routes, jobs and migrations have no bounded context, so they are never bounded here.
func DriverContext(ctx context.Context) context.Context {
	if _, ok := ctx.(fiber.Ctx); !ok {
		return ctx
	}
	if bounded, ok := DeadlineFrom(ctx); ok {
		return bounded
	}
	return ctx
}
