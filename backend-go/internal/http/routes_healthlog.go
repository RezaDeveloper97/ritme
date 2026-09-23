package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Daily health logs (backend/routes/api.php, DailyHealthLogController), all auth:api.
// Static /enums before the {date} param routes.
func init() {
	Register("healthlog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := healthlog.NewHandlers(healthlog.NewService(d.DB), clock.Real{})

		r.Get("/api/v1/health-logs/enums", locale, guard, h.Enums)
		r.Get("/api/v1/health-logs", locale, guard, h.Index)
		r.Post("/api/v1/health-logs", locale, guard, h.Store)
		r.Get("/api/v1/health-logs/:date", locale, guard, h.Show)
		r.Delete("/api/v1/health-logs/:date", locale, guard, h.Destroy)
	})
}
