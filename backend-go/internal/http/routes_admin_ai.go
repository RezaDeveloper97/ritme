package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/aicredits"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/ai/usage"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Admin AI under /api/admin/v1 — docs/go-migration/admin-api.md §17 and §29:
//   - GET /ai/usage: usage + cost aggregates and the credits spent (B-N6-05, goose 00125). Aggregates only (no user
//     ids, no content).
//   - the AI credits admin (internal/admin/aicredits): model catalogue (+ gateway sync), feature slots, credit packs,
//     a user's credits and manual adjustments. The sync asks the provider adapters of the same *ai.Client the user
//     routes use (keys from the env only; never part of this API).
func init() {
	Register("admin_ai", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}
		budget := usage.NewBudget(d.DB, d.Config.AI.DailyCostCapUSD, clock.Real{}, d.Logger)
		usage.NewAdminHandlers(d.DB, budget, clock.Real{}).Routes(route, kit)

		var gateway aicredits.Gateway
		if client := newAIPlatform(d, nil).client; client != nil { // a nil *ai.Client must stay a nil interface
			gateway = client
		}
		aicredits.New(d.DB, gateway, d.Logger).Routes(route, kit)
	})
}
