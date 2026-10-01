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

// Menopause mode (CB-MENO-02), Go only — no Laravel counterpart (deviations.md D-42). All auth:api, localized by
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
	})
}
