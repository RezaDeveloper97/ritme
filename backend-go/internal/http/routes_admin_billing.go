package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/billing"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/payments"
	"github.com/ritme/backend-go/internal/plus"
)

// Admin «اشتراک‌ها و پرداخت» (B-N2-09) under /api/admin/v1/plus/* — docs/go-migration/admin-api.md §15. Reads: any
// active admin; writes (plans, discount codes, settings, refunds, extensions): super admin. Refunds go through the
// same payment adapter as checkout (PAYMENT_PROVIDER); without one they can only be marked manually.
func init() {
	Register("admin_billing", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		var refunder billing.Refunder
		if gw := payments.New(paymentDeps(d)); gw != nil { // a nil *Gateway must stay a nil interface
			refunder = gw
		}
		svc := plus.NewService(d.DB, d.Config.Plus, nil, d.Logger) // settings only: no checkout from here
		billing.New(d.DB, svc, refunder, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
