package analysis_test

// /api/v1/analysis/* end to end on MariaDB: the reports read the caller's own cycle history and
// health_log_entries only (IDOR), Plus sections are locked for a free user and open during a trial,
// validation of range / ym / calendar, the 401 body and the request language.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/analysis"
	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/plus"
	"github.com/ritme/backend-go/resources/translations"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
)

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
}

func setup(t *testing.T) *env {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	languages := i18n.NewRegistry(i18nstore.New(db), nil, quiet)
	locale := i18n.Middleware(languages)
	h := analysis.NewHandlers(cycleservice.New(db, nil), healthlog.NewService(db),
		plus.NewService(db, config.Plus{TrialDays: 7}, nil, quiet),
		i18n.NewTranslationStore(translations.FS, ""), languages, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/analysis/summary", locale, guard, h.Summary)
	app.Get("/api/v1/analysis/cycle", locale, guard, h.Cycle)
	app.Get("/api/v1/analysis/period", locale, guard, h.Period)
	app.Get("/api/v1/analysis/symptoms", locale, guard, h.Symptoms)
	app.Get("/api/v1/analysis/correlations", locale, guard, h.Correlations)
	app.Get("/api/v1/analysis/body", locale, guard, h.Body)
	app.Get("/api/v1/analysis/monthly/:ym", locale, guard, h.Monthly)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // positive id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // positive id
}

// periods stores confirmed 5-day periods starting at each date.
func (e *env) periods(t *testing.T, uid uint64, starts ...string) {
	t.Helper()
	for _, s := range starts {
		d := civildate.MustParse(s)
		_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, period_end_date, is_confirmed, source, created_at, updated_at)
			VALUES (?, ?, ?, 1, 'user_logged', NOW(), NOW())`, uid, d, d.AddDays(4))
		require.NoError(t, err)
	}
}

func (e *env) entry(t *testing.T, uid uint64, date, cat, param, item, code, num string) {
	t.Helper()
	var c, n any
	if code != "" {
		c = code
	}
	if num != "" {
		n = num
	}
	_, err := e.db.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, value_num, value_text, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, 'manual', NOW(), NOW())`, uid, date, cat, param, item, c, n)
	require.NoError(t, err)
}

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func path(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		mm, _ := cur.(map[string]any)
		cur = mm[k]
	}
	return cur
}

func (e *env) get(t *testing.T, url, token, lang string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, strings.NewReader(""))
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	req.Header.Set(clock.Header, now)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

// seedA: four completed 28-day cycles, the current one from 2026-09-14, a month of sleep, mood and
// weight logs.
func seedA(t *testing.T, e *env, uid uint64) {
	e.periods(t, uid, "2026-05-25", "2026-06-22", "2026-07-20", "2026-08-17", "2026-09-14")
	start := civildate.MustParse("2026-08-01")
	for i := range 54 {
		d := start.AddDays(i).String()
		dur, mood := "6_9", "happy"
		if i%3 == 0 {
			dur, mood = "3_6", "irritable"
		}
		e.entry(t, uid, d, "sleep", "duration", "", dur, "")
		e.entry(t, uid, d, "mood", "moods", mood, "yes", "")
		e.entry(t, uid, d, "measurements", "weight", "", "", fmt.Sprintf("%.2f", 61-0.02*float64(i)))
	}
}

