package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/bannerads"
	contentstore "github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
	"github.com/ritme/backend-go/internal/teen"
)

// BannerEventsMax is the per-minute request limit of POST /banners/:id/events (own counter).
const BannerEventsMax = 60

// Sponsored home slots (bloom B-N10-02, D-93), Go only. Under /banners/ads and /banners/:id/events so the existing
// GET /banners stays byte-compatible. Registered before the content group's /banners route cannot clash (distinct paths).
func init() {
	Register("banner_ads", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := bannerads.New(contentstore.New(d.DB), d.Config.App.URL, teen.NewPolicy(d.DB), clock.Real{}, d.Logger)
		throttle := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			throttle = ratelimit.New(d.Cache, clock.Real{}).
				NamedWith("banner-events", BannerEventsMax, time.Minute, auth.ThrottleIdentity, writeThrottled)
		}
		r.Get("/api/v1/banners/ads", locale, guard, etag.ByBody(), h.List) // T-M12-34: ETag / 304
		r.Post("/api/v1/banners/:id/events", locale, guard, throttle, h.Event)
	})
}
