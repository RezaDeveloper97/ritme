package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/teen"
)

// Teen mode (CB-TEEN-01), Go only — no Laravel counterpart (deviations.md D-53). All auth:api, localized by
// Accept-Language, writes per-user throttled. Minors' health data: user-scoped only; the parent's only read is
// GET /teen/linked (active parent links, granted teen sections only, audited). Content (signs, readiness, FAQ, kit
// items) comes from the catalog (cached reader). Invites / grants / revoke of the parent link are bloom's
// /companions routes with type `parent` (routes_companion.go).
func init() {
	Register("teen", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		h := teen.NewHandlers(teen.NewService(d.DB, cat), clock.Real{})
		writes := writeThrottle(d)

		r.Get("/api/v1/teen/profile", locale, guard, h.ShowProfile)
		r.Put("/api/v1/teen/profile", locale, guard, writes, h.SaveProfile)
		r.Get("/api/v1/teen/today", locale, guard, h.Today)
		r.Put("/api/v1/teen/kit/:code", locale, guard, writes, h.UpdateKitItem)
		r.Put("/api/v1/teen/parent-note", locale, guard, writes, h.UpdateParentNote)
		r.Get("/api/v1/teen/linked", locale, guard, h.Linked)
	})
}
