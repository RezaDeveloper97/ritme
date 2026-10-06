package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	healthlogstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	"github.com/ritme/backend-go/internal/vitals"
	"github.com/ritme/backend-go/resources/translations"
)

// The pregnancy analysis (B-N3-12, internal/analysis/pregnancy*.go), Go only: one auth:api read-only report
// for the An_Hub_Preg hub and the An_PregWeight screen. Plus sections (glucose, symptoms by trimester)
// come back `locked: true` without data instead of a 402.
func init() {
	Register("analysis_pregnancy", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		h := analysis.NewPregnancyHandlers(
			store.New(d.DB),
			healthlogstore.New(d.DB),
			healthlog.NewService(d.DB),
			plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), // entitlements only
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath),
			languages,
			clock.Real{},
		).WithVitals(vitals.NewService(d.DB)) // B-N6-03: BP / glucose read vital_readings merged with the log sheet
		r.Get("/api/v1/analysis/pregnancy", i18n.Middleware(languages), guard, h.Pregnancy)
	})
}
