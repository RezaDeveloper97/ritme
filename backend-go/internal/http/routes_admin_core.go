package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/admins"
	adminauth "github.com/ritme/backend-go/internal/admin/auth"
	"github.com/ritme/backend-go/internal/admin/dashboard"
	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/overview"
	"github.com/ritme/backend-go/internal/admin/store"
	"github.com/ritme/backend-go/internal/admin/users"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/metrics"
	"github.com/ritme/backend-go/internal/okr"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/ratelimit"
)

// Admin API core (T-M2-20): auth, dashboard, users, admins under /api/admin/v1 — see
// docs/go-migration/admin-api.md. Every route carries its own httpadmin chain (host
// guard, CORS, default locale, session, CSRF, role), so other admin routes files
// (routes_admin_*.go) are protected the same way whatever order they mount in.
func init() {
	Register("admin_core", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		q := store.New(d.DB)
		var limiter *ratelimit.Limiter
		if d.Cache != nil {
			limiter = ratelimit.New(d.Cache, clock.Real{})
		}
		authH := adminauth.NewHandlers(kit, q, limiter)
		dash := dashboard.NewHandlers(q)
		usersH := users.NewHandlers(d.DB, d.Logger).WithStoragePath(d.Config.StoragePath) // B-N1-12: screenshots go with the user
		adminsH := admins.NewHandlers(d.DB, kit.Sessions(), d.Logger)

		p := httpadmin.Prefix
		get, post, put, del := fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodDelete
		h := func(method, path string, chain httpadmin.Chain) { httpadmin.Handle(r, method, p+path, chain) }

		// CORS preflights for the admin-web origin (the only OPTIONS route under the prefix).
		r.Options(p+"/*", kit.Preflight)

		h(post, "/auth/login", kit.Public(authH.Login))
		h(post, "/auth/logout", kit.Admin(authH.Logout))
		h(get, "/auth/me", kit.Admin(authH.Me))
		h(put, "/auth/password", kit.Admin(authH.ChangePassword))

		h(get, "/dashboard", kit.Admin(dash.Show))

		// B-N9-03: overview metrics snapshot (hourly aggregation job; capability «dashboard»).
		metricsSvc := metrics.New(d.DB, d.Logger)
		// B-N9-09: OKRs bound to the live metrics; the overview `okr` block is their summary.
		okrSvc := okr.New(d.DB, metricsSvc)
		okr.NewAdmin(okrSvc, d.Logger).Routes(okr.Route(h), kit)
		overview.New(metricsSvc, d.Logger).WithOKR(func(ctx context.Context, now time.Time) (any, error) {
			return okrSvc.Summary(ctx, now)
		}).Routes(overview.Route(h), kit)
		if d.DB != nil {
			mustJobs(metricsSvc.RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), tehranNow))
		}

		h(get, "/users", kit.Admin(usersH.List))
		h(get, "/users/directory", kit.Admin(usersH.Directory)) // B-N9-04
		h(get, "/users/directory/export", kit.Admin(usersH.ExportDirectory))
		h(get, "/users/:id", kit.Admin(usersH.Show))
		h(get, "/users/:id/overview", kit.Admin(usersH.Overview))
		h(put, "/users/:id", kit.Admin(usersH.Update))
		h(post, "/users/:id/block", kit.Admin(usersH.Block))
		h(post, "/users/:id/unblock", kit.Admin(usersH.Unblock))
		h(del, "/users/:id", kit.Admin(usersH.Destroy))

		h(get, "/admins", kit.Super(adminsH.List))
		h(post, "/admins", kit.Super(adminsH.Store))
		h(get, "/admins/:id", kit.Super(adminsH.Show))
		h(put, "/admins/:id", kit.Super(adminsH.Update))
		h(del, "/admins/:id", kit.Super(adminsH.Destroy))
	})
}
