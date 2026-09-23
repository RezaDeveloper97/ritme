package care_test

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
	"github.com/ritme/backend-go/internal/i18n"
	i18nstore "github.com/ritme/backend-go/internal/i18n/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/reminder"
)

func TestMain(m *testing.M) { testdb.Main(m) }

const (
	clientID = "0199c0de-0000-7000-8000-00000c0ffee1"
	// Wednesday 2026-09-23 (Saturday-based weekday 4).
	now = "2026-09-23T10:00:00+03:30"
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
	h := care.NewHandlers(db, clock.Real{})
	legacy := reminder.NewHandlers(db, clock.Real{})

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler(quiet)})
	app.Use(clock.Middleware(clock.Real{}, true))
	app.Get("/api/v1/care/enums", locale, guard, h.Enums)
	app.Get("/api/v1/care/medications", locale, guard, h.ListMedications)
	app.Post("/api/v1/care/medications", locale, guard, h.StoreMedication)
	app.Get("/api/v1/care/medications/:id", locale, guard, h.ShowMedication)
	app.Put("/api/v1/care/medications/:id", locale, guard, h.UpdateMedication)
	app.Delete("/api/v1/care/medications/:id", locale, guard, h.DestroyMedication)
	app.Post("/api/v1/care/medications/:id/intakes", locale, guard, h.TakeIntake)
	app.Delete("/api/v1/care/medications/:id/intakes", locale, guard, h.UntakeIntake)
	app.Get("/api/v1/care/appointments", locale, guard, h.ListAppointments)
	app.Post("/api/v1/care/appointments", locale, guard, h.StoreAppointment)
	app.Get("/api/v1/care/appointments/:id", locale, guard, h.ShowAppointment)
	app.Put("/api/v1/care/appointments/:id", locale, guard, h.UpdateAppointment)
	app.Delete("/api/v1/care/appointments/:id", locale, guard, h.DestroyAppointment)
	app.Post("/api/v1/care/appointments/:id/cancel", locale, guard, h.CancelAppointment)
	app.Patch("/api/v1/care/appointments/:id/prep/:itemId", locale, guard, h.TogglePrepItem)
	app.Get("/api/v1/reminders", locale, guard, legacy.Index)
	return &env{db: db, app: app, iss: passport.NewIssuer(key, q, clock.Real{}, 365)}
}

func (e *env) user(t *testing.T, mobile string) (uint64, string) {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, mobile)
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

const folic = `{"title":"فولیک اسید","dose":"400","unit":"mcg","form":"tablet","times":["20:00","08:00","08:00"],"notes":"بعد از صبحانه"}`

func (e *env) create(t *testing.T, token, body string) uint64 {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/care/medications", token, "fa", body)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	return uint64(r.data()["id"].(float64))
}

func medPath(id uint64, suffix string) string {
	return fmt.Sprintf("/api/v1/care/medications/%d%s", id, suffix)
}

