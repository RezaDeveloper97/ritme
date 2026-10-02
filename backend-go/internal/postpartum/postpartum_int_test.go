package postpartum_test

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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/postpartum"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-10-01T10:00:00+03:30"
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
	h := postpartum.NewHandlers(db, postpartum.NewService(db), clock.Real{})
	mh := messages.NewHandlers(db, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/postpartum", locale, guard, h.Show)
	app.Post("/api/v1/postpartum/activate", locale, guard, h.Activate)
	app.Get("/api/v1/postpartum/recovery", locale, guard, h.ShowRecovery)
	app.Put("/api/v1/postpartum/recovery", locale, guard, h.SaveRecovery)
	app.Get("/api/v1/postpartum/epds/questions", locale, guard, h.Questions)
	app.Get("/api/v1/postpartum/epds", locale, guard, h.History)
	app.Post("/api/v1/postpartum/epds", locale, guard, h.SaveCheck)
	app.Get("/api/v1/messages/daily", locale, guard, mh.Daily)
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

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

func (e *env) scalar(t *testing.T, query string, args ...any) string {
	t.Helper()
	var v sql.NullString
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&v))
	return v.String
}

// activated is a user in postpartum mode with a birth on birth (direct activation).
func (e *env) activated(t *testing.T, mobile, birth string) (uint64, string) {
	t.Helper()
	id, tok := e.user(t, mobile)
	r := e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en", `{"birth_date":"`+birth+`"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	return id, tok
}

func TestPostpartum_401(t *testing.T) {
	e := setup(t)
	for _, rt := range []struct{ m, p string }{
		{http.MethodGet, "/api/v1/postpartum"}, {http.MethodPost, "/api/v1/postpartum/activate"},
		{http.MethodGet, "/api/v1/postpartum/recovery"}, {http.MethodPut, "/api/v1/postpartum/recovery"},
		{http.MethodGet, "/api/v1/postpartum/epds/questions"}, {http.MethodGet, "/api/v1/postpartum/epds"},
		{http.MethodPost, "/api/v1/postpartum/epds"},
	} {
		r := e.do(t, rt.m, rt.p, "", "", "{}")
		assert.Equal(t, http.StatusUnauthorized, r.status, rt.p)
		assert.Equal(t, "unauthenticated", r.body["error_code"], rt.p)
	}
}

func TestPostpartum_ShowOutsideTheMode(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, "/api/v1/postpartum", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, false, d["active"])
	assert.Equal(t, "cycle", d["mode"])
	assert.Nil(t, d["profile"])
	assert.Nil(t, d["status"])
	assert.Equal(t, "کی فوراً تماس بگیرم؟", d["call_when"].(map[string]any)["title"])

	// recovery and a check need the mode
	r = e.do(t, http.MethodGet, "/api/v1/postpartum/recovery", tok, "en", "")
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, postpartum.ErrorCodeNotActive, r.body["error_code"])
	r = e.do(t, http.MethodPost, "/api/v1/postpartum/epds", tok, "en", `{"kind":"short","answers":{"q3":0,"q4":0,"q5":0}}`)
	assert.Equal(t, http.StatusConflict, r.status)
}

