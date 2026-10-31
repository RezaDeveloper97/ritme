package querybudget_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
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

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/enums"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/querybudget"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	keysDir     = "../../contract/fixtures/keys"
	clientID    = "0199c0de-0000-7000-8000-00000c0ffe48"
	requestNow  = "2026-10-21T10:00:00+03:30" // benchfixtures.Today, Tehran
	queriesHead = "X-Test-Queries"
)

// TestQueryBudgets runs every route of querybudget.Budgets cold (a fresh app over an empty Redis, config caches
// off) and warm (the same request again with every cache filled) and checks the counted queries against the
// budget. -v prints the measured table.
func TestQueryBudgets(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	iss := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)

	tokens := map[enums.LifeMode]string{}
	for i, mode := range []enums.LifeMode{enums.LifeModeCycle, enums.LifeModeTTC, enums.LifeModePregnancy} {
		u, ok := benchfixtures.UserFor(mode)
		require.True(t, ok, mode)
		uid := seedUser(t, db, u, fmt.Sprintf("0912000%04d", 8000+i))
		tok, err := iss.Issue(context.Background(), uid, time.Now())
		require.NoError(t, err)
		tokens[mode] = tok.AccessToken
	}

	defer configcache.SetEnabled(false)
	var report strings.Builder
	report.WriteString("\n| Route | Mode | Cold | Warm | Budget cold / warm |\n|---|---|---:|---:|---|\n")
	for _, b := range querybudget.Budgets {
		t.Run(b.Name, func(t *testing.T) {
			tok := tokens[b.Mode]
			require.NotEmpty(t, tok, "no fixture user for %s", b.Mode)

			configcache.SetEnabled(false) // cold: every config read goes to the database
			app := newApp(t, db, keysPath, b.Engine)
			cold, body := call(t, app, b.Path, tok)
			configcache.SetEnabled(true)
			_, _ = call(t, app, b.Path, tok) // fills the config caches
			warm, _ := call(t, app, b.Path, tok)
			if b.Name == "cycle today" || b.Name == "cycle today v6" { // the counts are only meaningful over the seeded history
				assert.Contains(t, body, `"cycle_day":10`, "the fixture's cycle day 10")
			}

			fmt.Fprintf(&report, "| `%s` | %s | %d | %d | %d / %d |\n", b.Path, b.Mode, cold, warm, b.Cold, b.Warm)
			assert.LessOrEqual(t, cold, b.Cold, "cold queries over budget for %s", b.Path)
			assert.LessOrEqual(t, warm, b.Warm, "warm queries over budget for %s", b.Path)
			if cold < b.Cold || warm < b.Warm {
				t.Logf("%s: measured %d / %d, below its budget %d / %d — lower the row in budgets.go", b.Name, cold, warm, b.Cold, b.Warm)
			}
		})
	}
	t.Log(report.String())
}

// TestAuthBudget is the guard's own share (querybudget.Auth) against MariaDB; the same pin as T-M9-09's
// verify_int_test.go, kept here so the whole budget table lives in one package.
func TestAuthBudget(t *testing.T) {
	db := testdb.New(t)
	keys, err := passport.LoadKeys(keysDir)
	require.NoError(t, err)
	exec(t, db, `INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`, clientID)
	u, _ := benchfixtures.UserFor(enums.LifeModeCycle)
	uid := seedUser(t, db, u, "09120009999")
	q := authstore.New(db)
	tok, err := passport.NewIssuer(keys.Private, q, clock.Real{}, 365).Issue(context.Background(), uid, time.Now())
	require.NoError(t, err)
	v := passport.NewVerifier(keys.Public, q, clock.Real{})
	verify := func() int64 {
		ctx, counter := metrics.WithQueryCounter(context.Background())
		_, err := v.Verify(ctx, "Bearer "+tok.AccessToken)
		require.NoError(t, err)
		return counter.Load()
	}
	b := querybudget.Auth
	cold := verify()
	warm := verify()
	t.Logf("| %s | cold %d | warm %d | budget %d / %d |", b.Name, cold, warm, b.Cold, b.Warm)
	assert.LessOrEqual(t, cold, b.Cold)
	assert.LessOrEqual(t, warm, b.Warm)
}

