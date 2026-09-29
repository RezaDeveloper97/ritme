package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/checkups"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Periodic checkups (docs/checkups/README.md), Go only — no Laravel counterpart. All auth:api,
// localized by Accept-Language. Static paths before {id}.
func init() {
	Register("checkups", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := checkups.NewHandlers(d.DB, clock.Real{})
		writes := writeThrottle(d) // per-user write limit (security audit M3-M7 #5)

		r.Get("/api/v1/checkups", locale, guard, h.List)
		r.Get("/api/v1/checkups/home", locale, guard, h.Home)
		r.Get("/api/v1/checkups/preview-next", locale, guard, h.PreviewNext)
		r.Get("/api/v1/checkups/records", locale, guard, h.ListRecords)
		r.Put("/api/v1/checkups/records/:recordId", locale, guard, writes, h.UpdateRecord)
		r.Delete("/api/v1/checkups/records/:recordId", locale, guard, writes, h.DestroyRecord)
		r.Post("/api/v1/checkups/custom", locale, guard, writes, h.StoreCustom)
		r.Put("/api/v1/checkups/custom/:id", locale, guard, writes, h.UpdateCustom)
		r.Delete("/api/v1/checkups/custom/:id", locale, guard, writes, h.DestroyCustom)
		r.Get("/api/v1/checkups/:id", locale, guard, h.Show)
		r.Post("/api/v1/checkups/:id/records", locale, guard, writes, h.StoreRecord)
		r.Put("/api/v1/checkups/:id/settings", locale, guard, writes, h.UpdateSettings)
	})
}
