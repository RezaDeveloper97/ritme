package ivf_test

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
	"github.com/ritme/backend-go/internal/companion"
	"github.com/ritme/backend-go/internal/companion/shared"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/ivf"
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
	cat := catalog.NewReader(catalogstore.New(db), nil, 0, quiet)
	h := ivf.NewHandlers(ivf.NewService(db, cat, companion.NewService(db, clock.Real{}, companion.Options{})), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/ivf", locale, guard, h.Show)
	app.Post("/api/v1/ivf/cycles", locale, guard, h.StartCycle)
	app.Put("/api/v1/ivf/cycles/current", locale, guard, h.UpdateCycle)
	app.Post("/api/v1/ivf/cycles/current/outcome", locale, guard, h.RecordOutcome)
	app.Get("/api/v1/ivf/meds", locale, guard, h.ShowMeds)
	app.Post("/api/v1/ivf/meds", locale, guard, h.AddMed)
	app.Put("/api/v1/ivf/meds/:id", locale, guard, h.UpdateMed)
	app.Delete("/api/v1/ivf/meds/:id", locale, guard, h.DeleteMed)
	app.Post("/api/v1/ivf/meds/:id/doses", locale, guard, h.LogDose)
	app.Delete("/api/v1/ivf/meds/:id/doses", locale, guard, h.UnlogDose)
	app.Get("/api/v1/ivf/scans", locale, guard, h.ShowScans)
	app.Put("/api/v1/ivf/scans/:date", locale, guard, h.SaveScan)
	app.Delete("/api/v1/ivf/scans/:date", locale, guard, h.DeleteScan)
	app.Get("/api/v1/ivf/tww", locale, guard, h.ShowTWW)
	app.Put("/api/v1/ivf/tww/:date", locale, guard, h.SaveMood)
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

func obj(m map[string]any, path ...string) map[string]any {
	for _, k := range path {
		m, _ = m[k].(map[string]any)
	}
	return m
}

func list(m map[string]any, key string) []map[string]any {
	l, _ := m[key].([]any)
	out := make([]map[string]any, 0, len(l))
	for _, x := range l {
		o, _ := x.(map[string]any)
		out = append(out, o)
	}
	return out
}

func (r response) errors() map[string]any {
	d, _ := r.body["errors"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token, body string) response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept-Language", "en")
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

func medID(t *testing.T, r response, name string) uint64 {
	t.Helper()
	for _, m := range list(r.data(), "meds") {
		if m["name"] == name {
			return uint64(m["id"].(float64))
		}
	}
	t.Fatalf("medicine %q not in %s", name, r.raw)
	return 0
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	r := e.do(t, http.MethodGet, "/api/v1/ivf", "", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestCycleLifecycle(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")

	r := e.do(t, http.MethodGet, "/api/v1/ivf", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["enabled"])
	assert.Nil(t, r.data()["cycle"])

	// writes without a cycle
	r = e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok, `{"name":"FSH","role":"stimulation","route":"subcutaneous","times":["20:00"]}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.errors(), "cycle")

	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok, `{"protocol":"nope"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.errors(), "protocol")

	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok, `{"protocol":"antagonist","started_on":"2026-09-10","stage":"stim","stim_started_on":"2026-09-17"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, true, r.data()["enabled"], "starting a cycle switches bloom's IVF/IUI flag on")
	c := obj(r.data(), "cycle")
	assert.EqualValues(t, 1, c["number"])
	assert.Equal(t, "stim", c["stage"])
	assert.EqualValues(t, 7, c["stage_day"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM user_life_profiles WHERE user_id = ? AND ivf_iui = 1", uid))

	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok, `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "one open cycle")
	assert.Contains(t, r.errors(), "cycle")

	// dates create care appointments; the next one shows on the home
	r = e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tok, `{"next_scan_at":"2026-09-24 09:00","beta_on":"2026-10-10"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	next := obj(r.data(), "next_appointment")
	assert.Equal(t, "scan", next["kind"])
	assert.Equal(t, "2026-09-24 09:00:00", obj(next, "appointment")["scheduled_at"])
	assert.Equal(t, "Follicle scan", obj(next, "appointment")["title"])
	assert.Equal(t, 2, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment'", uid))

	// moving a date moves the same reminder; clearing it removes it
	r = e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tok, `{"next_scan_at":"2026-09-25 10:30","beta_on":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment'", uid))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND scheduled_at = '2026-09-25 10:30:00'", uid))

	r = e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tok, `{"transfer_at":"2026-09-29 10:00","beta_on":"2026-09-28"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.errors(), "beta_on")

	r = e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tok, `{"notify_companion":true}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "no linked companion")
	assert.Contains(t, r.errors(), "notify_companion")

	// negative result: closed, next steps, upcoming appointments gone, cycle medicines' reminders off
	r = e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok, `{"name":"FSH","role":"stimulation","route":"subcutaneous","times":["20:00"]}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles/current/outcome", tok, `{"result":"negative"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"loss", "new_cycle"}, r.data()["next_steps"])
	assert.Equal(t, "closed", obj(r.data(), "cycle")["status"])
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'appointment'", uid))
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'medication' AND is_active = 1", uid))

	r = e.do(t, http.MethodGet, "/api/v1/ivf", tok, "")
	assert.Nil(t, r.data()["cycle"])
	assert.EqualValues(t, 1, r.data()["cycles_count"])
	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok, `{}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.EqualValues(t, 2, obj(r.data(), "cycle")["number"], "the next cycle is numbered after the last")
	assert.Equal(t, "prep", obj(r.data(), "cycle")["stage"])
}

func TestMedsSitesInventory(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000002")
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok,
		`{"started_on":"2026-09-10","stage":"stim","stim_started_on":"2026-09-17"}`).status)

	r := e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok,
		`{"name":"FSH","role":"stimulation","route":"subcutaneous","dose":150,"unit":"iu","times":["20:00"],"starts_on":"2026-09-17","stock_units":3,"stock_unit":"pen"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok,
		`{"name":"Antagonist","role":"suppression","route":"subcutaneous","times":["08:00"],"starts_on":"2026-09-21","stock_units":5,"stock_unit":"prefilled_syringe"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok,
		`{"name":"hCG","role":"trigger","route":"subcutaneous","trigger_at":"2026-09-25 22:30"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	fsh, anta := medID(t, r, "FSH"), medID(t, r, "Antagonist")

	// each medicine is a care medication reminder
	assert.Equal(t, 3, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND type = 'medication'", uid))
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND title = 'FSH' AND subtitle IS NOT NULL", uid))

	today := obj(r.data(), "today")
	assert.Len(t, list(today, "doses"), 2)
	assert.Len(t, list(obj(r.data(), "tomorrow"), "doses"), 2)
	trigger := obj(r.data(), "trigger")
	assert.Equal(t, "2026-09-25 22:30:00", trigger["trigger_at"])
	assert.Equal(t, false, trigger["taken"])
	sites := obj(r.data(), "sites")
	assert.Len(t, sites["codes"], 8)
	assert.Nil(t, sites["last"])
	assert.Equal(t, "abdomen_upper_right", sites["suggested"])
	for _, m := range list(r.data(), "meds") {
		if m["name"] == "FSH" {
			inv := obj(m, "inventory")
			assert.EqualValues(t, 3, inv["units_left"])
			assert.EqualValues(t, 3, inv["days_left"])
			assert.Equal(t, true, inv["low"])
		}
	}

	// log a dose with its site: care intake + site; rotation moves on
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", anta), tok, `{"slot":"08:00","site":"abdomen_upper_right"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM reminder_intakes WHERE user_id = ? AND intake_date = '2026-09-23' AND slot = '08:00'", uid))
	assert.Equal(t, "abdomen_upper_right", obj(r.data(), "sites", "last")["site"])
	assert.Equal(t, "abdomen_upper_left", obj(r.data(), "sites")["suggested"])
	d0 := list(obj(r.data(), "today"), "doses")[0]
	assert.Equal(t, true, d0["taken"])
	assert.Equal(t, "abdomen_upper_right", d0["site"])

	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", fsh), tok, `{"slot":"20:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	for _, m := range list(r.data(), "meds") {
		if m["name"] == "FSH" {
			assert.EqualValues(t, 2, obj(m, "inventory")["doses_left"], "a dose taken uses the stock")
		}
	}

	for name, tc := range map[string]struct{ body, field string }{
		"unknown slot":   {`{"slot":"09:00"}`, "slot"},
		"unknown site":   {`{"slot":"08:00","site":"elbow"}`, "site"},
		"future date":    {`{"slot":"08:00","date":"2026-09-24"}`, "date"},
		"not scheduled":  {`{"slot":"08:00","date":"2026-09-20"}`, "date"},
		"malformed slot": {`{"slot":"8am"}`, "slot"},
	} {
		r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", anta), tok, tc.body)
		assert.Equal(t, http.StatusUnprocessableEntity, r.status, name)
		assert.Contains(t, r.errors(), tc.field, name)
	}

	// undo
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", anta), tok, `{"date":"2026-09-23","slot":"08:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM ivf_dose_logs WHERE user_id = ? AND slot = '08:00'", uid))
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM reminder_intakes WHERE user_id = ? AND slot = '08:00'", uid))

	// sending GET's stock back keeps the count; a new count resets it
	r = e.do(t, http.MethodPut, fmt.Sprintf("/api/v1/ivf/meds/%d", fsh), tok,
		`{"name":"FSH","role":"stimulation","route":"subcutaneous","dose":"225","unit":"iu","times":["20:00"],"starts_on":"2026-09-17","stock_units":2,"stock_unit":"pen"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	for _, m := range list(r.data(), "meds") {
		if m["name"] == "FSH" {
			assert.Equal(t, "225", m["dose"])
			assert.EqualValues(t, 2, obj(m, "inventory")["doses_left"])
			assert.EqualValues(t, 3, obj(m, "inventory")["stock_units"], "count kept")
		}
	}

	// deleting the medicine removes its care reminder
	r = e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/ivf/meds/%d", fsh), tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, 0, e.count(t, "SELECT COUNT(*) FROM reminders WHERE user_id = ? AND title = 'FSH'", uid))
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, fmt.Sprintf("/api/v1/ivf/meds/%d", fsh), tok, "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, "/api/v1/ivf/meds/abc", tok, "").status)
}

func TestScansAndTWW(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000003")
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok,
		`{"started_on":"2026-09-10","stage":"stim","stim_started_on":"2026-09-17"}`).status)

	r := e.do(t, http.MethodPut, "/api/v1/ivf/scans/2026-09-23", tok,
		`{"right":{"lt_10":3,"10_14":4,"15_17":2,"18_plus":0},"left":{"lt_10":2,"10_14":5,"15_17":3,"18_plus":1},"endometrium_mm":8.5,"e2":1250}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	scans := list(r.data(), "scans")
	require.Len(t, scans, 1)
	assert.EqualValues(t, 7, scans[0]["stim_day"])
	assert.Equal(t, "8.5", scans[0]["endometrium_mm"])
	assert.Equal(t, "1250.00", scans[0]["e2"])
	assert.Equal(t, "pg_ml", scans[0]["e2_unit"])
	g := list(r.data(), "growth")[0]
	assert.EqualValues(t, 9, g["follicles_10_14"])
	assert.EqualValues(t, 6, g["follicles_15_plus"])

	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, "/api/v1/ivf/scans/2026-09-24", tok, `{}`).status, "future")
	r = e.do(t, http.MethodPut, "/api/v1/ivf/scans/2026-09-01", tok, `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.errors(), "date", "before the cycle start")

	r = e.do(t, http.MethodGet, "/api/v1/ivf", tok, "")
	assert.Equal(t, "2026-09-23", obj(r.data(), "latest_scan")["date"])

	r = e.do(t, http.MethodDelete, "/api/v1/ivf/scans/2026-09-23", tok, "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Empty(t, list(r.data(), "scans"))

	// two-week wait
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tok,
		`{"stage":"tww","retrieval_at":"2026-09-16 08:00","transfer_at":"2026-09-21 11:00","beta_on":"2026-10-02"}`).status)
	r = e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok,
		`{"name":"Progesterone","role":"luteal_support","route":"vaginal","times":["08:00","20:00"],"starts_on":"2026-09-16"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	progesterone := medID(t, r, "Progesterone")
	r = e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", progesterone), tok, `{"slot":"08:00","site":"thigh_left"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, "no site for a vaginal medicine")
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, fmt.Sprintf("/api/v1/ivf/meds/%d/doses", progesterone), tok, `{"slot":"08:00"}`).status)

	r = e.do(t, http.MethodPut, "/api/v1/ivf/tww/2026-09-23", tok, `{"mood":"hopeful"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "hopeful", obj(r.data(), "today")["mood"])
	assert.EqualValues(t, 2, r.data()["days_since_transfer"])
	assert.EqualValues(t, 9, r.data()["days_to_beta"])
	lut := list(r.data(), "luteal_support")
	require.Len(t, lut, 1)
	assert.EqualValues(t, 1, lut[0]["taken"])
	assert.EqualValues(t, 2, lut[0]["total"])

	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, "/api/v1/ivf/tww/2026-09-23", tok, `{"mood":"ecstatic"}`).status)
	r = e.do(t, http.MethodPut, "/api/v1/ivf/tww/2026-09-23", tok, `{"mood":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, obj(r.data(), "today")["mood"])

	// positive: closed, pregnancy setup next, luteal support keeps going
	r = e.do(t, http.MethodPost, "/api/v1/ivf/cycles/current/outcome", tok, `{"result":"positive"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"pregnancy_setup"}, r.data()["next_steps"])
	assert.Equal(t, 1, e.count(t, "SELECT COUNT(*) FROM reminders WHERE title = 'Progesterone' AND is_active = 1"))
}

func TestCompanionSharing(t *testing.T) {
	e := setup(t)
	owner, tok := e.user(t, "09120000004")
	partner, _ := e.user(t, "09120000005")
	res, err := e.db.Exec(`INSERT INTO companions (owner_id, companion_user_id, type, status, invited_at, accepted_at, created_at, updated_at)
		VALUES (?, ?, 'partner', 'active', '2026-09-20 10:00:00', '2026-09-20 10:05:00', '2026-09-20 10:00:00', '2026-09-20 10:05:00')`, owner, partner)
	require.NoError(t, err)
	link, _ := res.LastInsertId()
	_, err = e.db.Exec(`INSERT INTO companion_grants (companion_id, section, level, created_at, updated_at) VALUES (?, 'meds', 'view', NOW(), NOW())`, link)
	require.NoError(t, err)

	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tok, `{"notify_companion":true}`).status)
	r := e.do(t, http.MethodPost, "/api/v1/ivf/meds", tok, `{"name":"FSH","role":"stimulation","route":"subcutaneous","times":["20:00"]}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	r = e.do(t, http.MethodGet, "/api/v1/ivf", tok, "")
	comp := obj(r.data(), "companion")
	assert.Equal(t, true, comp["linked"])
	assert.Equal(t, true, comp["notify"])
	assert.Equal(t, true, comp["shares_meds"])
	assert.Equal(t, false, comp["shares_appointments"])

	// the companion's granted `meds` view (B-N4-02) includes the IVF medicine — it is a care medication
	view, err := shared.NewReader(e.db).Read(context.Background(), owner, companion.SectionMeds, "en", time.Now())
	require.NoError(t, err)
	raw, _ := json.Marshal(view)
	assert.Contains(t, string(raw), `"title":"FSH"`)
}

// User B can neither see nor change user A's IVF data.
func TestUserIsolation(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000006")
	_, tokB := e.user(t, "09120000007")
	require.Equal(t, http.StatusCreated, e.do(t, http.MethodPost, "/api/v1/ivf/cycles", tokA, `{"started_on":"2026-09-10"}`).status)
	r := e.do(t, http.MethodPost, "/api/v1/ivf/meds", tokA, `{"name":"FSH","role":"stimulation","route":"subcutaneous","times":["08:00"]}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	a := medID(t, r, "FSH")
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/ivf/scans/2026-09-20", tokA, `{"right":{"lt_10":2}}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/ivf/tww/2026-09-22", tokA, `{"mood":"calm"}`).status)

	r = e.do(t, http.MethodGet, "/api/v1/ivf", tokB, "")
	assert.Nil(t, r.data()["cycle"])
	assert.EqualValues(t, 0, r.data()["cycles_count"])
	r = e.do(t, http.MethodGet, "/api/v1/ivf/meds", tokB, "")
	assert.Empty(t, list(r.data(), "meds"))
	r = e.do(t, http.MethodGet, "/api/v1/ivf/scans", tokB, "")
	assert.Empty(t, list(r.data(), "scans"))
	r = e.do(t, http.MethodGet, "/api/v1/ivf/tww", tokB, "")
	assert.Empty(t, list(r.data(), "moods"))

	path := fmt.Sprintf("/api/v1/ivf/meds/%d", a)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodPut, path, tokB, `{"name":"x","role":"other","route":"oral","times":["09:00"]}`).status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, path, tokB, "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodPost, path+"/doses", tokB, `{"slot":"08:00"}`).status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodDelete, path+"/doses", tokB, `{"date":"2026-09-23","slot":"08:00"}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPut, "/api/v1/ivf/cycles/current", tokB, `{"stage":"stim"}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodDelete, "/api/v1/ivf/scans/2026-09-20", tokB, "").status)
	assert.Equal(t, http.StatusUnprocessableEntity, e.do(t, http.MethodPost, "/api/v1/ivf/cycles/current/outcome", tokB, `{"result":"negative"}`).status)

	// A's data untouched
	r = e.do(t, http.MethodGet, "/api/v1/ivf/meds", tokA, "")
	assert.Len(t, list(r.data(), "meds"), 1)
	assert.Equal(t, "open", obj(r.data(), "cycle")["status"])
	r = e.do(t, http.MethodGet, "/api/v1/ivf/scans", tokA, "")
	assert.Len(t, list(r.data(), "scans"), 1)
}
