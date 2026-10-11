package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/clinicalcfg"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/clinicalconfig"
)

// Versioned clinical thresholds & config with two medical approvals (bloom B-N9-11, D-88) under /api/admin/v1 —
// docs/go-migration/admin-api.md «Clinical thresholds & config». Mounting also installs the approved versions as the source of
// internal/clinicalconfig (the engines fall back to the code defaults until it loads or when it cannot).
func init() {
	Register("admin_clinical_config", func(r fiber.Router, d *Deps) {
		if d.DB != nil {
			clinicalconfig.Install(clinicalcfg.Loader(d.DB), d.Logger)
		}
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		clinicalcfg.NewHandlers(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
	})
}
