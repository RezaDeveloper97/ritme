package http

import (
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/notify"
	"github.com/ritme/backend-go/internal/profile"
)

// Profile / account (backend/routes/api.php, ProfileController). Middleware per route.
func init() {
	Register("profile", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := profile.NewHandlers(profile.Options{
			DB:       d.DB,
			Telegram: notify.NewTelegram(d.Config.Telegram, &http.Client{}, d.Logger),
			Debug:    d.Config.App.Debug,
			Logger:   d.Logger,
		})

		r.Get("/api/v1/profile", locale, guard, h.Show)
		r.Post("/api/v1/profile", locale, guard, h.Store)
		r.Get("/api/v1/profile/export", locale, guard, h.Export)
		r.Delete("/api/v1/account", locale, guard, h.DestroyAccount)
	})
}
