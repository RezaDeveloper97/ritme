package vitals_test

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
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/vitals"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// fixed is the request clock: Tuesday 2026-10-06 09:00 Tehran.
var fixed = time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)

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
	h := vitals.NewHandlers(vitals.NewService(db), clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/vitals"
	app.Get(p, locale, guard, h.Hub)
	app.Get(p+"/thresholds", locale, guard, h.Thresholds)
	app.Get(p+"/plan", locale, guard, h.Plan)
	app.Put(p+"/plan", locale, guard, h.SavePlan)
	app.Get(p+"/reports/:type", locale, guard, h.Report)
	app.Get(p+"/readings", locale, guard, h.Readings)
	app.Post(p+"/readings", locale, guard, h.Store)
	app.Get(p+"/readings/:id", locale, guard, h.Show)
	app.Put(p+"/readings/:id", locale, guard, h.Update)
	app.Delete(p+"/readings/:id", locale, guard, h.Destroy)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
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
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (e *env) do(t *testing.T, method, path, token, lang string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, body: m}
}

func TestVitals_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, path := range []string{"/api/v1/vitals", "/api/v1/vitals/readings", "/api/v1/vitals/plan", "/api/v1/vitals/reports/bp"} {
		r := e.do(t, http.MethodGet, path, "", "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestVitals_ReadingsCRUDAlertAndIDOR(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000001")
	_, tokB := e.user(t, "09120000002")

	r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tokA, "fa", map[string]any{
		"type": "bp", "systolic": 118, "diastolic": 76, "pulse": 70, "arm": "left", "position": "sitting",
		"measured_at": "2026-10-06 08:10",
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "ثبت شد", r.body["message"])
	reading := r.data()["reading"].(map[string]any)
	assert.Nil(t, r.data()["alert"])
	assert.Equal(t, "normal", reading["classification"].(map[string]any)["code"])
	assert.Equal(t, "08:10", reading["time"])
	assert.Equal(t, "morning", reading["period"])
	assert.Equal(t, "2026-10-06T08:10:00+03:30", reading["measured_at"])
	id := uint64(reading["id"].(float64))

	// Crisis BP → urgent modal with the seeded (admin-editable) copy.
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tokA, "en", map[string]any{
		"type": "bp", "systolic": 185, "diastolic": 112,
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	alert := r.data()["alert"].(map[string]any)
	assert.Equal(t, "bp_crisis", alert["rule"])
	assert.Equal(t, "Your blood pressure is in the crisis range", alert["title"])
	assert.Contains(t, alert["what_we_saw"], "185/112 mmHg")
	assert.Equal(t, "115", alert["actions"].([]any)[0].(map[string]any)["phone"])
	assert.Equal(t, true, r.data()["reading"].(map[string]any)["urgent"])

	// An admin edit of the copy is used on the next request.
	_, err := e.db.Exec(`UPDATE message_contents SET payload = JSON_SET(payload, '$.title', 'Edited title')
		WHERE ` + "`group`" + ` = 'vitals_alert' AND item_key = 'glucose_low' AND locale = 'en'`)
	require.NoError(t, err)
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tokA, "en", map[string]any{
		"type": "glucose", "value": 2.8, "unit": "mmol_l", "context": "random", "method": "glucometer",
	})
	require.Equal(t, http.StatusCreated, r.status, r.body)
	assert.Equal(t, "Edited title", r.data()["alert"].(map[string]any)["title"])
	g := r.data()["reading"].(map[string]any)["glucose"].(map[string]any)
	assert.InDelta(t, 2.8, g["value"], 1e-9)
	assert.InDelta(t, 50, g["mg_dl"], 1e-9) // 2.8 × 18 = 50.4
	assert.Equal(t, "mmol_l", g["unit"])

	// Validation (controller-style 422).
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tokA, "en", map[string]any{"type": "bp", "systolic": 400})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, false, r.body["success"])
	errs := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "systolic")
	assert.Contains(t, errs, "diastolic")
	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tokA, "en", map[string]any{"type": "hr", "bpm": 72, "context": "running"})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	// Owner reads / updates / deletes.
	path := fmt.Sprintf("/api/v1/vitals/readings/%d", id)
	r = e.do(t, http.MethodGet, path, tokA, "", nil)
	require.Equal(t, http.StatusOK, r.status)
	r = e.do(t, http.MethodPut, path, tokA, "en", map[string]any{"type": "bp", "systolic": 132, "diastolic": 86,
		"measured_at": "2026-10-05 22:00"})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "stage1", r.data()["reading"].(map[string]any)["classification"].(map[string]any)["code"])
	assert.Equal(t, "night", r.data()["reading"].(map[string]any)["period"])

	// IDOR: user B gets a uniform 404 on A's reading and sees none of A's data.
	for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		var b any
		if m == http.MethodPut {
			b = map[string]any{"type": "bp", "systolic": 120, "diastolic": 80}
		}
		r = e.do(t, m, path, tokB, "en", b)
		assert.Equal(t, http.StatusNotFound, r.status, m)
		assert.Equal(t, "vital_not_found", r.body["error_code"], m)
	}
	r = e.do(t, http.MethodGet, "/api/v1/vitals/readings", tokB, "", nil)
	assert.Equal(t, float64(0), r.data()["count"])
	r = e.do(t, http.MethodGet, path, tokA, "", nil)
	assert.Equal(t, float64(132), r.data()["blood_pressure"].(map[string]any)["systolic"], "B's PUT changed nothing")

	r = e.do(t, http.MethodDelete, path, tokA, "en", nil)
	require.Equal(t, http.StatusOK, r.status)
	r = e.do(t, http.MethodGet, path, tokA, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	r = e.do(t, http.MethodGet, "/api/v1/vitals/readings/abc", tokA, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)
}

