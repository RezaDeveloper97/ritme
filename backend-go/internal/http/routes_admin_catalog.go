package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
)

// Content catalog admin API (CB-CORE-03): /api/admin/v1/catalog[/{group}[/{id}]] — see
// docs/canvas-build/catalog.md. Every write flushes the group's public cache entry.
func init() {
	Register("admin_catalog", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		reader := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		catalog.NewAdmin(d.DB, reader, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
