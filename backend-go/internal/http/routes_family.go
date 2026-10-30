package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/family"
	"github.com/ritme/backend-go/internal/family/projections"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Family v6 (engine v6 M11, T-M11-05 / T-M11-06; D-193, D-194, D-204), Go only: GET /api/v1/family/me lists the
// caller's links as owner (grants with their state) and as member (effective domains only); GET
// /api/v1/family/owners/{ownerId}/domains/{domain} is the member projection of one domain (allow-list DTOs,
// internal/family/projections). Every member read and write of the family feature is decided by family.Decide;
// /companions/* stays the compatibility layer (routes_companion.go). Every /family/* answer is private, no-store.
func init() {
	Register("family", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		h := family.NewHandlers(d.DB, clock.Real{})
		p := projections.NewHandlers(d.DB, projections.New(d.DB).WithEngine(engineStates(r, d)),
			projections.NewAuditor(d.DB, family.NewReadAudit(d.Cache)), clock.Real{})
		r.Get("/api/v1/family/me", projections.NoStore, locale, guard, h.Me)
		r.Get("/api/v1/family/owners/:ownerId/domains/:domain", projections.NoStore, locale, guard, companionReadThrottle(d), p.Domain)
	})
}
