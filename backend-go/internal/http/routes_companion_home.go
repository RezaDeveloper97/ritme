package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	companionhome "github.com/ritme/backend-go/internal/companion/home"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Companion panel home (bloom B-N4-03, D-49), Go only. auth:api, localized by Accept-Language, read-only. A male
// (companion) account's home: per active link the granted, audited section views, phase tips (message_contents
// group companion_tip), reading and the shared child card; 403 not_companion_account for a woman's account.
func init() {
	Register("companion_home", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := companionhome.NewHandlers(d.DB, clock.Real{}, companionhome.Options{
			Reader: shared.NewReader(d.DB),
			AppURL: d.Config.App.URL,
			Logger: d.Logger,
		})
		r.Get("/api/v1/companion/home", locale, guard, h.Show)
	})
}
