package menopause_test

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

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	authstore "github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/catalog"
	catalogstore "github.com/ritme/backend-go/internal/catalog/store"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/resources/translations"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-10-01T10:00:00+03:30" // 9 Mehr 1405
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
	cat := catalog.NewReader(catalogstore.New(db), nil, 0, quiet)
	h := menopause.NewHandlers(menopause.NewService(db, cat), i18n.NewTranslationStore(translations.FS, ""), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/menopause/profile", locale, guard, h.ShowProfile)
	app.Put("/api/v1/menopause/profile", locale, guard, h.SaveProfile)
	app.Get("/api/v1/menopause/today", locale, guard, h.Today)
	app.Get("/api/v1/menopause/hot-flashes", locale, guard, h.ListFlashes)
	app.Post("/api/v1/menopause/hot-flashes", locale, guard, h.StartFlash)
	app.Post("/api/v1/menopause/hot-flashes/:id/stop", locale, guard, h.StopFlash)
	app.Get("/api/v1/menopause/scores", locale, guard, h.Scores)
	app.Post("/api/v1/menopause/scores", locale, guard, h.SaveScore)
	app.Get("/api/v1/menopause/patterns", locale, guard, h.Patterns)
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

func (r response) errors() map[string]any {
	d, _ := r.body["errors"].(map[string]any)
	return d
}

func (e *env) doAt(t *testing.T, at, method, path, token, lang, body string) response {
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
	req.Header.Set(clock.Header, at)
	resp, err := e.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func (e *env) do(t *testing.T, method, path, token, lang, body string) response {
	t.Helper()
	return e.doAt(t, now, method, path, token, lang, body)
}

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

// logEntry writes one health_log_entries row (item "" for a single/bool param).
func (e *env) logEntry(t *testing.T, userID uint64, date, category, param, item, code string) {
	t.Helper()
	var v any
	if code != "" {
		v = code
	}
	e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'manual', '2026-10-01 09:00:00', '2026-10-01 09:00:00')`, userID, date, category, param, item, v)
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/menopause/profile"},
		{http.MethodPut, "/api/v1/menopause/profile"},
		{http.MethodGet, "/api/v1/menopause/today"},
		{http.MethodGet, "/api/v1/menopause/hot-flashes"},
		{http.MethodPost, "/api/v1/menopause/hot-flashes"},
		{http.MethodPost, "/api/v1/menopause/hot-flashes/1/stop"},
		{http.MethodGet, "/api/v1/menopause/scores"},
		{http.MethodPost, "/api/v1/menopause/scores"},
		{http.MethodGet, "/api/v1/menopause/patterns"},
	} {
		r := e.do(t, c.method, c.path, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, c.path)
	}
}

func TestProfile(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")

	r := e.do(t, http.MethodGet, "/api/v1/menopause/profile", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["needs_stage"])
	assert.Nil(t, r.data()["stage"])
	assert.Nil(t, r.data()["tip"])

	// bloom's life profile row with the mode set: the menopause PUT leaves the other columns alone
	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, gender, created_at, updated_at) VALUES (?, 'menopause', 'female', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)

	r = e.do(t, http.MethodPut, "/api/v1/menopause/profile", tok, "en",
		`{"stage":"unsure","last_period":"2025-08-01","surgical":false,"hrt":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "Your menopause stage was saved", r.body["message"])
	assert.Equal(t, "meno", d["stage"])
	assert.Equal(t, "unsure", d["stored_stage"])
	assert.Equal(t, "2025-08-01", d["last_period"])
	assert.Equal(t, 14, num(d["months_without_period"]))
	assert.Equal(t, false, d["surgical"])
	assert.Equal(t, true, d["hrt"])
	assert.Equal(t, true, d["post_menopausal"])
	tip, _ := d["tip"].(map[string]any)
	require.NotNil(t, tip)
	assert.Equal(t, "stage_meno", tip["code"])
	assert.Equal(t, true, tip["needs_review"])

	var mode, gender string
	require.NoError(t, e.db.QueryRow(`SELECT life_mode, gender FROM user_life_profiles WHERE user_id = ?`, uid).Scan(&mode, &gender))
	assert.Equal(t, "menopause", mode)
	assert.Equal(t, "female", gender)

	// partial: clear hrt, change the stage to peri → meno suggested
	r = e.do(t, http.MethodPut, "/api/v1/menopause/profile", tok, "fa", `{"stage":"peri","hrt":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "peri", d["stage"])
	assert.Equal(t, "meno", d["suggested_stage"])
	assert.Nil(t, d["hrt"])
	assert.Equal(t, "2025-08-01", d["last_period"])
	assert.Equal(t, false, d["post_menopausal"])

	r = e.do(t, http.MethodPut, "/api/v1/menopause/profile", tok, "en", `{"stage":"later","last_period":"2026-10-02","hrt":"maybe"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Contains(t, r.errors(), "stage")
	assert.Equal(t, []any{"You cannot log a future time."}, r.errors()["last_period"])
	assert.Contains(t, r.errors(), "hrt")
}

func TestHotFlashFlow(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000002")

	r := e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	f := r.data()
	id := num(f["id"])
	assert.Equal(t, true, f["running"])
	assert.Equal(t, "2026-10-01T10:00:00+03:30", f["started_at"])
	assert.Equal(t, false, f["night"])
	assert.Equal(t, "Hot flash timer started", r.body["message"])

	// a second tap while it runs returns the same timer
	r = e.doAt(t, "2026-10-01T10:01:00+03:30", http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, id, num(r.data()["id"]))
	assert.Equal(t, 60, num(r.data()["elapsed_s"]))

	path := fmt.Sprintf("/api/v1/menopause/hot-flashes/%d/stop", id)
	r = e.doAt(t, "2026-10-01T10:01:42+03:30", http.MethodPost, path, tok, "en",
		`{"severity":"severe","sweat":true,"triggers":["warm_room","caffeine","caffeine"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	f = r.data()
	assert.Equal(t, false, f["running"])
	assert.Equal(t, 102, num(f["duration_s"]))
	assert.Equal(t, "severe", f["severity"])
	assert.Equal(t, true, f["sweat"])
	assert.Equal(t, []any{"caffeine", "warm_room"}, f["triggers"])

	// editing a stopped flash keeps its duration
	r = e.doAt(t, "2026-10-01T10:30:00+03:30", http.MethodPost, path, tok, "en", `{"triggers":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 102, num(r.data()["duration_s"]))
	assert.Equal(t, []any{}, r.data()["triggers"])
	assert.Equal(t, "severe", r.data()["severity"])

	// a finished flash logged after the fact (night by its hour)
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en",
		`{"started_at":"2026-10-01 03:20","duration_s":180,"severity":"moderate","sweat":true}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "Hot flash logged", r.body["message"])
	assert.Equal(t, true, r.data()["night"])
	assert.Equal(t, "2026-10-01T03:20:00+03:30", r.data()["started_at"])

	r = e.do(t, http.MethodGet, "/api/v1/menopause/hot-flashes", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2026-10-01", d["date"])
	assert.Equal(t, 2, num(d["count"]))
	assert.Equal(t, 1, num(d["night_count"]))
	assert.Equal(t, 141, num(d["avg_duration_s"]))
	assert.Nil(t, d["running"])
	items, _ := d["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, id, num(items[0].(map[string]any)["id"]), "newest first")

	r = e.do(t, http.MethodGet, "/api/v1/menopause/hot-flashes?date=2026-09-30", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, num(r.data()["count"]))

	// validation
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{"severity":"extreme","triggers":["wine"],"duration_s":0}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "severity")
	assert.Contains(t, r.errors(), "triggers.0")
	assert.Contains(t, r.errors(), "duration_s")
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{"started_at":"2026-10-01 12:00","duration_s":60}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"You cannot log a future time."}, r.errors()["started_at"])
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{"started_at":"2026-09-20 12:00","duration_s":60}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "started_at")
	r = e.do(t, http.MethodGet, "/api/v1/menopause/hot-flashes?date=yesterday", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)

	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes/abc/stop", tok, "en", `{}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes/999999/stop", tok, "en", `{}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Equal(t, "Hot flash not found", r.body["message"])
}

func TestForgottenTimerIsClosed(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000003")
	r := e.doAt(t, "2026-10-01T07:00:00+03:30", http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	old := num(r.data()["id"])

	r = e.do(t, http.MethodGet, "/api/v1/menopause/hot-flashes", tok, "en", "")
	assert.Nil(t, r.data()["running"], "three hours later it is not shown as running")

	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.NotEqual(t, old, num(r.data()["id"]))
	var dur int
	require.NoError(t, e.db.QueryRow(`SELECT duration_s FROM hot_flashes WHERE id = ?`, old).Scan(&dur))
	assert.Equal(t, menopause.MaxFlashSeconds, dur)
}

var boardAnswers = `{"hot_flashes":3,"heart_discomfort":1,"sleep_problems":2,"joint_muscle":1,"depressive_mood":1,
	"irritability":1,"anxiety":1,"exhaustion":1,"sexual_problems":1,"bladder_problems":0,"vaginal_dryness":2}`

func TestScores(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000004")

	// Shahrivar first: 18
	r := e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"month":"2026-09-01","answers":{"hot_flashes":4,"heart_discomfort":2,"sleep_problems":3,"joint_muscle":2,"depressive_mood":1,
	"irritability":1,"anxiety":1,"exhaustion":1,"sexual_problems":1,"bladder_problems":0,"vaginal_dryness":2}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-08-23", r.data()["month"])
	assert.Equal(t, 18, num(r.data()["total"]))
	assert.Nil(t, r.data()["delta"])

	r = e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"answers":`+boardAnswers+`}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "This month's questionnaire was saved", r.body["message"])
	assert.Equal(t, "2026-09-23", d["month"])
	assert.Equal(t, 1405, num(d["jalali_year"]))
	assert.Equal(t, 7, num(d["jalali_month"]))
	assert.Equal(t, 14, num(d["total"]))
	assert.Equal(t, 44, num(d["max"]))
	assert.Equal(t, -4, num(d["delta"]))
	assert.Equal(t, "2026-08-23", d["previous_month"])
	band, _ := d["band"].(map[string]any)
	assert.Equal(t, "moderate", band["code"])
	assert.Equal(t, "Moderate", band["title"])
	assert.Equal(t, []any{
		map[string]any{"code": "somatic", "score": 7.0, "max": 16.0},
		map[string]any{"code": "psychological", "score": 4.0, "max": 16.0},
		map[string]any{"code": "urogenital", "score": 3.0, "max": 12.0},
	}, d["domains"])

	// refilling replaces
	r = e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"month":"2026-10-01","answers":`+strings.Replace(boardAnswers, `"bladder_problems":0`, `"bladder_problems":1`, 1)+`}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 15, num(r.data()["total"]))
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM menopause_scores WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 2, n)

	r = e.do(t, http.MethodGet, "/api/v1/menopause/scores", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, 6, num(d["months"]))
	trend, _ := d["trend"].([]any)
	require.Len(t, trend, 6)
	assert.Equal(t, "2026-04-21", trend[0].(map[string]any)["month"]) // 1 Ordibehesht
	assert.Nil(t, trend[0].(map[string]any)["total"])
	assert.Equal(t, 18, num(trend[4].(map[string]any)["total"]))
	assert.Equal(t, "severe", trend[4].(map[string]any)["band"])
	assert.Equal(t, 15, num(trend[5].(map[string]any)["total"]))
	latest, _ := d["latest"].(map[string]any)
	assert.Equal(t, -3, num(latest["delta"]))
	assert.Len(t, d["items"], 2)
	assert.Len(t, d["bands"], 4)
	assert.Equal(t, "متوسط", latest["band"].(map[string]any)["title"])
	assert.Nil(t, d["hrt"])

	// HRT started in Shahrivar: the annotation compares with the score before it (none here)
	e.exec(t, `INSERT INTO treatment_items (user_id, kind, name, started_on, sort_order, created_at, updated_at)
		VALUES (?, 'hrt', 'Estradiol gel', '2026-09-05', 0, '2026-09-05 09:00:00', '2026-09-05 09:00:00')`, uid)
	r = e.do(t, http.MethodGet, "/api/v1/menopause/scores?months=2", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	hrt, _ := r.data()["hrt"].(map[string]any)
	require.NotNil(t, hrt)
	assert.Equal(t, "2026-09-05", hrt["started_on"])
	assert.Nil(t, hrt["baseline_total"])
	assert.Equal(t, 15, num(hrt["latest_total"]))
	assert.Nil(t, hrt["change"])
	assert.Len(t, r.data()["trend"], 2)

	// validation
	r = e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"answers":{"hot_flashes":5,"wine":1}}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "answers.hot_flashes")
	assert.Contains(t, r.errors(), "answers.anxiety")
	r = e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en",
		`{"answers":`+strings.Replace(boardAnswers, `{`, `{"wine":1,`, 1)+`}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "answers.wine")
	r = e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"month":"2026-11-01","answers":`+boardAnswers+`}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "month")
	r = e.do(t, http.MethodGet, "/api/v1/menopause/scores?months=25", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

func TestToday(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000005")

	r := e.do(t, http.MethodGet, "/api/v1/menopause/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2026-10-01", d["date"])
	assert.Equal(t, true, d["profile"].(map[string]any)["needs_stage"])
	assert.Equal(t, 0, num(d["hot_flashes"].(map[string]any)["count"]))
	assert.Nil(t, d["sleep"])
	assert.Nil(t, d["score"].(map[string]any)["latest"])
	assert.Len(t, d["score"].(map[string]any)["trend"], 6)
	assert.Equal(t, []any{}, d["treatment"])
	assert.IsType(t, []any{}, d["checkups"])
	assert.Equal(t, false, d["bleeding"].(map[string]any)["alert"])

	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, menopause_stage, menopause_last_period, created_at, updated_at)
		VALUES (?, 'menopause', 'meno', '2025-08-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	// last night: a sweaty flash at 03:20; today: sleep 3–6 h, night sweats logged; spotting 10 days ago
	e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{"started_at":"2026-10-01 03:20","duration_s":180,"sweat":true}`)
	e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tok, "en", `{"started_at":"2026-10-01 09:00","duration_s":120}`)
	e.logEntry(t, uid, "2026-10-01", "sleep", "duration", "", "3_6")
	e.logEntry(t, uid, "2026-10-01", "symptoms", "general", "night_sweats", "moderate")
	e.logEntry(t, uid, "2026-09-21", "bleeding", "presence", "", "spotting")
	e.logEntry(t, uid, "2026-09-25", "bleeding", "presence", "", "none")
	// HRT taken 6 of the last 7 days
	e.exec(t, `INSERT INTO treatment_items (id, user_id, kind, name, dose, schedule, started_on, review_on, sort_order, created_at, updated_at)
		VALUES (900001, ?, 'hrt', 'Estradiol gel', '1 pump', 'morning', '2026-08-01', '2026-12-01', 0, '2026-08-01 09:00:00', '2026-08-01 09:00:00')`, uid)
	for i := range 7 {
		if i == 3 {
			continue
		}
		d := civildate.MustParse("2026-10-01").AddDays(-i)
		e.exec(t, `INSERT INTO treatment_intakes (user_id, treatment_item_id, intake_date, taken_at, created_at, updated_at)
			VALUES (?, 900001, ?, ?, '2026-10-01 09:00:00', '2026-10-01 09:00:00')`, uid, d, d.String()+" 08:00:00")
	}
	e.do(t, http.MethodPost, "/api/v1/menopause/scores", tok, "en", `{"answers":`+boardAnswers+`}`)

	r = e.do(t, http.MethodGet, "/api/v1/menopause/today", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	p := d["profile"].(map[string]any)
	assert.Equal(t, "meno", p["stage"])
	assert.Equal(t, 14, num(p["months_without_period"]))
	hf := d["hot_flashes"].(map[string]any)
	assert.Equal(t, 2, num(hf["count"]))
	assert.Equal(t, 1, num(hf["night_count"]))
	assert.Equal(t, 150, num(hf["avg_duration_s"]))
	assert.Equal(t, map[string]any{"count": 1.0, "level": "moderate"}, d["night_sweats"])
	assert.Equal(t, map[string]any{"date": "2026-10-01", "code": "3_6", "hours": 4.5}, d["sleep"])
	sc := d["score"].(map[string]any)
	assert.Equal(t, 14, num(sc["latest"].(map[string]any)["total"]))
	tr := d["treatment"].([]any)
	require.Len(t, tr, 1)
	item := tr[0].(map[string]any)
	assert.Equal(t, true, item["taken_today"])
	assert.Equal(t, 6, num(item["days_taken"]))
	assert.Equal(t, 7, num(item["days"]))
	assert.Equal(t, 86, num(item["adherence_pct"]))
	assert.Equal(t, "2026-12-01", item["review_on"])
	b := d["bleeding"].(map[string]any)
	assert.Equal(t, true, b["alert"])
	assert.Equal(t, "2026-09-21", b["last_on"])
	alert, _ := b["alert_item"].(map[string]any)
	require.NotNil(t, alert)
	assert.Equal(t, "postmenopausal_bleeding", alert["code"])

	// perimenopause: the same spotting is no alert
	e.exec(t, `UPDATE user_life_profiles SET menopause_stage = 'peri', menopause_last_period = '2026-07-01' WHERE user_id = ?`, uid)
	r = e.do(t, http.MethodGet, "/api/v1/menopause/today", tok, "en", "")
	b = r.data()["bleeding"].(map[string]any)
	assert.Equal(t, false, b["alert"])
	assert.Equal(t, "2026-09-21", b["last_on"])
	assert.Nil(t, b["alert_item"])
}

