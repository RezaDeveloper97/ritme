package daylog_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/pregnancy"
	"github.com/ritme/backend-go/internal/pregnancy/store"
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
	dl := daylog.NewHandlers(q, pregnancyalerts.New(q), clock.Real{})
	v1 := pregnancy.NewHandlers(q)

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get(base+"/report", locale, guard, dl.Report)
	app.Get(base+"/days/:date", locale, guard, dl.Show)
	app.Put(base+"/days/:date", locale, guard, dl.Update)
	app.Get("/api/v1/pregnancy/symptoms/:date", locale, guard, v1.SymptomShow)
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

func TestDay_SaveMergeAndV1Visibility(t *testing.T) {
	h := setup(t)
	_, tok := h.user(t, "09120000801", true)

	r := h.do(t, http.MethodPut, base+"/days/2026-09-22", tok, "en",
		`{"mood":4,"water_glasses":6,"weight":62.5,"visit_note":"  ask about iron  ",
		  "symptoms":{"nausea":"moderate","heartburn":"mild","back_pain":"severe"}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2026-09-22", d["date"])
	assert.EqualValues(t, 12, d["week"]) // 11w6d
	assert.EqualValues(t, 4, d["mood"])
	assert.EqualValues(t, 6, d["water_glasses"])
	assert.EqualValues(t, 62.5, d["weight"])
	assert.Equal(t, "ask about iron", d["visit_note"])
	assert.Equal(t, map[string]any{"nausea": "moderate", "back_pain": "severe", "heartburn": "mild"}, d["symptoms"])
	lw, _ := d["last_weight"].(map[string]any)
	assert.EqualValues(t, 62.5, lw["value"])
	assert.Equal(t, "2026-09-22", lw["date"])

	// v1 sees the symptom log v2 wrote.
	r = h.do(t, http.MethodGet, "/api/v1/pregnancy/symptoms/2026-09-22", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	log, _ := r.data()["log"].(map[string]any)
	assert.Equal(t, true, log["has_nausea"])
	assert.Equal(t, "moderate", log["nausea_severity"])
	assert.Equal(t, true, log["has_back_pain"])
	assert.Equal(t, false, log["has_spotting"])

	// Explicit null clears; absent keeps.
	r = h.do(t, http.MethodPut, base+"/days/2026-09-22", tok, "en", `{"mood":null,"symptoms":{"nausea":"mild"}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Nil(t, d["mood"])
	assert.EqualValues(t, 6, d["water_glasses"])
	assert.Equal(t, map[string]any{"nausea": "mild"}, d["symptoms"])
	r = h.do(t, http.MethodPut, base+"/days/2026-09-22", tok, "en", `{"weight":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["weight"])
	assert.Nil(t, r.data()["last_weight"])

	// GET returns the same merged day; an empty day is all nulls.
	r = h.do(t, http.MethodGet, base+"/days/2026-09-22", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"nausea": "mild"}, r.data()["symptoms"])
	r = h.do(t, http.MethodGet, base+"/days/2026-09-01", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["mood"])
	assert.Empty(t, r.data()["symptoms"])
}

func TestDay_SpottingFiresV2AlertOnce(t *testing.T) {
	h := setup(t)
	uid, tok := h.user(t, "09120000802", true)
	today := "2026-09-23" // the pinned clock day
	body := `{"symptoms":{"spotting":"severe"}}`
	r := h.do(t, http.MethodPut, base+"/days/"+today, tok, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	alerts, _ := r.data()["alerts"].([]any)
	require.Len(t, alerts, 1, r.raw)
	a := alerts[0].(map[string]any)
	assert.Equal(t, "critical_symptom", a["rule_key"])
	assert.Equal(t, "Spotting logged", a["title"])
	assert.Equal(t, "urgent", a["level"])
	assert.NotEmpty(t, a["what_we_saw"])
	assert.NotEmpty(t, a["actions"])
	n := h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid)
	// The v1 rule keeps writing its row next to the v2 one.
	crit := h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ? AND alert_level = 'warning' AND title = 'Warning: Spotting'`, uid)
	assert.Positive(t, crit)

	// The offline outbox resends: no second write, no duplicate alert.
	r = h.do(t, http.MethodPut, base+"/days/"+today, tok, "en", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, r.data()["alerts"])
	assert.Equal(t, n, h.count(`SELECT COUNT(*) FROM pregnancy_alerts WHERE user_id = ?`, uid))
}

func TestDay_Validation(t *testing.T) {
	h := setup(t)
	_, tok := h.user(t, "09120000803", true)
	r := h.do(t, http.MethodPut, base+"/days/2026-09-24", tok, "en", `{"mood":3}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "date")
	r = h.do(t, http.MethodGet, base+"/days/nope", tok, "fa", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = h.do(t, http.MethodPut, base+"/days/2026-09-20", tok, "fa",
		`{"mood":9,"water_glasses":40,"weight":10,"symptoms":{"nausea":"extreme"}}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	errs, _ := r.body["errors"].(map[string]any)
	for _, k := range []string{"mood", "water_glasses", "weight", "symptoms.nausea"} {
		assert.Contains(t, errs, k)
	}
	r = h.do(t, http.MethodPut, base+"/days/2026-09-20", tok, "en", `{"symptoms":{"hiccups":"mild"}}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.body["errors"], "symptoms")
}

func TestDay_NotActiveAndAuth(t *testing.T) {
	h := setup(t)
	_, tok := h.user(t, "09120000804", false)
	r := h.do(t, http.MethodGet, base+"/days/2026-09-20", tok, "en", "")
	require.Equal(t, http.StatusConflict, r.status, r.raw)
	assert.Equal(t, "pregnancy_not_active", r.body["error_code"])
	r = h.do(t, http.MethodGet, base+"/days/2026-09-20", "", "en", "")
	require.Equal(t, http.StatusUnauthorized, r.status, r.raw)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestDay_UserIsolation(t *testing.T) {
	h := setup(t)
	_, a := h.user(t, "09120000805", true)
	_, b := h.user(t, "09120000806", true)
	r := h.do(t, http.MethodPut, base+"/days/2026-09-21", a, "en", `{"mood":2,"symptoms":{"fatigue":"mild"}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = h.do(t, http.MethodGet, base+"/days/2026-09-21", b, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["mood"])
	assert.Empty(t, r.data()["symptoms"])
	r = h.do(t, http.MethodGet, base+"/report?from=2026-09-01&to=2026-09-23", b, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, r.data()["days"])
}

func TestReport(t *testing.T) {
	h := setup(t)
	_, tok := h.user(t, "09120000807", true)
	for _, c := range []struct{ date, body string }{
		{"2026-09-10", `{"weight":60}`},
		{"2026-09-15", `{"mood":5,"symptoms":{"constipation":"moderate"}}`},
		{"2026-09-20", `{"visit_note":"bp ok","weight":61.2}`},
	} {
		r := h.do(t, http.MethodPut, base+"/days/"+c.date, tok, "en", c.body)
		require.Equal(t, http.StatusOK, r.status, r.raw)
	}
	r := h.do(t, http.MethodGet, base+"/report?from=2026-09-12&to=2026-09-23", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, map[string]any{"from": "2026-09-12", "to": "2026-09-23"}, d["range"])
	assert.Equal(t, "2027-04-07", d["due_date"])
	days, _ := d["days"].([]any)
	require.Len(t, days, 2)
	assert.Equal(t, "2026-09-15", days[0].(map[string]any)["date"])
	assert.Equal(t, map[string]any{"constipation": "moderate"}, days[0].(map[string]any)["symptoms"])
	weights, _ := d["weights"].([]any)
	require.Len(t, weights, 1)
	assert.EqualValues(t, 61.2, weights[0].(map[string]any)["value"])

	r = h.do(t, http.MethodGet, base+"/report?from=2026-09-20&to=2026-09-01", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	r = h.do(t, http.MethodGet, base+"/report", tok, "fa", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}
