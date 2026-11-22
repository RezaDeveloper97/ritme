package pregnancyalerts_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/cycle/revision"
	apihttp "github.com/ritme/backend-go/internal/http"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// T-M12-30c: GET /pregnancy/v2/today skips the daily alert evaluation only while its input revision is unchanged and
// the request clock is before the evaluation's next due instant. These run on the production wiring (internal/http.Mount
// with Redis), so the urgent alerts a skip could hide are checked end to end.

const (
	evalKeys   = "../../../contract/fixtures/keys"
	evalClient = "0199c0de-0000-7000-8000-00000c0ffe30"
	queriesHdr = "X-Test-Queries"
	// lmpWeek29 dates a pregnancy 29w3d on 2026-09-23 (kick counting from 28 completed weeks).
	lmpWeek29 = "2026-03-01"
)

type evalEnv struct {
	t   *testing.T
	db  *sql.DB
	app *fiber.App
	mr  *miniredis.Miniredis
	iss *passport.Issuer
}

func newEvalEnv(t *testing.T) *evalEnv {
	t.Helper()
	db := testdb.New(t)
	keysPath, err := filepath.Abs(evalKeys)
	require.NoError(t, err)
	keys, err := passport.LoadKeys(keysPath)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT IGNORE INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, evalClient)
	require.NoError(t, err)

	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(metrics.HTTP())
	app.Use(func(c fiber.Ctx) error {
		err := c.Next()
		c.Set(queriesHdr, strconv.FormatInt(metrics.RequestQueries(c), 10))
		return err
	})
	app.Use(clock.Middleware(clock.Real{}, true))
	apihttp.Mount(app, &apihttp.Deps{
		Config: &config.Config{
			App:         config.App{Env: "testing", URL: "http://localhost"},
			SMS:         config.SMS{Provider: "log"},
			Passport:    config.Passport{TokenLifetimeDays: 365, RefreshWindowDays: 30},
			StoragePath: keysPath,
		},
		DB: db, Cache: cache.NewFromClient(rdb, "ritme-go-eval:"), Logger: quiet,
	})
	return &evalEnv{t: t, db: db, app: app, mr: mr, iss: passport.NewIssuer(keys.Private, authstore.New(db), clock.Real{}, 365)}
}

func (e *evalEnv) exec(q string, args ...any) {
	e.t.Helper()
	_, err := e.db.Exec(q, args...)
	require.NoError(e.t, err)
}

func (e *evalEnv) count(q string, args ...any) int {
	e.t.Helper()
	var n int
	require.NoError(e.t, e.db.QueryRow(q, args...).Scan(&n))
	return n
}