func TestPatterns(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000006")

	r := e.do(t, http.MethodGet, "/api/v1/menopause/patterns", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, true, d["not_a_diagnosis"])
	assert.Equal(t, 0, num(d["found"]))
	assert.Equal(t, 20, num(d["min_days"]))
	disc, _ := d["disclaimer"].(map[string]any)
	require.NotNil(t, disc)
	assert.Equal(t, "patterns_disclaimer", disc["code"])
	items := d["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "not_enough_data", items[0].(map[string]any)["status"])

	// 30 logged days: caffeine every third day with 4 flashes, 1 flash otherwise
	for i := range 30 {
		day := civildate.MustParse("2026-10-01").AddDays(-i)
		count := 1
		if i%3 == 0 {
			e.logEntry(t, uid, day.String(), "menopause", "triggers", "caffeine", "")
			count = 4
		} else {
			e.logEntry(t, uid, day.String(), "symptoms", "general", "fatigue", "no")
		}
		for k := range count {
			e.exec(t, `INSERT INTO hot_flashes (user_id, started_at, duration_s, night, sweat, created_at, updated_at)
				VALUES (?, ?, 120, 0, 0, '2026-10-01 09:00:00', '2026-10-01 09:00:00')`, uid, fmt.Sprintf("%s %02d:00:00", day, 10+k))
		}
	}
	r = e.do(t, http.MethodGet, "/api/v1/menopause/patterns", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, 30, num(d["days_logged"]))
	assert.Equal(t, 1, num(d["found"]))
	caf := d["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "trigger_flashes", caf["key"])
	assert.Equal(t, "caffeine", caf["trigger"])
	assert.Equal(t, true, caf["found"])
	assert.Equal(t, "more", caf["direction"])
	assert.Equal(t, true, caf["needs_review"])
	assert.Equal(t, true, caf["not_a_diagnosis"])
	assert.Contains(t, caf["text"], "you had more hot flashes")
	assert.NotContains(t, caf["text"], ":trigger")
	assert.NotContains(t, caf["text"], "caffeine", "the trigger reads as its log-taxonomy label")

	r = e.do(t, http.MethodGet, "/api/v1/menopause/patterns", tok, "fa", "")
	caf = r.data()["items"].([]any)[0].(map[string]any)
	assert.Contains(t, caf["text"], "گرگرفتگی بیشتری داشتی")
}

// User A's rows never reach user B: flashes, stop, scores, profile, today, patterns.
func TestUserIsolation(t *testing.T) {
	e := setup(t)
	uidA, tokA := e.user(t, "09120000007")
	_, tokB := e.user(t, "09120000008")

	e.do(t, http.MethodPut, "/api/v1/menopause/profile", tokA, "en", `{"stage":"post","hrt":true}`)
	r := e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tokA, "en", `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	idA := num(r.data()["id"])
	e.do(t, http.MethodPost, "/api/v1/menopause/scores", tokA, "en", `{"answers":`+boardAnswers+`}`)
	e.logEntry(t, uidA, "2026-09-30", "bleeding", "presence", "", "bleeding")

	// B cannot stop (or edit) A's flash
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/menopause/hot-flashes/%d/stop", idA), tokB, "en", `{"severity":"mild"}`)
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	var running bool
	require.NoError(t, e.db.QueryRow(`SELECT duration_s IS NULL AND severity IS NULL FROM hot_flashes WHERE id = ?`, idA).Scan(&running))
	assert.True(t, running)

	r = e.do(t, http.MethodGet, "/api/v1/menopause/hot-flashes", tokB, "en", "")
	assert.Equal(t, 0, num(r.data()["count"]))
	assert.Nil(t, r.data()["running"])
	// B's start does not pick up A's running timer
	r = e.do(t, http.MethodPost, "/api/v1/menopause/hot-flashes", tokB, "en", `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.NotEqual(t, idA, num(r.data()["id"]))

	r = e.do(t, http.MethodGet, "/api/v1/menopause/scores", tokB, "en", "")
	assert.Nil(t, r.data()["latest"])
	assert.Equal(t, []any{}, r.data()["items"])

	r = e.do(t, http.MethodGet, "/api/v1/menopause/profile", tokB, "en", "")
	assert.Nil(t, r.data()["stage"])

	r = e.do(t, http.MethodGet, "/api/v1/menopause/today", tokB, "en", "")
	d := r.data()
	assert.Equal(t, 1, num(d["hot_flashes"].(map[string]any)["count"]), "only her own timer")
	assert.Nil(t, d["score"].(map[string]any)["latest"])
	assert.Equal(t, false, d["bleeding"].(map[string]any)["alert"])
	assert.Nil(t, d["bleeding"].(map[string]any)["last_on"])

	r = e.do(t, http.MethodGet, "/api/v1/menopause/patterns", tokB, "en", "")
	assert.Equal(t, 0, num(r.data()["days_logged"]))

	// A still sees her own
	r = e.do(t, http.MethodGet, "/api/v1/menopause/today", tokA, "en", "")
	assert.Equal(t, true, r.data()["bleeding"].(map[string]any)["alert"])
}
