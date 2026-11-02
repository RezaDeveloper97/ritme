package analysis_test

// T-M9-35 (D-129, A-01, E-09): the analysis tab under CYCLE_ENGINE=v6 over the contract fixture — the regularity
// verdict equals cycle_view.metrics.cycle_regularity for every persona, no lone-outlier exclusion, Go-native goldens
// (internal/analysis/testdata/golden-v6; rewrite with UPDATE_GOLDEN_V6=1 and review the diff by hand).
// Integration: `make test-int PKG=./internal/analysis/...`.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/engine"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/resources/translations"
)

const goldenV6Dir = "testdata/golden-v6"

type v6Env struct {
	t      *testing.T
	db     *sql.DB
	app    *fiber.App
	iss    *passport.Issuer
	states state.Source
	tokens map[uint64]string
}

func newV6Env(t *testing.T) *v6Env {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	testdb.LoadSQL(t, filepath.Join("..", "..", "contract", "fixtures", "dump.sql"))
	keysPath, err := filepath.Abs("../../contract/fixtures/keys")
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-t-m9-35a:")
	cq := cyclestore.New(db)
	p, err := state.New(cq, rc, clock.Real{}, state.Options{Enabled: true, L1Entries: 100, Build: "test", Logger: quiet})
	require.NoError(t, err)
	src := state.Source{P: p, Revs: cq}

	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)
	h := analysis.NewHandlers(cycleservice.New(db, nil), healthlog.NewService(db),
		plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet),
		i18n.NewTranslationStore(translations.FS, ""), languages, clock.Real{}).WithEngine(src)
	ttc := analysis.NewTTCHandlers(h, fertility.NewService(db))
	cfg := &config.Config{
		App:         config.App{Env: "testing", URL: "http://localhost"},
		Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
		StoragePath: keysPath,
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	guard := auth.MustGuard(app, cfg, db, quiet).RequireUser
	app.Get("/api/v1/analysis/summary", locale, guard, h.Summary)
	app.Get("/api/v1/analysis/cycle", locale, guard, h.Cycle)
	app.Get("/api/v1/analysis/ttc", locale, guard, ttc.TTC)
	app.Get("/api/v1/analysis/monthly/:ym", locale, guard, h.Monthly)
	return &v6Env{t: t, db: db, app: app, states: src, tokens: map[uint64]string{},
		iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *v6Env) get(uid uint64, path, lang string) (int, []byte) {
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
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(e.t, err)
	raw, err := io.ReadAll(resp.Body)
	require.NoError(e.t, err)
	_ = resp.Body.Close()
	return resp.StatusCode, raw
}

// engineRegularity is cycle_view.metrics.cycle_regularity of uid on the contract day (the same state the today
// payload encodes).
func (e *v6Env) engineRegularity(uid uint64) string {
	e.t.Helper()
	today := civildate.MustParse("2026-09-23")
	st, err := e.states.ForUser(context.Background(), uid, today)
	require.NoError(e.t, err)
	d := engine.DayOf(today)
	return st.Resolve(d, d).Metrics.CycleRegularity.String()
}

var analysisPersonas = []struct {
	name string
	uid  uint64
}{
	{"no_profile", 1001}, {"onboarding_declared", 1003}, {"regular", 1004}, {"irregular", 1005},
	{"short_outlier", 1006}, {"open_period_day3", 1007}, {"overdue_20", 1011}, {"ttc", 1012},
}

// Agreement (D-129): the tab's verdict is the engine's for every persona, in /analysis/cycle, the summary card and
// the TTC regularity card; the v1 spelling not_enough_data never appears.
func TestAnalysisV6_RegularityAgreesWithCycleView(t *testing.T) {
	e := newV6Env(t)
	for _, p := range analysisPersonas {
		t.Run(p.name, func(t *testing.T) {
			want := e.engineRegularity(p.uid)
			status, raw := e.getPlus(p.uid, "/api/v1/analysis/cycle?range=1y", "en")
			require.Equal(t, fiber.StatusOK, status, string(raw))
			var cyc struct {
				Data struct {
					Variation struct {
						Status string `json:"status"`
						Days   *int   `json:"days"`
						Max    int    `json:"max"`
					} `json:"variation"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(raw, &cyc))
			assert.Equal(t, want, cyc.Data.Variation.Status)
			if d := cyc.Data.Variation.Days; d != nil { // the shown variation never contradicts the verdict
				assert.Equal(t, *d > cyc.Data.Variation.Max, want == "irregular")
			}

			status, raw = e.get(p.uid, "/api/v1/analysis/summary", "en")
			require.Equal(t, fiber.StatusOK, status, string(raw))
			var sum struct {
				Data struct {
					Sections struct {
						Cycle struct {
							Data struct {
								Regularity string `json:"regularity"`
							} `json:"data"`
						} `json:"cycle"`
					} `json:"sections"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(raw, &sum))
			assert.Equal(t, want, sum.Data.Sections.Cycle.Data.Regularity)
			assert.NotContains(t, string(raw), `"regularity":"not_enough_data"`)

			status, raw = e.get(p.uid, "/api/v1/analysis/ttc", "en")
			require.Equal(t, fiber.StatusOK, status, string(raw))
			var ttc struct {
				Data struct {
					Regularity struct {
						Data *struct {
							Status string `json:"status"`
						} `json:"data"`
					} `json:"regularity"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(raw, &ttc))
			if d := ttc.Data.Regularity.Data; d != nil {
				assert.Equal(t, want, d.Status)
			}
		})
	}
}

// E-09: a lone outlier is no longer excluded (the v1 markOutlier); it stays counted and the verdict follows the
// engine (irregular: 28, 28, 28, 40 → 12 days over the adult band of 7).
func TestAnalysisV6_NoOutlierExclusion(t *testing.T) {
	e := newV6Env(t)
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Out', '09120035101', NOW(), NOW())`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // positive id
	for _, s := range []string{"2026-04-20", "2026-05-18", "2026-06-15", "2026-07-13", "2026-08-22"} {
		d := civildate.MustParse(s)
		_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, source, created_at, updated_at)
			VALUES (?, ?, ?, 1, 'user_logged', NOW(), NOW())`, uid, d, d.AddDays(4))
		require.NoError(t, err)
	}
	status, raw := e.getPlus(uid, "/api/v1/analysis/cycle?range=1y", "en")
	require.Equal(t, fiber.StatusOK, status, string(raw))
	var body struct {
		Data struct {
			Variation struct {
				Status string `json:"status"`
				Days   *int   `json:"days"`
			} `json:"variation"`
			Cycles []struct {
				Length   int     `json:"length"`
				Counted  bool    `json:"counted"`
				Excluded *string `json:"excluded"`
			} `json:"cycles"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &body))
	assert.Equal(t, "irregular", body.Data.Variation.Status)
	assert.Equal(t, e.engineRegularity(uid), body.Data.Variation.Status)
	require.NotNil(t, body.Data.Variation.Days)
	assert.Equal(t, 12, *body.Data.Variation.Days)
	for _, c := range body.Data.Cycles {
		assert.True(t, c.Counted, "length %d", c.Length)
		assert.Nil(t, c.Excluded)
	}
}

func TestAnalysisV6_Goldens(t *testing.T) {
	e := newV6Env(t)
	update := os.Getenv("UPDATE_GOLDEN_V6") != ""
	for _, p := range analysisPersonas {
		for _, c := range []struct{ name, path, lang string }{
			{"cycle_1y." + p.name + ".en", "/api/v1/analysis/cycle?range=1y", "en"},
			{"ttc." + p.name + ".en", "/api/v1/analysis/ttc", "en"},
		} {
			t.Run(c.name, func(t *testing.T) {
				get := e.get
				if strings.Contains(c.path, "range=1y") {
					get = e.getPlus
				}
				status, raw := get(p.uid, c.path, c.lang)
				require.Equal(t, fiber.StatusOK, status, string(raw))
				var pretty bytes.Buffer
				require.NoError(t, json.Indent(&pretty, raw, "", "  "))
				pretty.WriteByte('\n')
				file := filepath.Join(goldenV6Dir, c.name+".json")
				if update {
					require.NoError(t, os.MkdirAll(goldenV6Dir, 0o755))
					require.NoError(t, os.WriteFile(file, pretty.Bytes(), 0o644)) //nolint:gosec // golden fixture
					return
				}
				want, err := os.ReadFile(file) //nolint:gosec // test fixture path
				require.NoError(t, err, "no golden (run with UPDATE_GOLDEN_V6=1)")
				assert.JSONEq(t, string(want), string(raw))
			})
		}
	}
}

// getPlus is get for a Plus-only range (1y, T-M10-29 / D-178): the user holds a trial for this one request only,
// so every other endpoint (the TTC goldens) keeps reading her as a free user.
func (e *v6Env) getPlus(uid uint64, path, lang string) (int, []byte) {
	e.t.Helper()
	grantTrial(e.t, e.db, uid)
	defer revokeTrial(e.t, e.db, uid)
	return e.get(uid, path, lang)
}
