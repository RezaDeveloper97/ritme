package service_test

// T-M9-32 (D-120, X-03/X-05): /cycle/today and /cycle/date in each CYCLE_ENGINE mode, over the contract fixture
// (personas 1001…1018, seeded recommendations) on a router built like cmd/api (query counter → test clock → usertz →
// locale → auth guard).

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	cyclecache "github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/shadow"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

const (
	fixtureNow   = "2026-09-23T10:00:00+03:30" // the contract clock (cmd/contract DefaultNow)
	queriesHead  = "X-Test-Queries"
	contractKeys = "../../../contract/fixtures/keys"
)

// v6env is one router over the contract fixture in one CYCLE_ENGINE mode.
type v6env struct {
	t      *testing.T
	db     *sql.DB
	app    *fiber.App
	iss    *passport.Issuer
	eng    *service.Engine
	rc     *cache.Client
	tokens map[uint64]string
}

func newV6Env(t *testing.T, mode config.CycleEngine) *v6env {
	t.Helper()
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "..", "contract", "fixtures", "dump.sql"))
	configcache.SetEnabled(true) // production: languages, recommendations from the process snapshots
	t.Cleanup(func() { configcache.SetEnabled(false) })
	keysPath, err := filepath.Abs(contractKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-t-m9-32:")
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))

	// Verify off: the serving path is measured (README §9 verify runs in the state package's own tests).
	states, err := state.New(store.New(db), rc, clock.Real{}, state.Options{Enabled: true, L1Entries: 100, Build: "test", Logger: quiet})
	require.NoError(t, err)
	eng, err := service.NewEngine(service.EngineDeps{Mode: mode, States: states, DB: db, Redis: rc, Build: "test"}, nil)
	require.NoError(t, err)

	cfg := &config.Config{
		App:         config.App{Env: "testing", URL: "http://localhost"},
		Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
		StoragePath: keysPath,
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(metrics.HTTP())
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		c.Set(queriesHead, strconv.FormatInt(metrics.RequestQueries(c), 10))
		return err
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	tz := usertz.New(profilestore.New(db), quiet)
	t.Cleanup(tz.Wait)
	app.Use(tz.Handler())
	guard := auth.MustGuard(app, cfg, db, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), rc, quiet))
	h := service.NewHandlers(service.New(db, cyclecache.New(rc, true, quiet)), db, clock.Real{}).WithEngine(eng)
	app.Get("/api/v1/cycle/today", locale, guard, h.Today)
	app.Get("/api/v1/cycle/date/:date", locale, guard, h.ForDate)

	return &v6env{t: t, db: db, app: app, eng: eng, rc: rc, tokens: map[uint64]string{},
		iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *v6env) token(uid uint64) string {
	e.t.Helper()
	if tok, ok := e.tokens[uid]; ok {
		return tok
	}
	tok, err := e.iss.Issue(context.Background(), uid, time.Now())
	require.NoError(e.t, err)
	e.tokens[uid] = tok.AccessToken
	return tok.AccessToken
}

type v6reply struct {
	Status  int
	Raw     []byte
	Queries int64
	ETag    string
	Header  map[string]string
}

func (e *v6env) get(uid uint64, path string, hdr map[string]string) v6reply {
	e.t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+e.token(uid))
	req.Header.Set(clock.Header, fixtureNow)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	n, _ := strconv.ParseInt(resp.Header.Get(queriesHead), 10, 64)
	return v6reply{Status: resp.StatusCode, Raw: raw, Queries: n, ETag: resp.Header.Get(fiber.HeaderETag),
		Header: map[string]string{"cache-control": resp.Header.Get(fiber.HeaderCacheControl), "vary": resp.Header.Get(fiber.HeaderVary)}}
}

