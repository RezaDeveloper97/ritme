package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
)

// Ritme Plus (B-N2-04), Go only — no Laravel counterpart (deviations.md D-35). /plus/plans is public (the paywall
// shows prices before login); everything else is auth:api and user-scoped. Localized by Accept-Language.
//
// Payment provider: the in-package fake everywhere except production, where none is wired until B-N2-05 adds the
// adapter package with a real gateway behind config — checkout/verify then answer 503 payment_unavailable instead
// of granting Plus for free.
func init() {
	Register("plus", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		var gateway plus.Gateway
		if !d.Config.App.IsProduction() {
			gateway = plus.FakeGateway{}
		}
		h := plus.NewHandlers(plus.NewService(d.DB, d.Config.Plus, gateway, d.Logger), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/plus/plans", locale, h.Plans)
		r.Get("/api/v1/plus/status", locale, guard, h.Status)
		r.Get("/api/v1/plus/usage", locale, guard, h.Usage)
		r.Get("/api/v1/plus/history", locale, guard, h.History)
		r.Post("/api/v1/plus/trial/start", locale, guard, writes, h.StartTrial)
		r.Post("/api/v1/plus/checkout", locale, guard, writes, h.Checkout)
		r.Post("/api/v1/plus/verify", locale, guard, writes, h.Verify)
		r.Post("/api/v1/plus/cancel", locale, guard, writes, h.Cancel)
		r.Post("/api/v1/plus/restore", locale, guard, writes, h.Restore)
	})
}