// newApp is cmd/api's middleware order that matters for counting (metrics counter → test clock → module gate →
// the registry) on a fresh in-memory Redis, plus a probe that copies the request's query count into a header.
// engine is the CYCLE_ENGINE ("" = v1); the v6 state cache runs with verify sampled at 0 so the serving path is counted.
func newApp(t *testing.T, db *sql.DB, keysPath string, engine config.CycleEngine) *fiber.App {
	t.Helper()
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	rc := cache.NewFromClient(rdb, "ritme-go-qb:")

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(metrics.HTTP())
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		c.Set(queriesHead, strconv.FormatInt(metrics.RequestQueries(c), 10))
		return err
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Use(modules.Gate(modules.NewService(db, rc, quiet)))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
			CycleEngine: engine,
			EngineCache: config.EngineCache{Enabled: true, L1Entries: 1000, VerifySample: 0},
		},
		DB: db, Cache: rc, Logger: quiet,
	})
	return app
}

// call sends one GET as the token's user on the fixture day and returns the request's query count and body.
func call(t *testing.T, app *fiber.App, path, token string) (int64, string) {
	t.Helper()
	req := httptest.NewRequest(fiber.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(clock.Header, requestNow)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, fiber.StatusOK, resp.StatusCode, "%s: %s", path, body)
	n, err := strconv.ParseInt(resp.Header.Get(queriesHead), 10, 64)
	require.NoError(t, err, "query count header on %s", path)
	return n, string(body)
}