// data decodes {"success":true,"data":{…}}.
func data(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var env struct {
		Success bool                       `json:"success"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &env), string(raw))
	require.True(t, env.Success, string(raw))
	return env.Data
}

// expected is the engine's own §2.35 + today_ui output for uid on target, computed directly (inputs.Load → Prepare →
// Resolve/FutureCycles/TodayUI → AppendJSON), independent of the service.
func expected(t *testing.T, db *sql.DB, uid uint64, target, today civildate.Date) map[string]any {
	t.Helper()
	in, err := inputs.Load(context.Background(), store.New(db), uid, today)
	require.NoError(t, err)
	st := engine.Prepare(in, engineparams.For(today))
	s := st.Resolve(engine.DayOf(target), engine.DayOf(today))
	s.FutureCycles = st.FutureCycles(engine.DayOf(today))
	ui := st.TodayUI(s)
	var m map[string]any
	require.NoError(t, json.Unmarshal(engine.AppendJSON(nil, &s, &ui), &m))
	return m
}

var v6Personas = []struct {
	name string
	uid  uint64
}{
	{"regular", 1004}, {"irregular", 1005}, {"open_period_day3", 1007}, {"overdue_20", 1011},
	{"onboarding_declared", 1003}, {"pregnant_lmp_w8", 1014}, {"no_profile", 1001},
}

// The v6 cycle_view carries exactly the engine's output (every §2.35 key, limits, future_k, today_ui) plus the legacy
// keys; /cycle/date resolves the target with today's future_cycles.
func TestV6_PayloadEqualsEngine(t *testing.T) {
	e := newV6Env(t, config.CycleEngineV6)
	today := civildate.MustParse("2026-09-23")
	for _, p := range v6Personas {
		for _, target := range []string{"", "2026-10-12", "2026-09-14"} {
			t.Run(p.name+"/"+target, func(t *testing.T) {
				path, tgt := "/api/v1/cycle/today", today
				if target != "" {
					path, tgt = "/api/v1/cycle/date/"+target, civildate.MustParse(target)
				}
				r := e.get(p.uid, path, nil)
				require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
				d := data(t, r.Raw)
				var cv map[string]any
				require.NoError(t, json.Unmarshal(d["cycle_view"], &cv))
				want := expected(t, e.db, p.uid, tgt, today)
				for k, v := range want {
					assert.Equal(t, v, cv[k], "cycle_view.%s", k)
				}
				for _, k := range []string{"phase", "data_status", "predictions", "profile_values", "calculated_values",
					"effective_values", "data_quality_details", "daily_card"} {
					assert.Contains(t, cv, k, "legacy key %s (X-05)", k)
				}
				assert.JSONEq(t, `"completed"`, string(d["calculation_status"]))
				assert.JSONEq(t, `false`, string(d["is_recalculating"]))
				assert.Contains(t, d, "calculation")
			})
		}
	}
}

// Warm v6 /cycle/today = 1 query (the auth join); a matching If-None-Match → 304 with no body (1 query); a write bumps
// engine_rev → a new ETag and a fresh body.
func TestV6_WarmQueryETagAnd304(t *testing.T) {
	e := newV6Env(t, config.CycleEngineV6)
	const uid = 1004
	cold := e.get(uid, "/api/v1/cycle/today", nil)
	require.Equal(t, fiber.StatusOK, cold.Status)
	require.NotEmpty(t, cold.ETag)
	assert.True(t, strings.HasPrefix(cold.ETag, `W/"u`+strconv.FormatUint(uid, 36)+".e"), cold.ETag)
	assert.Equal(t, "private, no-cache", cold.Header["cache-control"])
	assert.Contains(t, cold.Header["vary"], "Authorization")
	t.Logf("cold /cycle/today: %d queries", cold.Queries)
	assert.LessOrEqual(t, cold.Queries, int64(9), "cold process (querybudget row)")

	warm := e.get(uid, "/api/v1/cycle/today", nil)
	require.Equal(t, fiber.StatusOK, warm.Status)
	assert.Equal(t, int64(1), warm.Queries, "warm v6 /cycle/today is the auth query only")
	assert.Equal(t, cold.ETag, warm.ETag)
	assert.Equal(t, string(cold.Raw), string(warm.Raw))

	date := e.get(uid, "/api/v1/cycle/date/2026-10-01", nil)
	require.Equal(t, fiber.StatusOK, date.Status)
	assert.Equal(t, int64(1), date.Queries, "warm v6 /cycle/date shares the state")

	nm := e.get(uid, "/api/v1/cycle/today", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusNotModified, nm.Status)
	assert.Empty(t, nm.Raw)
	assert.Equal(t, int64(1), nm.Queries, "304 costs the auth query only")
	assert.Equal(t, warm.ETag, nm.ETag)

	// A user's first request on a warm process (config snapshots loaded, X-Timezone sent): the auth join + the 4
	// input queries (README §8 cold ≤ 5). Her tag never revalidates another user's body.
	other := e.get(1005, "/api/v1/cycle/today", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag, usertz.Header: "Asia/Tehran"})
	assert.Equal(t, fiber.StatusOK, other.Status, "another user's tag never revalidates")
	t.Logf("cold user, warm process: %d queries", other.Queries)
	assert.LessOrEqual(t, other.Queries, int64(5))

	require.NoError(t, revision.Bump(context.Background(), e.db, uid))
	fresh := e.get(uid, "/api/v1/cycle/today", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusOK, fresh.Status, "a write bumps engine_rev → the old tag is stale")
	assert.NotEqual(t, warm.ETag, fresh.ETag)
}

