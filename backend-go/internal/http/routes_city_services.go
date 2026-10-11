package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// CityServicesPublicMax is the per-IP budget of the public directory reads per minute (CB-DIR-02).
const CityServicesPublicMax = 120

// City services (bloom B-N10-05, D-92), Go only: admin-managed classes / pools / playhouses near the user with a
// link-out, phone or request booking, plus the user's own «رزروها» list. Localized.
//
// CB-DIR-02 (mother & child directory, docs/canvas-build/directory.md): options, listings, listing and the new
// listings/{id}/slots are PUBLIC (browse without login, web SEO), IP-throttled and briefly cacheable; only published
// + active places are read. The bookings routes stay auth:api.
func init() {
	Register("city_services", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := cityservices.NewHandlers(cityservices.NewStore(d.DB), cityservices.NewNotifier("fake", d.Logger),
			d.Config.App.URL, clock.Real{}, d.Logger).
			WithCatalog(catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)) // CB-DIR-02
		writes := writeThrottle(d) // security audit (CB-DIR-03): per-user write budget on every POST
		public := func(c fiber.Ctx) error { return c.Next() }
		if d.Cache != nil {
			public = ratelimit.New(d.Cache, clock.Real{}).Named("city-services-public", CityServicesPublicMax, time.Minute, nil)
		}
		r.Get("/api/v1/city-services/options", public, locale, h.Options)
		r.Get("/api/v1/city-services/listings", public, locale, h.Listings)
		r.Get("/api/v1/city-services/listings/:id/slots", public, locale, h.Slots) // CB-DIR-02
		r.Get("/api/v1/city-services/listings/:id", public, locale, h.Listing)
		r.Post("/api/v1/city-services/listings/:id/report", locale, guard, writes, h.ReportListing) // CB-DIR-03
		// CB-DIR-03 extends bloom's booking routes in place (service / slot / child, ?scope=upcoming|past, cancel
		// window) and adds one booking, reschedule and review. Still a request the Ritme team confirms; no payment.
		r.Post("/api/v1/city-services/bookings", locale, guard, writes, h.StoreBooking)
		r.Get("/api/v1/city-services/bookings", locale, guard, h.Bookings)
		r.Get("/api/v1/city-services/bookings/:id", locale, guard, h.Booking) // CB-DIR-03
		r.Post("/api/v1/city-services/bookings/:id/cancel", locale, guard, writes, h.CancelBooking)
		r.Post("/api/v1/city-services/bookings/:id/reschedule", locale, guard, writes, h.RescheduleBooking) // CB-DIR-03
		r.Post("/api/v1/city-services/bookings/:id/review", locale, guard, writes, h.ReviewBooking)         // CB-DIR-03
	})
}
