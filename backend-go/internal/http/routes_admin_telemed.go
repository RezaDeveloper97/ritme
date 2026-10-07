package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/telemed"
)

// Admin doctors directory (bloom B-N7-02) under /api/admin/v1/telemed/* — docs/go-migration/admin-api.md §19: doctors
// CRUD + photo, visit types, weekly availability, time off, slot preview and review moderation. The console UI is
// B-N7-08. Photos go to the public disk on the backend-storage volume (STORAGE_PATH, mounted read-write).
func init() {
	Register("admin_telemed", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		reader := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		svc := telemed.NewService(d.DB, reader, nil, nil)
		telemed.NewAdmin(svc, media.NewDisk(d.Config.StoragePath), media.DefaultOptimizer, d.Config.App.URL, d.Logger).
			Routes(func(method, path string, chain httpadmin.Chain) {
				httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
			}, kit)
	})
}
