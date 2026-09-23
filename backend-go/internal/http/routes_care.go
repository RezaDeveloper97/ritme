package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Care reminders (docs/care-reminders/README.md), Go only — no Laravel counterpart
// (deviations.md). All auth:api, localized by Accept-Language. Static paths before {id}.
func init() {
	Register("care", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := care.NewHandlers(d.DB, clock.Real{})

		r.Get("/api/v1/care/enums", locale, guard, h.Enums)
		r.Get("/api/v1/care/medications", locale, guard, h.ListMedications)
		r.Post("/api/v1/care/medications", locale, guard, h.StoreMedication)
		r.Get("/api/v1/care/medications/:id", locale, guard, h.ShowMedication)
		r.Put("/api/v1/care/medications/:id", locale, guard, h.UpdateMedication)
		r.Delete("/api/v1/care/medications/:id", locale, guard, h.DestroyMedication)
		r.Post("/api/v1/care/medications/:id/intakes", locale, guard, h.TakeIntake)
		r.Delete("/api/v1/care/medications/:id/intakes", locale, guard, h.UntakeIntake)
	})
}
