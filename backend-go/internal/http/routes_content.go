package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	contentstore "github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/teen"
	"github.com/ritme/backend-go/resources/translations"
)

// Content (backend/routes/api.php: languages, info, privacy, banners, articles,
// cycle/phase-content) plus the public disk under /storage. Middleware is attached per
// route so nothing leaks onto other domains' /api/v1 routes.
func init() {
	Register("content", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		reg := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(reg)
		h := content.New(content.Deps{
			Queries:      contentstore.New(d.DB),
			Translations: i18n.NewTranslationStore(translations.FS, d.Config.StoragePath),
			AppURL:       d.Config.App.URL,
			StoragePath:  d.Config.StoragePath,
			Logger:       d.Logger,
			Commercial:   teen.NewPolicy(d.DB), // CB-TEEN-01: no banners for a teen-mode account
		})

		// Public.
		r.Get("/api/v1/languages", locale, h.Languages)
		r.Get("/api/v1/languages/:code/messages", locale, h.Messages)
		r.Get("/api/v1/info/:group", locale, h.Info)
		r.Get("/api/v1/privacy", locale, h.Privacy)
		r.Get("/api/v1/info-pages/:group", locale, h.InfoPage) // B-N1-12, Go only

		// auth:api.
		r.Get("/api/v1/banners", locale, guard, h.Banners)
		r.Get("/api/v1/articles", locale, guard, h.Articles)
		r.Get("/api/v1/articles/:slug", locale, guard, h.Article)
		r.Get("/api/v1/cycle/phase-content/:phase", locale, guard, h.PhaseContent)

		// Public disk (Apache served it through the storage:link symlink).
		r.Get("/storage/*", h.Storage)
	})
}