// Latency of the v6 day handler through the in-process router (README §8: warm p95 ≤ 15 ms, cold ≤ 60 ms). Logged, not
// asserted: the numbers depend on the machine's load; cold = the first request after a write (engine_rev bumped → the
// state is rebuilt from the 4 input queries).
func TestV6_Latency(t *testing.T) {
	e := newV6Env(t, config.CycleEngineV6)
	const uid, n = 1004, 200
	require.Equal(t, fiber.StatusOK, e.get(uid, "/api/v1/cycle/today", nil).Status)
	p95 := func(d []time.Duration) time.Duration {
		slices.Sort(d)
		return d[len(d)*95/100]
	}
	warm := make([]time.Duration, 0, n)
	for range n {
		start := time.Now()
		r := e.get(uid, "/api/v1/cycle/today", nil)
		warm = append(warm, time.Since(start))
		require.Equal(t, fiber.StatusOK, r.Status)
	}
	cold := make([]time.Duration, 0, n/4)
	for range n / 4 {
		require.NoError(t, revision.Bump(context.Background(), e.db, uid))
		start := time.Now()
		r := e.get(uid, "/api/v1/cycle/today", nil)
		cold = append(cold, time.Since(start))
		require.Equal(t, fiber.StatusOK, r.Status)
	}
	t.Logf("v6 /cycle/today latency: warm p95 %s (n=%d), cold-after-write p95 %s (n=%d)", p95(warm), n, p95(cold), n/4)
}

// v1 (and a nil engine) is the unchanged v1.1 payload; shadow serves the same bytes and records a classified diff
// after the response: lateness for a late user, o_plus_one on the day after ovulation.
func TestShadow_ServesV1AndClassifies(t *testing.T) {
	v1 := newV6Env(t, config.CycleEngineV1)
	assert.Nil(t, v1.eng, "v1 builds no engine")
	sh := newV6Env(t, config.CycleEngineShadow)
	require.NotNil(t, sh.eng.Shadow())

	const late = 1011 // overdue_20
	a := v1.get(late, "/api/v1/cycle/today", nil)
	b := sh.get(late, "/api/v1/cycle/today", nil)
	require.Equal(t, fiber.StatusOK, b.Status)
	assert.Equal(t, string(a.Raw), string(b.Raw), "shadow serves v1 unchanged")
	assert.Empty(t, b.ETag, "no ETag outside v6")
	sh.eng.Shadow().Wait()
	assert.Contains(t, sh.classes(t), shadow.ClassLateness)

	// O+1 of the regular persona's current cycle (v1.1 ovulation date + 1).
	var cv struct {
		Anchors struct {
			Ovulation string `json:"estimated_ovulation_date"`
		} `json:"anchors"`
	}
	require.NoError(t, json.Unmarshal(data(t, sh.get(1004, "/api/v1/cycle/today", nil).Raw)["cycle_view"], &cv))
	sh.eng.Shadow().Wait()
	o1 := civildate.MustParse(cv.Anchors.Ovulation).AddDays(1)
	r := sh.get(1004, "/api/v1/cycle/date/"+o1.String(), nil)
	require.Equal(t, fiber.StatusOK, r.Status)
	sh.eng.Shadow().Wait()
	assert.Contains(t, sh.classes(t), shadow.ClassOPlusOne, "O+1 = %s", o1)

	// Samples never carry the raw user id or absolute dates.
	raw, err := sh.rc.Redis().LRange(context.Background(), sh.rc.Key(shadow.SamplesKey), 0, -1).Result()
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	for _, s := range raw {
		assert.NotContains(t, s, "1011")
		assert.NotContains(t, s, "1004")
		assert.NotContains(t, s, "2026-09-", "no fixture dates")
	}
	var sb strings.Builder
	require.NoError(t, shadow.Report(context.Background(), sh.rc, &sb, 5))
	assert.Contains(t, sb.String(), "lateness")
	t.Log(sb.String())
}

// classes are the classes of every stored sample.
func (e *v6env) classes(t *testing.T) []shadow.Class {
	t.Helper()
	raw, err := e.rc.Redis().LRange(context.Background(), e.rc.Key(shadow.SamplesKey), 0, -1).Result()
	require.NoError(t, err)
	var out []shadow.Class
	for _, r := range raw {
		var s shadow.Sample
		require.NoError(t, json.Unmarshal([]byte(r), &s))
		out = append(out, s.Classes...)
	}
	return out
}

