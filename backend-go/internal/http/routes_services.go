package http

import (
	"context"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/checkups"
	checkupsstore "github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/enums"
	healthrecordstore "github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/services"
)

// Services hub «خدمات» (bloom B-N7-01), Go only — no Laravel counterpart (deviations.md D-69). auth:api, localized by
// Accept-Language. Sections, care tiles and programs come from the catalog groups services_sections / services_care
// / services_programs (cached reader, flushed by admin writes). The upcoming booking is the user's next confirmed
// visit from the telemedicine domain (telemed.Bookings as services.BookingSource, B-N7-03).
func init() {
	Register("services", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		checkupsQ := checkupsstore.New(d.DB)
		mode := func(ctx context.Context, userID uint64) (enums.LifeMode, error) {
			return checkups.UserLifeMode(ctx, checkupsQ, userID)
		}
		svc := services.NewService(cat, healthrecordstore.New(d.DB), mode, telemedBookings(d, false)) // B-N7-03 card
		h := services.NewHandlers(svc, clock.Real{})

		r.Get("/api/v1/services", locale, guard, h.Show)
	})
}
