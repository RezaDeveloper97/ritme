package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
)

// Content catalog (CB-CORE-03, docs/canvas-build/catalog.md), Go only — no Laravel counterpart.
// auth:api, localized by Accept-Language / ?locale=, cached per group in Redis.
func init() {
	Register("catalog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := catalog.NewHandlers(catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger))

		r.Get("/api/v1/catalog/:group", locale, guard, h.Group)
	})
}
