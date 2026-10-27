package fertility_test

// T-M9-35 (D-128): /fertility under CYCLE_ENGINE=v6 over the contract fixture plus one BBT/LH persona — Go-native
// goldens (internal/fertility/testdata/golden-v6; rewrite with UPDATE_GOLDEN_V6=1 and review the diff by hand), the
// insights query budget (≤ 4) and the ETag / 304 of today / bbt / insights.
// Integration: `make test-int PKG=./internal/fertility/...`.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/usertz"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

const (
	v6QueriesHead = "X-Test-Queries"
	v6GoldenDir   = "testdata/golden-v6"
)

type v6Env struct {
	t      *testing.T
	app    *fiber.App
	iss    *passport.Issuer
	tokens map[uint64]string
	db     *sql.DB
}

func (e *v6Env) exec(q string, args ...any) {
	e.t.Helper()
	_, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
}

// newV6Env mounts /fertility like routes_fertility.go with the v6 engine state source: metrics counter → request
// clock → user zone → locale → auth guard (revisions in the auth query) → handler.
func newV6Env(t *testing.T) *v6Env {
	t.Helper()
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	keysPath, err := filepath.Abs("../../contract/fixtures/keys")
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-t-m9-35:")
	cq := cyclestore.New(db)
	p, err := state.New(cq, rc, clock.Real{}, state.Options{Enabled: true, L1Entries: 100, Build: "test", Logger: quiet})
	require.NoError(t, err)
	h := fertility.NewHandlers(db, clock.Real{}).WithEngine(state.Source{P: p, Revs: cq})

	cfg := &config.Config{
		App:         config.App{Env: "testing", URL: "http://localhost"},
		Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
		StoragePath: keysPath,
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(metrics.HTTP())
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		c.Set(v6QueriesHead, strconv.FormatInt(metrics.RequestQueries(c), 10))
		return err
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	tz := usertz.New(profilestore.New(db), quiet)
	t.Cleanup(tz.Wait)
	app.Use(tz.Handler())
	guard := auth.MustGuard(app, cfg, db, slog.New(slog.NewTextHandler(io.Discard, nil))).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), rc, quiet))
	app.Get("/api/v1/fertility/today", locale, guard, h.Today)
	app.Get("/api/v1/fertility/bbt", locale, guard, h.BBT)
	app.Get("/api/v1/fertility/insights", locale, guard, h.Insights)
	app.Get("/api/v1/fertility/days/:date", locale, guard, h.ShowDay)
	app.Put("/api/v1/fertility/days/:date", locale, guard, h.UpdateDay)
	return &v6Env{t: t, app: app, tokens: map[uint64]string{},
		iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365), db: db}
}

type v6Reply struct {
	Status  int
	Raw     []byte
	Queries int64
	ETag    string
}

