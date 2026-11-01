package pregnancyalerts_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/platform/civildate"
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
	inboxalerts.Install(db, quiet) // the alerts inbox backend (T-M10-08), as tools.NewService installs it in the API
	t.Cleanup(func() { inboxalerts.Install(db, quiet) })
	v1 := pregnancy.NewHandlers(q)

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

// inbox counts the user's preg.* alerts inbox rows matching cond.
func (h *harness) inbox(uid uint64, cond string) int {
	return h.count(`SELECT COUNT(*) FROM message_inbox WHERE user_id = ? AND rule_key LIKE 'preg.%' AND `+cond, uid)
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
	// Daily: week 13 entered today (12w0d). No weight was ever logged, 84 days into the pregnancy: the weight
	// reminder is due (SPEC §5.4.1: ≥ 7 days without a weight).
	we := byRule(list, "week_entered")
	require.Len(t, we, 1)
	assert.Equal(t, "You've entered week 13", we[0]["title"])
	assert.Equal(t, "info", we[0]["level"])
	assert.Equal(t, "2026-09-23", we[0]["fact_date"])
	wm := byRule(list, "weight_missing_week")
	require.Len(t, wm, 1)
	assert.Equal(t, "suggestion", wm[0]["level"])
	// Once: the second read changes nothing.
	n := h.inbox(uid, "1")
	assert.Equal(t, 2, n)
	alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	assert.Equal(t, n, h.inbox(uid, "hit_count = 1"))
	// Localized (the rule's base fa copy).
	fa := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "fa", ""))
	assert.Equal(t, "وارد هفتهٔ ۱۳ شدی", byRule(fa, "week_entered")[0]["title"])
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid), "no new pregnancy_alerts rows")
}

func TestAlerts_VomitingThresholdAndDedupe(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000902", true)
	h.vomit(t, tok, "2026-09-22")
	// 1 day of 7, 1 severe, streak 1: below 4 / 2 / 3.
	assert.Zero(t, h.inbox(uid, "rule_key = 'preg.vomiting'"))
	r := h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"symptoms":{"vomiting":"severe"}}`)
	got := byRule(alertsOf(t, r), "vomiting_streak")
	require.Len(t, got, 1, r.raw)
	a := got[0]
	assert.Equal(t, "follow_up", a["level"])
	assert.Equal(t, "warning", a["alert_level"])
	assert.Equal(t, "Vomiting on 2 of the last 7 days", a["title"])
	assert.Equal(t, "Vomiting on 2 of the last 7 days, 2 of them severe", a["what_we_saw"])
	assert.Equal(t, "This pattern comes from your own logs, not a diagnosis", a["how_sure"])
	assert.NotEmpty(t, a["actions"])
	assert.Nil(t, a["contact"])
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.vomiting' AND level = 'worth_followup' AND state = 'active'"))
	// The same data again: the row is unchanged (same evidence).
	h.vomit(t, tok, "2026-09-23")
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.vomiting' AND hit_count = 1"))
}

func TestAlerts_LockedRuleOffAndEditTextsWithoutDeploy(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000903", true)
	// preg.critical_symptom is a locked safety rule (SPEC §4.2, T-M9-05): an inactive row no longer switches the path
	// off; the code-held definition with the embedded copy raises the alert, urgent.
	h.exec(`UPDATE message_rules SET is_active = 0 WHERE rule_key = 'preg.critical_symptom'`)
	r := h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"symptoms":{"spotting":"severe"}}`)
	got := byRule(alertsOf(t, r), "critical_symptom")
	require.Len(t, got, 1, r.raw)
	assert.Equal(t, "urgent", got[0]["level"])
	assert.Equal(t, "Spotting logged", got[0]["title"])
	assert.Contains(t, got[0]["contact"], "115")
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.critical_symptom' AND level = 'follow_soon'"))

	// Active again, with the advisor's edit of the en text: it shows on the next read, for the alert already raised.
	h.exec(`UPDATE message_rules SET is_active = 1,
		content = JSON_SET(content, '$.translations.en.a.title', 'Edited: {preg_symptoms}') WHERE rule_key = 'preg.critical_symptom'`)
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	crit := byRule(list, "critical_symptom")
	require.Len(t, crit, 1)
	assert.Equal(t, "Edited: Spotting", crit[0]["title"])
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.critical_symptom'"), "one alert history: the code-held hit kept the row id")
}

