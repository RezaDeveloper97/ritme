package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Contraception (CB-CONTRA-01), Go only — no Laravel counterpart (deviations.md D-36). All auth:api, localized by
// Accept-Language, writes per-user throttled. Health data: user-scoped only. Static paths before {date}.
func init() {
	Register("contraception", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := contraception.NewHandlers(contraception.NewService(d.DB), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/contraception", locale, guard, h.Show)
		r.Put("/api/v1/contraception/method", locale, guard, writes, h.SaveMethod)
		r.Delete("/api/v1/contraception/method", locale, guard, writes, h.StopMethod)
		r.Post("/api/v1/contraception/pills", locale, guard, writes, h.LogPill)
		r.Delete("/api/v1/contraception/pills/:date", locale, guard, writes, h.UnlogPill)
	})
}