func (e *v6Env) req(method string, uid uint64, path, lang, body string, hdr map[string]string) v6Reply {
	e.t.Helper()
	tok, ok := e.tokens[uid]
	if !ok {
		issued, err := e.iss.Issue(context.Background(), uid, time.Now())
		require.NoError(e.t, err)
		tok = issued.AccessToken
		e.tokens[uid] = tok
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", lang)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set(clock.Header, now)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	n, _ := strconv.ParseInt(resp.Header.Get(v6QueriesHead), 10, 64)
	return v6Reply{Status: resp.StatusCode, Raw: raw, Queries: n, ETag: resp.Header.Get(fiber.HeaderETag)}
}

func (e *v6Env) get(uid uint64, path, lang string, hdr map[string]string) v6Reply {
	return e.req(fiber.MethodGet, uid, path, lang, "", hdr)
}

// ttcPersona seeds a TTC user with three logged periods (07-16, 08-13, 09-10), biphasic BBT in the two finished
// cycles (shift on day 15 and 16), 13 flat readings in the current cycle, an LH positive on 08-01 (conflicting with
// the first cycle's BBT ovulation: BBT wins) and one on 09-22.
func (e *v6Env) ttcPersona(mobile string) uint64 {
	e.t.Helper()
	e.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('TTC v6', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	var id uint64
	require.NoError(e.t, e.db.QueryRow(`SELECT id FROM users WHERE mobile = ?`, mobile).Scan(&id))
	e.exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, last_period_start, pregnancy_intention, created_at, updated_at)
		VALUES (?, 5, 28, '2026-09-10', 'trying', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id)
	for _, s := range []string{"2026-07-16", "2026-08-13", "2026-09-10"} {
		end := civildate.MustParse(s).AddDays(4).String()
		e.exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source, created_at, updated_at)
			VALUES (?, ?, ?, 5, 1, 0, 'user_logged', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, s, end)
	}
	temps := func(from string, vs ...int) {
		d := civildate.MustParse(from)
		for i, v := range vs {
			e.exec(`INSERT INTO daily_health_logs (user_id, log_date, basal_body_temperature, created_at, updated_at)
				VALUES (?, ?, ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, d.AddDays(i).String(), fmt.Sprintf("%d.%02d", v/100, v%100))
		}
	}
	temps("2026-07-16", biphasic(15)...)
	temps("2026-08-13", biphasic(16)...)
	temps("2026-09-10", 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3645, 3630, 3625, 3635, 3630)
	for _, d := range []string{"2026-07-20", "2026-09-22"} {
		e.exec(`INSERT INTO fertility_logs (user_id, log_date, lh_test, created_at, updated_at)
			VALUES (?, ?, 'positive', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, d)
	}
	return id
}

var fertilityPersonas = []struct {
	name string
	uid  uint64
}{
	{"no_profile", 1001}, {"onboarding_declared", 1003}, {"regular", 1004}, {"irregular", 1005},
	{"short_outlier", 1006}, {"open_period_day3", 1007}, {"overdue_20", 1011}, {"ttc", 1012}, {"pregnant_lmp_w8", 1014},
}

func TestFertilityV6_Goldens(t *testing.T) {
	e := newV6Env(t)
	personas := slices.Clone(fertilityPersonas)
	personas = append(personas, struct {
		name string
		uid  uint64
	}{"ttc_bbt_lh", e.ttcPersona("09120035001")})
	update := os.Getenv("UPDATE_GOLDEN_V6") != ""
	for _, p := range personas {
		for _, c := range []struct{ name, path, lang string }{
			{"today." + p.name + ".fa", "/api/v1/fertility/today", "fa"},
			{"bbt3." + p.name + ".en", "/api/v1/fertility/bbt?range=3", "en"},
			{"insights." + p.name + ".fa", "/api/v1/fertility/insights", "fa"},
		} {
			t.Run(c.name, func(t *testing.T) {
				r := e.get(p.uid, c.path, c.lang, nil)
				require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
				var pretty bytes.Buffer
				require.NoError(t, json.Indent(&pretty, r.Raw, "", "  "))
				pretty.WriteByte('\n')
				file := filepath.Join(v6GoldenDir, c.name+".json")
				if update {
					require.NoError(t, os.MkdirAll(v6GoldenDir, 0o755))
					require.NoError(t, os.WriteFile(file, pretty.Bytes(), 0o644)) //nolint:gosec // golden fixture
					return
				}
				want, err := os.ReadFile(file) //nolint:gosec // test fixture path
				require.NoError(t, err, "no golden (run with UPDATE_GOLDEN_V6=1)")
				assert.JSONEq(t, string(want), string(r.Raw))
			})
		}
	}
}

// The seeded TTC persona: per-cycle ovulation from the engine (BBT; the 07-20 LH conflict resolved to BBT), the
// engine's BBT rule, ≤ 4 queries warm and the ETag / 304.
func TestFertilityV6_InsightsEngineBudgetAndETag(t *testing.T) {
	e := newV6Env(t)
	uid := e.ttcPersona("09120035002")

	cold := e.get(uid, "/api/v1/fertility/insights", "fa", nil)
	require.Equal(t, fiber.StatusOK, cold.Status, string(cold.Raw))
	warm := e.get(uid, "/api/v1/fertility/insights", "fa", nil)
	t.Logf("/fertility/insights v6: cold %d, warm %d queries", cold.Queries, warm.Queries)
	assert.LessOrEqual(t, warm.Queries, int64(4))
	assert.JSONEq(t, string(cold.Raw), string(warm.Raw))
	require.NotEmpty(t, warm.ETag)

	var body struct {
		Data struct {
			History []struct {
				Source       string `json:"source"`
				OvulationDay int    `json:"ovulation_day"`
				CycleStart   string `json:"cycle_start"`
			} `json:"history"`
			Evidence []struct {
				Key      string `json:"key"`
				Strength string `json:"strength"`
			} `json:"evidence"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(warm.Raw, &body))
	require.Len(t, body.Data.History, 2)
	assert.Equal(t, "bbt", body.Data.History[0].Source)
	assert.Equal(t, 15, body.Data.History[0].OvulationDay)
	assert.Equal(t, "bbt", body.Data.History[1].Source, "LH on day 5 conflicts with the BBT day 14: BBT wins")
	assert.Equal(t, 14, body.Data.History[1].OvulationDay)
	assert.Equal(t, "strong", body.Data.Evidence[2].Strength, "LH positive this cycle")

	for _, path := range []string{"/api/v1/fertility/insights", "/api/v1/fertility/today", "/api/v1/fertility/bbt?range=3"} {
		r := e.get(uid, path, "fa", nil)
		require.Equal(t, fiber.StatusOK, r.Status)
		require.NotEmpty(t, r.ETag, path)
		nm := e.get(uid, path, "fa", map[string]string{fiber.HeaderIfNoneMatch: r.ETag})
		assert.Equal(t, fiber.StatusNotModified, nm.Status, path)
		assert.Empty(t, nm.Raw)
		assert.LessOrEqual(t, nm.Queries, int64(2), "304 after the auth and user-zone queries only")
		other := e.get(uid, path, "en", map[string]string{fiber.HeaderIfNoneMatch: r.ETag})
		assert.Equal(t, fiber.StatusOK, other.Status, "another locale never revalidates")
	}
	b1 := e.get(uid, "/api/v1/fertility/bbt?range=1", "fa", nil)
	b3 := e.get(uid, "/api/v1/fertility/bbt?range=3", "fa", nil)
	assert.NotEqual(t, b1.ETag, b3.ETag, "the range is part of the tag")

	// A write moves the revision: the old tag no longer matches and the day carries the engine context.
	put := e.req(fiber.MethodPut, uid, "/api/v1/fertility/days/2026-09-23", "fa", `{"bbt":36.70}`, nil)
	require.Equal(t, fiber.StatusOK, put.Status, string(put.Raw))
	assert.Contains(t, string(put.Raw), `"cycle_day":14`)
	after := e.get(uid, "/api/v1/fertility/insights", "fa", map[string]string{fiber.HeaderIfNoneMatch: warm.ETag})
	assert.Equal(t, fiber.StatusOK, after.Status)
}
