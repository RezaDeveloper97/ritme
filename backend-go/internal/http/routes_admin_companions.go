package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/companions"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
)

// Admin companion «همدم» module (B-N4-07) under /api/admin/v1/companions/* — docs/go-migration/admin-api.md §16:
// the companion tips copy (message_contents group companion_tip) and a masked, read-only overview of links.
func init() {
	Register("admin_companions", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}
		companions.New(d.DB, d.Logger).Routes(route, kit)
	})
}
