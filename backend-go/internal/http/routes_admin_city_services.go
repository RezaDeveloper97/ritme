package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/cityservices"
	"github.com/ritme/backend-go/internal/files"
)

// City-services admin API (B-N10-05, D-92): /api/admin/v1/city-services/* (capability «content») and the booking
// request queue /api/admin/v1/city-service-bookings/* (capability «support»). CB-DIR-05 adds the directory admin to
// the same Admin (services, slots, reviews, reports) and the application queue /city-service-applications/* («content»).
func init() {
	Register("admin_city_services", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		// CB-DIR-05: the application review queue reads private licences (5-minute signed links) and deletes a rejected
		// application's photos through internal/files — the same vault policy as routes_city_services_join.go.
		current, previous, dev, missing := d.Config.Files.Resolve(d.Config.App, d.Config.LabFiles)
		if dev {
			current = files.DevKey("ritme-file-dev-key")
		}
		vault, err := files.NewVault(d.Config.StoragePath, current, previous, missing)
		if err != nil {
			panic(err) // config.Load already checked the key lengths
		}
		fileSvc := files.NewService(files.Options{DB: d.DB, Vault: vault, BaseURL: d.Config.App.URL})
		cityservices.NewAdmin(d.DB, media.NewDisk(d.Config.StoragePath), d.Config.App.URL, d.Logger).WithFiles(fileSvc).Routes(
			func(method, path string, chain httpadmin.Chain) {
				httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
			}, kit)
	})
}
