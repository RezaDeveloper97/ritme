package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	adminpregnancy "github.com/ritme/backend-go/internal/admin/pregnancy"
)

// Pregnancy v2 admin API (T-M7-06): week details, care plan and alert rules under
// /api/admin/v1 — see docs/go-migration/admin-api.md §13 and docs/pregnancy-v2/README.md.
// POST /messages (create in a registered group) lives with the messages editor
// (routes_admin_content.go).
func init() {
	Register("admin_pregnancy", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		adminpregnancy.New(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
