package care_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nt is an NT-scan appointment a week after `now` (Wednesday 2026-09-23 10:00 Tehran).
const nt = `{"kind":"in_person","with":"دکتر احمدی","specialty":"زنان و زایمان","topic":"ultrasound",
	"title":"سونوگرافی NT","scheduled_at":"2026-09-30 10:30:00","location":"مطب","remind_before":"1d",
	"add_to_calendar":true,"notes":"ناشتا","prep":[{"id":"tmp-1","text":"دفترچه بیمه","done":false},{"text":"جواب آزمایش قبلی"}]}`

func apptPath(id uint64, suffix string) string {
	return fmt.Sprintf("/api/v1/care/appointments/%d%s", id, suffix)
}

func (e *env) createAppt(t *testing.T, token, body string) uint64 {
	t.Helper()
	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", token, "fa", body)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	return uint64(r.data()["id"].(float64))
}

func ids(t *testing.T, r response) []uint64 {
	t.Helper()
	list, ok := r.body["data"].([]any)
	require.True(t, ok, r.raw)
	out := []uint64{}
	for _, x := range list {
		out = append(out, uint64(x.(map[string]any)["id"].(float64)))
	}
	return out
}

func TestAppointment_CRUD(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000101")

	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "fa", nt)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	assert.Equal(t, "نوبت ذخیره شد", r.body["message"])
	d := r.data()
	id := uint64(d["id"].(float64))
	assert.Equal(t, "appointment", d["type"])
	assert.Equal(t, "سونوگرافی NT", d["title"])
	assert.Equal(t, "دکتر احمدی · زنان و زایمان", d["subtitle"])
	assert.Equal(t, "ناشتا", d["notes"])
	assert.Equal(t, "in_person", d["kind"])
	assert.Equal(t, "دکتر احمدی", d["with"])
	assert.Equal(t, "زنان و زایمان", d["specialty"])
	assert.Equal(t, "ultrasound", d["topic"])
	assert.Equal(t, "مطب", d["location"])
	assert.Equal(t, "2026-09-30 10:30:00", d["scheduled_at"])
	assert.Equal(t, "1d", d["remind_before"])
	assert.Equal(t, "2026-09-29 10:30:00", d["remind_at"])
	assert.EqualValues(t, 7, d["days_until"])
	assert.Equal(t, true, d["add_to_calendar"])
	assert.Equal(t, "scheduled", d["status"])
	assert.Equal(t, true, d["is_active"])
	assert.Equal(t, []any{
		map[string]any{"id": "p1", "text": "دفترچه بیمه", "done": false},
		map[string]any{"id": "p2", "text": "جواب آزمایش قبلی", "done": false},
	}, d["prep"], "server ids; a client id is not kept on create")

	var meta, scheduled, recurrence string
	require.NoError(t, e.db.QueryRow(`SELECT meta, DATE_FORMAT(scheduled_at, '%Y-%m-%d %H:%i:%s'), recurrence FROM reminders WHERE id = ?`, id).
		Scan(&meta, &scheduled, &recurrence))
	assert.Equal(t, "2026-09-30 10:30:00", scheduled, "Tehran wall-clock in the column")
	assert.Equal(t, "none", recurrence)
	assert.JSONEq(t, `{"v":1,"kind":"in_person","with":"دکتر احمدی","specialty":"زنان و زایمان","topic":"ultrasound",
		"location":"مطب","remind_before":"1d","add_to_calendar":true,
		"prep":[{"id":"p1","text":"دفترچه بیمه","done":false},{"id":"p2","text":"جواب آزمایش قبلی","done":false}],
		"status":"scheduled"}`, meta)

	r = e.do(t, http.MethodGet, apptPath(id, ""), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "سونوگرافی NT", r.data()["title"])

	// The reminder switch: only is_active changes.
	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "fa", `{"is_active":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, false, d["is_active"])
	assert.Equal(t, "2026-09-30 10:30:00", d["scheduled_at"])
	assert.Equal(t, "سونوگرافی NT", d["title"])
	assert.Len(t, d["prep"], 2)

	// The frontend's prep toggle: the whole list through PUT; known ids stay, new items get new ids.
	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "fa",
		`{"prep":[{"id":"p2","text":"جواب آزمایش قبلی","done":true},{"id":"x9","text":"کارت ملی","done":false}]}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{
		map[string]any{"id": "p2", "text": "جواب آزمایش قبلی", "done": true},
		map[string]any{"id": "p3", "text": "کارت ملی", "done": false},
	}, r.data()["prep"])

	// Reschedule + remind 3h before; the untitled form falls back to the topic label.
	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "en",
		`{"scheduled_at":"2026-09-24 09:00","remind_before":"3h","title":null,"topic":"lab","with":"","specialty":null}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.Equal(t, "2026-09-24 09:00:00", d["scheduled_at"])
	assert.Equal(t, "2026-09-24 06:00:00", d["remind_at"])
	assert.EqualValues(t, 1, d["days_until"])
	assert.Equal(t, "Lab test", d["title"])
	assert.Nil(t, d["with"])
	assert.Nil(t, d["subtitle"])
	assert.Equal(t, "Appointment updated", r.body["message"])

	r = e.do(t, http.MethodDelete, apptPath(id, ""), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "نوبت حذف شد", r.body["message"])
	r = e.do(t, http.MethodGet, apptPath(id, ""), tok, "en", "")
	assert.Equal(t, http.StatusNotFound, r.status)
	assert.Equal(t, "Appointment not found", r.body["message"])
}

func TestAppointment_TitleFallsBackToTopic(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000102")
	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "fa",
		`{"kind":"phone","topic":"consult","title":"  ","scheduled_at":"2026-10-01 18:00:00"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "مشاوره", d["title"])
	assert.Equal(t, "1d", d["remind_before"], "default")
	assert.Equal(t, false, d["add_to_calendar"], "default")
	assert.Equal(t, []any{}, d["prep"])
	assert.Nil(t, d["subtitle"])
	assert.Nil(t, d["location"])
}

