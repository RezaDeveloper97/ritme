package reminders

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func d(s string) civildate.Date { return civildate.MustParse(s) }

func codes(occ []Occurrence) []notifications.Category {
	out := make([]notifications.Category, len(occ))
	for i, o := range occ {
		out[i] = o.Category
	}
	return out
}

func TestCycleFrom_ProjectsToUpcomingDates(t *testing.T) {
	// 29-day cycle anchored 2026-09-01: next period 09-30, PMS 09-27, ovulation 09-16 → window 09-11.
	c := CycleFrom(d("2026-09-01"), 29, 5, d("2026-09-05"))
	assert.Equal(t, d("2026-09-30"), c.NextPeriodStart)
	assert.Equal(t, d("2026-09-27"), c.PMSStart)
	assert.Equal(t, d("2026-09-11"), c.FertileWindowStart)

	// After this cycle's window, the next window is planned.
	c = CycleFrom(d("2026-09-01"), 29, 5, d("2026-09-20"))
	assert.Equal(t, d("2026-10-10"), c.FertileWindowStart)

	assert.Equal(t, Cycle{}, CycleFrom(civildate.Date{}, 29, 5, d("2026-09-20")))
}

func TestPMSCycleDay(t *testing.T) {
	assert.Equal(t, 27, PMSCycleDay(29))
	assert.Equal(t, 26, PMSCycleDay(28))
	assert.Equal(t, 0, PMSCycleDay(0))
}

func TestPlan_HonoursSwitchesTimesAndDaysBefore(t *testing.T) {
	c := CycleFrom(d("2026-09-01"), 29, 5, d("2026-09-05"))
	p := notifications.Defaults()

	// Default: before-period fires 2 days before (09-28) at 09:00; daily log every day; pill is opt-in.
	occ := Plan(p, c, d("2026-09-28"))
	require.Equal(t, []notifications.Category{notifications.BeforePeriod, notifications.DailyLog, notifications.Pill}, codes(occ))
	assert.True(t, occ[0].Decision.Send)
	assert.Equal(t, 9, occ[0].At.In(civildate.Tehran).Hour())
	assert.Equal(t, notifications.ReasonCategoryOff, occ[2].Decision.Reason)

	// Custom: 3 days before at 20:15, pill on at 21:30.
	p.Schedule.DaysBefore = 3
	p.Schedule.Times[notifications.BeforePeriod] = 20*60 + 15
	p.Schedule.Times[notifications.Pill] = 21*60 + 30
	p.Categories[notifications.Pill] = true
	occ = Plan(p, c, d("2026-09-27")) // also the PMS start
	require.Equal(t, []notifications.Category{
		notifications.BeforePeriod, notifications.PMS, notifications.DailyLog, notifications.Pill,
	}, codes(occ))
	at := occ[0].At.In(civildate.Tehran)
	assert.Equal(t, [2]int{20, 15}, [2]int{at.Hour(), at.Minute()})
	assert.True(t, occ[3].Decision.Send)
	assert.Equal(t, []notifications.Category{notifications.DailyLog, notifications.Pill}, codes(Plan(p, c, d("2026-09-28"))))
}

func TestPlan_QuietHoursDeferAndUnknownCycle(t *testing.T) {
	p := notifications.Defaults() // quiet 23:00 → 08:00
	p.Schedule.Times[notifications.DailyLog] = 23*60 + 30
	occ := Plan(p, Cycle{}, d("2026-09-10"))
	// No cycle → only the daily ones.
	require.Equal(t, []notifications.Category{notifications.DailyLog, notifications.Pill}, codes(occ))
	assert.False(t, occ[0].Decision.Send)
	assert.Equal(t, notifications.ReasonQuietHours, occ[0].Decision.Reason)
	assert.Equal(t, time.Date(2026, 9, 11, 8, 0, 0, 0, civildate.Tehran), occ[0].Decision.DeferUntil)
}
