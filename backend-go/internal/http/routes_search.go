package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/search"
	"github.com/ritme/backend-go/resources/translations"
)

// Global search (CB-NAV-01), Go only — no Laravel counterpart (deviations.md D-38). auth:api, localized by
// Accept-Language. User-scoped sources read only the requesting user's rows; the shop is not searched here.
func init() {
	Register("search", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := search.NewHandlers(search.NewService(search.NewSources(d.DB)), clock.Real{},
			i18n.NewTranslationStore(translations.FS, d.Config.StoragePath))

		r.Get("/api/v1/search", locale, guard, h.Search)
	})
}
