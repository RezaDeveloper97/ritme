package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/content"
	contentstore "github.com/ritme/backend-go/internal/content/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/resources/translations"
)

// languageRegistryTTL bounds how long a language edited in the Laravel admin takes to
// reach Go during the strangler period (the registry entry itself has no TTL).
const languageRegistryTTL = 5 * time.Minute

// Content (backend/routes/api.php: languages, info, privacy, banners, articles,
// cycle/phase-content) plus the public disk under /storage. Middleware is attached per
// route so nothing leaks onto other domains' /api/v1 routes.
func init() {
	Register("content", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		reg := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		locale := i18n.Middleware(reg)
		ttl := content.RegistryTTL(d.Cache, languageRegistryTTL)
		h := content.New(content.Deps{
			Queries:      contentstore.New(d.DB),
			Translations: i18n.NewTranslationStore(translations.FS, d.Config.StoragePath),
			AppURL:       d.Config.App.URL,
			StoragePath:  d.Config.StoragePath,
			Logger:       d.Logger,
		})

		// Public.
		r.Get("/api/v1/languages", ttl, locale, h.Languages)
		r.Get("/api/v1/languages/:code/messages", ttl, locale, h.Messages)
		r.Get("/api/v1/info/:group", ttl, locale, h.Info)
		r.Get("/api/v1/privacy", ttl, locale, h.Privacy)

		// auth:api.
		r.Get("/api/v1/banners", ttl, locale, guard, h.Banners)
		r.Get("/api/v1/articles", ttl, locale, guard, h.Articles)
		r.Get("/api/v1/articles/:slug", ttl, locale, guard, h.Article)
		r.Get("/api/v1/cycle/phase-content/:phase", ttl, locale, guard, h.PhaseContent)

		// Public disk (Apache served it through the storage:link symlink).
		r.Get("/storage/*", h.Storage)
	})
}
