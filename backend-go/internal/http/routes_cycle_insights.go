package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/insights"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Night & Bloom cycle read models (B-N1-08, Go only — no Laravel route): the cycle history screen
// and the symptom pattern over the typical cycle. Both auth:api, static paths only. Kept out of
// routes_cycle.go so the Laravel-mirrored route file stays as ported.
func init() {
	Register("cycle-insights", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := insights.NewHandlers(cycleservice.New(d.DB, nil), clock.Real{})

		r.Get("/api/v1/cycle/history", locale, guard, h.History)
		r.Get("/api/v1/cycle/symptom-pattern", locale, guard, h.SymptomPattern)
	})
}