// X-Timezone: Europe/Berlin at 21:30Z is still the 23rd there while Tehran is already on the 24th; the engine path
// uses the user's day.
func TestV6_TimezoneHeaderPicksTheLocalDay(t *testing.T) {
	e := newV6Env(t, config.CycleEngineV6)
	day := func(hdr map[string]string) string {
		r := e.get(1004, "/api/v1/cycle/today", hdr)
		require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
		var cv struct {
			Date string `json:"date"`
		}
		require.NoError(t, json.Unmarshal(data(t, r.Raw)["cycle_view"], &cv))
		return cv.Date
	}
	at := "2026-09-23T21:30:00Z"
	assert.Equal(t, "2026-09-23", day(map[string]string{clock.Header: at, usertz.Header: "Europe/Berlin"}))
	assert.Equal(t, "2026-09-24", day(map[string]string{clock.Header: at, usertz.Header: "Asia/Tehran"}))
}

// The pill persona (regular + an active pill method, I-03): the frontend's current zod schema (frontend/src/entities/
// cycle/api/schema.ts, copied as testdata/frontend_cycle_schema.json: the paths it parses without .nullable()) must
// see every field it needs; the paths that are null in v6 until T-M9-40 are pinned here and listed in PROGRESS.md.
func TestV6_PillPersonaAgainstFrontendSchema(t *testing.T) {
	e := newV6Env(t, config.CycleEngineV6)
	const uid = 1004
	// Saved before the fixture clock (2026-09-23): E-13 applies only from the method's start (T-M9-48d), and NOW()
	// would be after the contract day.
	_, err := e.db.Exec(`INSERT INTO contraception_methods (user_id, method, created_at, updated_at) VALUES (?, 'combined_pill', '2026-01-01 10:00:00', '2026-01-01 10:00:00')`, uid)
	require.NoError(t, err)
	require.NoError(t, revision.Bump(context.Background(), e.db, uid))

	r := e.get(uid, "/api/v1/cycle/today", nil)
	require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
	var body map[string]any
	require.NoError(t, json.Unmarshal(r.Raw, &body))
	cv := body["data"].(map[string]any)["cycle_view"].(map[string]any)
	limits := cv["limits"].(map[string]any)
	assert.Equal(t, true, limits["fertility_suppressed"])
	assert.Equal(t, "hormonal_contraception", limits["suppression_reason"])

	raw, err := os.ReadFile(filepath.Join("testdata", "frontend_cycle_schema.json"))
	require.NoError(t, err)
	var schema struct {
		NonNull []string `json:"non_null"`
	}
	require.NoError(t, json.Unmarshal(raw, &schema))
	var nulls []string
	for _, path := range schema.NonNull {
		v, present := lookup(body, path)
		require.True(t, present, "%s missing", path)
		if v == nil {
			nulls = append(nulls, path)
		}
	}
	sort.Strings(nulls)
	t.Logf("non-nullable in the current frontend schema, null in v6 for the pill persona: %v", nulls)
	// final_probability: v6 has no pregnancy probability; phase: hormonal contraception has no natural phase (unknown,
	// I-11 → no content phase).
	assert.Equal(t, []string{"data.calculation.final_probability", "data.calculation.phase"}, nulls,
		"T-M9-40 makes these nullable; until then they are the known gaps")

	// Every null value of the whole payload, for PROGRESS.md (legacy fields without a v6 value).
	var all []string
	walkNulls("", body, &all)
	sort.Strings(all)
	t.Logf("null fields (pill persona, today): %s", strings.Join(all, ", "))
	assert.Subset(t, all, []string{"data.calculation.final_probability", "data.calculation.text_flags",
		"data.cycle_view.anchors.estimated_ovulation_date", "data.cycle_view.predictions.fertile_window_start"})
}

func lookup(v any, path string) (any, bool) {
	for _, part := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		if v, ok = m[part]; !ok {
			return nil, false
		}
	}
	return v, true
}

func walkNulls(prefix string, v any, out *[]string) {
	switch x := v.(type) {
	case nil:
		*out = append(*out, prefix)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			walkNulls(p, x[k], out)
		}
	case []any:
		for i, it := range x {
			walkNulls(prefix+"["+strconv.Itoa(i)+"]", it, out)
		}
	}
}
