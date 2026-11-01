package http

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/admin/safetyreview"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/safety"
)

// Safety & clinical content dashboard and review queue (bloom B-N9-10, D-90) under /api/admin/v1/safety —
// docs/go-migration/admin-api.md «Safety & clinical content», plus the nightly safety self-test (internal/safety,
// T-M10-24; the CI twin is .github/workflows/safety-scenarios-nightly.yml).
func init() {
	Register("admin_safety", func(r fiber.Router, d *Deps) {
		kit := httpadmin.Wire(d.Config.App.Env, d.DB, d.Cache, d.Logger)
		safetyreview.New(d.DB, d.Logger).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
		// The persisted self-test runs (T-M10-24) and the nightly job safety.nightly (any process with workers runs
		// it; the result is read back from safety_scenario_runs).
		safety.NewRunsHandlers(d.DB).Routes(func(method, path string, chain httpadmin.Chain) {
			httpadmin.Handle(r, method, httpadmin.Prefix+path, chain)
		}, kit)
		mustJobs(safety.RegisterJobs(auth.Jobs(r, d.Cache, d.Logger), safety.NewRunner(d.DB, d.Logger), tehranNow))
	})
}

// tehranNow is the job clock of the routes files (Asia/Tehran wall clock, like the DB).
func tehranNow() time.Time { return time.Now().In(civildate.Tehran) }

// mustJobs fails the mount when a domain's job registration is wrong (a bad cron spec is a programming error).
func mustJobs(err error) {
	if err != nil {
		panic(err)
	}
}
