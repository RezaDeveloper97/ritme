package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
)

// Auth / OTP (backend/routes/api.php, OtpAuthController). Middleware is attached per route
// so nothing leaks onto other domains' /api/v1 routes.
func init() {
	Register("auth", func(r fiber.Router, d *Deps) {
		m := auth.MustModule(r, auth.Deps{Config: d.Config, DB: d.DB, Cache: d.Cache, Logger: d.Logger})
		if m == nil {
			return // the server refuses to start (FailStartup)
		}
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h, guard := m.Handlers, m.Guard.RequireUser

		r.Post("/api/v1/auth/send-otp", locale, m.Throttle(5), h.SendOTP)
		r.Post("/api/v1/auth/verify-otp", locale, m.Throttle(10), h.VerifyOTP)

		r.Post("/api/v1/auth/logout", locale, guard, h.Logout)
		r.Get("/api/v1/auth/user", locale, guard, h.User)
		r.Post("/api/v1/auth/refresh-session", locale, guard, m.Throttle(10), h.RefreshSession)
	})
}
