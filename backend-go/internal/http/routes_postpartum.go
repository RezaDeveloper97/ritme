package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/postpartum"
)

// Postpartum mode (bloom B-N5-01), Go only — no Laravel counterpart (deviations.md D-54). All auth:api, localized by
// Accept-Language, writes per-user throttled. Health data: user-scoped only; EPDS answers are never logged.
func init() {
	Register("postpartum", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := postpartum.NewHandlers(d.DB, postpartum.NewService(d.DB), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/postpartum", locale, guard, h.Show)
		r.Post("/api/v1/postpartum/activate", locale, guard, writes, h.Activate)
		r.Get("/api/v1/postpartum/recovery", locale, guard, h.ShowRecovery)
		r.Put("/api/v1/postpartum/recovery", locale, guard, writes, h.SaveRecovery)
		r.Get("/api/v1/postpartum/epds/questions", locale, guard, h.Questions)
		r.Get("/api/v1/postpartum/epds", locale, guard, h.History)
		r.Post("/api/v1/postpartum/epds", locale, guard, writes, h.SaveCheck)
	})
}
