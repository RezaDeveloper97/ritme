package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/cityservices"
)

// City-services admin API (B-N10-05, D-92): /api/admin/v1/city-services/* (capability «content») and the booking
// request queue /api/admin/v1/city-service-bookings/* (capability «support»).
func init() {
	Register("admin_city_services", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		cityservices.NewAdmin(d.DB, media.NewDisk(d.Config.StoragePath), d.Config.App.URL, d.Logger).Routes(
			func(method, path string, chain httpadmin.Chain) {
				httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
			}, kit)
	})
}
