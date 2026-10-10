package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/rbac"
)

// Admin roles & permissions, audit log and temporary health-data access (bloom B-N9-02) under /api/admin/v1 —
// docs/go-migration/admin-api.md §21.
func init() {
	Register("admin_rbac", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		rbac.NewHandlers(d.DB, kit, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
