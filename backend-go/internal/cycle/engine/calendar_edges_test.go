package engine_test

// T-M12-21 (findings T-08): Jalali and user-timezone edge tables for the cycle engine.
//
// The engine works on Gregorian days only (SPEC §2.3: no arithmetic on a Jalali string, no fixed month length), so
// the Jalali edges are pinned here as Gregorian inputs whose Jalali labels are checked against the backend's
// jalaali converter (checkups/engine, Borkowski break table: 1403 is leap, 1404 is not). The user-local day is
// civildate.LocalToday in the request zone (X-07); the rows below go through it, then through the engine.
//
// This is an external test package because checkups/engine imports cycle/engine.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	checkups "github.com/ritme/backend-go/internal/checkups/engine"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// jday is the engine Day of a Jalali date, checked against the Gregorian date the row was written with, so a row
// can never silently drift onto another day.
func jday(t *testing.T, jy, jm, jd int, gregorian string) engine.Day {
	t.Helper()
	g := checkups.FromJalali(jy, jm, jd)
	require.Equal(t, gregorian, g.String(), "Jalali %d-%02d-%02d", jy, jm, jd)
	return engine.DayOf(g)
}

func gday(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func TestEdge_JalaliCalendarFacts(t *testing.T) {
	// The rows below rely on these facts; if the converter changes, every table here must be re-read.
	assert.Equal(t, "2025-03-20", checkups.FromJalali(1403, 12, 30).String(), "1403 is leap: 30 Esfand exists")
	assert.Equal(t, "2025-03-21", checkups.FromJalali(1404, 1, 1).String(), "Nowruz 1404")
	assert.Equal(t, "2026-03-20", checkups.FromJalali(1404, 12, 29).String(), "1404 is not leap: 29 Esfand is last")
	assert.Equal(t, "2026-03-21", checkups.FromJalali(1405, 1, 1).String(), "Nowruz 1405")
	// Negative: 30 Esfand 1404 does not exist; the day after 29 Esfand 1404 is Nowruz.
	jy, jm, jd := checkups.ToJalali(civildate.MustParse("2026-03-21"))
	assert.Equal(t, [3]int{1405, 1, 1}, [3]int{jy, jm, jd})
	assert.Equal(t, "2026-03-20", checkups.JalaliMonthEnd(civildate.MustParse("2026-03-01")).String(),
		"Esfand 1404 ends on the 29th")
	assert.Equal(t, "2025-03-20", checkups.JalaliMonthEnd(civildate.MustParse("2025-03-01")).String(),
		"Esfand 1403 ends on the 30th")
}

// A period logged from 28 Esfand to 2 Farvardin has the same Jalali labels in 1403/1404 and 1404/1405, but it is 5
// days in the leap year and 4 days in the common one: the engine must count Gregorian days, never "Esfand = 30".
func TestEdge_PeriodAcrossEsfandIntoFarvardin(t *testing.T) {
	type dayRow struct {
		label     string
		target    engine.Day
		cycleDay  int
		menstrual bool
	}
	cases := []struct {
		name       string
		start, end engine.Day
		wantLen    int
		days       []dayRow
	}{
		{
			name: "leap 1403 (30 Esfand exists)", wantLen: 5,
			start: jday(t, 1403, 12, 28, "2025-03-18"), end: jday(t, 1404, 1, 2, "2025-03-22"),
			days: []dayRow{
				{"28 Esfand", jday(t, 1403, 12, 28, "2025-03-18"), 1, true},
				{"29 Esfand", jday(t, 1403, 12, 29, "2025-03-19"), 2, true},
				{"30 Esfand (leap day)", jday(t, 1403, 12, 30, "2025-03-20"), 3, true},
				{"1 Farvardin", jday(t, 1404, 1, 1, "2025-03-21"), 4, true},
				{"2 Farvardin (logged end)", jday(t, 1404, 1, 2, "2025-03-22"), 5, true},
				{"3 Farvardin (after the end)", jday(t, 1404, 1, 3, "2025-03-23"), 6, false},
			},
		},
		{
			name: "common 1404 (no 30 Esfand)", wantLen: 4,
			start: jday(t, 1404, 12, 28, "2026-03-19"), end: jday(t, 1405, 1, 2, "2026-03-22"),
			days: []dayRow{
				{"28 Esfand", jday(t, 1404, 12, 28, "2026-03-19"), 1, true},
				{"29 Esfand (last day)", jday(t, 1404, 12, 29, "2026-03-20"), 2, true},
				{"1 Farvardin", jday(t, 1405, 1, 1, "2026-03-21"), 3, true},
				{"2 Farvardin (logged end)", jday(t, 1405, 1, 2, "2026-03-22"), 4, true},
				{"3 Farvardin (after the end)", jday(t, 1405, 1, 3, "2026-03-23"), 5, false},
			},
		},
		{
			// Boundary: a period that starts on the leap day itself.
			name: "starts on 30 Esfand 1403", wantLen: 3,
			start: jday(t, 1403, 12, 30, "2025-03-20"), end: jday(t, 1404, 1, 2, "2025-03-22"),
			days: []dayRow{
				{"29 Esfand (before the start)", jday(t, 1403, 12, 29, "2025-03-19"), 0, false},
				{"30 Esfand (start)", jday(t, 1403, 12, 30, "2025-03-20"), 1, true},
				{"2 Farvardin (logged end)", jday(t, 1404, 1, 2, "2025-03-22"), 3, true},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.wantLen, int(c.end-c.start)+1, "inclusive period length")
			s := engine.Prepare(engine.Input{Periods: []engine.Period{{Start: c.start, End: c.end}}}, engine.DefaultParams())
			for _, r := range c.days {
				st := s.Resolve(r.target, r.target)
				if r.cycleDay == 0 {
					_, ok := st.CycleDay.Get()
					assert.False(t, ok, "%s: no cycle before the first period", r.label)
					continue
				}
				assert.Equal(t, engine.Some(r.cycleDay), st.CycleDay, r.label)
				assert.Equal(t, r.menstrual, st.Subphase == engine.SubphaseMenstrual, "%s: subphase %v", r.label, st.Subphase)
				assert.Equal(t, c.start, st.Anchors.CurrentPeriodStart, r.label)
			}
		})
	}
}

