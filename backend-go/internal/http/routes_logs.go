package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/resources/translations"
)

// Log taxonomy v2 (B-N3-01, internal/healthlog/taxonomy), Go only — no Laravel counterpart
// (deviations.md). All auth:api, localized by Accept-Language. Static paths before {date}; preferences and
// custom items (B-N3-02) too.
func init() {
	Register("logs", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(languages)
		h := healthlog.NewLogHandlers(healthlog.NewService(d.DB), clock.Real{},
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath), languages)
		writes := writeThrottle(d)

		r.Get("/api/v1/logs/taxonomy", locale, guard, h.Taxonomy)
		r.Get("/api/v1/logs/days", locale, guard, h.Days)
		r.Get("/api/v1/logs/days/:date", locale, guard, h.Day)
		r.Put("/api/v1/logs/days/:date", locale, guard, writes, h.Save)
		r.Delete("/api/v1/logs/days/:date", locale, guard, writes, h.Destroy)
		// B-N3-02: log preferences (order, visibility, quick tiles) and custom items
		r.Get("/api/v1/logs/preferences", locale, guard, h.Preferences)
		r.Put("/api/v1/logs/preferences", locale, guard, writes, h.SavePreferences)
		r.Delete("/api/v1/logs/preferences", locale, guard, writes, h.ResetPreferences)
		r.Get("/api/v1/logs/custom-items", locale, guard, h.CustomItems)
		r.Post("/api/v1/logs/custom-items", locale, guard, writes, h.AddCustomItem)
		r.Patch("/api/v1/logs/custom-items/:id", locale, guard, writes, h.RenameCustomItem)
		r.Delete("/api/v1/logs/custom-items/:id", locale, guard, writes, h.DeleteCustomItem)
	})
}
