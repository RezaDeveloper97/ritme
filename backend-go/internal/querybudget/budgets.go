// Package querybudget pins the number of DB queries each hot route makes (T-M9-08), auth and the module gate
// included, counted by the T-M9-07 per-request counter (metrics.QueryCounter) on the real router and MariaDB.
//
// The integration test (querybudget_int_test.go, `make test-int PKG=./internal/querybudget/...`) seeds one user
// per mode from internal/benchfixtures (24 periods, a year of logs with BBT and LH), calls every route below and
// fails when a count goes over its budget. The table started at the counts of T-M9-08 (the tests pass); a task
// that removes queries lowers its row in the same change, so a later regression is caught.
//
//   - Cold is the first request of a fresh process: empty Redis (engine cache, language registry, module
//     switches), new guards (OAuth client list not loaded) and the config caches off.
//   - Warm is the same request repeated with every cache filled.
//
// docs/engine-v6/perf-baseline.md shows the same table.
package querybudget

import (
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Budget is one route's query ceiling.
type Budget struct {
	// Name is the row label (also the subtest name).
	Name string
	// Route is the Fiber route template, as the metrics label it.
	Route string
	// Path is the request path (with query) the test sends.
	Path string
	// Mode is the fixture user that calls it.
	Mode enums.LifeMode
	// Cold and Warm are the ceilings (queries per request, auth included).
	Cold, Warm int64
	// Engine is the CYCLE_ENGINE the row runs with ("" = v1).
	Engine config.CycleEngine
}

// Budgets are the hot routes of docs/engine-v6/README.md §8, in display order.
var Budgets = []Budget{
	{"cycle today", "/api/v1/cycle/today", "/api/v1/cycle/today", enums.LifeModeCycle, 8, 4, ""},
	{"cycle date", "/api/v1/cycle/date/:date", "/api/v1/cycle/date/2026-10-25", enums.LifeModeCycle, 8, 4, ""},
	// T-M9-32 (README §8): the v6 path reads the engine state cache; warm = the auth query only. Cold here is a fresh
	// process with the config caches off: OAuth client list + token join + stored zone + languages + the 4 input
	// queries + recommendations; a new user on a warm process costs the auth join + the 4 input queries (5).
	{"cycle today v6", "/api/v1/cycle/today", "/api/v1/cycle/today", enums.LifeModeCycle, 9, 1, config.CycleEngineV6},
	{"cycle date v6", "/api/v1/cycle/date/:date", "/api/v1/cycle/date/2026-10-25", enums.LifeModeCycle, 9, 1, config.CycleEngineV6},
	{"cycle month calendar", "/api/v1/cycle/month/:year/:month", "/api/v1/cycle/month/2026/10?view=calendar", enums.LifeModeCycle, 6, 4, ""},
	{"cycle month full", "/api/v1/cycle/month/:year/:month", "/api/v1/cycle/month/2026/10", enums.LifeModeCycle, 8, 4, ""},
	{"home", "/api/v1/home", "/api/v1/home", enums.LifeModeCycle, 20, 11, ""},
	{"home cycle-overview", "/api/v1/home/cycle-overview", "/api/v1/home/cycle-overview", enums.LifeModeCycle, 9, 7, ""},
	// T-M9-34 (D-122): /home and the overview on the shared engine state (warm: no engine read, the message system's
	// life mode and 90-day logs from the home snapshot; cold adds the engine inputs). The overview's warm 5 = auth,
	// companion check, Plus banner and the 90-day logs (a 304 skips the logs). T-M9-34b: the challenge completions
	// (14-day window + today) and the doctor + medication reminders are one read each under v6.
	{"home v6", "/api/v1/home", "/api/v1/home", enums.LifeModeCycle, 21, 8, config.CycleEngineV6},
	{"home cycle-overview v6", "/api/v1/home/cycle-overview", "/api/v1/home/cycle-overview", enums.LifeModeCycle, 11, 5, config.CycleEngineV6},
	{"messages daily", "/api/v1/messages/daily", "/api/v1/messages/daily", enums.LifeModeCycle, 11, 8, ""},
	{"messages selected", "/api/v1/messages/selected", "/api/v1/messages/selected?placement=smart_tip", enums.LifeModeCycle, 10, 7, ""},
	{"fertility insights", "/api/v1/fertility/insights", "/api/v1/fertility/insights", enums.LifeModeTTC, 8, 5, ""},
	{"fertility bbt", "/api/v1/fertility/bbt", "/api/v1/fertility/bbt", enums.LifeModeTTC, 7, 4, ""},
	{"analysis summary", "/api/v1/analysis/summary", "/api/v1/analysis/summary?range=3m", enums.LifeModeCycle, 14, 11, ""},
	{"analysis monthly", "/api/v1/analysis/monthly/:ym", "/api/v1/analysis/monthly/2026-09", enums.LifeModeCycle, 17, 14, ""},
	{"analysis ttc", "/api/v1/analysis/ttc", "/api/v1/analysis/ttc", enums.LifeModeTTC, 15, 11, ""},
	{"analysis pregnancy", "/api/v1/analysis/pregnancy", "/api/v1/analysis/pregnancy", enums.LifeModePregnancy, 18, 15, ""},
}

// Auth is the auth step alone (passport Verify, T-M9-09): cold loads the OAuth client list once, warm is the one
// token-with-user JOIN. Every warm row above includes it.
var Auth = Budget{Name: "auth verify", Route: "(guard)", Cold: 2, Warm: 1}
