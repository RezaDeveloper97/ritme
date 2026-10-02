package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/resources/translations"
)

// The TTC analysis (B-N3-11, internal/analysis ttc.go), Go only — no Laravel route. auth:api, read-only,
// localized by Accept-Language. Its own file so the analysis and TTC tasks never edit the same routes file.
func init() {
	Register("analysis-ttc", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		h := analysis.NewTTCHandlers(analysis.NewHandlers(
			cycleservice.New(d.DB, nil),
			healthlog.NewService(d.DB),
			plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), // entitlements only: no gateway needed
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath),
			languages,
			clock.Real{},
		), fertility.NewService(d.DB))

		r.Get("/api/v1/analysis/ttc", locale, guard, h.TTC)
		r.Get("/api/v1/analysis/fertility", locale, guard, h.Fertility)
	})
}
