package pregnancyalerts_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
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
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/store"
	v2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/pregnancy/v2/daylog"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	now      = "2026-09-23T10:00:00+03:30"
	base     = "/api/v1/pregnancy/v2"
)

type response struct {
	status int
	body   map[string]any
	raw    string
}

func (r response) data() map[string]any { d, _ := r.body["data"].(map[string]any); return d }

type harness struct {
	app   *fiber.App
	iss   *passport.Issuer
	exec  func(q string, args ...any)
	count func(q string, args ...any) int
}

func setup(t *testing.T) *harness {
	t.Helper()
	db := testdb.New(t)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := db.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	aq := authstore.New(db)
	guard := auth.NewGuardWith(&key.PublicKey, aq, clock.Real{}, quiet).RequireUser
	locale := i18n.Middleware(i18n.NewRegistry(i18nstore.New(db), nil, quiet))
	q := store.New(db)
	al := pregnancyalerts.NewHandlers(q, clock.Real{})
	dl := daylog.NewHandlers(q, al.Engine(), clock.Real{})
	v1 := pregnancy.NewHandlers(q)
	v1.SetAfterLogSave(al.Engine().AfterSave)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Put(base+"/days/:date", locale, guard, dl.Update)
	app.Get(base+"/alerts", locale, guard, al.Index)
	app.Post(base+"/alerts/:id/actions/:action", locale, guard, al.Action)
	// As routes_pregnancy_v2.go mounts it (review #10, T-M7-20).
	app.Get(base+"/today", locale, guard, al.EvaluateFirst(quiet), v2.NewHandlers(q, clock.Real{}).Today)
	app.Post("/api/v1/pregnancy/symptoms", locale, guard, v1.SymptomStore)
	app.Post("/api/v1/pregnancy/weekly", locale, guard, v1.WeeklyStore)
	app.Get("/api/v1/pregnancy/alerts", locale, guard, v1.AlertIndex)
	// The same v1 handlers with a hook that always fails (review #5, T-M2-34).
	broken := pregnancy.NewHandlers(q)
	broken.SetAfterLogSave(func(context.Context, uint64, string, string, time.Time) error {
		return errors.New("lock wait timeout")
	})
	app.Post("/broken/symptoms", locale, guard, broken.SymptomStore)
	app.Post("/broken/weekly", locale, guard, broken.WeeklyStore)
	app.Post("/broken/fetal-movement", locale, guard, broken.FetalStore)
	return &harness{
		app: app, iss: passport.NewIssuer(key, aq, clock.Real{}, 365),
		exec: func(s string, args ...any) { _, err := db.Exec(s, args...); require.NoError(t, err) },
		count: func(s string, args ...any) int {
			var n int
			require.NoError(t, db.QueryRow(s, args...).Scan(&n))
			return n
		},
	}
}

// user creates a user in pregnancy mode dated by LMP 2026-07-01 (12w0d on 2026-09-23 → week 13).
func (h *harness) user(t *testing.T, mobile string, pregnant bool) (uint64, string) {
	t.Helper()
	var id int64
	h.exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('T', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	id = int64(h.count(`SELECT id FROM users WHERE mobile = ?`, mobile))
	if pregnant {
		h.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, lmp_date, created_at, updated_at)
			VALUES (?, 1, 'lmp', '2026-07-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, id)
	}
	tok, err := h.iss.Issue(context.Background(), uint64(id), time.Now()) //nolint:gosec // test id
	require.NoError(t, err)
	return uint64(id), tok.AccessToken //nolint:gosec // test id
}

func (h *harness) do(t *testing.T, method, path, token, lang, body string) response {
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
	resp, err := h.app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return response{status: resp.StatusCode, body: m, raw: string(raw)}
}

func alertsOf(t *testing.T, r response) []map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, r.status, r.raw)
	list, _ := r.data()["alerts"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, a := range list {
		out = append(out, a.(map[string]any))
	}
	return out
}

func byRule(list []map[string]any, rule string) []map[string]any {
	var out []map[string]any
	for _, a := range list {
		if a["rule_key"] == rule {
			out = append(out, a)
		}
	}
	return out
}