func TestAnalysis_OwnDataOnlyAndPlusLocks(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000001")
	b, tokB := e.user(t, "09120000002")
	seedA(t, e, a)
	e.periods(t, b, "2026-09-01")
	e.entry(t, b, "2026-09-20", "measurements", "weight", "", "", "80.00")
	e.entry(t, b, "2026-09-20", "pain", "location", "head", "severe", "")

	// A free user: the free cards carry A's data, every Plus card is locked without data.
	r := e.get(t, "/api/v1/analysis/summary", tokA, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "6m", path(r.data(), "range", "key"))
	assert.InDelta(t, 28, path(r.data(), "sections", "cycle", "data", "median_cycle"), 0)
	assert.Equal(t, true, path(r.data(), "sections", "mood_by_phase", "locked"))
	assert.Nil(t, path(r.data(), "sections", "mood_by_phase", "data"))
	assert.Equal(t, true, path(r.data(), "sections", "sleep_mood", "locked"))
	assert.Equal(t, true, path(r.data(), "sections", "labs", "locked"))
	assert.NotContains(t, r.raw, `"80`, "B's weight never shows in A's report")

	c := e.get(t, "/api/v1/analysis/correlations", tokA, "")
	require.Equal(t, 200, c.status, c.raw)
	assert.Equal(t, true, c.data()["locked"])
	assert.Equal(t, true, c.data()["not_causal"])
	assert.Nil(t, c.data()["data"])

	// B sees only B's own weight and symptom.
	bb := e.get(t, "/api/v1/analysis/body?range=1m", tokB, "")
	require.Equal(t, 200, bb.status, bb.raw)
	assert.InDelta(t, 80, path(bb.data(), "weight", "current"), 0.001)
	bs := e.get(t, "/api/v1/analysis/symptoms?range=1m", tokB, "")
	require.Equal(t, 200, bs.status)
	top, _ := bs.data()["top"].([]any)
	require.Len(t, top, 1)
	assert.Equal(t, "pain.location.head", top[0].(map[string]any)["key"])
	ab := e.get(t, "/api/v1/analysis/body?range=1m", tokA, "")
	assert.InDelta(t, 60, path(ab.data(), "weight", "current"), 1)

	// A running trial opens the Plus sections.
	_, err := e.db.Exec(`INSERT INTO plus_trials (user_id, started_at, ends_at, created_at, updated_at)
		VALUES (?, '2026-09-22 10:00:00', '2026-09-29 10:00:00', NOW(), NOW())`, a)
	require.NoError(t, err)
	c = e.get(t, "/api/v1/analysis/correlations?range=3m", tokA, "")
	require.Equal(t, 200, c.status, c.raw)
	assert.Equal(t, false, c.data()["locked"])
	items, _ := path(c.data(), "data", "items").([]any)
	require.Len(t, items, 4)
	sleep := items[0].(map[string]any)
	assert.Equal(t, "sleep_mood", sleep["key"])
	assert.Equal(t, "ready", sleep["status"])
	assert.Equal(t, "strong", sleep["strength"])
	assert.Equal(t, true, sleep["not_causal"])
	r = e.get(t, "/api/v1/analysis/summary", tokA, "")
	assert.Equal(t, false, path(r.data(), "sections", "sleep_mood", "locked"))
	// B is still free.
	c = e.get(t, "/api/v1/analysis/correlations", tokB, "")
	assert.Equal(t, true, c.data()["locked"])
}

func TestAnalysis_EveryReportAndLanguage(t *testing.T) {
	e := setup(t)
	a, tok := e.user(t, "09120000003")
	seedA(t, e, a)
	for _, url := range []string{
		"/api/v1/analysis/cycle?range=1y", "/api/v1/analysis/period?range=all", "/api/v1/analysis/symptoms?range=2w",
		"/api/v1/analysis/body?range=7d", "/api/v1/analysis/monthly/2026-09", "/api/v1/analysis/monthly/1405-06?calendar=jalali",
	} {
		r := e.get(t, url, tok, "")
		assert.Equal(t, 200, r.status, url+" "+r.raw)
	}
	m := e.get(t, "/api/v1/analysis/monthly/1405-06?calendar=jalali", tok, "")
	assert.Equal(t, "2026-08-23", path(m.data(), "month", "from"))
	assert.Equal(t, true, path(m.data(), "pdf", "locked"))

	en := e.get(t, "/api/v1/analysis/summary", tok, "en")
	assert.Contains(t, path(en.data(), "top_finding", "text"), "Your cycles are regular")
	fa := e.get(t, "/api/v1/analysis/summary", tok, "")
	assert.Contains(t, path(fa.data(), "top_finding", "text"), "منظم")
}

func TestAnalysis_ValidationAndAuth(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000004")
	for _, url := range []string{
		"/api/v1/analysis/summary?range=5y",
		"/api/v1/analysis/monthly/2026-13",
		"/api/v1/analysis/monthly/2026-9",
		"/api/v1/analysis/monthly/2026-10",
		"/api/v1/analysis/monthly/2026-09?calendar=lunar",
	} {
		r := e.get(t, url, tok, "")
		assert.Equal(t, 422, r.status, url+" "+r.raw)
	}
	r := e.get(t, "/api/v1/analysis/summary", "", "")
	assert.Equal(t, 401, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])

	// A brand-new user: no history, no logs → 200 with the no-data finding.
	r = e.get(t, "/api/v1/analysis/summary", tok, "")
	require.Equal(t, 200, r.status, r.raw)
	assert.Equal(t, "no_data", path(r.data(), "top_finding", "kind"))
}
