package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/reminder"
)

// Reminders (backend/routes/api.php, ReminderController), all auth:api.
// Static /enums before the {id} param routes.
func init() {
	Register("reminder", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := reminder.NewHandlers(d.DB, clock.Real{})

		r.Get("/api/v1/reminders/enums", locale, guard, h.Enums)
		r.Get("/api/v1/reminders", locale, guard, h.Index)
		r.Post("/api/v1/reminders", locale, guard, h.Store)
		r.Put("/api/v1/reminders/:id", locale, guard, h.Update)
		r.Delete("/api/v1/reminders/:id", locale, guard, h.Destroy)
	})
}
