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
	{"cycle today", "/api/v1/cycle/today", "/api/v1/cycle/today", enums.LifeModeCycle, 7, 4, ""},
	{"cycle date", "/api/v1/cycle/date/:date", "/api/v1/cycle/date/2026-10-25", enums.LifeModeCycle, 7, 4, ""},
	// T-M9-32 (README §8): the v6 path reads the engine state cache; warm = the auth query only. Cold here is a fresh
	// process with the config caches off: OAuth client list + token join + stored zone + languages + the 4 input
	// queries + recommendations; a new user on a warm process costs the auth join + the 4 input queries (5).
	{"cycle today v6", "/api/v1/cycle/today", "/api/v1/cycle/today", enums.LifeModeCycle, 8, 1, config.CycleEngineV6},
	{"cycle date v6", "/api/v1/cycle/date/:date", "/api/v1/cycle/date/2026-10-25", enums.LifeModeCycle, 8, 1, config.CycleEngineV6},
	{"cycle month calendar", "/api/v1/cycle/month/:year/:month", "/api/v1/cycle/month/2026/10?view=calendar", enums.LifeModeCycle, 5, 4, ""},
	// T-M9-49: v6 rows for the month calendar, fertility and analysis summary; every row lowered to the measured count.
	// T-M9-49c: analysis reads the default language from the locale middleware (one cold query less).
	{"cycle month calendar v6", "/api/v1/cycle/month/:year/:month", "/api/v1/cycle/month/2026/10?view=calendar", enums.LifeModeCycle, 8, 1, config.CycleEngineV6},
	{"cycle month full", "/api/v1/cycle/month/:year/:month", "/api/v1/cycle/month/2026/10", enums.LifeModeCycle, 7, 4, ""},
	{"home", "/api/v1/home", "/api/v1/home", enums.LifeModeCycle, 20, 11, ""},
	{"home cycle-overview", "/api/v1/home/cycle-overview", "/api/v1/home/cycle-overview", enums.LifeModeCycle, 8, 7, ""},
	// T-M9-34 (D-122): /home and the overview on the shared engine state (warm: no engine read, the message system's
	// life mode and 90-day logs from the home snapshot; cold adds the engine inputs). The overview's warm 5 = auth,
	// companion check, Plus banner and the 90-day logs (a 304 skips the logs). T-M9-34b: the challenge completions
	// (14-day window + today) and the doctor + medication reminders are one read each under v6.
	{"home v6", "/api/v1/home", "/api/v1/home", enums.LifeModeCycle, 21, 8, config.CycleEngineV6},
	{"home cycle-overview v6", "/api/v1/home/cycle-overview", "/api/v1/home/cycle-overview", enums.LifeModeCycle, 10, 5, config.CycleEngineV6},
	// T-M9-49b: the companion check, profile, life mode and pregnancy-profile presence are one account row, and the
	// daily log comes off the 90-day recent logs (daily warm 8 → 4); the feed reads the condition programs and day
	// entries and the open inbox rows off the fact profile row, and the translations off the compiled snapshot.
	{"messages daily", "/api/v1/messages/daily", "/api/v1/messages/daily", enums.LifeModeCycle, 6, 4, ""},
	// T-M10-06: selected is the feed of one slot; it always reads the event history (one grouped query), +1 on v1.
	{"messages selected", "/api/v1/messages/selected", "/api/v1/messages/selected?placement=smart_tip", enums.LifeModeCycle, 7, 5, ""},
	// T-M9-38b: daily and selected on the shared engine state (T-M9-38, D-130). Warm = v1's (the state comes from the
	// cache: no profile/history read for the cycle); cold adds the 4 engine input queries. The daily's "has cycle data"
	// gate reads the same engine state (its anchor), so it costs no query of its own.
	{"messages daily v6", "/api/v1/messages/daily", "/api/v1/messages/daily", enums.LifeModeCycle, 10, 4, config.CycleEngineV6},
	// T-M10-06: the feed path folds the companion check into the fact profile row (warm 7 → 4).
	{"messages selected v6", "/api/v1/messages/selected", "/api/v1/messages/selected?placement=smart_tip", enums.LifeModeCycle, 10, 4, config.CycleEngineV6},
	// T-M10-06 (D-163): every home slot in one call. v6 warm = auth + the fact profile (account facts, companion check
	// and the inbox counts in one row) + the engine-state revisions + the event history (SPEC: ≤ 3 + auth).
	{"messages feed", "/api/v1/messages/feed", "/api/v1/messages/feed", enums.LifeModeCycle, 8, 6, ""},
	{"messages feed v6", "/api/v1/messages/feed", "/api/v1/messages/feed", enums.LifeModeCycle, 13, 4, config.CycleEngineV6}, // T-M10-10: +2 cold = challenge catalog snapshot load; warm unchanged
	{"fertility insights", "/api/v1/fertility/insights", "/api/v1/fertility/insights", enums.LifeModeTTC, 7, 5, ""},
	{"fertility bbt", "/api/v1/fertility/bbt", "/api/v1/fertility/bbt", enums.LifeModeTTC, 6, 4, ""},
	{"analysis summary", "/api/v1/analysis/summary", "/api/v1/analysis/summary?range=3m", enums.LifeModeCycle, 14, 6, ""},
	{"fertility insights v6", "/api/v1/fertility/insights", "/api/v1/fertility/insights", enums.LifeModeTTC, 8, 2, config.CycleEngineV6},
	{"fertility bbt v6", "/api/v1/fertility/bbt", "/api/v1/fertility/bbt", enums.LifeModeTTC, 7, 1, config.CycleEngineV6},
	{"analysis summary v6", "/api/v1/analysis/summary", "/api/v1/analysis/summary?range=3m", enums.LifeModeCycle, 18, 6, config.CycleEngineV6},
	{"analysis monthly", "/api/v1/analysis/monthly/:ym", "/api/v1/analysis/monthly/2026-09", enums.LifeModeCycle, 14, 6, ""},
	{"analysis ttc", "/api/v1/analysis/ttc", "/api/v1/analysis/ttc", enums.LifeModeTTC, 15, 6, ""},
	{"analysis pregnancy", "/api/v1/analysis/pregnancy", "/api/v1/analysis/pregnancy", enums.LifeModePregnancy, 18, 6, ""},
	// T-M12-30 (P-05): the GETs the web home fires on every open (docs/quality-2026-11/perf.md §web-home fan-out).
	// Measured on the fixture users, cold = fresh process, warm = caches filled; each row is the measured count.
	// pregnancy v2 today (T-M12-30b profile, TestQueryAttribution): warm 21 = the Today read 6 (auth, profile, week
	// details, week state, today extras, tip payloads) + the daily alert evaluation 15 (pregnancyalerts.EvaluateFirst:
	// profile ×2, symptoms, extras, weekly logs, week log, fetal, contractions, kick sessions ×2, today extras, last
	// weight, vitals, log measurements, open inbox). Cutting the evaluation needs a change in
	// internal/messages/pregnancyalerts (time-driven rules, H-05), outside T-M12-30b's scope — see the task report.
	{"care today", "/api/v1/care/today", "/api/v1/care/today", enums.LifeModeCycle, 5, 4, ""},
	{"checkups home", "/api/v1/checkups/home", "/api/v1/checkups/home", enums.LifeModeCycle, 5, 4, ""},
	// T-M12-30b: the board, its item counts and the reminder switch's notification categories are one read (ListBoard).
	// v1 warm 5 = auth + board + the suggestion's v1 cycle load (profile, histories, logs; internal/cycle/service).
	// v6 warm 3 = auth + board + the engine-state revisions (the suggestion reads the cached state). Row growth:
	// TestHomeReadsConstantInRows (1 vs 20 lists, care rows, checkups, notifications).
	{"todo", "/api/v1/todo", "/api/v1/todo", enums.LifeModeCycle, 7, 5, ""},
	{"todo v6", "/api/v1/todo", "/api/v1/todo", enums.LifeModeCycle, 9, 3, config.CycleEngineV6},
	{"home notifications", "/api/v1/home/notifications", "/api/v1/home/notifications?per_page=50", enums.LifeModeCycle, 4, 3, ""},
	{"modules", "/api/v1/modules", "/api/v1/modules", enums.LifeModeCycle, 1, 0, ""},
	{"profile", "/api/v1/profile", "/api/v1/profile", enums.LifeModeCycle, 4, 3, ""},
	{"fertility today", "/api/v1/fertility/today", "/api/v1/fertility/today", enums.LifeModeTTC, 6, 4, ""},
	{"banners ads", "/api/v1/banners/ads", "/api/v1/banners/ads?mode=cycle", enums.LifeModeCycle, 4, 3, ""},
	{"learning courses", "/api/v1/learning/courses", "/api/v1/learning/courses", enums.LifeModeCycle, 5, 3, ""},
	{"logs taxonomy", "/api/v1/logs/taxonomy", "/api/v1/logs/taxonomy", enums.LifeModeCycle, 6, 4, ""},
	{"logs day", "/api/v1/logs/days/:date", "/api/v1/logs/days/2026-10-20", enums.LifeModeCycle, 3, 2, ""},
	{"pregnancy v2 today", "/api/v1/pregnancy/v2/today", "/api/v1/pregnancy/v2/today", enums.LifeModePregnancy, 27, 21, ""},
	{"companion home", "/api/v1/companion/home", "/api/v1/companion/home", enums.LifeModeCompanion, 9, 7, ""},
	{"teen today", "/api/v1/teen/today", "/api/v1/teen/today", enums.LifeModeTeen, 12, 7, ""},
}

// Auth is the auth step alone (passport Verify, T-M9-09): cold loads the OAuth client list once, warm is the one
// token-with-user JOIN. Every warm row above includes it.
var Auth = Budget{Name: "auth verify", Route: "(guard)", Cold: 2, Warm: 1}
