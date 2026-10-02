package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/companion"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/home"
	homestore "github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Home page, toggles and notifications (backend/routes/api.php prefix `home`, HomeController),
// all auth:api. Static paths before {param} paths.
func init() {
	Register("home", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := home.NewHandlers(home.Deps{
			Queries:   homestore.New(d.DB),
			Messages:  msgstore.New(d.DB),
			Pregnancy: pstore.New(d.DB),
			AppURL:    d.Config.App.URL,
			Logger:    d.Logger,
			Plus:      plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), // trial banner only: no gateway needed
			Accounts:  companion.NewAccounts(d.DB),                         // B-N4-03: 409 companion_account for a male account
		}, cycleservice.New(d.DB, nil), clock.Real{})

		g := r.Group("/api/v1/home")
		g.Get("", locale, guard, h.Index)
		g.Get("/cycle-overview", locale, guard, h.CycleOverview) // Go only (B-N1-06)
		g.Get("/sections/:section", locale, guard, h.Section)
		g.Post("/tasks/:task/toggle", locale, guard, h.ToggleTask)
		g.Post("/challenges/:challenge/toggle", locale, guard, h.ToggleChallenge)
		g.Get("/notifications", locale, guard, h.Notifications)
		g.Post("/notifications/read-all", locale, guard, h.MarkAllNotificationsRead)
		g.Post("/notifications/:notification/read", locale, guard, h.MarkNotificationRead)
	})
}
