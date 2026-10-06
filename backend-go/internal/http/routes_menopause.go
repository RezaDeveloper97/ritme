package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/resources/translations"
)

// Menopause mode (CB-MENO-02, treatment + report CB-MENO-03), Go only — no Laravel counterpart (deviations.md D-42). All auth:api, localized by
// Accept-Language, writes per-user throttled; lists and clinical copy come from the catalog (cached reader), trigger
// names from the log-taxonomy namespace. Health data: user-scoped only. Static paths before {id}.
func init() {
	Register("menopause", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := menopause.NewHandlers(menopause.NewService(d.DB, cat),
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/menopause/profile", locale, guard, h.ShowProfile)
		r.Put("/api/v1/menopause/profile", locale, guard, writes, h.SaveProfile)
		r.Get("/api/v1/menopause/today", locale, guard, h.Today)
		r.Get("/api/v1/menopause/hot-flashes", locale, guard, h.ListFlashes)
		r.Post("/api/v1/menopause/hot-flashes", locale, guard, writes, h.StartFlash)
		r.Post("/api/v1/menopause/hot-flashes/:id/stop", locale, guard, writes, h.StopFlash)
		r.Get("/api/v1/menopause/scores", locale, guard, h.Scores)
		r.Post("/api/v1/menopause/scores", locale, guard, writes, h.SaveScore)
		r.Get("/api/v1/menopause/patterns", locale, guard, h.Patterns)
		// Treatment & care + the doctor report preview (CB-MENO-03); HRT / supplements are care medication reminders.
		r.Get("/api/v1/menopause/treatment", locale, guard, h.Treatment)
		r.Post("/api/v1/menopause/treatment/items", locale, guard, writes, h.StoreItem)
		r.Put("/api/v1/menopause/treatment/items/:id", locale, guard, writes, h.UpdateItem)
		r.Delete("/api/v1/menopause/treatment/items/:id", locale, guard, writes, h.DestroyItem)
		r.Put("/api/v1/menopause/treatment/items/:id/intakes/:date", locale, guard, writes, h.LogIntake)
		r.Delete("/api/v1/menopause/treatment/items/:id/intakes/:date", locale, guard, writes, h.UnlogIntake)
		r.Put("/api/v1/menopause/treatment/side-effects/:date", locale, guard, writes, h.SaveSideEffects)
		r.Get("/api/v1/menopause/report", locale, guard, h.Report)
	})
}

// menopauseReportSection is the `menopause` section of the doctor report builder (bloom B-N6-04 report + share
// links): a healthrecord.SectionProvider, report-only, menopause mode only.
func menopauseReportSection(d *Deps, labels *i18n.TranslationStore) *menopause.ReportSection {
	cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
	return menopause.NewReportSection(menopause.NewService(d.DB, cat), labels)
}
