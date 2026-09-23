package fertility_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/fertility"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
	today    = "2026-09-23"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db      *sql.DB
	app     *fiber.App
	iss     *passport.Issuer
	queries *countingDB
}

// countingDB counts the statements the fertility handlers send (the /fertility/today budget).
type countingDB struct {
	*sql.DB
	n atomic.Int64
}

func (d *countingDB) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	d.n.Add(1)
	return d.DB.ExecContext(ctx, q, args...)
}

func (d *countingDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	d.n.Add(1)
	return d.DB.QueryContext(ctx, q, args...)
}

func (d *countingDB) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	d.n.Add(1)
	return d.DB.QueryRowContext(ctx, q, args...)
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testdb.New(t)
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	q := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, q, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	counter := &countingDB{DB: db}
	h := fertility.NewHandlers(counter, clock.Real{})
	logs := healthlog.NewHandlers(healthlog.NewService(db), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/fertility/today", locale, guard, h.Today)
	app.Get("/api/v1/fertility/days/:date", locale, guard, h.ShowDay)
	app.Put("/api/v1/fertility/days/:date", locale, guard, h.UpdateDay)
	app.Post("/api/v1/health-logs", locale, guard, logs.Store)
	app.Get("/api/v1/health-logs/:date", locale, guard, logs.Show)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), queries: counter}
}

// user creates a user; lmp != "" also gives them a profile (28-day cycle, 5-day period).
func (e *env) user(t *testing.T, mobile, lmp string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	if lmp != "" {
		_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, period_duration, cycle_duration, last_period_start, pregnancy_intention, created_at, updated_at)
			VALUES (?, 5, 28, ?, 'trying', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id, lmp)
		require.NoError(t, err)
	}
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
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