func TestMedication_CRUD(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000001")

	r := e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", folic)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, true, r.body["success"])
	d := r.data()
	id := uint64(d["id"].(float64))
	assert.Equal(t, "medication", d["type"])
	assert.Equal(t, "فولیک اسید", d["title"])
	assert.Equal(t, "۴۰۰ میکروگرم", d["subtitle"])
	assert.Equal(t, []any{"08:00", "20:00"}, d["times"], "sorted and de-duplicated")
	assert.Equal(t, []any{0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0}, d["weekdays"])
	assert.EqualValues(t, 1, d["amount"])
	assert.Equal(t, "ongoing", d["duration"])
	assert.Equal(t, true, d["notify"])
	assert.Equal(t, "2026-09-23", d["starts_on"], "starts today by default")
	assert.Nil(t, d["ends_on"])
	assert.Equal(t, true, d["is_active"])
	assert.Equal(t, "daily", d["recurrence"])
	assert.Equal(t, "08:00", d["recurrence_time"])
	assert.Equal(t, "بعد از صبحانه", d["notes"])

	var meta string
	require.NoError(t, e.db.QueryRow(`SELECT meta FROM reminders WHERE id = ?`, id).Scan(&meta))
	assert.JSONEq(t, `{"v":1,"dose":"400","unit":"mcg","form":"tablet","times":["08:00","20:00"],"weekdays":[0,1,2,3,4,5,6],"amount":1,"duration":"ongoing","notify":true}`, meta)

	r = e.do(t, http.MethodGet, medPath(id, ""), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "فولیک اسید", r.data()["title"])

	// The list switch: only is_active changes.
	r = e.do(t, http.MethodPut, medPath(id, ""), tok, "fa", `{"is_active":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, false, d["is_active"])
	assert.Equal(t, []any{"08:00", "20:00"}, d["times"])
	assert.Equal(t, "۴۰۰ میکروگرم", d["subtitle"])
	assert.Equal(t, "بعد از صبحانه", d["notes"])

	r = e.do(t, http.MethodPut, medPath(id, ""), tok, "en", `{"weekdays":[2,0,2],"dose":500,"unit":"mg","amount":2,"times":["21:30"]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, []any{0.0, 2.0}, d["weekdays"])
	assert.Equal(t, "weekly", d["recurrence"])
	assert.Equal(t, "21:30", d["recurrence_time"])
	assert.Equal(t, "500", d["dose"], "a numeric dose is kept as a string")
	assert.Equal(t, "500 mg", d["subtitle"])
	assert.EqualValues(t, 2, d["amount"])

	id2 := e.create(t, tok, `{"title":"آهن","form":"capsule","times":["13:00"]}`)
	r = e.do(t, http.MethodGet, "/api/v1/care/medications", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Len(t, r.body["data"], 2)
	r = e.do(t, http.MethodGet, "/api/v1/care/medications?active=1", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	list := r.body["data"].([]any)
	require.Len(t, list, 1)
	assert.EqualValues(t, id2, list[0].(map[string]any)["id"])
	assert.Nil(t, list[0].(map[string]any)["subtitle"], "no dose, no unit")

	r = e.do(t, http.MethodDelete, medPath(id, ""), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, true, r.body["success"])
	r = e.do(t, http.MethodGet, medPath(id, ""), tok, "en", "")
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, "Medication reminder not found", r.body["message"])
}

func TestMedication_ValidationFaEn(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000002")

	r := e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, "اطلاعات واردشده نامعتبر است", r.body["message"])
	errs := r.errors()
	assert.Contains(t, errs, "title")
	assert.Contains(t, errs, "form")
	assert.Contains(t, errs, "times")
	assert.Contains(t, errs["title"].([]any)[0], "نام دارو")

	r = e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "Validation failed", r.body["message"])
	assert.Equal(t, []any{"The medication name field is required."}, r.errors()["title"])

	cases := []struct {
		name, body, field string
	}{
		{"unknown form", `{"title":"x","form":"patch","times":["08:00"]}`, "form"},
		{"five times", `{"title":"x","form":"tablet","times":["01:00","02:00","03:00","04:00","05:00"]}`, "times"},
		{"no times", `{"title":"x","form":"tablet","times":[]}`, "times"},
		{"bad time", `{"title":"x","form":"tablet","times":["8:00"]}`, "times.0"},
		{"weekday out of range", `{"title":"x","form":"tablet","times":["08:00"],"weekdays":[7]}`, "weekdays.0"},
		{"empty weekdays", `{"title":"x","form":"tablet","times":["08:00"],"weekdays":[]}`, "weekdays"},
		{"amount zero", `{"title":"x","form":"tablet","times":["08:00"],"amount":0}`, "amount"},
		{"unknown duration", `{"title":"x","form":"tablet","times":["08:00"],"duration":"forever"}`, "duration"},
		{"until_date without ends_on", `{"title":"x","form":"tablet","times":["08:00"],"duration":"until_date"}`, "ends_on"},
		{"ends before start", `{"title":"x","form":"tablet","times":["08:00"],"duration":"until_date","starts_on":"2026-09-23","ends_on":"2026-09-01"}`, "ends_on"},
		{"bad start", `{"title":"x","form":"tablet","times":["08:00"],"starts_on":"23/09/2026"}`, "starts_on"},
	}
	for _, tc := range cases {
		r := e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "en", tc.body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, "%s: %s", tc.name, r.raw)
		assert.Contains(t, r.errors(), tc.field, "%s: %s", tc.name, r.raw)
	}

	r = e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", `{"title":"x","form":"tablet","times":["08:00"],"duration":"until_date"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, []any{"برای «تا تاریخ مشخص» تاریخ پایان الزامی است."}, r.errors()["ends_on"])

	// PUT validates the merged medication.
	id := e.create(t, tok, folic)
	r = e.do(t, http.MethodPut, medPath(id, ""), tok, "en", `{"times":[]}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "times")
	r = e.do(t, http.MethodPut, medPath(id, ""), tok, "en", `{"duration":"until_date","ends_on":"2026-10-01"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "2026-10-01", r.data()["ends_on"])
	r = e.do(t, http.MethodPut, medPath(id, ""), tok, "en", `{"duration":"ongoing"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["ends_on"])
}

func TestMedication_OwnershipAndUnauthenticated(t *testing.T) {
	e := setup(t)
	aID, a := e.user(t, "09120000003")
	_, b := e.user(t, "09120000004")
	id := e.create(t, a, folic)

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, medPath(id, ""), ""},
		{http.MethodPut, medPath(id, ""), `{"is_active":false}`},
		{http.MethodDelete, medPath(id, ""), ""},
		{http.MethodPost, medPath(id, "/intakes"), `{"date":"2026-09-23","slot":"08:00"}`},
		{http.MethodDelete, medPath(id, "/intakes"), `{"date":"2026-09-23","slot":"08:00"}`},
	} {
		r := e.do(t, tc.method, tc.path, b, "fa", tc.body)
		assert.Equal(t, http.StatusNotFound, r.status, "%s %s: %s", tc.method, tc.path, r.raw)
		assert.Equal(t, "یادآور دارو پیدا نشد", r.body["message"])
	}
	r := e.do(t, http.MethodGet, "/api/v1/care/medications", b, "fa", "")
	require.Equal(t, http.StatusOK, r.status)
	assert.Empty(t, r.body["data"])

	var active bool
	require.NoError(t, e.db.QueryRow(`SELECT is_active FROM reminders WHERE id = ?`, id).Scan(&active))
	assert.True(t, active, "B's PUT did not touch A's row")

	// A's non-medication reminder and a malformed id are misses too.
	res, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, recurrence, is_active, created_at, updated_at)
		VALUES (?, 'doctor', 'دکتر', 'none', 1, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, aID)
	require.NoError(t, err)
	doctorID, _ := res.LastInsertId()
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, medPath(uint64(doctorID), ""), a, "fa", "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, "/api/v1/care/medications/abc", a, "fa", "").status)

	r = e.do(t, http.MethodGet, "/api/v1/care/medications", "", "fa", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestIntake_Idempotent(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000005")
	id := e.create(t, tok, folic)
	body := `{"date":"2026-09-23","slot":"08:00"}`

	r := e.do(t, http.MethodPost, medPath(id, "/intakes"), tok, "fa", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.EqualValues(t, id, d["reminder_id"])
	assert.Equal(t, "2026-09-23", d["date"])
	assert.Equal(t, "08:00", d["slot"])
	assert.Equal(t, true, d["taken"])
	assert.Equal(t, "2026-09-23T06:30:00.000000Z", d["taken_at"])

	r = e.do(t, http.MethodPost, medPath(id, "/intakes"), tok, "fa", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminder_intakes WHERE reminder_id = ?`, id).Scan(&n))
	assert.Equal(t, 1, n)

	r = e.do(t, http.MethodDelete, medPath(id, "/intakes"), tok, "fa", body)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["taken"])
	assert.Nil(t, r.data()["taken_at"])
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminder_intakes WHERE reminder_id = ?`, id).Scan(&n))
	assert.Equal(t, 0, n)
	r = e.do(t, http.MethodDelete, medPath(id, "/intakes"), tok, "fa", body)
	assert.Equal(t, http.StatusOK, r.status, "untick is idempotent too")

	// The frontend unticks with the query string (no body).
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, medPath(id, "/intakes"), tok, "fa", body).status)
	r = e.do(t, http.MethodDelete, medPath(id, "/intakes?date=2026-09-23&slot=08:00"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["taken"])
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminder_intakes WHERE reminder_id = ?`, id).Scan(&n))
	assert.Equal(t, 0, n)

	// Deleting the medication cascades its intakes.
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPost, medPath(id, "/intakes"), tok, "fa", body).status)
	require.Equal(t, http.StatusOK, e.do(t, http.MethodDelete, medPath(id, ""), tok, "fa", "").status)
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminder_intakes`).Scan(&n))
	assert.Equal(t, 0, n)
}

func TestIntake_WeekdayAndWindow(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000006")
	// Saturdays and Mondays (0, 2), 2026-09-12 … 2026-09-21.
	id := e.create(t, tok, `{"title":"x","form":"tablet","times":["08:00"],"weekdays":[0,2],
		"starts_on":"2026-09-12","duration":"until_date","ends_on":"2026-09-21"}`)
	take := func(date, slot string) response {
		return e.do(t, http.MethodPost, medPath(id, "/intakes"), tok, "en", fmt.Sprintf(`{"date":%q,"slot":%q}`, date, slot))
	}

	assert.Equal(t, http.StatusOK, take("2026-09-12", "08:00").status, "Saturday, first day")
	assert.Equal(t, http.StatusOK, take("2026-09-14", "08:00").status, "Monday")
	assert.Equal(t, http.StatusOK, take("2026-09-19", "08:00").status, "Saturday")

	r := take("2026-09-16", "08:00")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, "Wednesday is not scheduled")
	assert.Equal(t, []any{"This medication is not scheduled on that day."}, r.errors()["date"])
	assert.Equal(t, http.StatusUnprocessableEntity, take("2026-09-05", "08:00").status, "Saturday before starts_on")
	assert.Equal(t, http.StatusOK, take("2026-09-21", "08:00").status, "Monday, the last day (ends_on inclusive)")
	r = take("2026-09-19", "09:00")
	require.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Equal(t, []any{"This time is not one of the medication's dose times."}, r.errors()["slot"])
	r = take("2026-09-24", "08:00")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, "future day")
	assert.Contains(t, r.errors(), "date")
	r = take("2026-9-19", "8:00")
	require.Equal(t, http.StatusUnprocessableEntity, r.status)
	assert.Contains(t, r.errors(), "date")
	assert.Contains(t, r.errors(), "slot")

	// After ends_on: move the window's end back.
	require.Equal(t, http.StatusOK, e.do(t, http.MethodPut, medPath(id, ""), tok, "en", `{"ends_on":"2026-09-13"}`).status)
	assert.Equal(t, http.StatusUnprocessableEntity, take("2026-09-14", "08:00").status, "Monday after ends_on")
}

func TestMedication_PregnancyEndUsesDueDate(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000007")
	body := `{"title":"فولیک اسید","form":"tablet","times":["08:00"],"duration":"pregnancy_end"}`

	r := e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", body)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Nil(t, r.data()["ends_on"], "no active pregnancy: open-ended")
	assert.Equal(t, "pregnancy_end", r.data()["duration"])

	_, err := e.db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, lmp_date, estimated_due_date, created_at, updated_at)
		VALUES (?, 1, '2026-06-01', '2027-03-08', '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, uid)
	require.NoError(t, err)
	r = e.do(t, http.MethodPost, "/api/v1/care/medications", tok, "fa", body)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "2027-03-08", r.data()["ends_on"])
}

func TestMedication_VisibleThroughLegacyReminders(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000008")
	e.create(t, tok, folic)

	r := e.do(t, http.MethodGet, "/api/v1/reminders", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	list := r.body["data"].([]any)
	require.Len(t, list, 1)
	row := list[0].(map[string]any)
	assert.Equal(t, "medication", row["type"])
	assert.Equal(t, "فولیک اسید", row["title"])
	assert.Equal(t, "۴۰۰ میکروگرم", row["subtitle"])
	assert.Equal(t, "daily", row["recurrence"])
	assert.Equal(t, "08:00:00", row["recurrence_time"])
	assert.Equal(t, true, row["is_active"])
}

func TestEnums_Localized(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000009")

	r := e.do(t, http.MethodGet, "/api/v1/care/enums", tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	for _, k := range []string{"forms", "units", "durations", "kinds", "topics", "remind_before"} {
		assert.NotEmpty(t, d[k], k)
	}
	assert.Equal(t, map[string]any{"value": "tablet", "label": "Tablet"}, d["forms"].([]any)[0])

	r = e.do(t, http.MethodGet, "/api/v1/care/enums", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, map[string]any{"value": "tablet", "label": "قرص"}, r.data()["forms"].([]any)[0])
	assert.Equal(t, map[string]any{"value": "1d", "label": "۱ روز قبل"}, r.data()["remind_before"].([]any)[2])
}
