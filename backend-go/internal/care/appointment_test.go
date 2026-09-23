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

func TestAssignPrepIDs(t *testing.T) {
	known := []PrepItem{{ID: "p1"}, {ID: "p4"}}
	got := AssignPrepIDs([]PrepItem{
		{ID: "p4", Text: "a"}, {ID: "tmp", Text: "b"}, {Text: "c"}, {ID: "p4", Text: "dup"}, {ID: "p1", Text: "d", Done: true},
	}, known)
	assert.Equal(t, []PrepItem{
		{ID: "p4", Text: "a"}, {ID: "p5", Text: "b"}, {ID: "p6", Text: "c"}, {ID: "p7", Text: "dup"}, {ID: "p1", Text: "d", Done: true},
	}, got)

	got = AssignPrepIDs([]PrepItem{{ID: "p1", Text: "a"}, {Text: "b"}}, nil)
	assert.Equal(t, []PrepItem{{ID: "p1", Text: "a"}, {ID: "p2", Text: "b"}}, got, "no stored list: every id is new")
}

func TestRemindOffset(t *testing.T) {
	assert.Equal(t, time.Hour, RemindOffset("1h"))
	assert.Equal(t, 3*time.Hour, RemindOffset("3h"))
	assert.Equal(t, 24*time.Hour, RemindOffset("1d"))
	assert.Equal(t, 48*time.Hour, RemindOffset("2d"))
}

func TestAppointmentMeta_Subtitle(t *testing.T) {
	assert.Equal(t, "دکتر احمدی · زنان", AppointmentMeta{With: ptr("دکتر احمدی"), Specialty: ptr("زنان")}.Subtitle().String)
	assert.Equal(t, "زنان", AppointmentMeta{With: ptr(""), Specialty: ptr("زنان")}.Subtitle().String)
	assert.False(t, AppointmentMeta{}.Subtitle().Valid)
}

func TestParseAppointment_LegacyRowDefaults(t *testing.T) {
	at := time.Date(2026, 9, 25, 23, 30, 0, 0, civildate.Tehran)
	a := ParseAppointment(store.Reminder{ID: 3, Type: TypeAppointment, Title: "دکتر", ScheduledAt: sql.NullTime{Time: at, Valid: true}})
	assert.Equal(t, AppointmentMeta{V: 1, Kind: "in_person", Topic: "other", RemindBefore: "1d", Prep: []PrepItem{}, Status: "scheduled"}, a.Meta)

	now := time.Date(2026, 9, 23, 23, 50, 0, 0, civildate.Tehran)
	j := a.JSON(now)
	v, _ := j.Get("days_until")
	assert.Equal(t, 2, v, "civil days, not 24h periods")
	v, _ = j.Get("remind_at")
	assert.Equal(t, "2026-09-24 23:30:00", v)
	assert.True(t, a.Upcoming(now))
	assert.False(t, a.Upcoming(at.Add(time.Second)))

	a = ParseAppointment(store.Reminder{Meta: rootdb.NullRawJSON{V: []byte(`{"v":1,"kind":"x","status":"cancelled","remind_before":"2d"}`), Valid: true}})
	assert.Equal(t, "in_person", a.Meta.Kind)
	assert.Equal(t, "cancelled", a.Meta.Status)
	assert.Equal(t, "2d", a.Meta.RemindBefore)
	assert.False(t, a.Upcoming(now), "no scheduled_at, cancelled")
	v, _ = a.JSON(now).Get("days_until")
	assert.Nil(t, v)
}

func TestParseScheduledAt(t *testing.T) {
	got, err := parseScheduledAt("2026-09-30 10:30")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 30, 10, 30, 0, 0, civildate.Tehran), got)
	_, err = parseScheduledAt("nope")
	assert.Error(t, err)
}