func TestAppointment_ValidationFaEn(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000103")

	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "fa", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, "اطلاعات واردشده نامعتبر است", r.body["message"])
	for _, f := range []string{"kind", "topic", "scheduled_at"} {
		assert.Contains(t, r.errors(), f)
	}
	assert.Contains(t, r.errors()["scheduled_at"].([]any)[0], "تاریخ و ساعت نوبت")

	r = e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, "Validation failed", r.body["message"])
	assert.Equal(t, []any{"The appointment date and time field is required."}, r.errors()["scheduled_at"])

	base := `"kind":"in_person","topic":"checkup","scheduled_at":"2026-10-01 10:00:00"`
	cases := []struct{ name, body, field string }{
		{"unknown kind", `{"kind":"home","topic":"checkup","scheduled_at":"2026-10-01 10:00:00"}`, "kind"},
		{"unknown topic", `{"kind":"online","topic":"surgery","scheduled_at":"2026-10-01 10:00:00"}`, "topic"},
		{"bad datetime", `{"kind":"online","topic":"lab","scheduled_at":"2026/10/01 10:00"}`, "scheduled_at"},
		{"date only", `{"kind":"online","topic":"lab","scheduled_at":"2026-10-01"}`, "scheduled_at"},
		{"unknown remind_before", `{` + base + `,"remind_before":"1w"}`, "remind_before"},
		{"prep not a list", `{` + base + `,"prep":"x"}`, "prep"},
		{"prep item without text", `{` + base + `,"prep":[{"done":true}]}`, "prep.0.text"},
		{"prep item blank", `{` + base + `,"prep":[{"text":"  "}]}`, "prep.0.text"},
		{"prep item string", `{` + base + `,"prep":["x"]}`, "prep.0"},
		{"bad calendar flag", `{` + base + `,"add_to_calendar":"maybe"}`, "add_to_calendar"},
	}
	for _, tc := range cases {
		r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "en", tc.body)
		require.Equal(t, http.StatusUnprocessableEntity, r.status, "%s: %s", tc.name, r.raw)
		assert.Contains(t, r.errors(), tc.field, "%s: %s", tc.name, r.raw)
	}

	id := e.createAppt(t, tok, nt)
	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "en", `{"scheduled_at":null}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "scheduled_at")

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=soon", tok, "en", "")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "scope")

	r = e.do(t, http.MethodPatch, apptPath(id, "/prep/p1"), tok, "en", `{}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "done")
}

