package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/babylog"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/children"
	childrenstore "github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/internal/postpartum"
	postpartumstore "github.com/ritme/backend-go/internal/postpartum/store"
	pregstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// The postpartum analysis hub (T-M10-27, D-176, internal/analysis/postpartum*.go), Go only: one auth:api read-only
// report for An_Hub_Post. The postpartum module switch is checked in the handler (403 module_disabled, as
// modules.Gate), the children module switch hides the child cards; Plus cards come back `locked: true` without data.
func init() {
	Register("analysis_postpartum", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		languages := i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger)
		cat := catalog.NewReader(catalogstore.New(d.DB), d.Cache, 0, d.Logger)
		kids := children.NewService(d.DB, cat)
		h := analysis.NewPostpartumHandlers(analysis.PostpartumDeps{
			Postpartum:      postpartum.NewService(d.DB),
			PostpartumStore: postpartumstore.New(d.DB),
			Pregnancy:       pregstore.New(d.DB),
			Logs:            healthlog.NewService(d.DB),
			Plus:            plus.NewService(d.DB, d.Config.Plus, nil, d.Logger), // entitlements only
			Modules:         modules.NewService(d.DB, d.Cache, d.Logger),
			ChildrenStore:   childrenstore.New(d.DB),
			Children:        kids,
			BabyLog:         babylog.NewService(d.DB, kids),
		}, clock.Real{})
		r.Get("/api/v1/analysis/postpartum", i18n.Middleware(languages), guard, h.Postpartum)
	})
}
