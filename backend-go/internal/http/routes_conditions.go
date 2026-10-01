package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/conditions"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Condition programs (CB-COND-01), Go only — no Laravel counterpart (deviations.md D-40). All auth:api, localized
// by Accept-Language, writes per-user throttled; lists and clinical copy come from the catalog (cached reader).
// Health data: user-scoped only. Static paths before {date}.
func init() {
	Register("conditions", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := conditions.NewHandlers(conditions.NewService(d.DB, cat), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/conditions", locale, guard, h.Show)
		r.Post("/api/v1/conditions/enrolments", locale, guard, writes, h.Enrol)
		r.Delete("/api/v1/conditions/enrolments/:program", locale, guard, writes, h.Leave)
		r.Get("/api/v1/conditions/pain/:date", locale, guard, h.ShowPain)
		r.Put("/api/v1/conditions/pain/:date", locale, guard, writes, h.SavePain)
		r.Get("/api/v1/conditions/pmdd/chart", locale, guard, h.PMDDChart)
		r.Get("/api/v1/conditions/pmdd/:date", locale, guard, h.ShowPMDD)
		r.Put("/api/v1/conditions/pmdd/:date", locale, guard, writes, h.SavePMDD)
		r.Get("/api/v1/conditions/pbac/:date", locale, guard, h.ShowPBAC)
		r.Put("/api/v1/conditions/pbac/:date", locale, guard, writes, h.SavePBAC)
	})
}