func TestAlerts_V1SaveRunsRules(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000904", true)
	r := h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":85}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	// The v1 response carries the alert of the save in the v1 shape (an inbox row).
	v1, _ := r.data()["alerts"].([]any)
	require.Len(t, v1, 1, r.raw)
	a := v1[0].(map[string]any)
	assert.Equal(t, "bp_high", a["alert_type"])
	urgent := pregnancyalerts.HighBPLevel(civildate.MustParse("2026-09-23")) == "urgent"
	assert.Equal(t, map[bool]string{true: "emergency", false: "warning"}[urgent], a["alert_level"])
	assert.Equal(t, []string{"user_id", "alert_level", "alert_type", "title", "message", "pregnancy_week",
		"trigger_symptoms", "recommended_actions", "updated_at", "created_at", "id"}, keys(a, r.raw))
	assert.Equal(t, map[string]any{"systolic": 150.0, "diastolic": 85.0}, a["trigger_symptoms"])
	key := map[bool]string{true: "preg.bp_high_urgent", false: "preg.bp_high"}[urgent]
	assert.Equal(t, 1, h.inbox(uid, "rule_key = '"+key+"'"))
	assert.EqualValues(t, h.count(`SELECT id FROM message_inbox WHERE user_id = ? AND rule_key = ?`, uid, key), a["id"])
	assert.Zero(t, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid), "the v1 rules are retired")
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	bp := byRule(list, "bp_high")
	require.Len(t, bp, 1)
	assert.Contains(t, bp[0]["what_we_saw"], "You logged 150/85 (")
	assert.Equal(t, urgent, bp[0]["contact"] != nil)
	// A re-save of the same reading raises nothing new.
	r = h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":85}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Empty(t, r.data()["alerts"])
}

// keys are the JSON keys of the alert object in raw, in order (the v1 shape is pinned by key order).
func keys(a map[string]any, raw string) []string {
	i := strings.Index(raw, `"alerts":[{`)
	if i < 0 {
		return nil
	}
	var out []string
	dec := json.NewDecoder(strings.NewReader(raw[i+len(`"alerts":[`):]))
	if _, err := dec.Token(); err != nil {
		return nil
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return out
		}
		k, _ := tok.(string)
		out = append(out, k)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return out
		}
	}
	_ = a
	return out
}

// Review #5 (T-M2-34) / D-21: the alert evaluation is best-effort for a v1 save — when it fails (here: no backend),
// the v1 log is committed and the save still answers 201 with an empty alerts array.
func TestAlerts_V1SaveSurvivesEvaluationFailure(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000907", true)
	pregnancyalerts.InstallBackend(nil)

	r := h.do(t, http.MethodPost, "/api/v1/pregnancy/weekly", tok, "en",
		`{"pregnancy_week":13,"log_date":"2026-09-23","systolic_pressure":150,"diastolic_pressure":85}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, true, r.body["success"])
	assert.Empty(t, r.data()["alerts"])
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_weekly_logs WHERE user_id = ?`, uid))

	r = h.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok, "en",
		`{"log_date":"2026-09-23","has_bleeding":true,"bleeding_severity":"mild"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, 1, h.count(`SELECT COUNT(*) FROM pregnancy_symptom_logs WHERE user_id = ?`, uid))
	assert.Zero(t, h.inbox(uid, "1"), "the failing evaluation wrote nothing")
}

// The v1 symptom save answers its urgent alert in the v1 shape, with the medical history flags (contract shape).
func TestAlerts_V1SymptomSaveShape(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000911", true)
	r := h.do(t, http.MethodPost, "/api/v1/pregnancy/symptoms", tok, "fa",
		`{"log_date":"2026-09-23","has_bleeding":true,"bleeding_severity":"mild"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	v1, _ := r.data()["alerts"].([]any)
	require.Len(t, v1, 1, r.raw)
	a := v1[0].(map[string]any)
	assert.Equal(t, []string{"user_id", "alert_level", "alert_type", "title", "message", "pregnancy_week",
		"trigger_symptoms", "medical_history_flags", "recommended_actions", "updated_at", "created_at", "id"}, keys(a, r.raw))
	assert.Equal(t, "emergency", a["alert_level"])
	assert.Equal(t, "critical_symptom", a["alert_type"])
	assert.Equal(t, map[string]any{"bleeding": true}, a["trigger_symptoms"])
	assert.Positive(t, a["pregnancy_week"], "the week comes from the dating when the rule text has no {preg_week}")
	assert.Equal(t, []any{}, a["medical_history_flags"])
	assert.NotEmpty(t, a["recommended_actions"])
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.critical_symptom' AND kind = 'safety' AND level = 'follow_soon'"))
}

