package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/labs"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/vitals"
	"github.com/ritme/backend-go/resources/translations"
)

// The «تحلیل» tab (B-N3-07, internal/analysis), Go only — no Laravel route. All auth:api, read-only,
// localized by Accept-Language (sentences from the `analysis` translation namespace). Plus sections
// are marked in the payload (`locked: true`, no data) instead of answering 402. Static paths before
// the {ym} path.
func init() {
	Register("analysis", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		h := analysis.NewHandlers(
			cycleservice.New(d.DB, nil),
			healthlog.NewService(d.DB),
			plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), // entitlements only: no gateway needed
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath),
			languages,
			clock.Real{},
		).WithLabs(labs.NewService(labs.Options{ // B-N6-06: the hub's labs card (reads only)
			DB: d.DB, Catalog: catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger), Logger: d.Logger,
		})).WithVitals(vitals.NewService(d.DB)) // B-N6-03: vitals averages read vital_readings merged with the log sheet

		r.Get("/api/v1/analysis/summary", locale, guard, h.Summary)
		r.Get("/api/v1/analysis/cycle", locale, guard, h.Cycle)
		r.Get("/api/v1/analysis/period", locale, guard, h.Period)
		r.Get("/api/v1/analysis/symptoms", locale, guard, h.Symptoms)
		r.Get("/api/v1/analysis/correlations", locale, guard, h.Correlations)
		r.Get("/api/v1/analysis/body", locale, guard, h.Body)
		r.Get("/api/v1/analysis/monthly/:ym", locale, guard, h.Monthly)
	})
}
