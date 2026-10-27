package service_test

// T-M9-33 (D-121): GET /cycle/month in each CYCLE_ENGINE mode over the contract fixture — Go-native goldens
// (contract/golden-v6/cycle-month, 6 personas × 3 months; rewrite with UPDATE_GOLDEN_V6=1 and review the diff by
// hand), cross-consumer parity with /cycle/date (8 personas × 120 days), the warm query budget and 304, v1 unchanged
// and the shadow month comparison.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	cyclecache "github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// newMonthEnv is newV6Env with GET /cycle/month mounted like routes_cycle.go (locale → auth guard → handler).
func newMonthEnv(t *testing.T, mode config.CycleEngine) *v6env {
	t.Helper()
	e := newV6Env(t, mode)
	keysPath, err := filepath.Abs(contractKeys)
	require.NoError(t, err)
	cfg := &config.Config{
		App:         config.App{Env: "testing", URL: "http://localhost"},
		Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
		StoragePath: keysPath,
	}
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	guard := auth.MustGuard(e.app, cfg, e.db, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(e.db), e.rc, quiet))
	h := service.NewHandlers(service.New(e.db, cyclecache.New(e.rc, true, quiet)), e.db, clock.Real{}).WithEngine(e.eng)
	e.app.Get("/api/v1/cycle/month/:year/:month", locale, guard, h.Month)
	return e
}

var monthGoldenPersonas = []struct {
	name string
	uid  uint64
}{
	{"regular", 1004}, {"irregular", 1005}, {"open_period_day3", 1007}, {"overdue_20", 1011},
	{"onboarding_declared", 1003}, {"pregnant_lmp_w8", 1014},
}

const goldenMonthDir = "../../../contract/golden-v6/cycle-month"

// Go-native goldens: the calendar view of Aug, Sep and Oct 2026 for six personas (today 2026-09-23); the full view
// of September for the regular persona.
func TestMonthV6_Goldens(t *testing.T) {
	e := newMonthEnv(t, config.CycleEngineV6)
	update := os.Getenv("UPDATE_GOLDEN_V6") != ""
	type golden struct{ name, path string }
	var cases []golden
	for _, p := range monthGoldenPersonas {
		for _, m := range []int{8, 9, 10} {
			cases = append(cases, golden{fmt.Sprintf("calendar.%s.2026-%02d", p.name, m),
				fmt.Sprintf("/api/v1/cycle/month/2026/%d?view=calendar", m)})
		}
	}
	cases = append(cases, golden{"full.regular.2026-09", "/api/v1/cycle/month/2026/9"})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var uid uint64
			for _, p := range monthGoldenPersonas {
				if bytes.Contains([]byte(c.name), []byte("."+p.name+".")) {
					uid = p.uid
				}
			}
			r := e.get(uid, c.path, nil)
			require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
			var pretty bytes.Buffer
			require.NoError(t, json.Indent(&pretty, r.Raw, "", "  "))
			pretty.WriteByte('\n')
			file := filepath.Join(goldenMonthDir, c.name+".json")
			if update {
				require.NoError(t, os.MkdirAll(goldenMonthDir, 0o755))
				require.NoError(t, os.WriteFile(file, pretty.Bytes(), 0o644)) //nolint:gosec // golden fixture
				return
			}
			want, err := os.ReadFile(file) //nolint:gosec // test fixture path
			require.NoError(t, err, "no golden (run with UPDATE_GOLDEN_V6=1)")
			assert.JSONEq(t, string(want), string(r.Raw))
		})
	}
}

type parityDay struct {
	Date         string `json:"calculation_date"`
	CycleDay     *int   `json:"cycle_day"`
	Fertile      bool   `json:"is_fertile_window"`
	MainPhase    string `json:"main_phase"`
	Fertility    string `json:"fertility_level"`
	DaysToPeriod *int   `json:"days_to_period"`
	DaysLate     *int   `json:"days_late"`
}

// Cross-consumer parity: for 8 personas × 120 days (2026-08-01 … 2026-11-28, today 2026-09-23) the month day equals
// the /cycle/date cycle_view of that date on phase, fertile flag, next start and lateness.
func TestMonthV6_ParityWithCycleView(t *testing.T) {
	e := newMonthEnv(t, config.CycleEngineV6)
	personas := []uint64{1004, 1005, 1007, 1011, 1003, 1014, 1012, 1006}
	first := civildate.MustParse("2026-08-01")
	for _, uid := range personas {
		month := map[string]parityDay{}
		for _, m := range []int{8, 9, 10, 11} {
			r := e.get(uid, fmt.Sprintf("/api/v1/cycle/month/2026/%d?view=calendar", m), nil)
			require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
			var days []parityDay
			require.NoError(t, json.Unmarshal(data(t, r.Raw)["calculations"], &days))
			for _, d := range days {
				month[d.Date] = d
			}
		}
		for i := range 120 {
			date := first.AddDays(i)
			md, ok := month[date.String()]
			require.True(t, ok, date.String())
			r := e.get(uid, "/api/v1/cycle/date/"+date.String(), nil)
			require.Equal(t, fiber.StatusOK, r.Status)
			var cv struct {
				CycleDay     *int   `json:"cycle_day"`
				MainPhase    string `json:"main_phase"`
				Fertility    string `json:"fertility_level"`
				DaysToPeriod *int   `json:"days_to_period"`
				DaysLate     *int   `json:"days_late"`
				Anchors      struct {
					Next *string `json:"predicted_next_period_start"`
				} `json:"anchors"`
				Limits struct {
					Suppressed bool `json:"fertility_suppressed"`
				} `json:"limits"`
			}
			require.NoError(t, json.Unmarshal(data(t, r.Raw)["cycle_view"], &cv))
			at := fmt.Sprintf("uid %d %s", uid, date)
			assert.Equal(t, cv.MainPhase, md.MainPhase, at)
			assert.Equal(t, cv.Fertility, md.Fertility, at)
			assert.Equal(t, cv.CycleDay, md.CycleDay, at)
			assert.Equal(t, cv.DaysLate, md.DaysLate, at)
			assert.Equal(t, cv.DaysToPeriod, md.DaysToPeriod, at)
			assert.Equal(t, cv.MainPhase == "fertile" && !cv.Limits.Suppressed, md.Fertile, at)
			if cv.Anchors.Next != nil && md.DaysLate != nil && *md.DaysLate == 0 {
				require.NotNil(t, md.DaysToPeriod, at)
				assert.Equal(t, *cv.Anchors.Next, date.AddDays(*md.DaysToPeriod).String(), "next start "+at)
			}
		}
	}
}

