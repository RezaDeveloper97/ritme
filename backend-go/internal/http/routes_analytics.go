package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analyticsevents"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/consent"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// AnalyticsEventsMax is the per-user request limit per minute of POST /analytics/events.
const AnalyticsEventsMax = 30

// First-party product analytics (bloom B-N9-08, D-89), Go only: the app posts batches of allow-listed, non-health
// events; recorded only for users who granted the `anonymous_stats` consent. Own throttle counter, localized 429.
func init() {
	Register("analytics_events", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := analyticsevents.NewHandlers(analyticsevents.NewStore(d.DB), consent.NewService(d.DB), clock.Real{})
		throttle := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			throttle = ratelimit.New(d.Cache, clock.Real{}).
				NamedWith("analytics-events", AnalyticsEventsMax, time.Minute, auth.ThrottleIdentity, writeThrottled)
		}
		r.Post("/api/v1/analytics/events", locale, guard, throttle, h.Ingest)
	})
}
