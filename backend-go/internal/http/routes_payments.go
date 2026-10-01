package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/payments"
)

// Payment gateway adapter (B-N2-05), Go only. Public, unauthenticated browser routes:
//
//   - GET /api/v1/payments/{provider}/return — the gateway sends the browser back here; it changes nothing and
//     303-redirects to an allow-listed web-app page (PLUS_CALLBACK_URL / PAYMENT_RETURN_URLS) with
//     ?reference&authority&status. Settlement is the authenticated POST /plus/verify (server to server).
//   - GET|POST /api/v1/payments/fake/pay/{authority} — the fake provider's TEST page; mounted only while
//     PAYMENT_PROVIDER=fake, which config refuses in production.
//
// Nothing is mounted when no provider is active (PAYMENT_PROVIDER=none, or a real provider without keys).
func init() {
	Register("payments", func(r fiber.Router, d *Deps) {
		gw := payments.New(paymentDeps(d))
		if gw == nil {
			return
		}
		r.Get("/api/v1/payments/:provider/return", gw.Return)
		if fake := payments.FakeHandlersFor(gw); fake != nil {
			r.Get("/api/v1/payments/fake/pay/:authority", fake.Page)
			r.Post("/api/v1/payments/fake/pay/:authority", writeThrottle(d), fake.Decide)
		}
	})
}

// paymentDeps builds the adapter's dependencies (routes_plus.go uses the same, so both see one provider; the fake's
// state is shared through Redis).
func paymentDeps(d *Deps) payments.Deps {
	return payments.Deps{App: d.Config.App, Config: d.Config.Payment, Plus: d.Config.Plus, Cache: d.Cache, Logger: d.Logger}
}