func (h *harness) vomit(t *testing.T, tok string, days ...string) {
	t.Helper()
	for _, d := range days {
		r := h.do(t, http.MethodPut, base+"/days/"+d, tok, "en", `{"symptoms":{"vomiting":"severe"}}`)
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
}

func TestAlerts_401AndNotActive(t *testing.T) {
	h := setup(t)
	r := h.do(t, http.MethodGet, base+"/alerts", "", "en", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
	_, tok := h.user(t, "09120000900", false)
	r = h.do(t, http.MethodGet, base+"/alerts", tok, "en", "")
	assert.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, "pregnancy_not_active", r.body["error_code"])
}

func TestAlerts_ListLegendAndDailyRules(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000901", true)
	r := h.do(t, http.MethodGet, base+"/alerts", tok, "en", "")
	list := alertsOf(t, r)
	d := r.data()
	assert.EqualValues(t, 7, d["window_days"])
	assert.Equal(t, "Based on your logs from the last 7 days", d["window_note"])
	legend := d["legend"].([]any)
	require.Len(t, legend, 4)
	assert.Equal(t, "info", legend[0].(map[string]any)["level"])
	assert.Equal(t, "urgent", legend[3].(map[string]any)["level"])
	// Daily: week 13 entered today (12w0d). Weight not logged, but "weight missing" waits for
	// day 5 of the week (review #11, T-M2-34).
	we := byRule(list, "week_entered")
	require.Len(t, we, 1)
	assert.Equal(t, "You've entered week 13", we[0]["title"])
	assert.Equal(t, "info", we[0]["level"])
	assert.Empty(t, byRule(list, "weight_missing_week"))
	// Once per window.
	n := h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid)
	alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	assert.Equal(t, n, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid))
	// Localized (fa row).
	fa := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "fa", ""))
	assert.NotEqual(t, we[0]["title"], byRule(fa, "week_entered")[0]["title"])
}

func TestAlerts_VomitingStreakThresholdAndDedupe(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000902", true)
	h.vomit(t, tok, "2026-09-22")
	// streak 1 on 09-22 (today not logged → counted from yesterday) with 1 severe day: below 3 / 2.
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:vomiting_streak'`, uid))
	r := h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"symptoms":{"vomiting":"severe"}}`)
	got := byRule(alertsOf(t, r), "vomiting_streak")
	require.Len(t, got, 1, r.raw)
	a := got[0]
	assert.Equal(t, "follow_up", a["level"])
	assert.Equal(t, "Vomiting logged 2 days in a row", a["title"])
	assert.Equal(t, "Vomiting in your last 2 logs, 2 of them severe", a["what_we_saw"])
	assert.NotEmpty(t, a["actions"])
	assert.Nil(t, a["contact"])
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:vomiting_streak' AND alert_level = 'warning'`, uid))
	h.vomit(t, tok, "2026-09-23")
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:vomiting_streak'`, uid))
}

func TestAlerts_DisableRuleAndEditTextsWithoutDeploy(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000903", true)
	h.exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.enabled', false)
		WHERE ` + "`group`" + ` = 'pregnancy_alert' AND item_key = 'critical_symptom'`)
	r := h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"symptoms":{"spotting":"severe"}}`)
	assert.Empty(t, byRule(alertsOf(t, r), "critical_symptom"))
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:critical_symptom'`, uid))

	h.exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.enabled', true, '$.level', 'follow_up')
		WHERE ` + "`group`" + ` = 'pregnancy_alert' AND item_key = 'critical_symptom'`)
	r = h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"symptoms":{"spotting":"severe","nausea":"mild"}}`)
	got := byRule(alertsOf(t, r), "critical_symptom")
	require.Len(t, got, 1, r.raw)
	assert.Equal(t, "follow_up", got[0]["level"])
	assert.Equal(t, "Spotting logged", got[0]["title"])

	// An admin edit of the texts shows on the next read, for alerts already raised.
	h.exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.title', 'Edited: {symptom}')
		WHERE ` + "`group`" + ` = 'pregnancy_alert' AND item_key = 'critical_symptom' AND locale = 'en'`)
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	assert.Equal(t, "Edited: Spotting", byRule(list, "critical_symptom")[0]["title"])
}

func TestAlerts_V1SaveRunsRules(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000904", true)
	r := h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":85}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	// The v1 response still carries only the v1 alert.
	v1, _ := r.data()["alerts"].([]any)
	require.Len(t, v1, 1)
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:bp_high' AND alert_level = 'emergency'`, uid))
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	bp := byRule(list, "bp_high")
	require.Len(t, bp, 1)
	assert.Equal(t, "You logged 150/85", bp[0]["what_we_saw"])
	assert.NotEmpty(t, bp[0]["contact"])
}

