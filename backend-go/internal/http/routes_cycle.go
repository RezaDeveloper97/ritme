package http

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	cyclecache "github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/periods"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/skips"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/idempotency"
)

// Cycle engine + period log (backend/routes/api.php `cycle` prefix, CycleCalculationController and
// PeriodLogController), all auth:api. Static paths before {param} paths. /cycle/phase-content/{phase}
// belongs to the content domain (routes_content.go).
//
// CYCLE_ENGINE_CACHE=off computes every engine result directly (the contract suite runs with the
// cache on and off); anything else keeps the Redis engine cache on.
//
// CYCLE_ENGINE (config.CycleEngine, T-M9-32, D-120): v1 serves the v1.1 payload; shadow serves it and compares v6
// after the response; v6 serves /cycle/today and /cycle/date from the v6 engine state cache with ETag/304.
func init() {
	Register("cycle", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		cacheOn := !strings.EqualFold(strings.TrimSpace(os.Getenv("CYCLE_ENGINE_CACHE")), "off")
		svc := cycleservice.New(d.DB, cyclecache.New(d.Cache, cacheOn, d.Logger))
		h := cycleservice.NewHandlers(svc, d.DB, clock.Real{}).WithEngine(cycleEngine(r, d))
		p := periods.NewHandlers(periods.NewService(d.DB), clock.Real{})
		// Optional Idempotency-Key on the period writes (D-131); the service also locks per user.
		idem := idempotency.New(d.Cache, auth.CurrentUserID, d.Logger).Handler()
		// «پریود نداشتم» (T-M9-29, E-15): Go-only (D-125), per-user write throttle like the other Go-only writes.
		sk := skips.NewHandlers(skips.NewService(d.DB), clock.Real{})
		writes := writeThrottle(d)

		c := r.Group("/api/v1/cycle")
		c.Get("/status", locale, guard, h.Status)
		c.Get("/today", locale, guard, h.Today)
		c.Get("/date/:date", locale, guard, h.ForDate)
		c.Get("/month/:year/:month", locale, guard, h.Month)
		c.Post("/recalculate", locale, guard, h.Recalculate)

		c.Get("/period/status", locale, guard, p.Status)
		c.Post("/period/start", locale, guard, idem, p.Start)
		c.Post("/period/end", locale, guard, idem, p.End)
		c.Get("/period/history", locale, guard, p.History)
		c.Post("/period", locale, guard, idem, p.Store)
		c.Put("/period/:period", locale, guard, p.Update)
		c.Delete("/period/:period", locale, guard, p.Destroy)

		c.Post("/skips", locale, guard, writes, sk.Store)
		c.Delete("/skips/:date", locale, guard, writes, sk.Destroy)
	})
}

// cycleEngine wires the CYCLE_ENGINE rollout (nil for v1).
func cycleEngine(r fiber.Router, d *Deps) *cycleservice.Engine {
	if d.Config == nil || d.Config.CycleEngine == "" || d.Config.CycleEngine == config.CycleEngineV1 {
		return nil
	}
	var kv state.KV
	if d.Cache != nil {
		kv = d.Cache
	}
	states, err := state.New(cyclestore.New(d.DB), kv, clock.Real{}, state.OptionsFrom(d.Config.EngineCache, etag.Build(), d.Logger))
	if err == nil {
		var eng *cycleservice.Engine
		if eng, err = cycleservice.NewEngine(cycleservice.EngineDeps{
			Mode: d.Config.CycleEngine, States: states, DB: d.DB, Redis: d.Cache, Build: etag.Build(),
		}, nil); err == nil {
			return eng
		}
	}
	auth.FailStartup(r, d.Logger, err)
	return nil
}