func TestVitals_ListMergesLogSheetAndReports(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000003")
	// Log-sheet day values (taxonomy measurements.*): 10-04 BP + glucose, 10-06 BP hidden by a timed reading.
	for _, v := range []struct {
		date, param, num string
	}{
		{"2026-10-04", "bp_systolic", "124"}, {"2026-10-04", "bp_diastolic", "80"},
		{"2026-10-04", "blood_sugar", "101.5"},
		{"2026-10-06", "bp_systolic", "150"}, {"2026-10-06", "bp_diastolic", "95"},
	} {
		_, err := e.db.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_num, source, created_at, updated_at)
			VALUES (?, ?, 'measurements', ?, '', ?, 'manual', '2026-10-06 08:00:00', '2026-10-06 08:00:00')`, uid, v.date, v.param, v.num)
		require.NoError(t, err)
	}
	for _, b := range []map[string]any{
		{"type": "bp", "systolic": 118, "diastolic": 76, "measured_at": "2026-10-06 08:10"},
		{"type": "bp", "systolic": 132, "diastolic": 86, "measured_at": "2026-10-05 22:00"},
		{"type": "glucose", "value": 94, "unit": "mg_dl", "context": "fasting", "measured_at": "2026-10-06 07:45"},
		{"type": "hr", "bpm": 72, "context": "resting", "measured_at": "2026-10-05 22:00"},
	} {
		r := e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", b)
		require.Equal(t, http.StatusCreated, r.status, r.body)
	}

	r := e.do(t, http.MethodGet, "/api/v1/vitals/readings?type=bp", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	items := r.data()["items"].([]any)
	require.Len(t, items, 3, "two timed + the 10-04 sheet value; the 10-06 sheet value is hidden by the timed one")
	last := items[2].(map[string]any)
	assert.Equal(t, "log", last["source"])
	assert.Nil(t, last["id"])

	r = e.do(t, http.MethodGet, "/api/v1/vitals/readings?from=2026-10-07&to=2026-10-01", tok, "en", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	r = e.do(t, http.MethodGet, "/api/v1/vitals/reports/bp?range=7d", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	d := r.data()
	assert.Equal(t, float64(3), d["readings"])
	assert.Equal(t, float64(125), d["average"].(map[string]any)["systolic"]) // (118+132+124)/3 = 124.67
	r = e.do(t, http.MethodGet, "/api/v1/vitals/reports/glucose", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "14d", r.data()["range"].(map[string]any)["key"])
	assert.Equal(t, float64(100), r.data()["time_in_range"].(map[string]any)["percent"])
	r = e.do(t, http.MethodGet, "/api/v1/vitals/reports/glucose?filter=lunch", tok, "en", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	r = e.do(t, http.MethodGet, "/api/v1/vitals/reports/weight", tok, "en", nil)
	assert.Equal(t, http.StatusNotFound, r.status)

	r = e.do(t, http.MethodGet, "/api/v1/vitals", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	latest := r.data()["latest"].(map[string]any)
	assert.Equal(t, float64(118), latest["bp"].(map[string]any)["blood_pressure"].(map[string]any)["systolic"])
	assert.Equal(t, float64(72), latest["hr"].(map[string]any)["heart_rate"].(map[string]any)["bpm"])
	assert.Len(t, r.data()["recent"], 5)
}

func TestVitals_Plan(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000004")
	_, tokB := e.user(t, "09120000005")

	r := e.do(t, http.MethodGet, "/api/v1/vitals/plan", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, []any{}, r.data()["items"])
	assert.Equal(t, map[string]any{"category": "vitals", "enabled": false}, r.data()["notifications"])

	r = e.do(t, http.MethodPut, "/api/v1/vitals/plan", tok, "en", map[string]any{"items": []any{
		map[string]any{"type": "bp", "slot": "morning", "days": []int{0, 1, 2, 3, 4, 5, 6}, "remind_at": "08:00"},
		map[string]any{"type": "bp", "slot": "evening", "days": []int{0, 3}},
		map[string]any{"type": "glucose", "slot": "fasting", "days": []int{3}, "remind_at": "07:30"},
	}})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, "Plan saved", r.body["message"])
	assert.Len(t, r.data()["items"], 3)
	week := r.data()["week"].(map[string]any)
	assert.Equal(t, float64(10), week["planned"])
	assert.Equal(t, "2026-10-03", week["from"])

	r = e.do(t, http.MethodPost, "/api/v1/vitals/readings", tok, "en", map[string]any{"type": "bp", "systolic": 118,
		"diastolic": 76, "measured_at": "2026-10-06 08:10"})
	require.Equal(t, http.StatusCreated, r.status)
	r = e.do(t, http.MethodGet, "/api/v1/vitals", tok, "", nil)
	assert.Equal(t, float64(1), r.data()["plan"].(map[string]any)["done"])

	r = e.do(t, http.MethodPut, "/api/v1/vitals/plan", tok, "en", map[string]any{"items": []any{
		map[string]any{"type": "hr", "slot": "fasting", "days": []int{1}},
	}})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.body["errors"], "items.0.slot")
	r = e.do(t, http.MethodPut, "/api/v1/vitals/plan", tok, "en", map[string]any{"items": []any{
		map[string]any{"type": "bp", "slot": "morning", "days": []int{9}},
	}})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)

	r = e.do(t, http.MethodGet, "/api/v1/vitals/plan", tokB, "", nil)
	assert.Equal(t, []any{}, r.data()["items"], "plans are per user")

	r = e.do(t, http.MethodPut, "/api/v1/vitals/plan", tok, "en", map[string]any{"items": []any{}})
	require.Equal(t, http.StatusOK, r.status, r.body)
	assert.Equal(t, []any{}, r.data()["items"])
}

func TestVitals_Thresholds(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000006")
	r := e.do(t, http.MethodGet, "/api/v1/vitals/thresholds", tok, "", nil)
	require.Equal(t, http.StatusOK, r.status)
	assert.Equal(t, vitals.ThresholdsVersion, r.data()["version"])
	assert.Equal(t, float64(54), r.data()["glucose"].(map[string]any)["urgent_below"])
}