// The cycle 15 Esfand → 13 Farvardin is 28 days in a leap year and 27 in a common one; the Feb 29 rows are the same
// check for the Gregorian leap day. CycleMin/CycleMax boundaries are taken across Nowruz so a Jalali month-length
// mistake would flip a cycle between valid and outlier.
func TestEdge_CycleLengthAcrossNowruz(t *testing.T) {
	p := engine.DefaultParams()
	cases := []struct {
		name        string
		start, next engine.Day
		want        int
		valid       bool
		short, long bool
	}{
		{"leap 1403: 15 Esfand → 13 Farvardin", jday(t, 1403, 12, 15, "2025-03-05"), jday(t, 1404, 1, 13, "2025-04-02"), 28, true, false, false},
		{"common 1404: 15 Esfand → 13 Farvardin", jday(t, 1404, 12, 15, "2026-03-06"), jday(t, 1405, 1, 13, "2026-04-02"), 27, true, false, false},
		{"Gregorian leap 2024: Feb 15 → Mar 14", gday("2024-02-15"), gday("2024-03-14"), 28, true, false, false},
		{"Gregorian common 2026: Feb 15 → Mar 14", gday("2026-02-15"), gday("2026-03-14"), 27, true, false, false},
		// Boundaries across Nowruz 1405.
		{"CycleMin across Nowruz is valid", gday("2026-03-10"), gday("2026-03-10") + engine.Day(p.CycleMin), p.CycleMin, true, false, false},
		{"CycleMin−1 across Nowruz is short", gday("2026-03-11"), gday("2026-03-11") + engine.Day(p.CycleMin-1), p.CycleMin - 1, false, true, false},
		{"CycleMax across Nowruz is valid", gday("2026-02-20"), gday("2026-02-20") + engine.Day(p.CycleMax), p.CycleMax, true, false, false},
		{"CycleMax+1 across Nowruz is long", gday("2026-02-19"), gday("2026-02-19") + engine.Day(p.CycleMax+1), p.CycleMax + 1, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := engine.Prepare(engine.Input{Periods: []engine.Period{
				{Start: c.start, End: c.start + 4},
				{Start: c.next, End: c.next + 4},
			}}, engine.DefaultParams())
			cycles := s.Cycles()
			require.Len(t, cycles, 1)
			got := cycles[0]
			assert.Equal(t, c.want, got.Length)
			assert.Equal(t, [3]bool{c.valid, c.short, c.long}, [3]bool{got.Valid, got.Short, got.Long}, "valid/short/long")
			// The day before the next start is the last cycle day of the completed cycle.
			st := s.Resolve(c.next-1, c.next)
			assert.Equal(t, engine.Some(c.want), st.CycleDay)
		})
	}
}

