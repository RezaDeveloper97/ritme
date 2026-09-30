package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/support"
)

// Admin support-reports inbox (B-N1-12b) under /api/admin/v1/support-reports — docs/go-migration/admin-api.md.
// The screenshot is streamed from private storage through the admin session chain only; it has no public URL.
func init() {
	Register("admin_support", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		route := func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}
		support.NewHandlers(d.DB, d.Config.StoragePath, d.Logger).Routes(route, kit)
	})
}
