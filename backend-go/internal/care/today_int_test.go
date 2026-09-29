package care_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// today reads GET /care/today and asserts the query budget (medications, intakes, appointments).
func (e *env) today(t *testing.T, token, lang, query string) response {
	t.Helper()
	e.queries.n.Store(0)
	r := e.do(t, http.MethodGet, "/api/v1/care/today"+query, token, lang, "")
	if r.status == http.StatusOK {
		assert.Equal(t, int64(3), e.queries.n.Load(), "query budget: medications, intakes, appointments")
	}
	return r
}

// phpJSON escapes non-ASCII like PHP's json_encode (\uXXXX, lowercase hex).
func phpJSON(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 {
			b.WriteRune(r)
			continue
		}
		fmt.Fprintf(&b, "\\u%04x", r)
	}
	return b.String()
}

// The clock is fixed at Wednesday 2026-09-23 10:00 Tehran (Saturday-based weekday 4).
func TestToday_Fixture(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000301")

	folic := e.create(t, tok, `{"title":"فولیک اسید","dose":"400","unit":"mcg","form":"tablet","times":["20:00","08:00"],"starts_on":"2026-09-01"}`)
	// Saturday and Monday only: not due on Wednesday.
	iron := e.create(t, tok, `{"title":"آهن","form":"capsule","times":["13:00"],"weekdays":[0,2],"starts_on":"2026-09-01"}`)
	// Starts on Friday.
	e.create(t, tok, `{"title":"ویتامین D","form":"drops","times":["09:00"],"starts_on":"2026-09-25"}`)
	// Ended yesterday.
	e.create(t, tok, `{"title":"امگا ۳","form":"capsule","times":["09:00"],"starts_on":"2026-09-01","duration":"until_date","ends_on":"2026-09-22"}`)
	// Paused.
	paused := e.create(t, tok, `{"title":"کلسیم","form":"tablet","times":["07:00"]}`)
	r := e.do(t, http.MethodPut, medPath(paused, ""), tok, "fa", `{"is_active":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	// Starts today, no dose/unit (no subtitle).
	vitc := e.create(t, tok, `{"title":"ویتامین C","form":"tablet","times":["08:00"]}`)

	r = e.do(t, http.MethodPost, medPath(folic, "/intakes"), tok, "fa", `{"date":"2026-09-23","slot":"08:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	// An intake moved to yesterday does not count today.
	r = e.do(t, http.MethodPost, medPath(folic, "/intakes"), tok, "fa", `{"date":"2026-09-23","slot":"20:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	_, err := e.db.Exec(`UPDATE reminder_intakes SET intake_date = '2026-09-22' WHERE reminder_id = ? AND slot = '20:00'`, folic)
	require.NoError(t, err)

	// Appointments: a cancelled one first, a past one, then the NT scan with its reminder switched off.
	cancelled := e.createAppt(t, tok, `{"kind":"phone","topic":"consult","scheduled_at":"2026-09-24 09:00:00"}`)
	r = e.do(t, http.MethodPost, apptPath(cancelled, "/cancel"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	e.createAppt(t, tok, `{"kind":"in_person","topic":"checkup","scheduled_at":"2026-09-23 09:30:00"}`)
	scan := e.createAppt(t, tok, nt)
	r = e.do(t, http.MethodPut, apptPath(scan, ""), tok, "fa", `{"is_active":false}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	e.createAppt(t, tok, `{"kind":"online","topic":"lab","scheduled_at":"2026-10-10 09:00:00"}`)

	r = e.today(t, tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	want := fmt.Sprintf(`{"success":true,"data":{"date":"2026-09-23","doses":[`+
		`{"reminder_id":%d,"title":"فولیک اسید ۴۰۰ میکروگرم","form":"tablet","slot":"08:00","taken":true},`+
		`{"reminder_id":%d,"title":"ویتامین C","form":"tablet","slot":"08:00","taken":false},`+
		`{"reminder_id":%d,"title":"فولیک اسید ۴۰۰ میکروگرم","form":"tablet","slot":"20:00","taken":false}],`+
		`"taken_count":1,"total":3,`+
		`"next_appointment":{"id":%d,"kind":"in_person","title":"سونوگرافی NT","with":"دکتر احمدی",`+
		`"scheduled_at":"2026-09-30 10:30:00","days_until":7,"location":"مطب","remind_before":"1d","is_active":false}}}`,
		folic, vitc, folic, scan)
	assert.JSONEq(t, want, r.raw)
	assert.Equal(t, phpJSON(want), r.raw, "byte for byte (non-ASCII escaped like PHP json_encode)")

	// ?date=: Monday brings the iron capsule back; the 20:00 folic intake belongs to Tuesday the 22nd.
	r = e.today(t, tok, "fa", "?date=2026-09-21")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d := r.data()
	assert.Equal(t, "2026-09-21", d["date"])
	doses := d["doses"].([]any)
	require.Len(t, doses, 4, r.raw)
	slots := []any{}
	for _, x := range doses {
		slots = append(slots, x.(map[string]any)["slot"])
	}
	assert.Equal(t, []any{"08:00", "09:00", "13:00", "20:00"}, slots, "folic, omega (still running), iron, folic")
	assert.EqualValues(t, iron, doses[2].(map[string]any)["reminder_id"])
	assert.Equal(t, "capsule", doses[2].(map[string]any)["form"])
	assert.EqualValues(t, 0, d["taken_count"])
	assert.EqualValues(t, scan, d["next_appointment"].(map[string]any)["id"], "next appointment is relative to now")

	r = e.today(t, tok, "fa", "?date=2026-09-22")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	d = r.data()
	assert.EqualValues(t, 1, d["taken_count"], "the 20:00 dose of the 22nd")
	assert.EqualValues(t, 3, d["total"], "folic ×2 + omega (last day)")

	// Friday: vitamin D has started (09:00).
	r = e.today(t, tok, "fa", "?date=2026-09-25")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 4, r.data()["total"])
}

func TestToday_EmptyAndCancelledOnly(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000302")

	r := e.today(t, tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, `{"success":true,"data":{"date":"2026-09-23","doses":[],"taken_count":0,"total":0,"next_appointment":null}}`, r.raw)

	id := e.createAppt(t, tok, nt)
	r = e.do(t, http.MethodPost, apptPath(id, "/cancel"), tok, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	r = e.today(t, tok, "en", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Nil(t, r.data()["next_appointment"], "cancelled appointments never count")
}

func TestToday_ValidationAndAuth(t *testing.T) {
	e := setup(t)
	_, tok := e.user(t, "09120000303")

	r := e.today(t, tok, "en", "?date=23-09-2026")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Equal(t, false, r.body["success"])
	assert.Equal(t, []any{"The date field must match the format Y-m-d."}, r.errors()["date"])

	r = e.today(t, tok, "fa", "?date=2026-09-31")
	require.Equal(t, http.StatusUnprocessableEntity, r.status, r.raw)
	assert.Contains(t, r.errors(), "date")

	r = e.today(t, "", "en", "")
	require.Equal(t, http.StatusUnauthorized, r.status, r.raw)
	assert.Equal(t, "unauthenticated", r.body["error_code"])
}

func TestToday_UserIsolation(t *testing.T) {
	e := setup(t)
	_, tokA := e.user(t, "09120000304")
	_, tokB := e.user(t, "09120000305")

	med := e.create(t, tokA, `{"title":"فولیک اسید","form":"tablet","times":["08:00"]}`)
	r := e.do(t, http.MethodPost, medPath(med, "/intakes"), tokA, "fa", `{"date":"2026-09-23","slot":"08:00"}`)
	require.Equal(t, http.StatusOK, r.status, r.raw)
	e.createAppt(t, tokA, nt)

	r = e.today(t, tokB, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.Equal(t, `{"success":true,"data":{"date":"2026-09-23","doses":[],"taken_count":0,"total":0,"next_appointment":null}}`, r.raw)

	r = e.today(t, tokA, "fa", "")
	require.Equal(t, http.StatusOK, r.status, r.raw)
	assert.EqualValues(t, 1, r.data()["taken_count"])
}
