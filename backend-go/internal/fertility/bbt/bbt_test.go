package bbt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

var start = civildate.MustParse("2026-09-01")

func ptr(v int) *int { return &v }

// series places values on consecutive cycle days from day 1; 0 = no reading that day.
func series(from civildate.Date, values ...int) []bbt.Reading {
	var out []bbt.Reading
	for i, v := range values {
		if v != 0 {
			out = append(out, bbt.Reading{Date: from.AddDays(i), Value: v})
		}
	}
	return out
}

func TestAnalyze(t *testing.T) {
	cycle := bbt.CycleInput{Start: start, End: start.AddDays(27), FertileWindow: &bbt.DayRange{FromDay: 10, ToDay: 15}}
	cases := []struct {
		name      string
		readings  []bbt.Reading
		today     civildate.Date
		coverline *int
		shiftDay  *int
		phase     bbt.Phase
		avg       *int
		logged    int
		soFar     int
		gaps      int
	}{
		{
			name: "clean biphasic",
			// days 1-12 low (max 3640), day 13 3665 (≥ +0.20), 14-15 above → shift day 13.
			readings: series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3640, 3630, 3625, 3635,
				3665, 3670, 3675, 3680),
			today: start.AddDays(15), coverline: ptr(3640), shiftDay: ptr(13), phase: bbt.PhasePostShift,
			avg: ptr(3627), logged: 16, soFar: 16, gaps: 0,
		},
		{
			name:     "no shift: flat readings, coverline is the latest six",
			readings: series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3620, 3645, 3630),
			today:    start.AddDays(9), coverline: ptr(3645), phase: bbt.PhasePreShift,
			avg: ptr(3627), logged: 10, soFar: 10,
		},
		{
			name: "rise that falls back is not a shift",
			// day 7 3665 clears 3640 by .25 but day 9 drops to the coverline → no shift.
			readings: series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3665, 3660, 3640, 3630, 3625, 3630),
			today:    start.AddDays(11), coverline: ptr(3665), phase: bbt.PhasePreShift,
			avg: ptr(3627), logged: 12, soFar: 12,
		},
		{
			name:     "rise just below 0.2 does not start a shift",
			readings: series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3659, 3662, 3665, 3660, 3661, 3659),
			today:    start.AddDays(11), coverline: ptr(3665), phase: bbt.PhasePreShift,
			avg: ptr(3627), logged: 12, soFar: 12,
		},
		{
			name:     "started rise without three readings yet: provisional coverline of that rise",
			readings: series(start, 3620, 3630, 3610, 3640, 3625, 3635, 3630, 3665, 3670),
			today:    start.AddDays(8), coverline: ptr(3640), phase: bbt.PhasePreShift,
			avg: ptr(3627), logged: 9, soFar: 9,
		},
		{
			name: "noisy with gaps: readings, not days, are counted",
			// gaps on days 3, 8, 9, 14, 17; a lone spike on day 6 (3670) is followed by a dip; the shift is
			// the 6-reading baseline before day 15 (max 3670 → needs ≥ 3690).
			readings: series(start, 3620, 3650, 0, 3610, 3640, 3670, 3600, 0, 0, 3630, 3650, 3620, 3640, 0,
				3695, 3700, 0, 3705),
			today: start.AddDays(17), coverline: ptr(3670), shiftDay: ptr(15), phase: bbt.PhasePostShift,
			avg: ptr(3632), logged: 13, soFar: 18, gaps: 5,
		},
		{
			name:     "fewer than 6 readings: no coverline, average of what exists",
			readings: series(start, 3620, 0, 3631, 0, 3640),
			today:    start.AddDays(6), phase: bbt.PhasePreShift,
			avg: ptr(3630), logged: 3, soFar: 7, gaps: 4,
		},
		{
			name:  "no readings",
			today: start.AddDays(2), phase: bbt.PhasePreShift, soFar: 3, gaps: 3,
		},
		{
			name:     "today before the cycle start: nothing so far",
			readings: series(start, 3620),
			today:    start.AddDays(-1), phase: bbt.PhasePreShift,
		},
		{
			name: "readings outside the cycle and after today are ignored",
			readings: append(append(series(start.AddDays(-3), 3700, 3700, 3700), series(start, 3620, 3630)...),
				bbt.Reading{Date: start.AddDays(28), Value: 3700}, bbt.Reading{Date: start.AddDays(5), Value: 3650}),
			today: start.AddDays(3), phase: bbt.PhasePreShift, avg: ptr(3625), logged: 2, soFar: 4, gaps: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := bbt.Analyze(cycle, tc.readings, tc.today)
			assert.Equal(t, start, c.Start)
			assert.Equal(t, cycle.FertileWindow, c.FertileWindow)
			assert.Equal(t, tc.coverline, c.Coverline, "coverline")
			assert.Equal(t, tc.shiftDay, c.ShiftDay, "shift day")
			assert.Equal(t, tc.phase, c.Phase)
			assert.Equal(t, tc.avg, c.Stats.PreOvulationAvg, "pre-ovulation avg")
			assert.Equal(t, tc.logged, c.Stats.LoggedDays, "logged")
			assert.Equal(t, tc.soFar, c.Stats.CycleDaysSoFar, "so far")
			assert.Equal(t, tc.gaps, c.Stats.Gaps, "gaps")
			assert.Len(t, c.Points, tc.logged)
			for i, p := range c.Points {
				assert.Equal(t, start.DiffDays(p.Date)+1, p.CycleDay)
				if i > 0 {
					assert.Greater(t, p.CycleDay, c.Points[i-1].CycleDay, "sorted by cycle day")
				}
			}
			if tc.shiftDay != nil {
				assert.Equal(t, ptr(*tc.shiftDay-1), c.OvulationDay())
			} else {
				assert.Nil(t, c.OvulationDay())
			}
		})
	}
}

