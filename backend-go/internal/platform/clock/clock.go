// Package clock is the injectable "now" of the API.
//
// Production uses Real (the wall clock, in Asia/Tehran). Tests use Fixed. The
// contract recorder freezes time per request with the X-Test-Now header, which
// Middleware honours under exactly the rule of the Laravel TestClock middleware
// (backend/app/Http/Middleware/TestClock.php): APP_ENV ∈ {local, testing, contract}
// AND TEST_CLOCK_ENABLED=true. Production can never match the first condition.
package clock

import (
	"context"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Clock tells the current instant.
type Clock interface {
	Now() time.Time
}

// Real is the wall clock, reported in Asia/Tehran (Carbon::now() with the app tz).
type Real struct{}

// Now implements Clock.
func (Real) Now() time.Time { return time.Now().In(civildate.Tehran) }

// Fixed always returns the same instant.
type Fixed time.Time

// Now implements Clock.
func (f Fixed) Now() time.Time { return time.Time(f) }

// At returns a Fixed clock at t.
func At(t time.Time) Fixed { return Fixed(t) }

// Header is the request header carrying the frozen instant.
const Header = "X-Test-Now"

// Environments are the APP_ENV values in which the test clock may be enabled.
var Environments = []string{"local", "testing", "contract"}

// TestClockEnabled is TestClock::isEnabled(): appEnv is one of Environments and flag
// is a PHP FILTER_VALIDATE_BOOLEAN truthy value ("1", "true", "on", "yes").
func TestClockEnabled(appEnv, flag string) bool {
	if !slices.Contains(Environments, appEnv) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(flag)) {
	case "1", "true", "on", "yes":
		return true
	}
	return false
}

// TestClockEnabledFromEnv reads APP_ENV and TEST_CLOCK_ENABLED from the process env.
func TestClockEnabledFromEnv() bool {
	return TestClockEnabled(os.Getenv("APP_ENV"), os.Getenv("TEST_CLOCK_ENABLED"))
}

type ctxKey struct{}

// WithClock returns ctx carrying c.
func WithClock(ctx context.Context, c Clock) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// FromContext returns the clock stored in ctx (a request's fiber.Ctx or its
// Context()), or fallback when there is none.
func FromContext(ctx context.Context, fallback Clock) Clock {
	if ctx != nil {
		if c, ok := ctx.Value(ctxKey{}).(Clock); ok {
			return c
		}
	}
	return fallback
}

// invalidHeaderBody is the Laravel body for an unparseable header (compact JSON).
const invalidHeaderBody = `{"message":"Invalid X-Test-Now header."}`

// Middleware stores the request clock: base, or a Fixed clock parsed from X-Test-Now
// when enabled is true. Handlers read it with FromContext(c, base).
//
// The header is parsed like Carbon::parse($value, 'Asia/Tehran'): ISO-8601, a value
// without an offset is Tehran wall-clock. An unparseable header is a 400 rather than a
// silent fallback to the real clock, as in Laravel.
func Middleware(base Clock, enabled bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		clk := base
		if enabled {
			if v := c.Get(Header); v != "" {
				t, err := civildate.ParseLenient(v, base.Now(), civildate.Tehran)
				if err != nil {
					c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
					return c.Status(fiber.StatusBadRequest).SendString(invalidHeaderBody)
				}
				clk = Fixed(t.In(civildate.Tehran))
			}
		}
		c.Locals(ctxKey{}, clk)
		c.SetContext(WithClock(c.Context(), clk))
		return c.Next()
	}
}