// A predicted next start that lands on Nowruz, on the leap day and on the day after the last day of a common Esfand.
func TestEdge_PredictionLandsOnJalaliBoundaries(t *testing.T) {
	cases := []struct {
		name string
		last engine.Day
		want engine.Day
	}{
		{"→ 30 Esfand 1403 (leap day)", gday("2025-02-20"), jday(t, 1403, 12, 30, "2025-03-20")},
		{"→ 1 Farvardin 1404", gday("2025-02-21"), jday(t, 1404, 1, 1, "2025-03-21")},
		{"→ 29 Esfand 1404 (last day)", gday("2026-02-20"), jday(t, 1404, 12, 29, "2026-03-20")},
		{"→ 1 Farvardin 1405", gday("2026-02-21"), jday(t, 1405, 1, 1, "2026-03-21")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := engine.Input{
				Periods:  []engine.Period{{Start: c.last, End: c.last + 4}},
				Declared: engine.Declared{CycleLength: engine.Some(28), PeriodLength: engine.Some(5), Manual: true},
			}
			s := engine.Prepare(in, engine.DefaultParams())
			st := s.Resolve(c.last+3, c.last+3)
			assert.Equal(t, c.want, st.Anchors.PredictedNextPeriodStart)
			// Negative: the day before the prediction is still this cycle (day 28), the prediction day is not.
			before := s.Resolve(c.want-1, c.want-1)
			assert.Equal(t, engine.Some(28), before.CycleDay)
			assert.NotEqual(t, engine.SubphaseMenstrual, before.Subphase)
		})
	}
}

// localDay is the user-local day of an instant in zone, through the same helper the handlers use (X-07).
func localDay(t *testing.T, at time.Time, zone string) engine.Day {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)
	ctx := civildate.WithZone(context.Background(), func(context.Context) *time.Location { return loc })
	return engine.DayOf(civildate.LocalToday(ctx, clock.At(at)))
}

// A period start logged at 23:30 vs 00:30 local belongs to two different days, in the user's zone — not Tehran's
// and not UTC. Rows cover the Tehran Nowruz midnight, a Berlin user whose 23:30 is already tomorrow in Tehran, both
// Berlin DST changes, and zones with odd offsets.
func TestEdge_LogAt2330vs0030Local(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	cases := []struct {
		name string
		at   time.Time
		zone string
		want string
	}{
		{"Tehran 23:30 on 29 Esfand 1404", time.Date(2026, 3, 20, 23, 30, 0, 0, civildate.Tehran), "Asia/Tehran", "2026-03-20"},
		{"Tehran 23:59:59 on 29 Esfand 1404", time.Date(2026, 3, 20, 23, 59, 59, 0, civildate.Tehran), "Asia/Tehran", "2026-03-20"},
		{"Tehran 00:00:00 on Nowruz 1405", time.Date(2026, 3, 21, 0, 0, 0, 0, civildate.Tehran), "Asia/Tehran", "2026-03-21"},
		{"Tehran 00:30 on Nowruz 1405", time.Date(2026, 3, 21, 0, 30, 0, 0, civildate.Tehran), "Asia/Tehran", "2026-03-21"},
		{"Tehran 23:30 on 30 Esfand 1403 (leap day)", time.Date(2025, 3, 20, 23, 30, 0, 0, civildate.Tehran), "Asia/Tehran", "2025-03-20"},
		// Berlin 23:30 CET = 02:00 Tehran the next day: the user's day wins.
		{"Berlin 23:30 on 2026-03-20", time.Date(2026, 3, 20, 23, 30, 0, 0, berlin), "Europe/Berlin", "2026-03-20"},
		{"same instant read in Tehran (negative)", time.Date(2026, 3, 20, 23, 30, 0, 0, berlin), "Asia/Tehran", "2026-03-21"},
		{"Berlin 00:30 on 2026-03-21", time.Date(2026, 3, 21, 0, 30, 0, 0, berlin), "Europe/Berlin", "2026-03-21"},
		// Spring forward 2026-03-29 02:00 → 03:00.
		{"Berlin 00:30 on the spring-forward day", time.Date(2026, 3, 29, 0, 30, 0, 0, berlin), "Europe/Berlin", "2026-03-29"},
		{"Berlin 23:30 CEST on the spring-forward day", time.Date(2026, 3, 29, 23, 30, 0, 0, berlin), "Europe/Berlin", "2026-03-29"},
		{"Berlin 00:00 the day after spring-forward", time.Date(2026, 3, 30, 0, 0, 0, 0, berlin), "Europe/Berlin", "2026-03-30"},
		// Fall back 2026-10-25 03:00 → 02:00: the 25-hour day.
		{"Berlin 00:30 on the fall-back day", time.Date(2026, 10, 25, 0, 30, 0, 0, berlin), "Europe/Berlin", "2026-10-25"},
		{"Berlin second 02:30 on the fall-back day", time.Date(2026, 10, 25, 0, 30, 0, 0, time.UTC), "Europe/Berlin", "2026-10-25"},
		{"Berlin 23:30 CET on the fall-back day", time.Date(2026, 10, 25, 23, 30, 0, 0, berlin), "Europe/Berlin", "2026-10-25"},
		{"Berlin 23:59:59 CET on the fall-back day", time.Date(2026, 10, 25, 22, 59, 59, 0, time.UTC), "Europe/Berlin", "2026-10-25"},
		{"Berlin 00:00 the day after fall-back", time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC), "Europe/Berlin", "2026-10-26"},
		// Odd offsets and the date line.
		{"Kathmandu (+05:45) 00:30", time.Date(2026, 3, 20, 18, 45, 0, 0, time.UTC), "Asia/Kathmandu", "2026-03-21"},
		{"Kiritimati (+14) 00:30", time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC), "Pacific/Kiritimati", "2026-03-21"},
		{"Pago Pago (−11) 23:30, same instant", time.Date(2026, 3, 20, 10, 30, 0, 0, time.UTC), "Pacific/Pago_Pago", "2026-03-19"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, gday(c.want), localDay(t, c.at, c.zone))
		})
	}
}