func TestAnalyze_PastCycleStopsAtItsEnd(t *testing.T) {
	c := bbt.Analyze(bbt.CycleInput{Start: start, End: start.AddDays(4)}, series(start, 3620, 3630, 3640, 3650, 3660, 3670, 3680),
		start.AddDays(40))
	assert.Equal(t, 5, c.Stats.CycleDaysSoFar)
	assert.Equal(t, 5, c.Stats.LoggedDays)
	assert.Nil(t, c.Coverline)
}

func TestBoundaries(t *testing.T) {
	d := civildate.MustParse
	cur, end := d("2026-09-01"), d("2026-09-28")
	starts := []civildate.Date{d("2026-07-05"), d("2026-08-03"), d("2026-08-03"), d("2026-06-07"), d("2026-09-01"),
		d("2026-09-10"), d("2026-05-10")}

	got := bbt.Boundaries(cur, end, starts, 3)
	assert.Equal(t, []bbt.CycleInput{
		{Start: cur, End: end},
		{Start: d("2026-08-03"), End: d("2026-08-31")},
		{Start: d("2026-07-05"), End: d("2026-08-02")},
	}, got, "newest first, duplicates / current / later starts skipped, capped at count")

	all := bbt.Boundaries(cur, end, starts, 6)
	require.Len(t, all, 5, "only as many as the history has")
	assert.Equal(t, bbt.CycleInput{Start: d("2026-05-10"), End: d("2026-06-06")}, all[4])

	assert.Equal(t, []bbt.CycleInput{{Start: cur, End: end}}, bbt.Boundaries(cur, end, nil, 6))
	assert.Empty(t, bbt.Boundaries(civildate.Date{}, end, starts, 6), "no current cycle")
	assert.Empty(t, bbt.Boundaries(cur, end, starts, 0))
}

// Range 3 / 6: each cycle's points are numbered from its own start, so the overlaid charts align
// by cycle day; past shift days come from every previous cycle, whatever the range.
func TestBuild_RangeAlignment(t *testing.T) {
	d := civildate.MustParse
	biphasic := func(from civildate.Date, shiftDay int) []bbt.Reading {
		vals := make([]int, 0, 26)
		for day := 1; day <= 26; day++ {
			v := 3620 + (day%3)*5
			if day >= shiftDay {
				v = 3680
			}
			vals = append(vals, v)
		}
		return series(from, vals...)
	}
	starts := []civildate.Date{d("2026-06-06"), d("2026-07-04"), d("2026-08-02")}
	cycles := bbt.Boundaries(d("2026-08-30"), d("2026-09-26"), starts, 6)
	require.Len(t, cycles, 4)
	var readings []bbt.Reading
	readings = append(readings, biphasic(d("2026-06-06"), 14)...)
	readings = append(readings, biphasic(d("2026-07-04"), 16)...)
	readings = append(readings, biphasic(d("2026-08-02"), 15)...)
	readings = append(readings, series(d("2026-08-30"), 3620, 3630, 3625)...)
	today := d("2026-09-01")

	for _, tc := range []struct {
		rangeSize, cycles int
	}{{1, 1}, {3, 3}, {6, 4}} {
		res := bbt.Build(cycles, readings, today, tc.rangeSize)
		require.Len(t, res.Cycles, tc.cycles, "range %d", tc.rangeSize)
		assert.Equal(t, []int{15, 16, 14}, res.PastShiftDays, "range %d", tc.rangeSize)
		for i, c := range res.Cycles {
			assert.Equal(t, cycles[i].Start, c.Start)
			require.NotEmpty(t, c.Points)
			assert.Equal(t, 1, c.Points[0].CycleDay, "day 1 = own start")
			assert.Equal(t, c.Start, c.Points[0].Date)
		}
	}
	res := bbt.Build(cycles, readings, today, 3)
	assert.Equal(t, bbt.PhasePreShift, res.Cycles[0].Phase)
	assert.Equal(t, 3, res.Cycles[0].Stats.CycleDaysSoFar)
	assert.Equal(t, ptr(15), res.Cycles[1].ShiftDay)
	assert.Equal(t, 26, res.Cycles[2].Stats.LoggedDays, "a past cycle keeps only its own readings")

	empty := bbt.Build(nil, nil, today, 6)
	assert.Equal(t, []bbt.Cycle{}, empty.Cycles)
	assert.Equal(t, []int{}, empty.PastShiftDays)
}

func TestParseAndFormat(t *testing.T) {
	for in, want := range map[string]int{"36.55": 3655, "36.5": 3650, "36": 3600, "37.05": 3705, "36.": 3600} {
		v, err := bbt.ParseValue(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, v, in)
	}
	for _, bad := range []string{"", "abc", "36.555", "-36.5", "36.x", "36.-1"} {
		_, err := bbt.ParseValue(bad)
		assert.Error(t, err, bad)
	}
	assert.Equal(t, "36.55", bbt.Format(3655))
	assert.Equal(t, "36.05", bbt.Format(3605))
	assert.Equal(t, "37.00", bbt.Format(3700))
}
