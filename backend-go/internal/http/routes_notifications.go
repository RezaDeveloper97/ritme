package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/clock"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Notification settings (B-N1-11): categories, quiet hours, neutral lock-screen copy. Go only — no Laravel
// counterpart. auth:api, localized by Accept-Language; the write is per-user throttled.
func init() {
	Register("notifications", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := notifications.NewHandlers(profilestore.New(d.DB), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/profile/notification-settings", locale, guard, h.Show)
		r.Put("/api/v1/profile/notification-settings", locale, guard, writes, h.Update)
	})
}