func TestPostpartum_ActivateFromPregnancyClosesItAsDelivered(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000002")
	e.exec(t, `INSERT INTO user_profiles (user_id, user_goal, pregnancy_intention, created_at, updated_at) VALUES (?, 'non_ttc', 'pregnant', NOW(), NOW())`, id)
	e.exec(t, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, onboarding_completed, created_at, updated_at) VALUES (?, 1, 0, 1, NOW(), NOW())`, id)
	pid := e.scalar(t, `SELECT id FROM pregnancy_profiles WHERE user_id = ?`, id)

	r := e.do(t, http.MethodGet, "/api/v1/postpartum", tok, "en", "")
	assert.Equal(t, "pregnancy", r.data()["mode"])

	r = e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en",
		`{"birth_date":"2026-09-14","delivery_type":"vaginal","baby_count":1}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Postpartum mode is on", r.body["message"])
	d := r.data()
	assert.Equal(t, true, d["active"])
	assert.Equal(t, false, d["setup_required"])
	p := d["profile"].(map[string]any)
	assert.Equal(t, "2026-09-14", p["birth_date"])
	assert.Equal(t, "vaginal", p["delivery_type"])
	assert.Equal(t, "Vaginal birth", p["delivery_type_label"])
	assert.Equal(t, postpartum.SourcePregnancy, p["source"])
	assert.NotNil(t, p["pregnancy_closed_at"])
	s := d["status"].(map[string]any)
	assert.EqualValues(t, 17, s["days_since_birth"]) // nbl_v15_Main: «۲ هفته و ۳ روز»
	assert.EqualValues(t, 2, s["weeks"])
	assert.EqualValues(t, 3, s["days"])
	assert.EqualValues(t, 3, s["week"])
	assert.Equal(t, "puerperium", s["phase"])
	assert.EqualValues(t, 25, s["puerperium_days_left"])
	assert.Equal(t, "Bleeding gets lighter", d["week_tip"].(map[string]any)["title"])
	assert.Equal(t, "full", d["checkin"].(map[string]any)["due"], "day 17, no full check yet")

	// the pregnancy is closed (not deleted), the mode and the legacy goal follow
	assert.Equal(t, "0", e.scalar(t, `SELECT pregnancy_mode FROM pregnancy_profiles WHERE id = ?`, pid))
	assert.Equal(t, pid, e.scalar(t, `SELECT pregnancy_profile_id FROM postpartum_profiles WHERE user_id = ?`, id))
	assert.Equal(t, "postpartum", e.scalar(t, `SELECT life_mode FROM user_life_profiles WHERE user_id = ?`, id))
	assert.Equal(t, "", e.scalar(t, `SELECT pregnancy_intention FROM user_profiles WHERE user_id = ?`, id))

	// activating again edits the birth (one row per user), the closed pregnancy stays recorded
	r = e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en", `{"birth_date":"2026-09-15","delivery_type":"cesarean"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "1", e.scalar(t, `SELECT COUNT(*) FROM postpartum_profiles WHERE user_id = ?`, id))
	assert.Equal(t, "cesarean", r.data()["profile"].(map[string]any)["delivery_type"])
	assert.Equal(t, postpartum.SourceDirect, r.data()["profile"].(map[string]any)["source"],
		"no active pregnancy any more: a later activation is direct")
}

func TestPostpartum_ActivateDirectAndValidation(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000003")
	for body, field := range map[string]string{
		`{}`:                          "birth_date",
		`{"birth_date":"2026-10-02"}`: "birth_date",
		`{"birth_date":"2025-09-30"}`: "birth_date",
		`{"birth_date":"14-09-2026"}`: "birth_date",
		`{"birth_date":"2026-09-14","delivery_type":"other"}`: "delivery_type",
		`{"birth_date":"2026-09-14","baby_count":5}`:          "baby_count",
	} {
		r := e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en", body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, body)
		assert.Equal(t, false, r.body["success"], body)
		assert.Contains(t, r.errors(), field, body)
	}
	r := e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "fa", `{"birth_date":"2026-10-02"}`)
	assert.Equal(t, "تاریخ زایمان نمی‌تواند در آینده باشد.", r.errors()["birth_date"].([]any)[0]) //nolint:staticcheck // ST1018: Persian copy (ZWNJ) verbatim

	r = e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en", `{"birth_date":"2026-10-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	p := r.data()["profile"].(map[string]any)
	assert.Equal(t, postpartum.SourceDirect, p["source"])
	assert.Nil(t, p["delivery_type"])
	assert.EqualValues(t, 1, p["baby_count"])
	assert.Equal(t, "short", r.data()["checkin"].(map[string]any)["due"], "first two weeks: the short check")
	assert.Equal(t, "", e.scalar(t, `SELECT pregnancy_profile_id FROM postpartum_profiles WHERE user_id = ?`, id))
}

func TestPostpartum_ModeWithoutBirthNeedsSetup(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000004")
	e.exec(t, `INSERT INTO user_life_profiles (user_id, life_mode, created_at, updated_at) VALUES (?, 'postpartum', NOW(), NOW())`, id)
	r := e.do(t, http.MethodGet, "/api/v1/postpartum", tok, "en", "")
	d := r.data()
	assert.Equal(t, true, d["active"])
	assert.Equal(t, true, d["setup_required"])
	assert.Nil(t, d["status"])
	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", `{"lochia_amount":"light"}`)
	assert.Equal(t, http.StatusConflict, r.status)
}

