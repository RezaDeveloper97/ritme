package notifications

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/profile/store"
)

// B-N1-09: reminder times / days-before and the settings-only pill category.
func TestSchedule_DefaultsParseAndStore(t *testing.T) {
	p := Defaults()
	m, ok := p.TimeOf(DailyLog)
	assert.True(t, ok)
	assert.Equal(t, 22*60, m)
	_, ok = p.TimeOf(Articles)
	assert.False(t, ok, "untimed category")
	assert.Equal(t, DefaultDaysBefore, p.DaysBeforePeriod())
	assert.True(t, Known(Pill))
	assert.False(t, p.Enabled(Pill), "pill is opt-in")
	for _, g := range Groups {
		assert.NotContains(t, g.Categories, Pill, "pill is not a notification-settings switch")
	}

	row := store.NotificationPreference{
		Schedule: db.NullRawJSON{Valid: true, V: []byte(
			`{"before_period":{"days_before":3,"time":"20:15"},"pill":{"time":"7:05"},"articles":{"time":"10:00"},"pms":{"time":"99:00"}}`)},
	}
	p = FromRow(row)
	bp, _ := p.TimeOf(BeforePeriod)
	assert.Equal(t, 20*60+15, bp)
	assert.Equal(t, 3, p.DaysBeforePeriod())
	pill, _ := p.TimeOf(Pill)
	assert.Equal(t, 7*60+5, pill)
	pms, _ := p.TimeOf(PMS)
	assert.Equal(t, 9*60, pms, "unreadable time falls back")
	assert.JSONEq(t, `{"before_period":{"days_before":3,"time":"20:15"},"pill":{"time":"07:05"}}`, string(p.ScheduleJSON()))

	p.Schedule.DaysBefore = 12
	assert.Equal(t, DefaultDaysBefore, p.DaysBeforePeriod(), "out of range → default")
}
