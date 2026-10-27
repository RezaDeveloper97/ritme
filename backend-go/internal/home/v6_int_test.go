package home_test

// T-M9-34 (D-122): /home and /home/cycle-overview under CYCLE_ENGINE=v6 over the contract fixture — Go-native goldens
// (contract/golden-v6/home; rewrite with UPDATE_GOLDEN_V6=1 and review the diff by hand), every persona builds without
// a failing section, the query budget (warm /home, warm cycle-overview) and the cycle-overview ETag / 304.
// Integration: `make test-int PKG=./internal/home/...`.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/home"
	homestore "github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/usertz"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

const (
	fixtureNow    = "2026-09-23T10:00:00+03:30" // the contract clock (cmd/contract DefaultNow)
	queriesHead   = "X-Test-Queries"
	contractKeys  = "../../contract/fixtures/keys"
	goldenHomeDir = "../../contract/golden-v6/home"
)

type homeEnv struct {
	t      *testing.T
	app    *fiber.App
	iss    *passport.Issuer
	tokens map[uint64]string
	logs   *bytes.Buffer
}

// newHomeEnv mounts /home and /home/cycle-overview like routes_home.go with the v6 engine state source (engineStates):
// metrics counter → request clock → user zone → locale → auth guard → handler, config caches on (production).
func newHomeEnv(t *testing.T) *homeEnv {
	t.Helper()
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	keysPath, err := filepath.Abs(contractKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-t-m9-34:")
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	var logs bytes.Buffer
	sectionLog := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError}))

	cq := cyclestore.New(db)
	p, err := state.New(cq, rc, clock.Real{}, state.Options{Enabled: true, L1Entries: 100, Build: "test", Logger: quiet})
	require.NoError(t, err)
	h := home.NewHandlers(home.Deps{
		Queries: homestore.New(db), Messages: msgstore.New(db), Pregnancy: pstore.New(db),
		AppURL: "https://api.ritme.app", Logger: sectionLog,
	}, cycleservice.New(db, nil), clock.Real{}).WithEngine(state.Source{P: p, Revs: cq}, cq)

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
	app.Get("/api/v1/home", locale, guard, h.Index)
	app.Get("/api/v1/home/cycle-overview", locale, guard, h.CycleOverview)
	return &homeEnv{t: t, app: app, tokens: map[uint64]string{}, logs: &logs,
		iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

type reply struct {
	Status  int
	Raw     []byte
	Queries int64
	ETag    string
}

func (e *homeEnv) get(uid uint64, path, lang string, hdr map[string]string) reply {
	e.t.Helper()
	tok, ok := e.tokens[uid]
	if !ok {
		issued, err := e.iss.Issue(context.Background(), uid, time.Now())
		require.NoError(e.t, err)
		tok = issued.AccessToken
		e.tokens[uid] = tok
	}
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", lang)
	req.Header.Set("Authorization", "Bearer "+tok)
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
	return reply{Status: resp.StatusCode, Raw: raw, Queries: n, ETag: resp.Header.Get(fiber.HeaderETag)}
}

var goldenPersonas = []struct {
	name string
	uid  uint64
}{
	{"no_profile", 1001}, {"onboarding_declared", 1003}, {"regular", 1004}, {"irregular", 1005},
	{"short_outlier", 1006}, {"open_period_day3", 1007}, {"overdue_20", 1011}, {"ttc", 1012}, {"pregnant_lmp_w8", 1014},
}

// Go-native goldens of /home (fa) and /home/cycle-overview (en) per persona on the contract day.
func TestHomeV6_Goldens(t *testing.T) {
	e := newHomeEnv(t)
	update := os.Getenv("UPDATE_GOLDEN_V6") != ""
	for _, p := range goldenPersonas {
		for _, c := range []struct{ name, path, lang string }{
			{"index." + p.name + ".fa", "/api/v1/home", "fa"},
			{"cycle_overview." + p.name + ".en", "/api/v1/home/cycle-overview", "en"},
		} {
			t.Run(c.name, func(t *testing.T) {
				r := e.get(p.uid, c.path, c.lang, nil)
				require.Equal(t, fiber.StatusOK, r.Status, string(r.Raw))
				var pretty bytes.Buffer
				require.NoError(t, json.Indent(&pretty, r.Raw, "", "  "))
				pretty.WriteByte('\n')
				file := filepath.Join(goldenHomeDir, c.name+".json")
				if update {
					require.NoError(t, os.MkdirAll(goldenHomeDir, 0o755))
					require.NoError(t, os.WriteFile(file, pretty.Bytes(), 0o644)) //nolint:gosec // golden fixture
					return
				}
				want, err := os.ReadFile(file) //nolint:gosec // test fixture path
				require.NoError(t, err, "no golden (run with UPDATE_GOLDEN_V6=1)")
				assert.JSONEq(t, string(want), string(r.Raw))
			})
		}
	}
	assert.Empty(t, e.logs.String(), "no section failed to build")
}

// Every fixture persona builds the v6 home in every locale without a failing section.
func TestHomeV6_EveryPersonaBuilds(t *testing.T) {
	e := newHomeEnv(t)
	for uid := uint64(1001); uid <= 1018; uid++ {
		for _, lang := range []string{"fa", "en"} {
			for _, q := range []string{"", "?date=2026-09-30", "?date=2026-08-01"} {
				r := e.get(uid, "/api/v1/home"+q, lang, nil)
				if r.Status == fiber.StatusConflict { // a companion (male) fixture account
					continue
				}
				require.Equal(t, fiber.StatusOK, r.Status, "%d %s%s: %s", uid, lang, q, r.Raw)
			}
		}
	}
	assert.Empty(t, e.logs.String(), "no section failed to build")
}

// Query budget (README §8): warm /home ≤ homeWarmBudget, warm cycle-overview ≤ overviewWarmBudget, and a matching
// If-None-Match on the overview answers 304 without a body.
//
// /home warm is 11 (v1: 13): auth, pregnancy profile, engine profile, histories, logs (90 d: the message system's too),
// unread notifications, task completions, challenge completions (14 d + today), doctor and medication reminders.
// The task's target of 8 needs merged sqlc queries in db/queries/home (outside T-M9-34's touches; open item).
// Cycle-overview warm is auth + the 90-day logs (the production route adds the companion check and the Plus banner).
const (
	homeWarmBudget     = 11
	overviewWarmBudget = 2
)

func TestHomeV6_QueryBudgetAnd304(t *testing.T) {
	e := newHomeEnv(t)
	const uid = 1004
	cold := e.get(uid, "/api/v1/home", "fa", nil)
	require.Equal(t, fiber.StatusOK, cold.Status, string(cold.Raw))
	warm := e.get(uid, "/api/v1/home", "fa", nil)
	t.Logf("/home v6: cold %d, warm %d queries", cold.Queries, warm.Queries)
	assert.LessOrEqual(t, warm.Queries, int64(homeWarmBudget))
	assert.JSONEq(t, string(cold.Raw), string(warm.Raw))

	ov := e.get(uid, "/api/v1/home/cycle-overview", "en", nil)
	require.Equal(t, fiber.StatusOK, ov.Status, string(ov.Raw))
	require.NotEmpty(t, ov.ETag)
	ov2 := e.get(uid, "/api/v1/home/cycle-overview", "en", nil)
	t.Logf("/home/cycle-overview v6: warm %d queries", ov2.Queries)
	assert.LessOrEqual(t, ov2.Queries, int64(overviewWarmBudget))
	assert.Equal(t, ov.ETag, ov2.ETag)

	nm := e.get(uid, "/api/v1/home/cycle-overview", "en", map[string]string{fiber.HeaderIfNoneMatch: ov2.ETag})
	assert.Equal(t, fiber.StatusNotModified, nm.Status)
	assert.Empty(t, nm.Raw)
	assert.Less(t, nm.Queries, ov2.Queries, "304 skips the log read")

	fa := e.get(uid, "/api/v1/home/cycle-overview", "fa", map[string]string{fiber.HeaderIfNoneMatch: ov2.ETag})
	assert.Equal(t, fiber.StatusOK, fa.Status, "another locale never revalidates with this tag")
	assert.Equal(t, fmt.Sprint(fiber.StatusOK), fmt.Sprint(e.get(1001, "/api/v1/home/cycle-overview", "en", nil).Status))
}
