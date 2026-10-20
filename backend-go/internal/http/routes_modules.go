package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/modules"
)

// App module kill-switches: GET /api/v1/modules (public, read by onboarding and the app shell) and the admin
// «ماژول‌ها» screen under /api/admin/v1/modules (config capability). The 403 gate on the modules' own routes is
// installed app-wide in cmd/api (modules.Gate), before any domain mounts.
func init() {
	Register("modules", func(r fiber.Router, d *Deps) {
		h := modules.NewHandlers(modules.NewService(d.DB, d.Cache, d.Logger), d.Logger)
		r.Get("/api/v1/modules", h.Public)

		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		httpadmin.Handle(r, fiber.MethodGet, httpadmin.Prefix+"/modules", kit.Admin(h.AdminShow))
		httpadmin.Handle(r, fiber.MethodPut, httpadmin.Prefix+"/modules", kit.Admin(h.AdminUpdate))
	})
}