func TestPostpartum_CompanionAccountRefused(t *testing.T) {
	e := setup(t)
	id, tok := e.user(t, "09120000005")
	e.exec(t, `INSERT INTO user_life_profiles (user_id, gender, created_at, updated_at) VALUES (?, 'male', NOW(), NOW())`, id)
	r := e.do(t, http.MethodPost, "/api/v1/postpartum/activate", tok, "en", `{"birth_date":"2026-09-14"}`)
	assert.Equal(t, http.StatusConflict, r.status)
	assert.Equal(t, "companion_account", r.body["error_code"])
	assert.Equal(t, "0", e.scalar(t, `SELECT COUNT(*) FROM postpartum_profiles WHERE user_id = ?`, id))
}

func TestPostpartum_RecoveryLogOnTheTaxonomy(t *testing.T) {
	e := setup(t)
	id, tok := e.activated(t, "09120000006", "2026-09-14")

	r := e.do(t, http.MethodGet, "/api/v1/postpartum/recovery", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2026-10-01", d["date"])
	assert.Nil(t, d["lochia_amount"])
	assert.Nil(t, d["breasts"])
	assert.Equal(t, []any{}, d["pain_locations"])

	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", `{"lochia_amount":"heavy","lochia_color":"pink_brown",
		"pain_level":"moderate","pain_locations":["stitches","back"],"breasts":["nipple_pain"],"feeds_count":8,"sleep_hours":4.5}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "heavy", d["lochia_amount"])
	assert.Equal(t, "pink_brown", d["lochia_color"])
	assert.Equal(t, "moderate", d["pain_level"])
	assert.Equal(t, []any{"stitches", "back"}, d["pain_locations"])
	assert.Equal(t, []any{"nipple_pain"}, d["breasts"])
	assert.EqualValues(t, 8, d["feeds_count"])
	assert.EqualValues(t, 4.5, d["sleep_hours"])
	alerts := d["alerts"].([]any)
	require.Len(t, alerts, 1)
	a := alerts[0].(map[string]any)
	assert.Equal(t, "heavy_bleeding", a["key"])
	assert.Equal(t, "115", a["action"].(map[string]any)["number"])

	// stored as taxonomy v2 slots (the postpartum log sheet reads the same rows)
	slot := func(cat, param, item string) string {
		return e.scalar(t, `SELECT COALESCE(value_code, value_num) FROM health_log_entries
			WHERE user_id = ? AND log_date = '2026-10-01' AND category = ? AND param = ? AND item = ?`, id, cat, param, item)
	}
	assert.Equal(t, "heavy", slot("bleeding", "lochia_amount", ""))
	assert.Equal(t, "moderate", slot("pain", "location", "stitches"))
	assert.Equal(t, "no", slot("breasts", "symptoms", "engorgement"))
	assert.Equal(t, "yes", slot("breasts", "symptoms", "nipple_pain"))
	assert.Equal(t, "8.00", slot("baby", "feeds_count", ""))
	assert.Equal(t, "4.50", slot("sleep", "hours", ""))
	assert.Equal(t, "3_6", slot("sleep", "duration", ""), "the bucket follows for the analysis engine")

	// partial: keys not sent stay, null clears, [] = normal breasts, pain none clears the locations
	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", `{"lochia_amount":"light","sleep_hours":null,"breasts":[],"pain_level":"none"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "light", d["lochia_amount"])
	assert.Equal(t, "pink_brown", d["lochia_color"])
	assert.Nil(t, d["sleep_hours"])
	assert.Equal(t, []any{}, d["breasts"])
	assert.Equal(t, "none", d["pain_level"])
	assert.Equal(t, []any{}, d["pain_locations"])
	assert.Equal(t, []any{}, d["alerts"])
	assert.Equal(t, "0", e.scalar(t, `SELECT COUNT(*) FROM health_log_entries WHERE user_id = ? AND category = 'sleep'`, id))

	// a past day by date
	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", `{"date":"2026-09-30","feeds_count":0}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/postpartum/recovery?date=2026-09-30", tok, "en", "")
	assert.EqualValues(t, 0, r.data()["feeds_count"])
}

func TestPostpartum_RecoveryValidation(t *testing.T) {
	e := setup(t)
	_, tok := e.activated(t, "09120000007", "2026-09-14")
	for body, field := range map[string]string{
		`{"lochia_amount":"gushing"}`:                     "lochia_amount",
		`{"pain_level":"severe"}`:                         "pain_locations",
		`{"pain_level":"mild","pain_locations":["knee"]}`: "pain_locations.0",
		`{"feeds_count":31}`:                              "feeds_count",
		`{"sleep_hours":25}`:                              "sleep_hours",
		`{"sleep_hours":4.2}`:                             "sleep_hours",
		`{"date":"2026-10-02","lochia_amount":"light"}`:   "date",
	} {
		r := e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, body+" "+r.raw)
		assert.Contains(t, r.errors(), field, body)
	}
}