// pregnant is a pregnant user dated by lmp.
func (e *evalEnv) pregnant(mobile, lmp string) (uint64, string) {
	e.t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(e.t, err)
	id, err := res.LastInsertId()
	require.NoError(e.t, err)
	uid := uint64(id) //nolint:gosec // auto-increment id
	e.exec(`INSERT INTO user_life_profiles (user_id, gender, life_mode, onboarding_completed_at, created_at, updated_at)
		VALUES (?, 'female', 'pregnancy', '2026-09-01 09:00:00', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
		VALUES (?, 1, 'lmp', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid, lmp)
	tok, err := e.iss.Issue(context.Background(), uid, time.Now())
	require.NoError(e.t, err)
	return uid, tok.AccessToken
}

// call sends one request at the Tehran wall clock at ("2026-09-23T10:00") and returns its status and query count.
func (e *evalEnv) call(method, path, tok, at string) (int, int64) {
	e.t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set(clock.Header, at+":00+03:30")
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 60 * time.Second})
	require.NoError(e.t, err)
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	n, err := strconv.ParseInt(resp.Header.Get(queriesHdr), 10, 64)
	require.NoError(e.t, err)
	return resp.StatusCode, n
}

// today is GET /pregnancy/v2/today at at; it returns the request's query count.
func (e *evalEnv) today(tok, at string) int64 {
	e.t.Helper()
	status, n := e.call(fiber.MethodGet, "/api/v1/pregnancy/v2/today", tok, at)
	require.Equal(e.t, 200, status)
	return n
}

// open is the number of open inbox rows of rule for uid.
func (e *evalEnv) open(uid uint64, rule string) int {
	return e.count(`SELECT COUNT(*) FROM message_inbox WHERE user_id = ? AND rule_key = ? AND open_flag = 1`, uid, rule)
}

// warmMax is the skipped read's ceiling (querybudget "pregnancy v2 today" warm): the Today read plus the input stamp.
const warmMax = 8

// A repeat read skips the evaluation; the read after it is within the warm budget.
func TestEvalCache_RepeatReadSkips(t *testing.T) {
	e := newEvalEnv(t)
	_, tok := e.pregnant("09120300001", lmpWeek29)
	first := e.today(tok, "2026-09-23T10:00")
	again := e.today(tok, "2026-09-23T10:05")
	assert.Greater(t, first, int64(warmMax), "the first read evaluates")
	assert.LessOrEqual(t, again, int64(warmMax), "the repeat read skips the evaluation")
}

// A qualifying log written by a revision-bumping writer that does not evaluate itself (data inserted, analysis_rev
// bumped as revision.BumpAnalysis does on every pregnancy log save) raises the urgent alert on the very next read.
func TestEvalCache_UrgentRightAfterLog(t *testing.T) {
	e := newEvalEnv(t)
	uid, tok := e.pregnant("09120300002", lmpWeek29)
	e.today(tok, "2026-09-23T10:00")
	require.LessOrEqual(t, e.today(tok, "2026-09-23T10:01"), int64(warmMax), "cached before the log")
	require.Zero(t, e.open(uid, "preg.critical_symptom"))

	e.exec(`INSERT INTO pregnancy_symptom_logs (user_id, log_date, has_bleeding, bleeding_severity, created_at, updated_at)
		VALUES (?, '2026-09-23', 1, 'severe', '2026-09-23 10:02:00', '2026-09-23 10:02:00')`, uid)
	require.NoError(t, revision.BumpAnalysis(context.Background(), e.db, uid))

	e.today(tok, "2026-09-23T10:03")
	assert.Equal(t, 1, e.open(uid, "preg.critical_symptom"), "the urgent alert is raised on the next read")
}

// H-05: a kick session started (no revision bump) and left running short of the target raises the urgent alert on the
// first read after its 2 h window ends, with no write in between.
func TestEvalCache_KickWindowExpiresWithoutWrite(t *testing.T) {
	e := newEvalEnv(t)
	uid, tok := e.pregnant("09120300003", lmpWeek29)
	e.today(tok, "2026-09-23T09:00")

	status, _ := e.call(fiber.MethodPost, "/api/v1/pregnancy/kick-sessions", tok, "2026-09-23T10:00")
	require.Equal(t, 201, status)

	e.today(tok, "2026-09-23T10:01") // the new session moves the stamp: evaluated, due = 12:00
	assert.LessOrEqual(t, e.today(tok, "2026-09-23T11:30"), int64(warmMax), "inside the window the read skips")
	require.Zero(t, e.open(uid, "preg.fetal_movement_kicks"), "no alert while the session is counting")

	e.today(tok, "2026-09-23T12:00")
	assert.Equal(t, 1, e.open(uid, "preg.fetal_movement_kicks"), "the window ended short of the target: urgent alert")
}

// The local day change is a due instant: the read after midnight that enters a new week raises week_entered for it.
func TestEvalCache_WeekChange(t *testing.T) {
	e := newEvalEnv(t)
	uid, tok := e.pregnant("09120300004", lmpWeek29)
	week := func() int {
		return e.count(`SELECT COALESCE(MAX(CAST(JSON_EXTRACT(evidence, '$.vars.preg_week') AS UNSIGNED)), 0)
			FROM message_inbox WHERE user_id = ? AND rule_key = 'preg.week_entered' AND open_flag = 1`, uid)
	}
	// 2026-09-26 is 29w6d, 2026-09-27 is 30w0d (week 31).
	e.today(tok, "2026-09-26T23:00")
	require.Equal(t, 30, week())
	require.LessOrEqual(t, e.today(tok, "2026-09-26T23:50"), int64(warmMax), "same day: skipped")

	e.today(tok, "2026-09-27T00:10")
	assert.Equal(t, 31, week(), "the new week's alert after midnight")
}

// Redis down: every read evaluates (fail safe), so an input with no revision bump is still seen.
func TestEvalCache_RedisDownEvaluates(t *testing.T) {
	e := newEvalEnv(t)
	uid, tok := e.pregnant("09120300005", lmpWeek29)
	e.today(tok, "2026-09-23T10:00")
	e.mr.Close()

	e.exec(`INSERT INTO pregnancy_symptom_logs (user_id, log_date, has_bleeding, bleeding_severity, created_at, updated_at)
		VALUES (?, '2026-09-23', 1, 'severe', '2026-09-23 10:02:00', '2026-09-23 10:02:00')`, uid)
	e.today(tok, "2026-09-23T10:03")
	assert.Equal(t, 1, e.open(uid, "preg.critical_symptom"))
}

// NextDue enumerates the time-based rules: the next Tehran midnight, a running kick session's window end, the next
// appointment's own instant (the earliest wins).
func TestNextDue(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, civildate.Tehran)
		require.NoError(t, err)
		return v
	}
	now := at("2026-09-23 10:00")
	assert.Equal(t, at("2026-09-24 00:00"), pregnancyalerts.NextDue(pregnancyalerts.Facts{}, time.Time{}, now))
	running := pregnancyalerts.Facts{Kicks: []pregnancyalerts.Kick{{ID: 1, Start: at("2026-09-23 09:30")}}}
	assert.Equal(t, at("2026-09-23 11:30"), pregnancyalerts.NextDue(running, time.Time{}, now))
	ended := pregnancyalerts.Facts{Kicks: []pregnancyalerts.Kick{{ID: 1, Start: at("2026-09-23 07:00"), End: at("2026-09-23 08:00")}}}
	assert.Equal(t, at("2026-09-24 00:00"), pregnancyalerts.NextDue(ended, time.Time{}, now))
	assert.Equal(t, at("2026-09-23 15:00"), pregnancyalerts.NextDue(pregnancyalerts.Facts{}, at("2026-09-23 15:00"), now))
	assert.Equal(t, at("2026-09-24 00:00"), pregnancyalerts.NextDue(pregnancyalerts.Facts{}, at("2026-09-23 09:00"), now),
		"a past appointment is no due")
}
