package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Admin AI usage + cost aggregates (B-N6-05) under /api/admin/v1/ai/usage — docs/go-migration/admin-api.md §17.
// Aggregates only (no user ids, no content); the admin-web screen comes with B-N9.
func init() {
	Register("admin_ai", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}
		budget := usage.NewBudget(d.DB, d.Config.AI.DailyCostCapUSD, clock.Real{}, d.Logger)
		usage.NewAdminHandlers(d.DB, budget, clock.Real{}).Routes(route, kit)
	})
}
