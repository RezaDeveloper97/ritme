package pelvic_test

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
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/pelvic"
	pelvicstore "github.com/ritme/backend-go/internal/pelvic/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30" // a Wednesday
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db  *sql.DB
	app *fiber.App
	iss *passport.Issuer
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
	levels := catalog.NewReader(catalogstore.New(db), nil, 0, quiet) // no cache: admin edits show at once
	h := pelvic.NewHandlers(pelvic.NewService(pelvicstore.New(db), levels), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/pelvic", locale, guard, h.Show)
	app.Post("/api/v1/pelvic/program", locale, guard, h.StartProgram)
	app.Delete("/api/v1/pelvic/program", locale, guard, h.StopProgram)
	app.Post("/api/v1/pelvic/sessions", locale, guard, h.AddSession)
	app.Get("/api/v1/pelvic/diary/:date", locale, guard, h.ShowDiary)
	app.Put("/api/v1/pelvic/diary/:date", locale, guard, h.UpdateDiary)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
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

func (r response) program() map[string]any {
	p, _ := r.data()["program"].(map[string]any)
	return p
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

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/pelvic"},
		{http.MethodPost, "/api/v1/pelvic/program"},
		{http.MethodDelete, "/api/v1/pelvic/program"},
		{http.MethodPost, "/api/v1/pelvic/sessions"},
		{http.MethodGet, "/api/v1/pelvic/diary/2026-09-23"},
		{http.MethodPut, "/api/v1/pelvic/diary/2026-09-23"},
	} {
		r := e.do(t, c.method, c.path, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, c.path)
	}
}

func TestProgram_FlowWithSeededLevels(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000701")

	r := e.do(t, http.MethodGet, "/api/v1/pelvic", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["program"])
	assert.JSONEq(t, `{"date":"2026-09-23","leak":null,"night_voids":null,"uti_symptoms":[],"uti_alert":false}`,
		mustJSON(t, r.data()["diary_today"]))

	// A session without a program is refused.
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tok, "en", `{"sets_completed":3,"duration_sec":300}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, []any{"Start the pelvic floor program first."}, r.errors()["program"])

	// Started two weeks ago → week 3 → seeded level_2 (board: «هفته ۳ · سطح ۲», 5 minutes, hold 5 s × 10 × 3).
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/program", tok, "fa", `{"started_on":"2026-09-09"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "برنامهٔ تمرین کف لگن شروع شد", r.body["message"])
	p := r.program()
	assert.Equal(t, "2026-09-09", p["started_on"])
	assert.EqualValues(t, 3, p["week"])
	assert.EqualValues(t, 8, p["weeks_total"])
	assert.Equal(t, false, p["completed"])
	assert.EqualValues(t, 0, p["streak_days"])
	assert.Equal(t, false, p["today_done"])
	level, _ := p["level"].(map[string]any)
	require.NotNil(t, level, r.raw)
	assert.Equal(t, "level_2", level["code"])
	assert.EqualValues(t, 2, level["number"])
	assert.Equal(t, "سطح ۲", level["title"])
	assert.Contains(t, level["body"], "نفست را حبس نکن")
	assert.EqualValues(t, 5, level["hold_sec"])
	assert.EqualValues(t, 5, level["rest_sec"])
	assert.EqualValues(t, 10, level["reps"])
	assert.EqualValues(t, 3, level["sets"])
	assert.EqualValues(t, 300, level["session_sec"])
	assert.Equal(t, true, level["needs_review"])
	assert.Regexp(t, `^\{"success":true,"message":".*","data":\{"program":\{"started_on":.*"week":.*"weeks_total":.*"completed":.*"level":\{"code":.*"streak_days":.*"today_done":.*"week_days":\[.*\]\},"diary_today":\{`, r.raw)

	// Three trained days in a row, two sessions today.
	for _, d := range []string{"2026-09-21", "2026-09-22", "2026-09-23"} {
		r = e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tok, "en", `{"date":"`+d+`","sets_completed":3,"duration_sec":300}`)
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tok, "en", `{"sets_completed":2,"duration_sec":200}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Session saved", r.body["message"])
	p = r.program()
	assert.EqualValues(t, 3, p["streak_days"])
	assert.Equal(t, true, p["today_done"])
	assert.Equal(t, "Level 2", p["level"].(map[string]any)["title"])
	assert.Equal(t,
		`[{"date":"2026-09-19","done":false},{"date":"2026-09-20","done":false},{"date":"2026-09-21","done":true},{"date":"2026-09-22","done":true},{"date":"2026-09-23","done":true},{"date":"2026-09-24","done":false},{"date":"2026-09-25","done":false}]`,
		mustJSON(t, p["week_days"]))

	var count, sets, dur int
	var code string
	require.NoError(t, e.db.QueryRow(`SELECT sessions_count, sets_completed, duration_sec, level_code FROM pelvic_sessions WHERE session_date = '2026-09-23'`).
		Scan(&count, &sets, &dur, &code))
	assert.Equal(t, []any{2, 5, 500, "level_2"}, []any{count, sets, dur, code})

	// Validation.
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tok, "en", `{"date":"2026-09-24","sets_completed":-1}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "date")
	assert.Contains(t, r.errors(), "sets_completed")
	assert.Contains(t, r.errors(), "duration_sec")

	// Stop: the program goes, the trained days stay.
	r = e.do(t, http.MethodDelete, "/api/v1/pelvic/program", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Your pelvic floor program has stopped", r.body["message"])
	assert.Nil(t, r.data()["program"])
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pelvic_sessions`).Scan(&n))
	assert.Equal(t, 3, n)

	// Restart today: week 1, level_1, the streak is still there.
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/program", tok, "en", `{}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	p = r.program()
	assert.Equal(t, "2026-09-23", p["started_on"])
	assert.EqualValues(t, 1, p["week"])
	assert.Equal(t, "level_1", p["level"].(map[string]any)["code"])
	assert.EqualValues(t, 3, p["streak_days"])
}

