package healthrecord_test

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
	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
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
	svc *healthrecord.Service
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
	svc := healthrecord.NewService(db, nil)
	h := healthrecord.NewHandlers(svc, clock.Fixed(fixed))

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	p := "/api/v1/health-record"
	app.Get(p, locale, guard, h.Show)
	app.Put(p+"/basics", locale, guard, h.UpdateBasics)
	app.Post(p+"/pregnancies", locale, guard, h.StorePregnancy)
	app.Put(p+"/pregnancies/:id", locale, guard, h.UpdatePregnancy)
	app.Delete(p+"/pregnancies/:id", locale, guard, h.DestroyPregnancy)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365), svc: svc}
}

func (e *env) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	_, err := e.db.Exec(query, args...)
	require.NoError(t, err)
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Maryam', ?, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	tok, err := e.iss.Issue(context.Background(), uint64(id), time.Now())
	require.NoError(t, err)
	return uint64(id), tok.AccessToken
}

type response struct {
	status int
	raw    string
	body   map[string]any
}

func (r response) data() map[string]any {
	d, _ := r.body["data"].(map[string]any)
	return d
}

func (r response) section(t *testing.T, key string) map[string]any {
	t.Helper()
	secs, _ := r.data()["sections"].([]any)
	for _, s := range secs {
		m, _ := s.(map[string]any)
		if m["key"] == key {
			return m
		}
	}
	t.Fatalf("section %s missing", key)
	return nil
}

func (e *env) do(t *testing.T, method, path, token string, body any) response {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rd = strings.NewReader(string(b))
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := e.app.Test(req)
	require.NoError(t, err)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), string(raw))
	return response{status: res.StatusCode, raw: string(raw), body: m}
}

func TestHealthRecord_Unauthenticated(t *testing.T) {
	e := setup(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/health-record"}, {http.MethodPut, "/api/v1/health-record/basics"},
		{http.MethodPost, "/api/v1/health-record/pregnancies"}, {http.MethodDelete, "/api/v1/health-record/pregnancies/1"},
	} {
		r := e.do(t, c.method, c.path, "", nil)
		assert.Equal(t, http.StatusUnauthorized, r.status, c.path)
		assert.Equal(t, "unauthenticated", r.body["error_code"])
	}
}

func TestHealthRecord_EmptyRecord(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")
	r := e.do(t, http.MethodGet, "/api/v1/health-record", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "owner", r.data()["audience"])
	secs, _ := r.data()["sections"].([]any)
	require.Len(t, secs, len(healthrecord.Sections))
	for i, key := range healthrecord.Sections {
		s, _ := secs[i].(map[string]any)
		assert.Equal(t, key, s["key"])
		assert.Equal(t, true, s["empty"], key)
	}
	assert.Equal(t, true, r.section(t, "basics")["editable"])
	assert.Equal(t, false, r.section(t, "vitals")["editable"])
}