func epds(answers map[string]int) string {
	b, _ := json.Marshal(answers)
	return string(b)
}

func fullAnswers(q10 int, rest ...int) map[string]int {
	m := map[string]int{"q10": q10}
	for i := 1; i <= 9; i++ {
		m[fmt.Sprintf("q%d", i)] = rest[i-1]
	}
	return m
}

func TestPostpartum_EPDSQuestions(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000008")
	r := e.do(t, http.MethodGet, "/api/v1/postpartum/epds/questions", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "short", d["kind"])
	assert.EqualValues(t, 9, d["max"])
	items := d["items"].([]any)
	require.Len(t, items, 3)
	q4 := items[1].(map[string]any)
	assert.Equal(t, "q4", q4["code"])
	assert.EqualValues(t, 4, q4["number"])
	assert.Equal(t, "بی‌دلیل احساس نگرانی یا اضطراب داشته‌ام", q4["text"]) //nolint:staticcheck // ST1018: Persian copy (ZWNJ) verbatim
	assert.NotEmpty(t, d["disclaimer"])

	r = e.do(t, http.MethodGet, "/api/v1/postpartum/epds/questions?kind=full", tok, "en", "")
	items = r.data()["items"].([]any)
	require.Len(t, items, 10)
	q10 := items[9].(map[string]any)
	opts := q10["options"].([]any)
	require.Len(t, opts, 4)
	assert.EqualValues(t, 3, opts[0].(map[string]any)["score"])
	assert.Equal(t, "Never", opts[3].(map[string]any)["label"])
	assert.EqualValues(t, 0, opts[3].(map[string]any)["score"])

	r = e.do(t, http.MethodGet, "/api/v1/postpartum/epds/questions?kind=long", tok, "en", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
}

func TestPostpartum_EPDSSafetyPath(t *testing.T) {
	e := setup(t)
	id, tok := e.activated(t, "09120000009", "2026-09-01")
	post := func(kind string, answers map[string]int) response {
		return e.do(t, http.MethodPost, "/api/v1/postpartum/epds", tok, "en", `{"kind":"`+kind+`","answers":`+epds(answers)+`}`)
	}

	// Q10 positive at a low total → urgent with the call actions
	r := post("full", fullAnswers(1, 0, 0, 0, 0, 0, 0, 0, 0, 0))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	chk := d["check"].(map[string]any)
	assert.EqualValues(t, 1, chk["total"])
	assert.Equal(t, true, chk["urgent"])
	assert.Equal(t, true, chk["self_harm"])
	assert.Equal(t, "low", chk["band"])
	assert.EqualValues(t, 5, chk["week"])
	safety := d["safety"].(map[string]any)
	assert.Equal(t, "urgent", safety["level"])
	assert.Equal(t, []any{"self_harm"}, safety["reasons"])
	var numbers []any
	for _, a := range safety["actions"].([]any) {
		assert.Equal(t, "call", a.(map[string]any)["type"])
		assert.NotEmpty(t, a.(map[string]any)["label"])
		numbers = append(numbers, a.(map[string]any)["number"])
	}
	assert.Equal(t, []any{"115", "123", "1480"}, numbers)

	// the same day again replaces the check; total 13 alone is urgent too
	r = post("full", fullAnswers(0, 2, 2, 2, 1, 1, 1, 1, 1, 2))
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.EqualValues(t, 13, d["check"].(map[string]any)["total"])
	assert.Equal(t, "likely", d["check"].(map[string]any)["band"])
	assert.Equal(t, false, d["check"].(map[string]any)["self_harm"])
	assert.Equal(t, []any{"score"}, d["safety"].(map[string]any)["reasons"])
	assert.Equal(t, "1", e.scalar(t, `SELECT COUNT(*) FROM epds_checks WHERE user_id = ?`, id))
	assert.Nil(t, d["checkin"].(map[string]any)["due"])

	// 10–12 → advice without call actions; a short check ≥ 6 → follow-up to the full one
	r = post("short", map[string]int{"q3": 2, "q4": 2, "q5": 2})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "follow_up", d["safety"].(map[string]any)["level"])
	assert.Equal(t, "full", d["follow_up"])
	assert.Nil(t, d["check"].(map[string]any)["self_harm"])
	assert.Nil(t, d["checkin"].(map[string]any)["due"], "a full check was already taken today (the schedule tests cover the elevated-short rule)")

	// low short → no safety message
	r = post("short", map[string]int{"q3": 0, "q4": 1, "q5": 0})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["safety"])

	// the answers are stored, the history shows totals only, newest first
	var stored string
	require.NoError(t, e.db.QueryRow(`SELECT answers FROM epds_checks WHERE user_id = ? AND kind = 'full'`, id).Scan(&stored))
	assert.JSONEq(t, `{"q1":2,"q2":2,"q3":2,"q4":1,"q5":1,"q6":1,"q7":1,"q8":1,"q9":2,"q10":0}`, stored)
	h := e.do(t, http.MethodGet, "/api/v1/postpartum/epds", tok, "en", "")
	require.Equal(t, http.StatusOK, h.status, h.raw)
	checks := h.data()["checks"].([]any)
	require.Len(t, checks, 2)
	assert.NotContains(t, h.raw, "answers")
	assert.NotContains(t, h.raw, "q10")
}