func TestProgram_LevelsFollowTheCatalog(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000702")
	r := e.do(t, http.MethodPost, "/api/v1/pelvic/program", tok, "en", `{"started_on":"2026-07-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 8, r.program()["week"])
	assert.Equal(t, true, r.program()["completed"])
	assert.Equal(t, "level_4", r.program()["level"].(map[string]any)["code"])

	// An admin hides level 4 → the last level that has started is level 3.
	_, err := e.db.Exec(`UPDATE catalog_items SET is_active = 0 WHERE ` + "`group`" + ` = 'pelvic_levels' AND code = 'level_4'`)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/v1/pelvic", tok, "en", "")
	assert.Equal(t, "level_3", r.program()["level"].(map[string]any)["code"])

	// No valid level at all → level null, the program still works.
	_, err = e.db.Exec(`UPDATE catalog_items SET is_active = 0 WHERE ` + "`group`" + ` = 'pelvic_levels'`)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, "/api/v1/pelvic", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.program()["level"])
	r = e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tok, "en", `{"sets_completed":1,"duration_sec":60}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var code sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT level_code FROM pelvic_sessions`).Scan(&code))
	assert.False(t, code.Valid)
}

func diaryPath(date string) string { return "/api/v1/pelvic/diary/" + date }

func TestDiary_RoundTripAndClear(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000703")

	r := e.do(t, http.MethodPut, diaryPath("2026-09-22"), tok, "fa",
		`{"leak":"cough","night_voids":2,"uti_symptoms":["frequency","burning","burning"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "دفترچهٔ مثانه ذخیره شد", r.body["message"])
	assert.JSONEq(t, `{"date":"2026-09-22","leak":"cough","night_voids":2,"uti_symptoms":["burning","frequency"],"uti_alert":true}`,
		mustJSON(t, r.data()))

	g := e.do(t, http.MethodGet, diaryPath("2026-09-22"), tok, "fa", "")
	require.Equal(t, http.StatusOK, g.status, g.raw)
	assert.Equal(t, r.data(), g.data())
	assert.Regexp(t, `^\{"success":true,"data":\{"date":"2026-09-22","leak":"cough","night_voids":2,"uti_symptoms":\["burning","frequency"\],"uti_alert":true\}\}$`, g.raw)

	// Partial: an omitted key stays, null clears.
	r = e.do(t, http.MethodPut, diaryPath("2026-09-22"), tok, "en", `{"uti_symptoms":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"date":"2026-09-22","leak":"cough","night_voids":2,"uti_symptoms":[],"uti_alert":false}`, mustJSON(t, r.data()))

	// Everything cleared → the row is deleted.
	r = e.do(t, http.MethodPut, diaryPath("2026-09-22"), tok, "en", `{"leak":null,"night_voids":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pelvic_bladder_logs`).Scan(&n))
	assert.Zero(t, n)

	// Today's diary rides on GET /pelvic.
	r = e.do(t, http.MethodPut, diaryPath("2026-09-23"), tok, "en", `{"leak":"none","night_voids":0}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/pelvic", tok, "en", "")
	assert.JSONEq(t, `{"date":"2026-09-23","leak":"none","night_voids":0,"uti_symptoms":[],"uti_alert":false}`, mustJSON(t, r.data()["diary_today"]))
}

func TestDiary_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000704")

	r := e.do(t, http.MethodPut, diaryPath("2026-09-24"), tok, "en", `{"leak":"sometimes","night_voids":"x","uti_symptoms":["fever"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "Validation failed", r.body["message"])
	assert.Equal(t, []any{"You cannot log a future day."}, r.errors()["date"])
	for _, k := range []string{"leak", "night_voids", "uti_symptoms.0"} {
		assert.Contains(t, r.errors(), k)
	}
	r = e.do(t, http.MethodGet, diaryPath("23-09-2026"), tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "date")
}

// User B never sees or changes user A's program, trained days or diary.
func TestIsolationBetweenUsers(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000705")
	_, tokB := e.user(t, "09120000706")

	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, "/api/v1/pelvic/program", tokA, "en", `{"started_on":"2026-09-09"}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, "/api/v1/pelvic/sessions", tokA, "en", `{"sets_completed":3,"duration_sec":300}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, diaryPath("2026-09-23"), tokA, "en", `{"leak":"urgency"}`).status)

	r := e.do(t, http.MethodGet, "/api/v1/pelvic", tokB, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["program"])
	assert.Nil(t, r.data()["diary_today"].(map[string]any)["leak"])
	g := e.do(t, http.MethodGet, diaryPath("2026-09-23"), tokB, "en", "")
	assert.Nil(t, g.data()["leak"])

	// B's writes and stop touch only B's rows.
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, diaryPath("2026-09-23"), tokB, "en", `{"leak":null,"night_voids":1}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/pelvic/program", tokB, "en", "").status)
	r = e.do(t, http.MethodGet, "/api/v1/pelvic", tokA, "en", "")
	require.NotNil(t, r.program())
	assert.Equal(t, true, r.program()["today_done"])
	assert.Equal(t, "urgency", r.data()["diary_today"].(map[string]any)["leak"])
	assert.Nil(t, r.data()["diary_today"].(map[string]any)["night_voids"])

	// Account deletion cascades (health data leaves with the user).
	_, err := e.db.Exec(`DELETE FROM users WHERE id = ?`, a)
	require.NoError(t, err)
	for _, table := range []string{"pelvic_programs", "pelvic_sessions", "pelvic_bladder_logs"} {
		var n int
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, a).Scan(&n))
		assert.Zero(t, n, table)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}