func TestHealthRecord_AggregatesSources(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")
	e.exec(t, `INSERT INTO user_profiles (user_id, birthday, weight, height, created_at, updated_at)
		VALUES (?, '1993-05-01', 58.00, 164, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO user_life_profiles (user_id, gender, life_mode, chronic_illnesses, gyn_conditions, medications, created_at, updated_at)
		VALUES (?, 'female', 'cycle', '["thyroid"]', '["pcos"]', '[]', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO reminders (user_id, type, title, subtitle, notes, recurrence, is_active, created_at, updated_at)
		VALUES (?, 'medication', 'Levothyroxine', '50 mcg', 'empty stomach', 'daily', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO reminders (user_id, type, title, recurrence, is_active, created_at, updated_at)
		VALUES (?, 'medication', 'Old pill', 'daily', 0, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO vital_readings (user_id, type, measured_at, systolic, diastolic, created_at, updated_at)
		VALUES (?, 'bp', '2026-10-05 08:00:00', 120, 80, '2026-10-05 08:00:00', '2026-10-05 08:00:00')`, uid)
	e.exec(t, `INSERT INTO vital_readings (user_id, type, measured_at, glucose_mg_dl, glucose_unit, context, created_at, updated_at)
		VALUES (?, 'glucose', '2026-10-05 07:30:00', 94.0, 'mg_dl', 'fasting', '2026-10-05 08:00:00', '2026-10-05 08:00:00')`, uid)
	for _, v := range []struct{ date, param, num string }{
		{"2026-10-05", "bp_systolic", "150"}, {"2026-10-05", "bp_diastolic", "95"}, // hidden by the timed reading
		{"2026-10-03", "bp_systolic", "130"}, {"2026-10-03", "bp_diastolic", "84"},
		{"2026-08-01", "bp_systolic", "180"}, {"2026-08-01", "bp_diastolic", "110"}, // outside 30 days
	} {
		e.exec(t, `INSERT INTO health_log_entries (user_id, log_date, category, param, item, value_num, source, created_at, updated_at)
			VALUES (?, ?, 'measurements', ?, '', ?, 'manual', '2026-10-06 08:00:00', '2026-10-06 08:00:00')`, uid, v.date, v.param, v.num)
	}
	var pap uint64
	require.NoError(t, e.db.QueryRow("SELECT id FROM checkup_types WHERE user_id IS NULL ORDER BY sort_order, id LIMIT 1").Scan(&pap))
	e.exec(t, `INSERT INTO checkup_records (user_id, checkup_type_id, done_on, result, note, created_at, updated_at)
		VALUES (?, ?, '2026-09-10', 'normal', 'private note', '2026-09-10 09:00:00', '2026-09-10 09:00:00')`, uid, pap)

	r := e.do(t, http.MethodGet, "/api/v1/health-record", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	person, _ := r.data()["person"].(map[string]any)
	assert.Equal(t, "Maryam", person["name"])
	assert.EqualValues(t, 33, person["age"])

	basics, _ := r.section(t, "basics")["data"].(map[string]any)
	assert.EqualValues(t, 164, basics["height_cm"])
	assert.EqualValues(t, 58, basics["weight_kg"])
	bmi, _ := basics["bmi"].(map[string]any)
	assert.EqualValues(t, 21.6, bmi["value"])
	assert.Nil(t, basics["blood_type"])

	cond, _ := r.section(t, "conditions")["data"].(map[string]any)
	assert.Equal(t, []any{"thyroid"}, cond["chronic_illnesses"])
	assert.Equal(t, []any{"pcos"}, cond["gyn_conditions"])

	meds, _ := r.section(t, "medications")["data"].(map[string]any)
	items, _ := meds["items"].([]any)
	require.Len(t, items, 1, "only active medications")
	assert.Equal(t, "Levothyroxine", items[0].(map[string]any)["title"])
	assert.Equal(t, "empty stomach", items[0].(map[string]any)["notes"])

	vit, _ := r.section(t, "vitals")["data"].(map[string]any)
	bp, _ := vit["blood_pressure"].(map[string]any)
	assert.EqualValues(t, 2, bp["readings"], "timed 10-05 hides that day's log value; 08-01 is outside the window")
	assert.EqualValues(t, 125, bp["systolic"])
	assert.EqualValues(t, 82, bp["diastolic"])
	fasting, _ := vit["glucose_fasting"].(map[string]any)
	assert.EqualValues(t, 94, fasting["avg"])
	assert.EqualValues(t, 100, fasting["in_target_percent"])
	assert.Nil(t, vit["heart_rate"])

	chk, _ := r.section(t, "checkups")["data"].(map[string]any)
	ci, _ := chk["items"].([]any)
	require.Len(t, ci, 1)
	assert.Equal(t, "2026-09-10", ci[0].(map[string]any)["done_on"])
	assert.NotContains(t, r.raw, "private note", "checkup notes are not part of the record")
}

func TestHealthRecord_ShareAudienceDropsNotesAndEditable(t *testing.T) {
	e := setup(t)
	uid, _ := e.user(t, "09120000001")
	e.exec(t, `INSERT INTO reminders (user_id, type, title, notes, recurrence, is_active, created_at, updated_at)
		VALUES (?, 'medication', 'Folic acid', 'my note', 'daily', 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	e.exec(t, `INSERT INTO health_record_pregnancies (user_id, outcome, ended_on, created_at, updated_at)
		VALUES (?, 'vaginal', '2024-01-01', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid)
	rec, err := e.svc.Build(context.Background(), uid, healthrecord.AudienceShare,
		healthrecord.Options{Today: civildate.InTehran(fixed), Locale: "en", DefaultLocale: "fa"})
	require.NoError(t, err)
	b, err := json.Marshal(rec.JSON())
	require.NoError(t, err)
	assert.NotContains(t, string(b), "my note")
	assert.NotContains(t, string(b), `"editable":true`)
	preg, ok := rec.Section(healthrecord.SectionPregnancies)
	require.True(t, ok)
	items, _ := preg.Data.Get("items")
	first := items.([]*jsonx.OrderedMap)[0]
	id, _ := first.Get("id")
	assert.Nil(t, id, "share view carries no row ids")
}

func TestHealthRecord_LossPrivacy(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")
	e.exec(t, `INSERT INTO pregnancy_losses (user_id, loss_type, occurred_on, private_note, next_step, created_at, updated_at)
		VALUES (?, 'ectopic', '2025-11-20', 'Q0lQSEVSVEVYVA==', 'ttc', '2025-11-21 09:00:00', '2025-11-21 09:00:00')`, uid)
	var lossID int64
	require.NoError(t, e.db.QueryRow("SELECT id FROM pregnancy_losses WHERE user_id = ?", uid).Scan(&lossID))
	e.exec(t, `INSERT INTO pregnancy_loss_moods (user_id, loss_id, log_date, mood, created_at, updated_at)
		VALUES (?, ?, '2025-11-22', 'numb', '2025-11-22 09:00:00', '2025-11-22 09:00:00')`, uid, lossID)
	e.exec(t, `INSERT INTO postpartum_profiles (user_id, birth_date, delivery_type, baby_count, source, created_at, updated_at)
		VALUES (?, '2024-06-10', 'cesarean', 1, 'direct', '2024-06-11 09:00:00', '2024-06-11 09:00:00')`, uid)

	r := e.do(t, http.MethodGet, "/api/v1/health-record", tok, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	for _, leak := range []string{"ectopic", "2025-11-20", "2025-11-21", "2025-11-22", "Q0lQSEVSVEVYVA", "numb", "ttc", "loss"} {
		assert.NotContains(t, r.raw, leak)
	}
	preg, _ := r.section(t, "pregnancies")["data"].(map[string]any)
	assert.EqualValues(t, 2, preg["pregnancies_count"])
	assert.EqualValues(t, 1, preg["births_count"])
	items, _ := preg["items"].([]any)
	require.Len(t, items, 2)
	birth, _ := items[0].(map[string]any)
	assert.Equal(t, "cesarean", birth["outcome"])
	assert.Equal(t, "2024-06-10", birth["date"])
	ended, _ := items[1].(map[string]any)
	assert.Equal(t, "ended", ended["outcome"])
	assert.Nil(t, ended["date"])
	assert.Nil(t, ended["id"])
	assert.Equal(t, false, ended["editable"])
}

func TestHealthRecord_BasicsEditAndValidation(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001")
	e.exec(t, `INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, blood_type, rh_factor, created_at, updated_at)
		VALUES (?, 0, 0, 'lmp', 'O', 'negative', '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, uid)
	r := e.do(t, http.MethodGet, "/api/v1/health-record", tok, nil)
	basics, _ := r.section(t, "basics")["data"].(map[string]any)
	assert.Equal(t, "O-", basics["blood_type"], "pregnancy profile fallback")
	assert.Equal(t, "pregnancy", basics["blood_type_source"])

	r = e.do(t, http.MethodPut, "/api/v1/health-record/basics", tok, map[string]any{"blood_type": "Z+", "allergies": "x"})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	errs, _ := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "blood_type")
	assert.Contains(t, errs, "allergies")

	r = e.do(t, http.MethodPut, "/api/v1/health-record/basics", tok,
		map[string]any{"blood_type": "A+", "allergies": []any{" Penicillin ", "penicillin", "Pollen"}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	basics, _ = r.section(t, "basics")["data"].(map[string]any)
	assert.Equal(t, "A+", basics["blood_type"])
	assert.Equal(t, "record", basics["blood_type_source"])
	all, _ := r.section(t, "allergies")["data"].(map[string]any)
	assert.Equal(t, []any{"Penicillin", "Pollen"}, all["items"])

	// An absent key keeps its value; [] = «none».
	r = e.do(t, http.MethodPut, "/api/v1/health-record/basics", tok, map[string]any{"allergies": []any{}})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	basics, _ = r.section(t, "basics")["data"].(map[string]any)
	assert.Equal(t, "A+", basics["blood_type"])
	all, _ = r.section(t, "allergies")["data"].(map[string]any)
	assert.Equal(t, []any{}, all["items"])
	assert.Equal(t, true, all["answered"])

	r = e.do(t, http.MethodPut, "/api/v1/health-record/basics", tok, map[string]any{"allergies": []any{"   "}})
	assert.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
}

func TestHealthRecord_PregnanciesCRUDAndIDOR(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000001")
	_, tokB := e.user(t, "09120000002")

	r := e.do(t, http.MethodPost, "/api/v1/health-record/pregnancies", tokA,
		map[string]any{"outcome": "twins", "ended_on": "2030-01-01", "baby_count": 5})
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	errs, _ := r.body["errors"].(map[string]any)
	for _, f := range []string{"outcome", "ended_on", "baby_count"} {
		assert.Contains(t, errs, f)
	}

	r = e.do(t, http.MethodPost, "/api/v1/health-record/pregnancies", tokA,
		map[string]any{"outcome": "vaginal", "ended_on": "2024-01-01", "baby_count": 1})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	id := uint64(r.data()["id"].(float64))
	assert.Equal(t, "manual", r.data()["source"])
	assert.Equal(t, true, r.data()["editable"])

	r = e.do(t, http.MethodPost, "/api/v1/health-record/pregnancies", tokA, map[string]any{"outcome": "ended", "baby_count": 2})
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Nil(t, r.data()["baby_count"], "an ended pregnancy has no babies")

	path := fmt.Sprintf("/api/v1/health-record/pregnancies/%d", id)
	// B cannot see, change or delete A's entry.
	r = e.do(t, http.MethodPut, path, tokB, map[string]any{"outcome": "cesarean"})
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "record_pregnancy_not_found", r.body["error_code"])
	r = e.do(t, http.MethodDelete, path, tokB, nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	r = e.do(t, http.MethodGet, "/api/v1/health-record", tokB, nil)
	pregB, _ := r.section(t, "pregnancies")["data"].(map[string]any)
	assert.EqualValues(t, 0, pregB["pregnancies_count"])

	r = e.do(t, http.MethodPut, path, tokA, map[string]any{"outcome": "cesarean", "ended_on": "2023-01-01"})
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "cesarean", r.data()["outcome"])
	assert.Equal(t, "2023-01-01", r.data()["date"])

	r = e.do(t, http.MethodGet, "/api/v1/health-record", tokA, nil)
	pregA, _ := r.section(t, "pregnancies")["data"].(map[string]any)
	assert.EqualValues(t, 2, pregA["pregnancies_count"])
	assert.EqualValues(t, 1, pregA["births_count"])

	r = e.do(t, http.MethodDelete, path, tokA, nil)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.do(t, http.MethodDelete, path, tokA, nil)
	assert.Equal(t, http.StatusNotFound, r.status)
	r = e.do(t, http.MethodDelete, "/api/v1/health-record/pregnancies/abc", tokA, nil)
	assert.Equal(t, http.StatusNotFound, r.status)
}
