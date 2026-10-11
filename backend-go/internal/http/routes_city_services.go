package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// City services (bloom B-N10-05, D-92), Go only: admin-managed classes / pools / playhouses near the user with a
// link-out, phone or request booking, plus the user's own «رزروها» list. auth:api, localized.
func init() {
	Register("city_services", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := cityservices.NewHandlers(cityservices.NewStore(d.DB), cityservices.NewNotifier("fake", d.Logger),
			d.Config.App.URL, clock.Real{}, d.Logger)
		r.Get("/api/v1/city-services/options", locale, guard, h.Options)
		r.Get("/api/v1/city-services/listings", locale, guard, h.Listings)
		r.Get("/api/v1/city-services/listings/:id", locale, guard, h.Listing)
		r.Post("/api/v1/city-services/bookings", locale, guard, h.StoreBooking)
		r.Get("/api/v1/city-services/bookings", locale, guard, h.Bookings)
		r.Post("/api/v1/city-services/bookings/:id/cancel", locale, guard, h.CancelBooking)
	})
}
