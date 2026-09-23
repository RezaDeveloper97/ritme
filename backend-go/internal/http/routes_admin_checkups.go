package http

import (
	"github.com/gofiber/fiber/v3"

	admincheckups "github.com/ritme/backend-go/internal/admin/checkups"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
)

// Checkups admin API (T-M4-03): the checkup-type catalog under /api/admin/v1/checkup-types —
// see docs/go-migration/admin-api.md §12 and docs/checkups/README.md.
func init() {
	Register("admin_checkups", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		admincheckups.New(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
