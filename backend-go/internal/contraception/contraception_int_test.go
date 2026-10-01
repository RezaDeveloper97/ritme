package contraception_test

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
	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
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
	h := contraception.NewHandlers(contraception.NewService(db), clock.Real{})
	ch := care.NewHandlers(db, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/contraception", locale, guard, h.Show)
	app.Put("/api/v1/contraception/method", locale, guard, h.SaveMethod)
	app.Delete("/api/v1/contraception/method", locale, guard, h.StopMethod)
	app.Post("/api/v1/contraception/pills", locale, guard, h.LogPill)
	app.Delete("/api/v1/contraception/pills/:date", locale, guard, h.UnlogPill)
	app.Get("/api/v1/care/appointments", locale, guard, ch.ListAppointments)
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

func (r response) obj(key string) map[string]any {
	m, _ := r.data()[key].(map[string]any)
	return m
}

func (r response) reminders() []map[string]any {
	list, _ := r.data()["reminders"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, x := range list {
		m, _ := x.(map[string]any)
		out = append(out, m)
	}
	return out
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

func (e *env) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(query, args...).Scan(&n))
	return n
}

// pillPrefs is the B-N1-09 pill reminder as stored: enabled switch and time ("" when unset).
func (e *env) pillPrefs(t *testing.T, userID uint64) (enabled *bool, at string) {
	t.Helper()
	var cats, sched sql.NullString
	err := e.db.QueryRow(`SELECT categories, schedule FROM notification_preferences WHERE user_id = ?`, userID).Scan(&cats, &sched)
	if err == sql.ErrNoRows {
		return nil, ""
	}
	require.NoError(t, err)
	var c map[string]bool
	_ = json.Unmarshal([]byte(cats.String), &c)
	if v, ok := c["pill"]; ok {
		enabled = &v
	}
	var s map[string]struct {
		Time string `json:"time"`
	}
	_ = json.Unmarshal([]byte(sched.String), &s)
	return enabled, s["pill"].Time
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/contraception"},
		{http.MethodPut, "/api/v1/contraception/method"},
		{http.MethodDelete, "/api/v1/contraception/method"},
		{http.MethodPost, "/api/v1/contraception/pills"},
		{http.MethodDelete, "/api/v1/contraception/pills/2026-09-23"},
	} {
		r := e.do(t, c.method, c.path, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, c.path)
	}
}

func TestShow_Empty(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, "/api/v1/contraception", tok, "", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"data":{"tracking":false,"method":null,"pill":null,"reminders":[]}}`, r.raw)
}

func TestPillMethod_OneScheduleAndPack(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000002")
	// The user already timed the pill reminder on the cycle-settings screen (B-N1-09), switched off.
	_, err := e.db.Exec(`INSERT INTO notification_preferences (user_id, categories, schedule, created_at, updated_at)
		VALUES (?, '{"pill":false,"articles":true}', '{"pill":{"time":"20:15"}}', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	require.NoError(t, err)

	r := e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "fa",
		`{"method":"combined_pill","pack_type":"21_7","pack_started_on":"2026-09-16","packs_left":1}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "روش پیشگیری ذخیره شد", r.body["message"])
	assert.Equal(t, true, r.data()["tracking"])
	m := r.obj("method")
	assert.Equal(t, "combined_pill", m["method"])
	assert.Equal(t, map[string]any{"enabled": true, "time": "20:15"}, m["reminder"], "the B-N1-09 time is reused")
	assert.InDelta(t, 1, m["packs_left"], 0)

	enabled, at := e.pillPrefs(t, uid)
	require.NotNil(t, enabled)
	assert.True(t, *enabled)
	assert.Equal(t, "20:15", at)
	assert.Equal(t, 1, e.count(t, `SELECT track_contraception FROM user_life_profiles WHERE user_id = ?`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM notification_preferences WHERE user_id = ?`, uid), "one row, one schedule")
	var cats string
	require.NoError(t, e.db.QueryRow(`SELECT categories FROM notification_preferences WHERE user_id = ?`, uid).Scan(&cats))
	assert.JSONEq(t, `{"pill":true,"articles":true}`, cats, "other categories untouched")

	p := r.obj("pill")
	assert.InDelta(t, 1, p["pack_number"], 0)
	assert.InDelta(t, 8, p["pack_day"], 0)
	assert.Equal(t, "2026-10-14", p["next_pack_on"])
	assert.Equal(t, "2026-11-11", p["runs_out_on"])
	assert.Equal(t, "2026-11-06", p["refill_on"])
	assert.Equal(t, map[string]any{"date": "2026-09-23", "kind": "active", "status": "pending"}, p["today"])
	assert.InDelta(t, 0, p["missed_count"], 0, "days before the setup are untracked, not missed")
	days, _ := p["days"].([]any)
	require.Len(t, days, 28)
	assert.Equal(t, "untracked", days[0].(map[string]any)["status"])

	rem := r.reminders()
	require.Len(t, rem, 1)
	assert.Equal(t, "pill_refill", rem[0]["kind"])
	assert.Equal(t, "custom", rem[0]["type"])
	assert.Equal(t, "خرید بسته بعدی قرص", rem[0]["title"])
	assert.Equal(t, "2026-11-06", rem[0]["due_on"])

	// Change the time on this screen → the same (only) schedule moves.
	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en",
		`{"method":"combined_pill","pack_type":"21_7","pack_started_on":"2026-09-16","packs_left":1,"reminder_time":"21:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	_, at = e.pillPrefs(t, uid)
	assert.Equal(t, "21:00", at)
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ?`, uid), "refill reminder kept, not duplicated")

	// Log today's pill, then undo.
	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Pill logged", r.body["message"])
	p = r.obj("pill")
	assert.Equal(t, "taken", p["today"].(map[string]any)["status"])
	assert.InDelta(t, 1, p["streak_days"], 0)

	r = e.do(t, http.MethodDelete, "/api/v1/contraception/pills/2026-09-23", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "pending", r.obj("pill")["today"].(map[string]any)["status"])
}

