package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/media"
	"github.com/ritme/backend-go/internal/shop"
	shopadmin "github.com/ritme/backend-go/internal/shop/admin"
)

// Shop admin API (B-N10-03, D-91): categories and link-out products under /api/admin/v1/shop (capability «content»).
func init() {
	Register("admin_shop", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		shop.NewAdmin(d.DB, media.NewDisk(d.Config.StoragePath), d.Config.App.URL, d.Logger).Routes(
			func(method, path string, chain httpadmin.Chain) {
				httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
			}, kit)
		// CB-SHOP-05 (additive): the internal shop's sellers, commerce fields, variants/stock, slots, shipment queue,
		// returns, reviews, discount codes and layette template — same /shop section, routes registered after bloom's.
		shopadmin.New(d.DB, media.NewDisk(d.Config.StoragePath), d.Config.App.URL, d.Logger).Routes(
			func(method, path string, chain httpadmin.Chain) {
				httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
			}, kit)
	})
}