func TestPostpartum_EPDSValidation(t *testing.T) {
	e := setup(t)
	_, tok := e.activated(t, "09120000010", "2026-09-01")
	for body, field := range map[string]string{
		`{}`:                             "kind",
		`{"kind":"weekly","answers":{}}`: "kind",
		`{"kind":"short"}`:               "answers",
		`{"kind":"short","answers":{"q3":1,"q4":1}}`:                "answers.q5",
		`{"kind":"short","answers":{"q3":1,"q4":1,"q5":4}}`:         "answers.q5",
		`{"kind":"short","answers":{"q3":1,"q4":1,"q5":1,"q10":2}}`: "answers.q10",
		`{"kind":"full","answers":{"q3":1,"q4":1,"q5":1}}`:          "answers.q1",
	} {
		r := e.do(t, http.MethodPost, "/api/v1/postpartum/epds", tok, "en", body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, body+" "+r.raw)
		assert.Contains(t, r.errors(), field, body)
	}
}

// IDOR: every read and write is the caller's own — another user's checks and recovery days never show.
func TestPostpartum_UserIsolation(t *testing.T) {
	e := setup(t)
	_, a := e.activated(t, "09120000011", "2026-09-01")
	_, b := e.activated(t, "09120000012", "2026-09-10")
	r := e.do(t, http.MethodPost, "/api/v1/postpartum/epds", a, "en", `{"kind":"short","answers":{"q3":3,"q4":3,"q5":3}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", a, "en", `{"lochia_amount":"heavy"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	h := e.do(t, http.MethodGet, "/api/v1/postpartum/epds", b, "en", "")
	assert.Equal(t, []any{}, h.data()["checks"])
	rb := e.do(t, http.MethodGet, "/api/v1/postpartum/recovery", b, "en", "")
	assert.Nil(t, rb.data()["lochia_amount"])
	sb := e.do(t, http.MethodGet, "/api/v1/postpartum", b, "en", "")
	assert.Equal(t, "2026-09-10", sb.data()["profile"].(map[string]any)["birth_date"])
	assert.Nil(t, sb.data()["checkin"].(map[string]any)["last"])
}

// The message engine serves postpartum content (no more empty result) with the day's alert as the override.
func TestPostpartum_DailyMessage(t *testing.T) {
	e := setup(t)
	_, tok := e.activated(t, "09120000013", "2026-09-14")
	r := e.do(t, http.MethodGet, "/api/v1/messages/daily", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "postpartum", d["mode"])
	pm := d["primary_message"].(map[string]any)
	assert.EqualValues(t, 3, pm["week"])
	assert.Equal(t, "checkin_full", pm["override_type"])

	r = e.do(t, http.MethodPut, "/api/v1/postpartum/recovery", tok, "en", `{"lochia_amount":"heavy"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/messages/daily", tok, "fa", "")
	pm = r.data()["primary_message"].(map[string]any)
	assert.Equal(t, "heavy_bleeding", pm["override_type"])
	assert.Equal(t, "خون‌ریزی زیاد ثبت کردی", pm["short_message"]) //nolint:staticcheck // ST1018: Persian copy (ZWNJ) verbatim
}