func TestLogPill_Validation(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000003")

	r := e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "method")

	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en",
		`{"method":"combined_pill","pack_type":"21_7","pack_started_on":"2026-09-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{"date":"2026-09-22"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, "day 22 = break day: %s", r.raw)
	assert.Equal(t, []any{"There is no pill on a break day."}, r.errors()["date"])

	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{"date":"2026-08-31"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"This day is before the start of your pack."}, r.errors()["date"])

	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{"date":"2026-09-24"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"This date cannot be in the future."}, r.errors()["date"])

	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{"date":"2026-09-21","status":"missed"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 1, r.obj("pill")["missed_count"], 0, "day 21 logged missed; earlier days predate the setup")
	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tok, "en", `{"date":"2026-09-20"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.InDelta(t, 1, r.obj("pill")["missed_count"], 0, "the last active pill (day 21) missed")
	assert.InDelta(t, 2, r.obj("pill")["streak_days"], 0, "break days 22–23 since the missed pill on day 21")

	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en", `{"method":"combined_pill"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "pack_type")
	assert.Contains(t, r.errors(), "pack_started_on")

	r = e.do(t, http.MethodDelete, "/api/v1/contraception/pills/2026-13-01", tok, "en", "")
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

func TestLongActing_CareReminders(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000004")

	// Start on the pill (refill reminder), then switch to an IUD: the refill reminder goes, the pill reminder goes off.
	r := e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en",
		`{"method":"progestin_pill","pack_started_on":"2026-09-10","packs_left":0}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	require.Len(t, r.reminders(), 1)

	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en",
		`{"method":"copper_iud","inserted_on":"2026-09-01","iud_lifetime_years":10}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["pill"])
	m := r.obj("method")
	assert.Nil(t, m["reminder"])
	assert.Nil(t, m["pack_started_on"])
	assert.Equal(t, "2026-10-13", m["followup_on"])
	assert.Equal(t, "2036-09-01", m["iud_replace_on"])
	enabled, _ := e.pillPrefs(t, uid)
	require.NotNil(t, enabled)
	assert.False(t, *enabled, "one pill reminder: off without a pill method")

	rem := r.reminders()
	require.Len(t, rem, 3)
	kinds := []string{}
	for _, x := range rem {
		kinds = append(kinds, x["kind"].(string))
	}
	assert.ElementsMatch(t, []string{"iud_string_check", "iud_followup", "iud_replacement"}, kinds)
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ? AND title = 'Buy your next pill pack'`, uid))

	// The visits are care appointments: listed by /care/appointments like any other.
	ca := e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=all", tok, "en", "")
	require.Equal(t, http.StatusOK, ca.status, ca.raw)
	assert.Contains(t, ca.raw, "IUD check-up visit")
	assert.Contains(t, ca.raw, "IUD replacement")

	// The user switches the string check off; a new insertion date moves the dates and keeps her switch.
	var checkID uint64
	for _, x := range rem {
		if x["kind"] == "iud_string_check" {
			checkID = uint64(x["reminder_id"].(float64))
		}
	}
	_, err := e.db.Exec(`UPDATE reminders SET is_active = 0 WHERE id = ?`, checkID)
	require.NoError(t, err)
	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en",
		`{"method":"copper_iud","inserted_on":"2026-09-10","iud_lifetime_years":5,"followup_done":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	rem = r.reminders()
	require.Len(t, rem, 2, "follow-up done → its reminder goes")
	for _, x := range rem {
		if x["kind"] == "iud_string_check" {
			assert.InDelta(t, checkID, x["reminder_id"], 0, "same row")
			assert.Equal(t, false, x["is_active"])
			assert.Equal(t, "2026-10-10", x["due_on"])
			assert.Equal(t, "monthly", x["recurrence"])
		} else {
			assert.Equal(t, "iud_replacement", x["kind"])
			assert.Equal(t, "2031-09-10", x["due_on"])
		}
	}

	// Injection: next one 12 weeks after the last.
	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tok, "en", `{"method":"injection","injected_on":"2026-08-04"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-10-27", r.obj("method")["next_injection_on"])
	rem = r.reminders()
	require.Len(t, rem, 1)
	assert.Equal(t, "injection_next", rem[0]["kind"])
	assert.Equal(t, "2026-10-27 09:00:00", rem[0]["scheduled_at"])

	// Stop: method and its reminders go, the switch goes off; the pill log stays.
	_, err = e.db.Exec(`INSERT INTO contraception_pill_logs (user_id, log_date, status, logged_at, created_at, updated_at)
		VALUES (?, '2026-09-11', 'taken', '2026-09-11 21:00:00', NULL, NULL)`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodDelete, "/api/v1/contraception/method", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"tracking":false,"method":null,"pill":null,"reminders":[]}`, mustJSON(t, r.data()))
	assert.Equal(t, 0, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ?`, uid))
	assert.Equal(t, 0, e.count(t, `SELECT track_contraception FROM user_life_profiles WHERE user_id = ?`, uid))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM contraception_pill_logs WHERE user_id = ?`, uid))
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func TestUserIsolation(t *testing.T) {
	e := setup(t)
	a, tokA := e.user(t, "09120000005")
	_, tokB := e.user(t, "09120000006")

	r := e.do(t, http.MethodPut, "/api/v1/contraception/method", tokA, "en",
		`{"method":"combined_pill","pack_type":"28","pack_started_on":"2026-09-20","packs_left":2}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/contraception/pills", tokA, "en", `{}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	// B sees nothing of A's.
	r = e.do(t, http.MethodGet, "/api/v1/contraception", tokB, "en", "")
	require.Equal(t, http.StatusOK, r.status)
	assert.JSONEq(t, `{"tracking":false,"method":null,"pill":null,"reminders":[]}`, mustJSON(t, r.data()))

	// B's writes never touch A's rows.
	r = e.do(t, http.MethodDelete, "/api/v1/contraception/pills/2026-09-23", tokB, "en", "")
	require.Equal(t, http.StatusOK, r.status)
	r = e.do(t, http.MethodDelete, "/api/v1/contraception/method", tokB, "en", "")
	require.Equal(t, http.StatusOK, r.status)
	r = e.do(t, http.MethodPut, "/api/v1/contraception/method", tokB, "en", `{"method":"condom"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)

	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM contraception_pill_logs WHERE user_id = ?`, a))
	assert.Equal(t, 1, e.count(t, `SELECT COUNT(*) FROM reminders WHERE user_id = ?`, a))
	assert.Equal(t, "combined_pill", func() string {
		var s string
		require.NoError(t, e.db.QueryRow(`SELECT method FROM contraception_methods WHERE user_id = ?`, a).Scan(&s))
		return s
	}())
	r = e.do(t, http.MethodGet, "/api/v1/contraception", tokA, "en", "")
	assert.Equal(t, "taken", r.obj("pill")["today"].(map[string]any)["status"])
	assert.Equal(t, fmt.Sprint(true), fmt.Sprint(r.data()["tracking"]))
}
