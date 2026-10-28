package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Family v6 (engine v6 M11, T-M11-05; D-193, D-204), Go only: GET /api/v1/family/me lists the caller's links as owner
// (grants with their state) and as member (effective domains only). Every member read and write of the family
// feature is decided by family.Decide; /companions/* stays the compatibility layer (routes_companion.go).
func init() {
	Register("family", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := family.NewHandlers(d.DB, clock.Real{})
		r.Get("/api/v1/family/me", locale, guard, h.Me)
	})
}
