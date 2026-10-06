package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/vitals"
)

// Vitals (bloom B-N6-01, D-63), Go only: timed BP / glucose / heart-rate readings with classification, reports, the
// weekly measurement plan and the urgent safety modal. All auth:api, localized by Accept-Language, writes per-user
// throttled; every row is scoped by the user (a foreign id is a uniform 404). Readings are never logged.
func init() {
	Register("vitals", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := vitals.NewHandlers(vitals.NewService(d.DB), clock.Real{})
		writes := writeThrottle(d)

		p := "/api/v1/vitals"
		r.Get(p, locale, guard, h.Hub)
		r.Get(p+"/thresholds", locale, guard, h.Thresholds)
		r.Get(p+"/plan", locale, guard, h.Plan)
		r.Put(p+"/plan", locale, guard, writes, h.SavePlan)
		r.Get(p+"/reports/:type", locale, guard, h.Report)
		r.Get(p+"/readings", locale, guard, h.Readings)
		r.Post(p+"/readings", locale, guard, writes, h.Store)
		r.Get(p+"/readings/:id", locale, guard, h.Show)
		r.Put(p+"/readings/:id", locale, guard, writes, h.Update)
		r.Delete(p+"/readings/:id", locale, guard, writes, h.Destroy)
	})
}
