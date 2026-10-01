package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/profile"
)

// Onboarding v2 and the life-stage modes (B-N2-01): the step-by-step answers of the new onboarding and the
// mode / IVF-IUI / contraception switches. Go only — no Laravel counterpart. auth:api, localized by
// Accept-Language; writes are per-user throttled.
func init() {
	Register("onboarding", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := profile.NewOnboardingHandlers(d.DB, clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/onboarding", locale, guard, h.Show)
		r.Post("/api/v1/onboarding/complete", locale, guard, writes, h.Complete)
		r.Put("/api/v1/onboarding/steps/:step", locale, guard, writes, h.UpdateStep)
		r.Get("/api/v1/profile/life-stage", locale, guard, h.ShowLifeStage)
		r.Put("/api/v1/profile/life-stage", locale, guard, writes, h.UpdateLifeStage)
	})
}