// The local day of the log moves the period start, hence the cycle day of every later target and the length of
// the cycle it closes. Two Berlin users log the same period start at 23:30 and 00:30 local (both are 2026-03-21 in
// Tehran): one day apart in the engine.
func TestEdge_LocalLogDayDrivesCycleDayAndLength(t *testing.T) {
	berlin, _ := time.LoadLocation("Europe/Berlin")
	prev := gday("2026-02-21")
	late := localDay(t, time.Date(2026, 3, 20, 23, 30, 0, 0, berlin), "Europe/Berlin")
	early := localDay(t, time.Date(2026, 3, 21, 0, 30, 0, 0, berlin), "Europe/Berlin")
	require.Equal(t, gday("2026-03-20"), late)
	require.Equal(t, gday("2026-03-21"), early)

	target := gday("2026-03-25")
	for _, c := range []struct {
		name     string
		start    engine.Day
		length   int
		cycleDay int
	}{
		{"logged 23:30 local (29 Esfand)", late, 27, 6},
		{"logged 00:30 local (Nowruz)", early, 28, 5},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := engine.Prepare(engine.Input{Periods: []engine.Period{
				{Start: prev, End: prev + 4},
				{Start: c.start, End: c.start + 4},
			}}, engine.DefaultParams())
			require.Len(t, s.Cycles(), 1)
			assert.Equal(t, c.length, s.Cycles()[0].Length)
			assert.Equal(t, engine.Some(c.cycleDay), s.Resolve(target, target).CycleDay)
		})
	}

	// The "today" the engine gets for a Berlin user at 23:30 is the Berlin day: cycle day 1 of a period logged that
	// evening, not day 2 (which is what the Tehran day of the same instant would give).
	s := engine.Prepare(engine.Input{Periods: []engine.Period{{Start: prev, End: prev + 4}, {Start: late, End: NoEnd}}},
		engine.DefaultParams())
	today := localDay(t, time.Date(2026, 3, 20, 23, 30, 0, 0, berlin), "Europe/Berlin")
	assert.Equal(t, engine.Some(1), s.Resolve(today, today).CycleDay)
	tehranToday := localDay(t, time.Date(2026, 3, 20, 23, 30, 0, 0, berlin), "Asia/Tehran")
	assert.Equal(t, engine.Some(2), s.Resolve(tehranToday, tehranToday).CycleDay, "negative: the server zone is a day ahead")
}

// NoEnd is an open period.
const NoEnd = engine.NoDay
