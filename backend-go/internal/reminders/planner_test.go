package reminders

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
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

// cycleOn is the planner's cycle from a v6 engine state: one real period starting at anchor, the user's own
// (manual) lengths, today as given.
func cycleOn(anchor string, cycleLen, periodLen int, today string, mutate ...func(*engine.Input)) Cycle {
	start := engine.DayOf(d(anchor))
	in := engine.Input{
		Periods:  []engine.Period{{Start: start, End: start + engine.Day(periodLen-1)}},
		Declared: engine.Declared{CycleLength: engine.Some(cycleLen), PeriodLength: engine.Some(periodLen), Manual: true},
	}
	for _, m := range mutate {
		m(&in)
	}
	return CycleFrom(engine.Prepare(in, engine.DefaultParams()), d(today))
}

func TestCycleFrom_ProjectsToUpcomingDates(t *testing.T) {
	// 29-day cycle anchored 2026-09-01: next period 09-30, PMS 09-27 (next − 3), ovulation 09-16 → window 09-11.
	c := cycleOn("2026-09-01", 29, 5, "2026-09-05")
	assert.Equal(t, d("2026-09-30"), c.NextPeriodStart)
	assert.Equal(t, d("2026-09-27"), c.PMSStart)
	assert.Equal(t, d("2026-09-11"), c.FertileWindowStart)
	assert.Equal(t, 29, c.CycleLength)

	// After this cycle's window, the next window is planned.
	c = cycleOn("2026-09-01", 29, 5, "2026-09-20")
	assert.Equal(t, d("2026-10-10"), c.FertileWindowStart)

	// 15 days late: the engine's display chain starts after today (E-04) or is empty past the cap, never the past.
	c = cycleOn("2026-09-01", 29, 5, "2026-10-15")
	assert.True(t, c.NextPeriodStart.IsZero() || c.NextPeriodStart.After(d("2026-10-15")))

	assert.Equal(t, Cycle{}, CycleFrom(engine.Prepare(engine.Input{}, engine.DefaultParams()), d("2026-09-20")))
}

func TestCycleFrom_HormonalHasNoFertileWindow(t *testing.T) {
	c := cycleOn("2026-09-01", 29, 5, "2026-09-05", func(in *engine.Input) { in.Hormonal = true })
	assert.Equal(t, d("2026-09-30"), c.NextPeriodStart)
	assert.True(t, c.FertileWindowStart.IsZero(), "E-13: no fertile dates under hormonal contraception")
}

func TestPMSCycleDay(t *testing.T) {
	assert.Equal(t, 27, PMSCycleDay(29))
	assert.Equal(t, 26, PMSCycleDay(28))
	assert.Equal(t, 0, PMSCycleDay(0))
}

func TestPlan_HonoursSwitchesTimesAndDaysBefore(t *testing.T) {
	c := cycleOn("2026-09-01", 29, 5, "2026-09-05")
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

// T-M12-21 — Jalali and quiet-hours edge tables for the planner. The planner works on Gregorian days; the Jalali
// labels are comments (1403 is leap: 30 Esfand 1403 = 2025-03-20; 1404 is not: 29 Esfand 1404 = 2026-03-20).

// Before-period (2 days before by default) and PMS (next − 3) fire on the right Gregorian day when the next period
// lands on Nowruz or on the leap day, and on no neighbouring day.
func TestPlan_CycleRemindersAcrossEsfandAndNowruz(t *testing.T) {
	cases := []struct {
		name              string
		anchor, today     string
		next, before, pms string
		quiet             []string // days on which neither fires (negative rows)
	}{
		{
			name: "next = 1 Farvardin 1405 (common Esfand)", anchor: "2026-02-21", today: "2026-02-25",
			next: "2026-03-21", before: "2026-03-19" /* 28 Esfand */, pms: "2026-03-18", /* 27 Esfand */
			quiet: []string{"2026-03-17", "2026-03-20", "2026-03-21"},
		},
		{
			name: "next = 1 Farvardin 1404 (leap Esfand)", anchor: "2025-02-21", today: "2025-02-25",
			next: "2025-03-21", before: "2025-03-19" /* 29 Esfand */, pms: "2025-03-18", /* 28 Esfand */
			quiet: []string{"2025-03-17", "2025-03-20", "2025-03-21"},
		},
		{
			name: "next = 30 Esfand 1403 (the leap day)", anchor: "2025-02-20", today: "2025-02-25",
			next: "2025-03-20", before: "2025-03-18", pms: "2025-03-17",
			quiet: []string{"2025-03-16", "2025-03-19", "2025-03-20"},
		},
		{
			name: "next = 2 Farvardin 1405: reminders on 29 Esfand and 1 Farvardin", anchor: "2026-02-22", today: "2026-02-25",
			next: "2026-03-22", before: "2026-03-20", pms: "2026-03-19",
			quiet: []string{"2026-03-18", "2026-03-21", "2026-03-22"},
		},
	}
	p := notifications.Defaults()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cy := cycleOn(c.anchor, 28, 5, c.today)
			require.Equal(t, d(c.next), cy.NextPeriodStart)
			assert.Equal(t, d(c.pms), cy.PMSStart)
			assert.Contains(t, codes(Plan(p, cy, d(c.before))), notifications.BeforePeriod)
			assert.Contains(t, codes(Plan(p, cy, d(c.pms))), notifications.PMS)
			for _, q := range c.quiet {
				got := codes(Plan(p, cy, d(q)))
				assert.NotContains(t, got, notifications.BeforePeriod, q)
				assert.NotContains(t, got, notifications.PMS, q)
			}
		})
	}
}