func (r response) errors() map[string]any {
	d, _ := r.body["errors"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token, lang, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

func dayPath(date string) string { return "/api/v1/fertility/days/" + date }

const fullDay = `{"lh":"positive","mucus":"egg_white","bbt":36.55,"bbt_time":"06:45","intercourse":"unprotected",` +
	`"symptoms":["bloating","ovarian_pain","ovarian_pain"],"note":"صبح زود"}`

func TestDay_RoundTrip(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000501", "2026-09-10")

	r := e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "fa", fullDay)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.body["success"])
	assert.Equal(t, "ثبت روز ذخیره شد", r.body["message"])
	assert.NotContains(t, r.body, "warning")
	d := r.data()
	assert.Equal(t, "2026-09-22", d["date"])
	assert.EqualValues(t, 13, d["cycle_day"])
	assert.Equal(t, "positive", d["lh"])
	assert.Equal(t, "egg_white", d["mucus"])
	assert.Equal(t, "36.55", d["bbt"])
	assert.Equal(t, "06:45", d["bbt_time"])
	assert.Equal(t, "unprotected", d["intercourse"])
	assert.Equal(t, []any{"ovarian_pain", "bloating"}, d["symptoms"], "known order, de-duplicated")
	assert.Equal(t, "صبح زود", d["note"])
	chance, _ := d["chance"].(map[string]any)
	require.NotNil(t, chance)
	assert.NotNil(t, chance["level"])
	assert.NotEmpty(t, chance["label"])

	// GET returns the same day; key order is the contract's.
	g := e.do(t, http.MethodGet, dayPath("2026-09-22"), tok, "fa", "")
	require.Equal(t, http.StatusOK, g.status, g.raw)
	assert.Equal(t, d, g.data())
	assert.Regexp(t, `^\{"success":true,"data":\{"date":.*"cycle_day":.*"lh":.*"mucus":.*"bbt":.*"bbt_time":.*"intercourse":.*"symptoms":.*"note":.*"chance":\{"level":.*"label":.*"bars":`, g.raw)

	// Stored where the README says.
	var lh, mucus, bbtTime string
	require.NoError(t, e.db.QueryRow(`SELECT lh_test, cervical_mucus, bbt_time FROM fertility_logs WHERE log_date = '2026-09-22'`).Scan(&lh, &mucus, &bbtTime))
	assert.Equal(t, []string{"positive", "egg_white", "06:45:00"}, []string{lh, mucus, bbtTime})
	var ovarian, bloating string
	var breast sql.NullString
	var spotting sql.NullBool
	require.NoError(t, e.db.QueryRow(`SELECT ovarian_pain_intensity, bloating_intensity, breast_sensitivity_intensity, spotting
		FROM daily_health_logs WHERE log_date = '2026-09-22'`).Scan(&ovarian, &bloating, &breast, &spotting))
	assert.Equal(t, "low", ovarian)
	assert.Equal(t, "low", bloating)
	assert.False(t, breast.Valid)
	assert.False(t, spotting.Valid)

	// A partial PUT leaves the other fields alone.
	r = e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "en", `{"lh":"faint"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Day saved", r.body["message"])
	d = r.data()
	assert.Equal(t, "faint", d["lh"])
	assert.Equal(t, "egg_white", d["mucus"])
	assert.Equal(t, "36.55", d["bbt"])
	assert.Equal(t, []any{"ovarian_pain", "bloating"}, d["symptoms"])
}

func TestDay_SymptomKeepsLoggedIntensity(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000502", "2026-09-10")
	r := e.do(t, http.MethodPost, "/api/v1/health-logs", tok, "fa", `{"log_date":"2026-09-20","ovarian_pain_intensity":"high"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)

	r = e.do(t, http.MethodPut, dayPath("2026-09-20"), tok, "fa", `{"symptoms":["ovarian_pain","breast_sensitivity"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var ovarian, breast string
	require.NoError(t, e.db.QueryRow(`SELECT ovarian_pain_intensity, breast_sensitivity_intensity FROM daily_health_logs WHERE log_date = '2026-09-20'`).Scan(&ovarian, &breast))
	assert.Equal(t, "high", ovarian, "an intensity logged through the full health log is kept")
	assert.Equal(t, "low", breast)
}

func TestDay_ExplicitNullClears(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000503", "2026-09-10")
	r := e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "fa", fullDay)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	r = e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "fa",
		`{"lh":null,"mucus":null,"bbt":null,"bbt_time":null,"intercourse":null,"symptoms":null,"note":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	for _, k := range []string{"lh", "mucus", "bbt", "bbt_time", "intercourse", "note"} {
		assert.Nil(t, d[k], k)
	}
	assert.Equal(t, []any{}, d["symptoms"])

	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM fertility_logs`).Scan(&n))
	assert.Zero(t, n, "an emptied fertility_logs row is deleted")
	var bbt, intercourse, notes, ovarian, bloating sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT basal_body_temperature, intercourse_type, notes, ovarian_pain_intensity, bloating_intensity
		FROM daily_health_logs WHERE log_date = '2026-09-22'`).Scan(&bbt, &intercourse, &notes, &ovarian, &bloating))
	for _, v := range []sql.NullString{bbt, intercourse, notes, ovarian, bloating} {
		assert.False(t, v.Valid)
	}

	// Empty string is null too (ConvertEmptyStringsToNull), and [] clears the symptoms.
	r = e.do(t, http.MethodPut, dayPath("2026-09-21"), tok, "fa", `{"note":"x","symptoms":["spotting"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPut, dayPath("2026-09-21"), tok, "fa", `{"note":"","symptoms":[]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["note"])
	assert.Equal(t, []any{}, r.data()["symptoms"])
}

// Spotting written here runs the same health-log side effects as POST /health-logs: two users
// with the same profile, one logs through each endpoint; warning, cycle history and profile
// (LMP, recalculation mark) must end up the same.
func TestDay_SpottingRunsHealthLogSideEffects(t *testing.T) {
	e := setup(t)
	idA, tokA := e.user(t, "09120000504", "2026-09-01")
	idB, tokB := e.user(t, "09120000505", "2026-09-01")

	// Cycle day 22 of 28: luteal spotting.
	a := e.do(t, http.MethodPut, dayPath("2026-09-22"), tokA, "fa", `{"symptoms":["spotting"]}`)
	require.Equal(t, http.StatusOK, a.status, a.raw)
	b := e.do(t, http.MethodPost, "/api/v1/health-logs", tokB, "fa", `{"log_date":"2026-09-22","spotting":true}`)
	require.Equal(t, http.StatusCreated, b.status, b.raw)

	require.NotNil(t, b.body["warning"])
	assert.Equal(t, b.body["warning"], a.body["warning"], "same luteal-spotting warning")

	state := func(userID uint64) []any {
		var version int
		var status, lmp string
		var histories int
		require.NoError(t, e.db.QueryRow(`SELECT calculation_version, calculation_status, last_period_start FROM user_profiles WHERE user_id = ?`, userID).
			Scan(&version, &status, &lmp))
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM cycle_histories WHERE user_id = ?`, userID).Scan(&histories))
		return []any{version, status, lmp[:10], histories}
	}
	assert.Equal(t, state(idB), state(idA))
	assert.Equal(t, 1, state(idA)[0], "recalculation marked")

	// A bleeding day logged the classic way after a fertility-logged day still starts a period.
	c := e.do(t, http.MethodPost, "/api/v1/health-logs", tokA, "fa", `{"log_date":"2026-09-23","bleeding_intensity":"medium"}`)
	require.Equal(t, http.StatusCreated, c.status, c.raw)
	assert.Equal(t, "2026-09-23", state(idA)[2])
}

func TestDay_LegacyHealthLogSeesWrites(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000506", "2026-09-10")
	r := e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "fa", `{"bbt":"36.4","intercourse":"protected","note":"n"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	g := e.do(t, http.MethodGet, "/api/v1/health-logs/2026-09-22", tok, "fa", "")
	require.Equal(t, http.StatusOK, g.status, g.raw)
	assert.Equal(t, "36.40", g.data()["basal_body_temperature"])
	assert.Equal(t, "protected", g.data()["intercourse_type"])
	assert.Equal(t, "n", g.data()["notes"])

	// LH/mucus only: no daily_health_logs row is created.
	r = e.do(t, http.MethodPut, dayPath("2026-09-21"), tok, "fa", `{"lh":"negative"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	g = e.do(t, http.MethodGet, "/api/v1/health-logs/2026-09-21", tok, "fa", "")
	assert.Equal(t, http.StatusNotFound, g.status)
}

func TestDay_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000507", "2026-09-10")

	r := e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "fa",
		`{"bbt":38.6,"lh":"maybe","symptoms":["ovarian_pain","headache"],"bbt_time":"7am","note":["x"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, "اطلاعات واردشده نامعتبر است", r.body["message"])
	errs := r.errors()
	assert.Equal(t, []any{"دما باید بین ۳۵٫۰۰ و ۳۸٫۵۰ درجه باشد."}, errs["bbt"])
	for _, k := range []string{"lh", "symptoms.1", "bbt_time", "note"} {
		assert.Contains(t, errs, k)
	}

	r = e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "en", `{"bbt":34.99}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "Validation failed", r.body["message"])
	assert.Equal(t, []any{"The temperature must be between 35.00 and 38.50 °C."}, r.errors()["bbt"])

	// Boundaries are accepted.
	r = e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "en", `{"bbt":38.5}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPut, dayPath("2026-09-22"), tok, "en", `{"bbt":35}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "35.00", r.data()["bbt"])

	// Future day.
	r = e.do(t, http.MethodPut, dayPath("2026-09-24"), tok, "en", `{"lh":"negative"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"You cannot log a future day."}, r.errors()["date"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM fertility_logs`).Scan(&n))
	assert.Zero(t, n)

	// A malformed date is a 422 on date; a future GET is fine (predicted chance).
	r = e.do(t, http.MethodGet, dayPath("22-09-2026"), tok, "fa", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "date")
	r = e.do(t, http.MethodGet, dayPath("2026-10-01"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["lh"])
}

func TestDay_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/fertility/today"},
		{http.MethodGet, dayPath(today)},
		{http.MethodPut, dayPath(today)},
	} {
		r := e.do(t, c.method, c.path, "", "fa", `{"lh":"positive"}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw)
	}
}

func TestDay_OtherUsersDataIsInvisible(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000508", "2026-09-10")
	_, tokB := e.user(t, "09120000509", "")
	r := e.do(t, http.MethodPut, dayPath(today), tokA, "fa", fullDay)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	g := e.do(t, http.MethodGet, dayPath(today), tokB, "fa", "")
	require.Equal(t, http.StatusOK, g.status, g.raw)
	d := g.data()
	assert.Nil(t, d["lh"])
	assert.Nil(t, d["bbt"])
	assert.Equal(t, []any{}, d["symptoms"])
	assert.Nil(t, d["cycle_day"], "no profile, no cycle")

	// B writing the same date does not touch A's rows.
	r = e.do(t, http.MethodPut, dayPath(today), tokB, "fa", `{"lh":null,"bbt":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	g = e.do(t, http.MethodGet, dayPath(today), tokA, "fa", "")
	assert.Equal(t, "positive", g.data()["lh"])
	assert.Equal(t, "36.55", g.data()["bbt"])
}

func TestToday(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000510", "2026-09-10")

	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, "/api/v1/fertility/today", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, int64(3), e.queries.n.Load(), "query budget: profile, histories, merged day")
	d := r.data()
	assert.Equal(t, today, d["date"])
	assert.EqualValues(t, 14, d["cycle_day"])
	assert.Equal(t, map[string]any{"value": nil, "label": nil}, d["lh"])
	assert.Equal(t, map[string]any{"value": nil}, d["bbt"])
	assert.Equal(t, map[string]any{"value": nil, "label": nil}, d["intercourse"])
	chance := d["chance"].(map[string]any)
	assert.Contains(t, []any{"high", "peak"}, chance["level"], "cycle day 14 of 28 is the fertile peak")
	assert.GreaterOrEqual(t, chance["bars"], 4.0)

	r = e.do(t, http.MethodPut, dayPath(today), tok, "fa", `{"lh":"positive","bbt":36.7,"intercourse":"unprotected"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/fertility/today", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, map[string]any{"value": "positive", "label": "مثبت"}, d["lh"])
	assert.Equal(t, map[string]any{"value": "36.70"}, d["bbt"])
	assert.Equal(t, map[string]any{"value": "unprotected", "label": "بدون محافظت"}, d["intercourse"])
	assert.Regexp(t, `^\{"success":true,"data":\{"date":.*"cycle_day":.*"chance":.*"lh":.*"bbt":.*"intercourse":`, r.raw)

	r = e.do(t, http.MethodGet, "/api/v1/fertility/today", tok, "en", "")
	assert.Equal(t, "Positive", r.data()["lh"].(map[string]any)["label"])

	// No profile: no cycle day, unknown chance.
	_, tokNone := e.user(t, "09120000511", "")
	r = e.do(t, http.MethodGet, "/api/v1/fertility/today", tokNone, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["cycle_day"])
	assert.Equal(t, map[string]any{"level": nil, "label": "Unknown", "bars": 0.0}, r.data()["chance"])
}