// Warm /cycle/month?view=calendar = 1 query (the auth join); the same If-None-Match → 304 with no body; another
// month has its own tag.
func TestMonthV6_WarmQueryAnd304(t *testing.T) {
	e := newMonthEnv(t, config.CycleEngineV6)
	const uid, path = 1004, "/api/v1/cycle/month/2026/9?view=calendar"
	cold := e.get(uid, path, nil)
	require.Equal(t, fiber.StatusOK, cold.Status, string(cold.Raw))
	require.NotEmpty(t, cold.ETag)
	assert.Equal(t, "private, no-cache", cold.Header["cache-control"])
	t.Logf("cold /cycle/month calendar: %d queries", cold.Queries)
	assert.LessOrEqual(t, cold.Queries, int64(9))

	warm := e.get(uid, path, nil)
	require.Equal(t, fiber.StatusOK, warm.Status)
	assert.Equal(t, int64(1), warm.Queries, "warm v6 month is the auth query only")
	assert.Equal(t, cold.ETag, warm.ETag)
	assert.Equal(t, string(cold.Raw), string(warm.Raw))

	nm := e.get(uid, path, map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusNotModified, nm.Status)
	assert.Empty(t, nm.Raw)
	assert.Equal(t, int64(1), nm.Queries)

	other := e.get(uid, "/api/v1/cycle/month/2026/10?view=calendar", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusOK, other.Status, "another month never revalidates with this tag")
	full := e.get(uid, "/api/v1/cycle/month/2026/9", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusOK, full.Status, "another view never revalidates with this tag")

	// Validation still answers before the engine.
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.get(uid, "/api/v1/cycle/month/2026/13", nil).Status)
	assert.Equal(t, fiber.StatusUnprocessableEntity, e.get(uid, "/api/v1/cycle/month/2026/9?view=x", nil).Status)
}

// Latency through the in-process router (budget: warm p95 ≤ 20 ms, cold ≤ 80 ms). Logged, not asserted.
func TestMonthV6_Latency(t *testing.T) {
	e := newMonthEnv(t, config.CycleEngineV6)
	const uid, n = 1004, 200
	path := "/api/v1/cycle/month/2026/9?view=calendar"
	require.Equal(t, fiber.StatusOK, e.get(uid, path, nil).Status)
	warm := make([]time.Duration, 0, n)
	for range n {
		start := time.Now()
		require.Equal(t, fiber.StatusOK, e.get(uid, path, nil).Status)
		warm = append(warm, time.Since(start))
	}
	cold := make([]time.Duration, 0, n/4)
	for range n / 4 {
		require.NoError(t, revision.Bump(context.Background(), e.db, uid))
		start := time.Now()
		r := e.get(uid, path, nil)
		cold = append(cold, time.Since(start))
		require.Equal(t, fiber.StatusOK, r.Status)
		assert.LessOrEqual(t, r.Queries, int64(5), "cold after a write: auth + the input queries")
	}
	slices.Sort(warm)
	slices.Sort(cold)
	t.Logf("v6 /cycle/month calendar: warm p95 %s (n=%d), cold-after-write p95 %s (n=%d)",
		warm[n*95/100], n, cold[len(cold)*95/100], n/4)
}

// v1 is the unchanged legacy month (no ETag); shadow serves the same bytes and classifies the month after it.
func TestMonth_V1UnchangedAndShadowClassifies(t *testing.T) {
	v1 := newMonthEnv(t, config.CycleEngineV1)
	sh := newMonthEnv(t, config.CycleEngineShadow)
	for _, path := range []string{"/api/v1/cycle/month/2026/9?view=calendar", "/api/v1/cycle/month/2026/10"} {
		a := v1.get(1011, path, nil)
		b := sh.get(1011, path, nil)
		require.Equal(t, fiber.StatusOK, b.Status)
		assert.Equal(t, string(a.Raw), string(b.Raw), "shadow serves v1 unchanged")
		assert.Empty(t, a.ETag)
		assert.Empty(t, b.ETag)
		service.WaitMonthShadow()
	}
	assert.Positive(t, counter(t, "engine_shadow_month_jobs_total", "compared"))
	assert.Positive(t, counter(t, "engine_shadow_month_diff_total", "lateness"), "overdue_20: the legacy day wraps")
}

// counter reads one labelled counter from the process registry.
func counter(t *testing.T, name, label string) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetValue() == label {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
