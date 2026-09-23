package http

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	cyclecache "github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/periods"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Cycle engine + period log (backend/routes/api.php `cycle` prefix, CycleCalculationController and
// PeriodLogController), all auth:api. Static paths before {param} paths. /cycle/phase-content/{phase}
// belongs to the content domain (routes_content.go).
//
// CYCLE_ENGINE_CACHE=off computes every engine result directly (the contract suite runs with the
// cache on and off); anything else keeps the Redis engine cache on.
func init() {
	Register("cycle", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cacheOn := !strings.EqualFold(strings.TrimSpace(os.Getenv("CYCLE_ENGINE_CACHE")), "off")
		svc := cycleservice.New(d.DB, cyclecache.New(d.Cache, cacheOn, d.Logger))
		h := cycleservice.NewHandlers(svc, d.DB, clock.Real{})
		p := periods.NewHandlers(periods.NewService(d.DB), clock.Real{})

		c := r.Group("/api/v1/cycle")
		c.Get("/status", locale, guard, h.Status)
		c.Get("/today", locale, guard, h.Today)
		c.Get("/date/:date", locale, guard, h.ForDate)
		c.Get("/month/:year/:month", locale, guard, h.Month)
		c.Post("/recalculate", locale, guard, h.Recalculate)

		c.Get("/period/status", locale, guard, p.Status)
		c.Post("/period/start", locale, guard, p.Start)
		c.Post("/period/end", locale, guard, p.End)
		c.Get("/period/history", locale, guard, p.History)
		c.Post("/period", locale, guard, p.Store)
		c.Put("/period/:period", locale, guard, p.Update)
		c.Delete("/period/:period", locale, guard, p.Destroy)
	})
}
