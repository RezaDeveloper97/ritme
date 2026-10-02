package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/teen"
)

// Ritme Plus (B-N2-04, gating/trial offer B-N2-06), Go only — no Laravel counterpart (deviations.md D-35). /plus/plans is public (the paywall
// shows prices before login); everything else is auth:api and user-scoped. Localized by Accept-Language.
//
// Payment provider: the internal/payments adapter chosen by PAYMENT_PROVIDER (B-N2-05) — the fake TEST gateway by
// default outside production, Zarinpal once configured, none in production until then (checkout/verify answer 503
// payment_unavailable instead of granting Plus for free). The gateway's return/TEST-page routes: routes_payments.go.
func init() {
	Register("plus", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		var gateway plus.Gateway
		if gw := payments.New(paymentDeps(d)); gw != nil { // a nil *Gateway must stay a nil interface
			gateway = gw
		}
		h := plus.NewHandlers(plus.NewService(d.DB, d.Config.Plus, gateway, d.Logger), clock.Real{}).
			WithCommercial(teen.NewPolicy(d.DB)) // CB-TEEN-01: no offer, trial or checkout for a teen-mode account
		writes := writeThrottle(d)

		r.Get("/api/v1/plus/plans", locale, h.Plans)
		r.Get("/api/v1/plus/status", locale, guard, h.Status)
		r.Get("/api/v1/plus/usage", locale, guard, h.Usage)
		r.Get("/api/v1/plus/history", locale, guard, h.History)
		r.Get("/api/v1/plus/trial", locale, guard, h.TrialSheet) // trial sheet + offer (B-N2-06)
		r.Post("/api/v1/plus/trial/start", locale, guard, writes, h.StartTrial)
		r.Post("/api/v1/plus/checkout", locale, guard, writes, h.Checkout)
		r.Post("/api/v1/plus/verify", locale, guard, writes, h.Verify)
		r.Post("/api/v1/plus/cancel", locale, guard, writes, h.Cancel)
		r.Post("/api/v1/plus/restore", locale, guard, writes, h.Restore)
	})
}
