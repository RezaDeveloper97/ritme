package http

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/checkups"
	checkupsstore "github.com/ritme/backend-go/internal/checkups/store"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/companion"
	contentstore "github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/enums"
	healthrecordstore "github.com/ritme/backend-go/internal/healthrecord/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	insurancestore "github.com/ritme/backend-go/internal/insurance/store"
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
		// CB-NAV-03: a teen's doctor / lab tiles need an active parent link (companion type parent, CB-TEEN-01).
		parentLinked := func(ctx context.Context, userID uint64) (bool, error) {
			links, err := companion.NewService(d.DB, clock.Real{}, companion.Options{}).ListForOwner(ctx, userID)
			if err != nil {
				return false, err
			}
			for _, l := range links {
				if l.Type == companion.TypeParent && l.Status == companion.StatusActive {
					return true, nil
				}
			}
			return false, nil
		}
		svc := services.NewService(cat, healthrecordstore.New(d.DB), mode, telemedBookings(d, false)).WithTeenParent(parentLinked).WithInsurancePage(insurancePublished(d)).WithCityServices(cityServicesLive(d)) // B-N7-03 card, B-N10-06 tile
		h := services.NewHandlers(svc, clock.Real{})

		r.Get("/api/v1/services", locale, guard, h.Show)
	})
}

// insurancePublished is B-N10-06's tile switch: the `insurance` info page has at least one active box.
// CB-INS-03: the tile also shows once the user keeps her own policy (the tracker lives at the same /insurance route).
func insurancePublished(d *Deps) services.InsurancePublished {
	q := contentstore.New(d.DB)
	iq := insurancestore.New(d.DB)
	return func(ctx context.Context, userID uint64) (bool, error) {
		rows, err := q.ListActiveInfoPageSections(ctx, "insurance")
		if err != nil || len(rows) > 0 {
			return len(rows) > 0, err
		}
		if _, err := iq.GetUserPolicy(ctx, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
}

// cityServicesLive is B-N10-05's section switch: at least one visible city-service listing.
func cityServicesLive(d *Deps) services.CityServicesLive {
	st := cityservices.NewStore(d.DB)
	return st.HasActive
}
