package care

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	rootdb "github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func med(id uint64, title, subtitle, meta string, active bool, starts, ends string) store.Reminder {
	r := store.Reminder{ID: id, Type: TypeMedication, Title: title, IsActive: active,
		Meta: rootdb.NullRawJSON{V: []byte(meta), Valid: true}}
	if subtitle != "" {
		r.Subtitle = sql.NullString{String: subtitle, Valid: true}
	}
	if starts != "" {
		r.StartsOn = civildate.NullDate{Date: civildate.MustParse(starts), Valid: true}
	}
	if ends != "" {
		r.EndsOn = civildate.NullDate{Date: civildate.MustParse(ends), Valid: true}
	}
	return r
}

func TestTodayDoses(t *testing.T) {
	wed := civildate.MustParse("2026-09-23") // Saturday-based weekday 4
	meds := []store.Reminder{
		med(1, "فولیک اسید", "۴۰۰ میکروگرم", `{"v":1,"form":"tablet","times":["08:00","20:00"],"weekdays":[0,1,2,3,4,5,6]}`, true, "2026-09-01", ""),
		med(2, "آهن", "", `{"v":1,"form":"capsule","times":["13:00"],"weekdays":[0,2]}`, true, "", ""),
		med(3, "کلسیم", "", `{"v":1,"form":"tablet","times":["07:00"],"weekdays":[4]}`, false, "", ""),
		med(4, "امگا", "", `{"v":1,"form":"capsule","times":["08:00"],"weekdays":[4]}`, true, "", "2026-09-22"),
		med(5, "ویتامین", "", `{"v":1,"form":"drops","times":["08:00"],"weekdays":[4]}`, true, "2026-09-24", ""),
		med(0, "زینک", "", `{"v":1,"form":"tablet","times":["08:00"],"weekdays":[4]}`, true, "", ""),
	}
	got := TodayDoses(wed, meds, []store.ListIntakesOnDateRow{{ReminderID: 1, Slot: "20:00"}, {ReminderID: 2, Slot: "13:00"}}, "fa")
	assert.Equal(t, []Dose{
		{ReminderID: 0, Title: "زینک", Form: "tablet", Slot: "08:00"},
		{ReminderID: 1, Title: "فولیک اسید ۴۰۰ میکروگرم", Form: "tablet", Slot: "08:00"},
		{ReminderID: 1, Title: "فولیک اسید ۴۰۰ میکروگرم", Form: "tablet", Slot: "20:00", Taken: true},
	}, got, "weekday, window and is_active filtered; sorted by slot then id")

	assert.Equal(t, []Dose{}, TodayDoses(wed, nil, nil, "fa"))
}

func TestNextAppointment(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	appt := func(id uint64, at time.Time, meta string, active bool) store.Reminder {
		return store.Reminder{ID: id, Type: TypeAppointment, IsActive: active, ScheduledAt: sql.NullTime{Time: at, Valid: true},
			Meta: rootdb.NullRawJSON{V: []byte(meta), Valid: true}}
	}
	rows := []store.Reminder{
		appt(1, now.Add(-time.Hour), `{"v":1,"status":"scheduled"}`, true),
		appt(2, now.Add(time.Hour), `{"v":1,"status":"cancelled"}`, false),
		appt(3, now.Add(48*time.Hour), `{"v":1,"status":"scheduled"}`, false),
		appt(4, now.Add(72*time.Hour), `{"v":1,"status":"scheduled"}`, true),
	}
	next := NextAppointment(rows, now)
	require.NotNil(t, next)
	assert.EqualValues(t, 3, next.Row.ID, "reminder off still counts; cancelled and past skipped")
	assert.Nil(t, NextAppointment(rows[:2], now))

	j := TodayJSON(civildate.MustParse("2026-09-23"), nil, next, now)
	v, _ := j.Get("next_appointment")
	b, err := v.(interface{ MarshalJSON() ([]byte, error) }).MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":3,"kind":"in_person","title":"","with":null,"scheduled_at":"2026-09-25 10:00:00",
		"days_until":2,"location":null,"remind_before":"1d"}`, string(b))
}