func TestAlerts_ActionsAndIDOR(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000905", true)
	_, other := h.user(t, "09120000906", true)
	h.vomit(t, tok, "2026-09-21", "2026-09-22", "2026-09-23")
	list := alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	a := byRule(list, "vomiting_streak")
	require.Len(t, a, 1)
	id := fmt.Sprint(int64(a[0]["id"].(float64)))
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.vomiting' AND hit_count = 2"), "the 3rd day was new evidence: one row, two hits")

	// Another user can neither act on it nor see it.
	r := h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/ack", other, "en", "")
	assert.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Empty(t, byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", other, "en", "")), "vomiting_streak"))
	// Unknown action.
	assert.Equal(t, http.StatusNotFound, h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/delete", tok, "en", "").status)

	h.do(t, http.MethodPut, base+"/days/2026-09-23", tok, "en", `{"visit_note":"ask about iron"}`)
	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/add_to_visit_note", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	want := "ask about iron\nVomiting on 3 of the last 7 days, 3 of them severe"
	assert.Equal(t, want, r.data()["visit_note"])
	assert.Equal(t, true, r.data()["alert"].(map[string]any)["is_read"])
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.vomiting' AND state = 'added_to_visit_note'"))
	// Idempotent.
	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/add_to_visit_note", tok, "en", "")
	assert.Equal(t, want, r.data()["visit_note"])

	r = h.do(t, http.MethodPost, base+"/alerts/"+id+"/actions/ack", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.data()["alert"].(map[string]any)["is_acked"])
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.vomiting' AND state = 'acknowledged'"))
	// Acked alerts stay in the v2 window (is_acked).
	assert.Len(t, byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", "")), "vomiting_streak"), 1)
}

// Review #10 (T-M2-34): HEAD /alerts (answered by the GET route) never runs the evaluation.
func TestAlerts_HeadIsReadOnly(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000908", true)
	r := h.do(t, http.MethodHead, base+"/alerts", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Zero(t, h.inbox(uid, "1"))
	alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", ""))
	assert.Positive(t, h.inbox(uid, "1"), "GET still evaluates (week_entered)")
}

// GET /today evaluates the calendar alerts without the Alerts screen being opened first (review #10), deduplicated;
// HEAD stays read-only.
func TestAlerts_TodayEvaluatesCalendarRules(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000909", true) // 12w0d on 2026-09-23: first day of week 13
	today := func(method string) {
		t.Helper()
		r := h.do(t, method, base+"/today", tok, "en", "")
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	today(http.MethodHead)
	assert.Zero(t, h.inbox(uid, "1"), "HEAD writes nothing")
	today(http.MethodGet)
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.week_entered'"))
	assert.Equal(t, 1, h.inbox(uid, "rule_key = 'preg.weight_missing'"))
	today(http.MethodGet)
	assert.Equal(t, 2, h.inbox(uid, "hit_count = 1"), "deduplicated")
}

// QA 2026-09-29-c L5: re-dating (manual → LMP) in Setup refreshes the current week's week_entered alert — basis in the
// text and fact date (the week start is its evidence) — instead of keeping the facts frozen. The read state survives,
// and no second alert is created.
func TestAlerts_WeekEnteredFollowsReDating(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000910", false)
	// Manual entry today: 12w0d → week 13 starts today (2026-09-23).
	h.exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, age_source, manual_weeks, manual_days, manual_entry_date,
		created_at, updated_at) VALUES (?, 1, 'manual', 12, 0, '2026-09-23', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, uid)
	we := byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", "")), "week_entered")
	require.Len(t, we, 1)
	assert.Equal(t, "2026-09-23", we[0]["fact_date"])
	assert.Contains(t, we[0]["advice"], "Manual Entry")
	id := we[0]["id"]
	r := h.do(t, http.MethodPost, base+"/alerts/"+fmt.Sprint(int64(id.(float64)))+"/actions/ack", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)

	// Setup re-dates by LMP: 12w2d, so week 13 started two days ago.
	h.exec(`UPDATE pregnancy_profiles SET age_source = 'lmp', lmp_date = '2026-06-29' WHERE user_id = ?`, uid)
	we = byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "en", "")), "week_entered")
	require.Len(t, we, 1, "refreshed, not duplicated")
	assert.Equal(t, id, we[0]["id"])
	assert.EqualValues(t, 13, we[0]["pregnancy_week"])
	assert.Equal(t, "2026-09-21", we[0]["fact_date"])
	assert.Contains(t, we[0]["advice"], "Last Menstrual Period")
	assert.NotContains(t, we[0]["advice"], "Manual Entry")
	assert.Equal(t, true, we[0]["is_read"])
	fa := byRule(alertsOf(t, h.do(t, http.MethodGet, base+"/alerts", tok, "fa", "")), "week_entered")
	require.Len(t, fa, 1)
	assert.NotContains(t, fa[0]["advice"], "ورود دستی")
}
