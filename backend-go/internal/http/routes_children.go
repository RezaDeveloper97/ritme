package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Children (bloom B-N5-02, D-56), Go only — no Laravel counterpart. All auth:api, localized by Accept-Language,
// writes per-user throttled. A child is visible to its owner and, read-only and audited, to the owner's active spouse
// through a shared family; anything else is a uniform 404. Child data is never logged.
func init() {
	Register("children", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		svc := children.NewService(d.DB, cat)
		h := children.NewHandlers(svc, clock.Real{})
		h.SetToday(babylog.NewService(d.DB, svc)) // the child home's «امروز» card (B-N5-03)
		writes := writeThrottle(d)

		p := "/api/v1/children"
		r.Get(p, locale, guard, h.Index)
		r.Post(p, locale, guard, writes, h.Store)
		r.Get(p+"/reminders", locale, guard, h.Reminders)
		r.Get(p+"/:id", locale, guard, h.Show)
		r.Put(p+"/:id", locale, guard, writes, h.Update)
		r.Delete(p+"/:id", locale, guard, writes, h.Destroy)
		r.Get(p+"/:id/measurements", locale, guard, h.Measurements)
		r.Post(p+"/:id/measurements", locale, guard, writes, h.StoreMeasurement)
		r.Put(p+"/:id/measurements/:mid", locale, guard, writes, h.UpdateMeasurement)
		r.Delete(p+"/:id/measurements/:mid", locale, guard, writes, h.DestroyMeasurement)
		r.Get(p+"/:id/growth", locale, guard, h.Growth)
		r.Get(p+"/:id/vaccines", locale, guard, h.Vaccines)
		r.Post(p+"/:id/vaccines/visits/:visit", locale, guard, writes, h.MarkVisit)
		r.Put(p+"/:id/vaccines/:code", locale, guard, writes, h.MarkDose)
		r.Delete(p+"/:id/vaccines/:code", locale, guard, writes, h.UnmarkDose)
		r.Get(p+"/:id/milestones", locale, guard, h.Milestones)
		r.Put(p+"/:id/milestones/:code", locale, guard, writes, h.Check)
		r.Get(p+"/:id/learn", locale, guard, h.Learn)
	})
}
