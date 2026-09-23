package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Fertility / TTC tracking (docs/fertility-ttc/README.md), Go only — no Laravel counterpart
// (deviations.md D-19). All auth:api, localized by Accept-Language. Static paths before {date}.
func init() {
	Register("fertility", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := fertility.NewHandlers(d.DB, clock.Real{})

		r.Get("/api/v1/fertility/today", locale, guard, h.Today)
		r.Get("/api/v1/fertility/bbt", locale, guard, h.BBT)
		r.Get("/api/v1/fertility/days/:date", locale, guard, h.ShowDay)
		r.Put("/api/v1/fertility/days/:date", locale, guard, h.UpdateDay)
	})
}