// Review #5 (T-M2-34): the v2 hook is best-effort. A hook failure after the v1 log committed
// is logged, and the v1 save still answers 201 with its v1 body (D-21).
func TestAlerts_V1SaveSurvivesHookFailure(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000907", true)

	r := h.do(t, http.MethodPost, "/broken/weekly", tok, "en",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":85}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, true, r.body["success"])
	v1, _ := r.data()["alerts"].([]any)
	assert.Len(t, v1, 1, "v1 alerts unchanged")
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_weekly_logs WHERE user_id = ?`, uid))

	r = h.do(t, http.MethodPost, "/broken/symptoms", tok, "en",
		`{"log_date":"2026-09-23","has_nausea":true,"nausea_severity":"mild"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_symptom_logs WHERE user_id = ?`, uid))

	r = h.do(t, http.MethodPost, "/broken/fetal-movement", tok, "en",
		`{"log_date":"2026-09-23","pregnancy_week":13,"movement_status":"felt","movement_count":2}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)

	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type LIKE 'v2:%'`, uid),
		"the failing hook wrote nothing")
}

func TestAlerts_ActionsAndIDOR(t *testing.T) {
	h := setup(t)
	_, tok := h.user(t, "09120000905", true)
	_, other := h.user(t, "09120000906", true)
	h.vomit(t, tok, "2026-09-21", "2026-09-22", "2026-09-23")
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	a := byRule(list, "vomiting_streak")
	require.Len(t, a, 1)
	id := fmt.Sprint(int64(a[0]["id"].(float64)))

	// Another user can neither act on it nor see it.
	r := h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/ack", other, "en", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Empty(t, byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", other, "en", "")), "vomiting_streak"))
	// Unknown action.
	assert.Equal(t, http.StatusNotFound, h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/delete", tok, "en", "").status)

	h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"visit_note":"ask about iron"}`)
	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/add_to_visit_note", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	want := "ask about iron\nVomiting in your last 2 logs, 2 of them severe" // fired at the 2nd severe day; the 3rd is deduped
	assert.Equal(t, want, r.data()["visit_note"])
	assert.Equal(t, true, r.data()["alert"].(map[string]any)["is_read"])
	// Idempotent.
	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/add_to_visit_note", tok, "en", "")
	assert.Equal(t, want, r.data()["visit_note"])

	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/ack", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["alert"].(map[string]any)["is_acked"])
	// Acked alerts leave the v1 active list but stay in the v2 window (is_acked).
	v1 := h.do(t, http.MethodGet, "/api/v1/pregnancy/alerts", tok, "en", "")
	for _, x := range v1.data()["alerts"].([]any) {
		assert.NotEqual(t, "v2:vomiting_streak", x.(map[string]any)["alert_type"])
	}
}

// Review #10 (T-M2-34): HEAD /alerts (answered by the GET route) never runs the evaluation.
func TestAlerts_HeadIsReadOnly(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000908", true)
	r := h.do(t, http.MethodHead, base+"/alerts", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid))
	alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	assert.Positive(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid), "GET still evaluates (week_entered)")
}

// Today's unread badge counts the calendar alerts without the Alerts screen being opened first
// (review #10): GET /today evaluates them (deduplicated, so a second GET adds nothing); HEAD stays
// read-only. weight_missing_week reads the admin-editable params.from_weekday (review #11).
func TestAlerts_TodayEvaluatesCalendarRules(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000909", true) // 12w0d on 2026-09-23: first day of week 13
	unread := func(method string) any {
		t.Helper()
		r := h.do(t, method, base+"/today", tok, "en", "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
		return r.data()["unread_alerts"]
	}

	unread(http.MethodHead)
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid), "HEAD writes nothing")

	// Day 0 of the week: week_entered fires; weight_missing_week waits for from_weekday (default 5).
	assert.EqualValues(t, 1, unread(http.MethodGet))
	assert.EqualValues(t, 1, unread(http.MethodGet), "deduplicated")

	// An admin sets from_weekday = 0: the weight reminder fires today as well.
	h.exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.params.from_weekday', 0)
		WHERE ` + "`group`" + ` = 'pregnancy_alert' AND item_key = 'weight_missing_week'`)
	assert.EqualValues(t, 2, unread(http.MethodGet))
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_type = 'v2:weight_missing_week'`, uid))
}