// A reminder exactly at the quiet-hours start is deferred to the window's end on the next day (across the
// 29 Esfand → Nowruz midnight); one exactly at the end goes out; the minute before the end waits until it.
func TestPlan_QuietHoursBoundariesAcrossNowruz(t *testing.T) {
	esfand29 := d("2026-03-20")
	cases := []struct {
		name       string
		minute     int
		send       bool
		deferUntil time.Time
	}{
		{"22:59 sends", 22*60 + 59, true, time.Time{}},
		{"23:00 (quiet start) defers to Nowruz 08:00", 23 * 60, false, time.Date(2026, 3, 21, 8, 0, 0, 0, civildate.Tehran)},
		{"23:59 defers to Nowruz 08:00", 23*60 + 59, false, time.Date(2026, 3, 21, 8, 0, 0, 0, civildate.Tehran)},
		{"00:00 defers to the same day 08:00", 0, false, time.Date(2026, 3, 20, 8, 0, 0, 0, civildate.Tehran)},
		{"07:59 defers to the same day 08:00", 7*60 + 59, false, time.Date(2026, 3, 20, 8, 0, 0, 0, civildate.Tehran)},
		{"08:00 (quiet end) sends", 8 * 60, true, time.Time{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := notifications.Defaults()
			p.Schedule.Times[notifications.DailyLog] = c.minute
			occ := Plan(p, Cycle{}, esfand29)
			require.Equal(t, notifications.DailyLog, occ[0].Category)
			at := occ[0].At.In(civildate.Tehran)
			assert.Equal(t, esfand29, civildate.FromTime(at), "the occurrence stays on its own day")
			assert.Equal(t, c.minute, at.Hour()*60+at.Minute())
			assert.Equal(t, c.send, occ[0].Decision.Send)
			assert.Equal(t, c.deferUntil, occ[0].Decision.DeferUntil)
		})
	}
}

// Plan has no zone input: the times are on the Tehran clock (README §5, X-07 scope). Across Berlin's DST changes
// the daily reminder therefore stays 24 h apart and at 09:00 Tehran; following the user's zone instead (M-08)
// is an open product question.
func TestPlan_DailyTimeAcrossBerlinDST(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	p := notifications.Defaults()
	p.Schedule.Times[notifications.DailyLog] = 9 * 60
	for _, days := range [][]string{
		{"2026-03-28", "2026-03-29", "2026-03-30"},
		{"2026-10-24", "2026-10-25", "2026-10-26"},
	} {
		var prev time.Time
		for i, day := range days {
			at := Plan(p, Cycle{}, d(day))[0].At
			assert.Equal(t, 9, at.In(civildate.Tehran).Hour(), day)
			assert.Equal(t, d(day), civildate.InTehran(at), day)
			assert.Equal(t, d(day), civildate.FromTime(at.In(berlin)), "%s: the Berlin day is the same day", day)
			if i > 0 {
				assert.Equal(t, 24*time.Hour, at.Sub(prev), day)
			}
			prev = at
		}
	}
}
