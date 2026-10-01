package conditions_test

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
	"github.com/ritme/backend-go/internal/conditions"
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
	h := conditions.NewHandlers(conditions.NewService(db, cat), clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/conditions", locale, guard, h.Show)
	app.Post("/api/v1/conditions/enrolments", locale, guard, h.Enrol)
	app.Delete("/api/v1/conditions/enrolments/:program", locale, guard, h.Leave)
	app.Get("/api/v1/conditions/pain/:date", locale, guard, h.ShowPain)
	app.Put("/api/v1/conditions/pain/:date", locale, guard, h.SavePain)
	app.Get("/api/v1/conditions/pmdd/chart", locale, guard, h.PMDDChart)
	app.Get("/api/v1/conditions/pmdd/:date", locale, guard, h.ShowPMDD)
	app.Put("/api/v1/conditions/pmdd/:date", locale, guard, h.SavePMDD)
	app.Get("/api/v1/conditions/pbac/:date", locale, guard, h.ShowPBAC)
	app.Put("/api/v1/conditions/pbac/:date", locale, guard, h.SavePBAC)
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

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

func (e *env) enrol(t *testing.T, tok, program string) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/conditions/enrolments", tok, "en", `{"program":"`+program+`"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
}

// logRows are the user's health_log_entries of a day as "cat.param.item=code|num" strings.
func (e *env) logRows(t *testing.T, userID uint64, date string) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT category, param, item, COALESCE(value_code, ''), COALESCE(value_num, '')
		FROM health_log_entries WHERE user_id = ? AND log_date = ? ORDER BY category, param, item`, userID, date)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var c, p, i, code, num string
		require.NoError(t, rows.Scan(&c, &p, &i, &code, &num))
		out = append(out, c+"."+p+"."+i+"="+code+"|"+num)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestUnauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/conditions"},
		{http.MethodPost, "/api/v1/conditions/enrolments"},
		{http.MethodDelete, "/api/v1/conditions/enrolments/endo"},
		{http.MethodGet, "/api/v1/conditions/pain/2026-09-23"},
		{http.MethodPut, "/api/v1/conditions/pain/2026-09-23"},
		{http.MethodGet, "/api/v1/conditions/pmdd/chart"},
		{http.MethodGet, "/api/v1/conditions/pmdd/2026-09-23"},
		{http.MethodPut, "/api/v1/conditions/pmdd/2026-09-23"},
		{http.MethodGet, "/api/v1/conditions/pbac/2026-09-23"},
		{http.MethodPut, "/api/v1/conditions/pbac/2026-09-23"},
	} {
		r := e.do(t, c.method, c.path, "", "", `{}`)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.JSONEq(t, `{"message":"Unauthenticated.","error_code":"unauthenticated"}`, r.raw, c.path)
	}
}

func TestEnrolment_Flow(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000801")

	r := e.do(t, http.MethodGet, "/api/v1/conditions", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"success":true,"data":{"programs":[
		{"code":"endo","enrolled":false,"enrolled_on":null},{"code":"pmdd","enrolled":false,"enrolled_on":null},
		{"code":"heavy_bleeding","enrolled":false,"enrolled_on":null},{"code":"pcos","enrolled":false,"enrolled_on":null}]}}`, r.raw)

	r = e.do(t, http.MethodPost, "/api/v1/conditions/enrolments", tok, "fa", `{"program":"pcos","enrolled_on":"2026-09-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "به برنامه پیوستی", r.body["message"])
	assert.Contains(t, r.raw, `{"code":"pcos","enrolled":true,"enrolled_on":"2026-09-01"}`)

	// joining again keeps the first date
	r = e.do(t, http.MethodPost, "/api/v1/conditions/enrolments", tok, "en", `{"program":"pcos"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Contains(t, r.raw, `{"code":"pcos","enrolled":true,"enrolled_on":"2026-09-01"}`)

	r = e.do(t, http.MethodPost, "/api/v1/conditions/enrolments", tok, "en", `{"program":"diabetes","enrolled_on":"2026-09-30"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Contains(t, r.errors(), "program")
	assert.Equal(t, []any{"You cannot log a future day."}, r.errors()["enrolled_on"])

	r = e.do(t, http.MethodDelete, "/api/v1/conditions/enrolments/pcos", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "You left the program; your entries are kept", r.body["message"])
	assert.Contains(t, r.raw, `{"code":"pcos","enrolled":false,"enrolled_on":null}`)

	r = e.do(t, http.MethodDelete, "/api/v1/conditions/enrolments/diabetes", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

func TestPain_DiaryUsesTheLogTaxonomy(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000802")
	const path = "/api/v1/conditions/pain/2026-09-22"
	full := `{"score":7,"locations":["pelvis","back"],"relief":["heat","painkiller"],"types":["cramping","stabbing"],
		"associated":["dyspareunia","dyschezia","bloating"],"missed_activity":true,"analgesic":"Ibuprofen",
		"analgesic_time":"08:00","analgesic_effect":"a_little"}`

	r := e.do(t, http.MethodPut, path, tok, "en", full)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"Join this program first."}, r.errors()["program"])

	e.enrol(t, tok, "endo")
	r = e.do(t, http.MethodPut, path, tok, "fa", full)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "دفترچه درد ذخیره شد", r.body["message"])
	assert.JSONEq(t, `{"date":"2026-09-22","score":7,"locations":["pelvis","back"],"relief":["heat","painkiller"],
		"types":["cramping","stabbing"],"associated":["dyspareunia","dyschezia","bloating"],"missed_activity":true,
		"analgesic":"Ibuprofen","analgesic_time":"08:00","analgesic_effect":"a_little"}`, mustJSON(t, r.data()))

	// the taxonomy half is in health_log_entries (no parallel log) …
	assert.Equal(t, []string{
		"pain.location.back=severe|7.00", "pain.location.pelvis=severe|7.00",
		"pain.relief.heat=yes|", "pain.relief.painkiller=yes|",
		"sex.symptoms.pain_during_intercourse=yes|",
		"symptoms.digestive.bloating=yes|",
	}, e.logRows(t, uid, "2026-09-22"))
	// … and written back to the legacy day log the cycle engine reads
	var pelvic, back sql.NullString
	require.NoError(t, e.db.QueryRow(`SELECT pelvic_pain_intensity, back_pain_intensity FROM daily_health_logs WHERE user_id = ? AND log_date = '2026-09-22'`, uid).Scan(&pelvic, &back))
	assert.Equal(t, "high", pelvic.String)
	assert.Equal(t, "high", back.String)
	// only the slot-less code is in the program row
	var assoc string
	require.NoError(t, e.db.QueryRow(`SELECT associated FROM condition_pain_entries WHERE user_id = ?`, uid).Scan(&assoc))
	assert.JSONEq(t, `["dyschezia"]`, assoc)

	// partial: 0 = no pain clears the locations; deselecting bloating clears its slot; the rest stays
	r = e.do(t, http.MethodPut, path, tok, "en", `{"score":0,"associated":["dyschezia"],"analgesic":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, 0, d["score"])
	assert.Equal(t, []any{}, d["locations"])
	assert.Equal(t, []any{"dyschezia"}, d["associated"])
	assert.Equal(t, []any{"cramping", "stabbing"}, d["types"])
	assert.Nil(t, d["analgesic"])
	assert.Equal(t, "08:00", d["analgesic_time"])
	assert.Equal(t, []string{"pain.none.=yes|", "pain.relief.heat=yes|", "pain.relief.painkiller=yes|"}, e.logRows(t, uid, "2026-09-22"))

	// a symptom logged from the log sheet shows up as associated
	_, err := e.db.Exec(`INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_code, source, created_at, updated_at)
		VALUES (?, '2026-09-22', 'symptoms', 'digestive', 'nausea', 'severe', 'manual', NOW(), NOW())`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodGet, path, tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{"dyschezia", "nausea"}, r.data()["associated"])

	// score and locations must fit together
	r = e.do(t, http.MethodPut, path, tok, "en", `{"score":5}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"Choose where it hurts for a pain above 0."}, r.errors()["locations"])
	r = e.do(t, http.MethodPut, path, tok, "en", `{"score":null,"locations":["back"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"Choose how strong the pain is."}, r.errors()["score"])

	// validation
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pain/2026-09-24", tok, "en", `{"score":3}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"You cannot log a future day."}, r.errors()["date"])
	r = e.do(t, http.MethodPut, path, tok, "en", `{"score":11,"locations":["stitches"],"types":["sharp"]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	for _, k := range []string{"score", "locations.0", "types.0"} {
		assert.Contains(t, r.errors(), k)
	}

	// clearing everything removes the program row
	r = e.do(t, http.MethodPut, path, tok, "en", `{"score":null,"relief":null,"types":null,"associated":null,"missed_activity":null,"analgesic_time":null,"analgesic_effect":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM condition_pain_entries WHERE user_id = ?`, uid).Scan(&n))
	assert.Equal(t, 0, n)
	assert.Equal(t, []any{}, r.data()["associated"], "associated null clears the slot-backed symptoms too")
	assert.Empty(t, e.logRows(t, uid, "2026-09-22"))
}

func (e *env) period(t *testing.T, userID uint64, start string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO cycle_histories (user_id, period_start_date, bleeding_length, is_confirmed, created_at, updated_at)
		VALUES (?, ?, 5, 1, NOW(), NOW())`, userID, start)
	require.NoError(t, err)
}

func TestPMDD_QuestionnaireAndChart(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000803")

	r := e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/2026-09-23", tok, "en", `{"scores":{"sadness":3}}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "program")

	e.enrol(t, tok, "pmdd")
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/2026-09-23", tok, "en",
		`{"scores":{"concentration":2,"sadness":5,"anxiety":4,"mood_swings":6,"anger":3,"loss_of_interest":4}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "Today's questionnaire saved", r.body["message"])
	assert.Regexp(t, `"scores":\{"sadness":5,"anxiety":4,"mood_swings":6,"anger":3,"loss_of_interest":4,"concentration":2\},"mean":4\}`, r.raw)

	r = e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/2026-09-23", tok, "en", `{"scores":{"anger":null,"sadness":1}}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"sadness":1,"anxiety":4,"mood_swings":6,"loss_of_interest":4,"concentration":2}`, mustJSON(t, r.data()["scores"]))

	r = e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/2026-09-23", tok, "fa", `{"scores":{"joy":3,"sadness":7}}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "scores.sadness")

	// no recorded period: no cycles, not enough data; the crisis note is always there
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pmdd/chart", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, []any{}, d["cycles"])
	assert.Equal(t, "not_enough_data", d["pattern"])
	assert.Equal(t, false, d["ready"])
	codes := []string{}
	for _, a := range d["alerts"].([]any) {
		codes = append(codes, a.(map[string]any)["code"].(string))
	}
	assert.Equal(t, []string{"pmdd_not_enough_data", "pmdd_needs_two_cycles", "pmdd_crisis"}, codes)
	crisis := d["alerts"].([]any)[2].(map[string]any)
	assert.Contains(t, crisis["body"], "۱۴۸۰")
	assert.Equal(t, true, crisis["needs_review"])

	// two closed cycles fully rated (low after the period, high in the last 10 days) + the current one
	e.period(t, uid, "2026-07-30")
	e.period(t, uid, "2026-08-27")
	e.period(t, uid, "2026-09-20")
	_, err := e.db.Exec(`DELETE FROM pmdd_entries WHERE user_id = ?`, uid)
	require.NoError(t, err)
	rate := func(from string, days, score int) {
		start, _ := time.Parse("2006-01-02", from)
		for i := 0; i < days; i++ {
			s := `{"scores":{"sadness":` + string(rune('0'+score)) + `,"anxiety":` + string(rune('0'+score)) + `}}`
			r := e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/"+start.AddDate(0, 0, i).Format("2006-01-02"), tok, "en", s)
			require.Equal(t, http.StatusOK, r.status, r.raw)
		}
	}
	rate("2026-08-02", 7, 1)  // cycle 07-30: days 4–10
	rate("2026-08-17", 10, 5) // its last 10 days (08-17 … 08-26)
	rate("2026-08-30", 7, 2)  // cycle 08-27: days 4–10
	rate("2026-09-10", 10, 6) // its last 10 days (09-10 … 09-19)

	r = e.do(t, http.MethodGet, "/api/v1/conditions/pmdd/chart", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, true, d["ready"])
	assert.EqualValues(t, 2, d["complete_cycles"])
	assert.Equal(t, "luteal", d["pattern"])
	cycles := d["cycles"].([]any)
	require.Len(t, cycles, 3)
	cur := cycles[0].(map[string]any)
	assert.Equal(t, "2026-09-20", cur["start"])
	assert.Equal(t, false, cur["closed"])
	prev := cycles[1].(map[string]any)
	assert.Equal(t, "2026-08-27", prev["start"])
	assert.Equal(t, "2026-09-19", prev["end"])
	assert.JSONEq(t, `{"from":"2026-09-10","to":"2026-09-19"}`, mustJSON(t, prev["late_luteal"]))
	assert.JSONEq(t, `{"from":"2026-08-27","to":"2026-08-31"}`, mustJSON(t, prev["period"]))
	assert.EqualValues(t, 6, prev["luteal_mean"])
	assert.EqualValues(t, 2, prev["follicular_mean"])
	first := prev["days"].([]any)[0].(map[string]any)
	assert.EqualValues(t, 4, first["cycle_day"])
	assert.Equal(t, true, first["in_period"])
	codes = codes[:0]
	for _, a := range d["alerts"].([]any) {
		codes = append(codes, a.(map[string]any)["code"].(string))
	}
	assert.Equal(t, []string{"pmdd_pattern_luteal", "pmdd_crisis"}, codes)
}

func TestPBAC_ChartAndAlert(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000804")

	r := e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tok, "en", `{"light":1}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "program")

	e.enrol(t, tok, "heavy_bleeding")
	e.period(t, uid, "2026-09-21")

	// day 1: 2 medium + 2 fully soaked = 50, a large clot +5, flooding +5 → 60
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-21", tok, "en", `{"medium":2,"heavy":2,"clots":"large","flooding":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 60, r.data()["score"])
	assert.Nil(t, r.data()["alert"])
	assert.Equal(t, []string{"bleeding.clot_size.=large|", "bleeding.clots.=yes|"}, e.logRows(t, uid, "2026-09-21"), "clots live in the log taxonomy")

	// day 2 (board): 1 light, 2 medium, 3 heavy = 71 → period 131 ≥ 100
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tok, "fa", `{"light":1,"medium":2,"heavy":3,"clots":"none"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "جدول خونریزی ذخیره شد", r.body["message"])
	d := r.data()
	assert.EqualValues(t, 71, d["score"])
	assert.Equal(t, "none", d["clots"])
	assert.JSONEq(t, `{"start":"2026-09-21","end":"2026-09-30","day":2,"score":131,"days_logged":2}`, mustJSON(t, d["period"]))
	alert := d["alert"].(map[string]any)
	assert.Equal(t, "pbac_over_100", alert["code"])
	assert.Equal(t, "امتیاز این پریود از ۱۰۰ بیشتر شده", alert["title"])
	assert.Contains(t, alert["body"], "فریتین")

	// partial update keeps the other counts; null clears the clot chip
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tok, "en", `{"heavy":0,"clots":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.EqualValues(t, 11, d["score"])
	assert.Nil(t, d["clots"])
	assert.Nil(t, d["alert"])
	assert.Empty(t, e.logRows(t, uid, "2026-09-22"))

	r = e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tok, "en", `{"light":51,"clots":"huge"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "light")
	assert.Contains(t, r.errors(), "clots")

	// an unlogged day outside any period
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pbac/2026-08-01", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.JSONEq(t, `{"date":"2026-08-01","light":0,"medium":0,"heavy":0,"clots":null,"flooding":false,"score":0,"period":null,"alert_threshold":100,"alert":null}`, mustJSON(t, r.data()))
}

func TestUserIsolation(t *testing.T) {
	e := setup(t)
	uidA, tokA := e.user(t, "09120000805")
	uidB, tokB := e.user(t, "09120000806")
	for _, p := range []string{"endo", "pmdd", "heavy_bleeding"} {
		e.enrol(t, tokA, p)
	}
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/conditions/pain/2026-09-22", tokA, "en", `{"score":6,"locations":["back"],"types":["burning"]}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/conditions/pmdd/2026-09-22", tokA, "en", `{"scores":{"sadness":5}}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tokA, "en", `{"heavy":2}`).status)

	// B sees none of A's data and no enrolment
	r := e.do(t, http.MethodGet, "/api/v1/conditions", tokB, "en", "")
	assert.NotContains(t, r.raw, `"enrolled":true`)
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pain/2026-09-22", tokB, "en", "")
	assert.Nil(t, r.data()["score"])
	assert.Equal(t, []any{}, r.data()["types"])
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pmdd/2026-09-22", tokB, "en", "")
	assert.JSONEq(t, `{}`, mustJSON(t, r.data()["scores"]))
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pbac/2026-09-22", tokB, "en", "")
	assert.EqualValues(t, 0, r.data()["score"])

	// B can't write without her own enrolment, and her writes/leave never touch A's rows
	r = e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tokB, "en", `{"heavy":0}`)
	assert.Equal(t, http.StatusUnprocessableEntity, r.status)
	e.enrol(t, tokB, "heavy_bleeding")
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, "/api/v1/conditions/pbac/2026-09-22", tokB, "en", `{"light":1}`).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, "/api/v1/conditions/enrolments/heavy_bleeding", tokB, "en", "").status)

	var heavy int
	require.NoError(t, e.db.QueryRow(`SELECT heavy_count FROM pbac_entries WHERE user_id = ? AND entry_date = '2026-09-22'`, uidA).Scan(&heavy))
	assert.Equal(t, 2, heavy)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM condition_enrolments WHERE user_id = ?`, uidA).Scan(&n))
	assert.Equal(t, 3, n)
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM pbac_entries WHERE user_id = ?`, uidB).Scan(&n))
	assert.Equal(t, 1, n)
	r = e.do(t, http.MethodGet, "/api/v1/conditions/pain/2026-09-22", tokA, "en", "")
	assert.EqualValues(t, 6, r.data()["score"])

	// account deletion cascades
	_, err := e.db.Exec(`DELETE FROM users WHERE id = ?`, uidA)
	require.NoError(t, err)
	for _, table := range []string{"condition_enrolments", "condition_pain_entries", "pmdd_entries", "pbac_entries"} {
		require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, uidA).Scan(&n))
		assert.Equal(t, 0, n, table)
	}
}
