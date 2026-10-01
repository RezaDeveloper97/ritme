package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Smart messages (backend/routes/api.php, prefix messages, auth:api).
func init() {
	Register("messages", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := messages.NewHandlers(d.DB, clock.Real{})
		const p = "/api/v1/messages"

		r.Get(p+"/daily", locale, guard, h.Daily)
		r.Get(p+"/mode", locale, guard, h.Mode)

		// Heavy pain / heavy bleeding nudges of the condition programs (CB-COND-06b): Go-only, D-41.
		nudges := conditionnudges.NewHandlers(d.DB, clock.Real{})
		r.Get(p+"/nudges", locale, guard, nudges.Index)
	})
}