// seedUser writes u's account, profile, life-stage row, history and logs (benchfixtures data) and returns its id.
func seedUser(t *testing.T, db *sql.DB, u benchfixtures.User, mobile string) uint64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Bench', ?, '2025-09-01 09:00:00', '2025-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // auto-increment id

	periods := benchfixtures.Periods(u.Periods)
	var intention any
	if u.Mode == enums.LifeModeTTC {
		intention = "trying"
	}
	exec(t, db, `INSERT INTO user_profiles (user_id, birthday, weight, height, period_duration, cycle_duration, last_period_start,
		user_goal, pregnancy_intention, calculation_status, calculation_version, created_at, updated_at)
		VALUES (?, ?, '61.00', 165, 5, 28, ?, ?, ?, 'completed', 7, NOW(), NOW())`,
		uid, benchfixtures.Birthday, periods[0].Start, string(u.Goal), intention)
	exec(t, db, `INSERT INTO user_life_profiles (user_id, gender, life_mode, onboarding_completed_at, created_at, updated_at)
		VALUES (?, 'female', ?, NOW(), NOW(), NOW())`, uid, string(u.Mode))
	if u.Mode == enums.LifeModePregnancy {
		exec(t, db, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
			VALUES (?, 1, 'lmp', ?, NOW(), NOW())`, uid, u.LMP)
	}

	var rows [][]any
	for _, p := range periods {
		if u.Mode == enums.LifeModePregnancy && !p.Start.Before(u.LMP.AddDays(1)) {
			continue // no period after the LMP
		}
		rows = append(rows, []any{uid, p.Start, p.End(), p.Bleeding, 1, 0, string(enums.DataSourceUserLogged)})
	}
	bulk(t, db, "cycle_histories", "user_id, period_start_date, period_end_date, bleeding_length, is_confirmed, is_estimated, source", rows)

	days := benchfixtures.Days(u.LogDays)
	rows = rows[:0]
	for _, r := range benchfixtures.DailyLogRows(uid, days) {
		rows = append(rows, []any{uid, r.LogDate, r.BleedingIntensity, nullJSON(r.Moods), r.StomachAcheIntensity,
			r.BloatingIntensity, r.SleepQuality, r.BasalBodyTemperature})
	}
	bulk(t, db, "daily_health_logs", "user_id, log_date, bleeding_intensity, moods, stomach_ache_intensity, bloating_intensity, sleep_quality, basal_body_temperature", rows)

	rows = rows[:0]
	var fert [][]any
	for _, d := range days {
		for _, e := range d.Entries {
			rows = append(rows, []any{uid, d.Date, e.Category, e.Param, e.Item, e.Code, e.Num})
		}
		if d.LH != "" || d.Mucus != "" {
			fert = append(fert, []any{uid, d.Date, nullStr(d.LH), nullStr(d.Mucus)})
		}
	}
	bulk(t, db, "health_log_entries", "user_id, log_date, category, param, item, value_code, value_num", rows)
	bulk(t, db, "fertility_logs", "user_id, log_date, lh_test, cervical_mucus", fert)
	return uid
}

func exec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	_, err := db.Exec(query, args...)
	require.NoError(t, err)
}

// bulk inserts rows (plus created_at / updated_at = NOW()) in multi-row statements.
func bulk(t *testing.T, db *sql.DB, table, cols string, rows [][]any) {
	t.Helper()
	const chunk = 200
	for start := 0; start < len(rows); start += chunk {
		part := rows[start:min(start+chunk, len(rows))]
		var sb strings.Builder
		args := make([]any, 0, len(part)*len(part[0]))
		fmt.Fprintf(&sb, "INSERT INTO `%s` (%s, created_at, updated_at) VALUES ", table, cols)
		for i, r := range part {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(" + strings.Repeat("?, ", len(r)) + "NOW(), NOW())")
			args = append(args, r...)
		}
		_, err := db.Exec(sb.String(), args...)
		require.NoError(t, err, table)
	}
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullJSON(v sql.Null[json.RawMessage]) any {
	if !v.Valid {
		return nil
	}
	return string(v.V)
}

// TestMessagesSelectedConstantInRules: /messages/selected reads the active rules from the configcache snapshot and
// their history in one grouped query (T-M9-13), so its warm count does not grow with the number of rules (T-M9-13b).
func TestMessagesSelectedConstantInRules(t *testing.T) {
	db := testdb.New(t)
	keysPath, err := filepath.Abs(keysDir)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	exec(t, db, `INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`, clientID)
	u, _ := benchfixtures.UserFor(enums.LifeModeCycle)
	uid := seedUser(t, db, u, "09120007777")
	tok, err := passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365).Issue(context.Background(), uid, time.Now())
	require.NoError(t, err)

	const content = `{"a":{"title":"Hi","body":"body","cta_label":"Go","cta_target":"ritme://x"}}`
	addRules := func(n int) {
		for i := range n {
			// capped and cooled-down rules: each one needs its own history (the probes T-M9-13 folded into one query);
			// approved clinical, the one kind smart_tip serves (SPEC §5.4, T-M10-04)
			exec(t, db, `INSERT INTO message_rules (title, level, kind, priority, placement, content, variants_enabled, ab_split,
				conditions, exceptions, cap_count, cap_period, cooldown_days, audience_percent, status, is_active, version,
				approved_version, created_at, updated_at)
				VALUES (?, 'info', 'clinical', ?, 'smart_tip', ?, 0, 50, '[]', '[]', 3, 'week', 1, 100, 'approved', 1, 1, 1, NOW(), NOW())`,
				fmt.Sprintf("rule %d", i), 1+i%50, content)
		}
		configcache.Changed(context.Background(), "message_rules")
	}
	path := querybudget.Budgets[slices.IndexFunc(querybudget.Budgets, func(b querybudget.Budget) bool { return b.Name == "messages selected" })]

	configcache.SetEnabled(true)
	defer configcache.SetEnabled(false)
	app := newApp(t, db, keysPath, "")
	warm := func() int64 {
		_, _ = call(t, app, path.Path, tok.AccessToken) // reloads the snapshot after a change
		n, _ := call(t, app, path.Path, tok.AccessToken)
		return n
	}
	addRules(1)
	one := warm()
	addRules(49)
	fifty := warm()
	t.Logf("messages selected warm: 1 rule %d queries, 50 rules %d queries", one, fifty)
	assert.Equal(t, one, fifty, "query count constant in the number of rules")
	// The feed (T-M10-06) reads the history in one ListUserMessageEventStats whatever the rules, so the table row
	// holds: no +1 for capped candidates.
	assert.LessOrEqual(t, fifty, path.Warm)
}
