package http

import (
	"github.com/gofiber/fiber/v3"

	adminchildren "github.com/ritme/backend-go/internal/admin/children"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
)

// Admin «کودک» module (B-N5-09) under /api/admin/v1/children/* — docs/go-migration/admin-api.md §18: the read-only
// WHO growth-standard viewer. The child catalogs themselves are edited through /api/admin/v1/catalog/{group}
// (routes_admin_catalog.go, super-only writes).
func init() {
	Register("admin_children", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}
		adminchildren.New(d.Logger).Routes(route, kit)
	})
}
