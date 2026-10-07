package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/telemed"
)

// Doctors directory «پزشکان و ماماها» (bloom B-N7-02, D-67), Go only: the directory with filters, a doctor's profile,
// free slots and reviews. All auth:api and localized by Accept-Language; review writes per-user throttled; a user can
// delete only their own review (anything else is a uniform 404). Booking (B-N7-03) plugs its Busy / VisitChecker in
// here; until then nothing is busy and nobody may review.
func init() {
	Register("telemed", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		reader := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := telemed.NewHandlers(telemed.NewService(d.DB, reader, nil, nil), clock.Real{}, d.Config.App.URL)
		writes := writeThrottle(d)

		p := "/api/v1/telemed/doctors"
		r.Get(p, locale, guard, h.List)
		r.Get(p+"/filters", locale, guard, h.Filters)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Get(p+"/:id/slots", locale, guard, h.Slots)
		r.Get(p+"/:id/reviews", locale, guard, h.Reviews)
		r.Post(p+"/:id/reviews", locale, guard, writes, h.StoreReview)
		r.Delete(p+"/:id/reviews/:review", locale, guard, writes, h.DestroyReview)
	})
}