func TestAppointment_ScopeAndCancel(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000104")
	body := func(at string) string {
		return fmt.Sprintf(`{"kind":"in_person","topic":"checkup","scheduled_at":%q}`, at)
	}
	later := e.createAppt(t, tok, body("2026-10-10 09:00:00"))
	soon := e.createAppt(t, tok, body("2026-09-25 09:00:00"))
	earlierToday := e.createAppt(t, tok, body("2026-09-23 09:30:00"))
	lastWeek := e.createAppt(t, tok, body("2026-09-16 09:00:00"))
	toCancel := e.createAppt(t, tok, body("2026-09-28 09:00:00"))

	r := e.do(t, http.MethodPost, apptPath(toCancel, "/cancel"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "نوبت لغو شد", r.body["message"])
	assert.Equal(t, "cancelled", r.data()["status"])
	assert.Equal(t, false, r.data()["is_active"])
	var active bool
	require.NoError(t, e.db.QueryRow(`SELECT is_active FROM reminders WHERE id = ?`, toCancel).Scan(&active))
	assert.False(t, active, "row kept, reminder off")
	r = e.do(t, http.MethodPost, apptPath(toCancel, "/cancel"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, "cancel is idempotent")

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []uint64{soon, later}, ids(t, r), "default upcoming: soonest first, cancelled and past excluded")
	first := r.body["data"].([]any)[0].(map[string]any)
	assert.EqualValues(t, 2, first["days_until"])

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=upcoming", tok, "fa", "")
	assert.Equal(t, []uint64{soon, later}, ids(t, r))

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=past", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []uint64{toCancel, earlierToday, lastWeek}, ids(t, r), "latest first, cancelled included")
	var todayDays, lastWeekDays any
	for _, x := range r.body["data"].([]any) {
		row := x.(map[string]any)
		switch uint64(row["id"].(float64)) {
		case earlierToday:
			todayDays = row["days_until"]
		case lastWeek:
			lastWeekDays = row["days_until"]
		}
	}
	assert.EqualValues(t, 0, todayDays)
	assert.EqualValues(t, -7, lastWeekDays)

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=all", tok, "fa", "")
	assert.Equal(t, []uint64{lastWeek, earlierToday, soon, toCancel, later}, ids(t, r))

	// A PUT on a cancelled appointment keeps it cancelled.
	r = e.do(t, http.MethodPut, apptPath(toCancel, ""), tok, "fa", `{"status":"scheduled","location":"بیمارستان"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "cancelled", r.data()["status"])
	assert.Equal(t, "بیمارستان", r.data()["location"])
}

func TestAppointment_PrepToggle(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000105")
	id := e.createAppt(t, tok, nt)

	r := e.do(t, http.MethodPatch, apptPath(id, "/prep/p2"), tok, "fa", `{"done":true}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, []any{
		map[string]any{"id": "p1", "text": "دفترچه بیمه", "done": false},
		map[string]any{"id": "p2", "text": "جواب آزمایش قبلی", "done": true},
	}, r.data()["prep"])
	assert.Equal(t, "سونوگرافی NT", r.data()["title"], "nothing else changes")

	r = e.do(t, http.MethodPatch, apptPath(id, "/prep/p2"), tok, "fa", `{"done":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, false, r.data()["prep"].([]any)[1].(map[string]any)["done"])

	r = e.do(t, http.MethodPatch, apptPath(id, "/prep/p9"), tok, "fa", `{"done":true}`)
	require.Equal(t, http.StatusNotFound, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, "این مورد در فهرست آمادگی پیدا نشد", r.body["message"])
	r = e.do(t, http.MethodPatch, apptPath(id, "/prep/p9"), tok, "en", `{"done":true}`)
	assert.Equal(t, "Checklist item not found", r.body["message"])
}

func TestAppointment_OwnershipAndUnauthenticated(t *testing.T) {
	e := setup(t)
	_, a := e.user(t, "09120000106")
	_, b := e.user(t, "09120000107")
	id := e.createAppt(t, a, nt)

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, apptPath(id, ""), ""},
		{http.MethodPut, apptPath(id, ""), `{"is_active":false}`},
		{http.MethodDelete, apptPath(id, ""), ""},
		{http.MethodPost, apptPath(id, "/cancel"), ""},
		{http.MethodPatch, apptPath(id, "/prep/p1"), `{"done":true}`},
	} {
		r := e.do(t, tc.method, tc.path, b, "fa", tc.body)
		assert.Equal(t, http.StatusNotFound, r.status, "%s %s: %s", tc.method, tc.path, r.raw)
		assert.Equal(t, "نوبت پیدا نشد", r.body["message"])
	}
	r := e.do(t, http.MethodGet, "/api/v1/care/appointments?scope=all", b, "fa", "")
	require.Equal(t, http.StatusOK, r.status)
	assert.Empty(t, r.body["data"])

	var active bool
	var meta string
	require.NoError(t, e.db.QueryRow(`SELECT is_active, meta FROM reminders WHERE id = ?`, id).Scan(&active, &meta))
	assert.True(t, active, "B's PUT/cancel did not touch A's row")
	assert.Contains(t, meta, `"status":"scheduled"`)
	assert.Contains(t, meta, `"done":false`)

	// A's medication is not an appointment, a malformed id is a miss too.
	medID := e.create(t, a, folic)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, apptPath(medID, ""), a, "fa", "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, "/api/v1/care/appointments/abc", a, "fa", "").status)
	assert.Equal(t, http.StatusNotFound, e.do(t, http.MethodGet, medPath(id, ""), a, "fa", "").status,
		"an appointment is not a medication")

	r = e.do(t, http.MethodGet, "/api/v1/care/appointments", "", "fa", "")
	assert.Equal(t, http.StatusUnauthorized, r.status)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestAppointment_VisibleThroughLegacyReminders(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000108")
	e.createAppt(t, tok, nt)

	r := e.do(t, http.MethodGet, "/api/v1/reminders", tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	list := r.body["data"].([]any)
	require.Len(t, list, 1)
	row := list[0].(map[string]any)
	assert.Equal(t, "appointment", row["type"])
	assert.Equal(t, "سونوگرافی NT", row["title"])
	assert.Equal(t, "دکتر احمدی · زنان و زایمان", row["subtitle"])
	assert.Equal(t, "none", row["recurrence"])
	assert.Equal(t, "2026-09-30T07:00:00.000000Z", row["scheduled_at"], "Tehran 10:30 as Eloquent UTC")
	assert.Equal(t, true, row["is_active"])
}

// Pregnancy v2 visit link (T-M7-05): care_item_key / stage / result_note are optional, kept on PUT
// when absent, validated, and absent from the stored meta when unset (v1 rows unchanged).
func TestAppointment_VisitStage(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000131")

	r := e.do(t, http.MethodPost, "/api/v1/care/appointments", tok, "en",
		`{"kind":"phone","topic":"consult","scheduled_at":"2026-10-01 18:00:00"}`)
	require.Equal(t, http.StatusCreated, r.status, r.raw)
	d := r.data()
	id := uint64(d["id"].(float64))
	assert.Nil(t, d["care_item_key"])
	assert.Nil(t, d["stage"])
	assert.Nil(t, d["result_note"])
	var meta string
	require.NoError(t, e.db.QueryRow(`SELECT meta FROM reminders WHERE id = ?`, id).Scan(&meta))
	assert.NotContains(t, meta, "stage")

	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "en", `{"care_item_key":"nt_scan","stage":"booked"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "nt_scan", r.data()["care_item_key"])
	assert.Equal(t, "booked", r.data()["stage"])

	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "en", `{"stage":"result","result_note":"All normal"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, "nt_scan", r.data()["care_item_key"], "kept when absent")
	assert.Equal(t, "result", r.data()["stage"])
	assert.Equal(t, "All normal", r.data()["result_note"])

	r = e.do(t, http.MethodPut, apptPath(id, ""), tok, "en", `{"stage":"maybe","care_item_key":"a b"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	errs, _ := r.body["errors"].(map[string]any)
	assert.Contains(t, errs, "stage")
	assert.Contains(t, errs, "care_item_key")
}
