package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/cycle/snapshots"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/idempotency"
)

// Daily health logs (backend/routes/api.php, DailyHealthLogController), all auth:api.
// Static /enums before the {date} param routes.
func init() {
	Register("healthlog", func(r fiber.Router, d *Deps) {
		guard := auth.MustGuard(r, d.Config, d.DB, d.Logger).RequireUser
		locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(d.DB), d.Cache, d.Logger))
		svc := healthlog.NewService(d.DB).WithEngine(engineStates(r, d))
		if d.DB != nil {
			// Prediction log (T-M9-39b, D-137): a bleeding log that starts a new real period queues the same job as the
			// period button (the handler is registered by routes_cycle.go on the shared job queue).
			svc.WithStartHook(snapshots.Hook(auth.Jobs(r, d.Cache, d.Logger), d.Logger))
		}
		h := healthlog.NewHandlers(svc, clock.Real{})
		// Optional Idempotency-Key on the save (D-131).
		idem := idempotency.New(d.Cache, auth.CurrentUserID, d.Logger).Handler()

		r.Get("/api/v1/health-logs/enums", locale, guard, h.Enums)
		r.Get("/api/v1/health-logs", locale, guard, h.Index)
		r.Post("/api/v1/health-logs", locale, guard, idem, h.Store)
		r.Get("/api/v1/health-logs/:date", locale, guard, h.Show)
		r.Delete("/api/v1/health-logs/:date", locale, guard, h.Destroy)
	})
}
