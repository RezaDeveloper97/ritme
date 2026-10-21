package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

func day(s string) Day { return DayOf(civildate.MustParse(s)) }

// history builds closed periods of periodLen days separated by the given cycle lengths, starting 2025-01-01.
func history(periodLen int, lengths ...int) []Period {
	s := day("2025-01-01")
	out := []Period{{Start: s, End: s + Day(periodLen-1)}}
	for _, l := range lengths {
		s += Day(l)
		out = append(out, Period{Start: s, End: s + Day(periodLen-1)})
	}
	return out
}

func prep(in Input) *State { return Prepare(in, DefaultParams()) }

func TestPrepare_EffectiveCycleLength(t *testing.T) {
	cases := []struct {
		name     string
		lengths  []int
		declared Opt[int]
		want     int
		src      LengthSource
	}{
		{"§2.9 median(28,29)=28.5 → 29", []int{28, 29}, None[int](), 29, LengthHistory},
		{"median(27,28)=27.5 → 28", []int{27, 28}, None[int](), 28, LengthHistory},
		{"median of 3", []int{28, 30, 35}, None[int](), 30, LengthHistory},
		{"one record", []int{31}, None[int](), 31, LengthHistory},
		{"none + declared", nil, Some(31), 31, LengthDeclared},
		{"none → default", nil, None[int](), 28, LengthDefault},
		{"four valid → only the last 3", []int{21, 28, 30, 35}, None[int](), 30, LengthHistory},
		{"declared never enters history medians", []int{28}, Some(35), 28, LengthHistory},
		{"declared outside 21..45 is ignored", nil, Some(60), 28, LengthDefault},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := prep(Input{Periods: history(5, c.lengths...), Declared: Declared{CycleLength: c.declared}})
			got, src := s.CycleLength()
			assert.Equal(t, c.want, got)
			assert.Equal(t, c.src, src)
			assert.False(t, s.Contradictory())
		})
	}
}

func TestPrepare_EffectivePeriodLength(t *testing.T) {
	s := day("2026-01-01")
	closed := func(start Day, n int) Period { return Period{Start: start, End: start + Day(n-1)} }

	st := prep(Input{Periods: []Period{closed(s, 4), closed(s+28, 5), closed(s+56, 6)}})
	n, src := st.PeriodLength()
	assert.Equal(t, 5, n)
	assert.Equal(t, LengthHistory, src)

	st = prep(Input{Periods: []Period{closed(s, 5), closed(s+28, 6)}})
	n, _ = st.PeriodLength()
	assert.Equal(t, 6, n, "median(5,6)=5.5 → 6")

	st = prep(Input{})
	n, src = st.PeriodLength()
	assert.Equal(t, 5, n)
	assert.Equal(t, LengthDefault, src)

	st = prep(Input{Declared: Declared{PeriodLength: Some(7)}})
	n, src = st.PeriodLength()
	assert.Equal(t, 7, n)
	assert.Equal(t, LengthDeclared, src)
}

func TestPrepare_SkippedCycle(t *testing.T) {
	// §2.8 / E-15: starts 01-01, 01-29, 02-26, skip 03-26, start 04-23.
	in := Input{
		Periods: []Period{
			{Start: day("2026-01-01"), End: day("2026-01-05")},
			{Start: day("2026-01-29"), End: day("2026-02-02")},
			{Start: day("2026-02-26"), End: day("2026-03-02")},
			{Start: day("2026-04-23"), End: NoDay},
		},
		Skips: []Day{day("2026-03-26")},
	}
	s := prep(in)
	require.Len(t, s.Cycles(), 3)
	gap := s.Cycles()[2]
	assert.Equal(t, 56, gap.Length)
	assert.True(t, gap.Skipped)
	assert.False(t, gap.Valid)
	assert.False(t, gap.Short)
	assert.False(t, gap.Long, "a skipped gap is not an outlier")
	assert.Equal(t, 2, s.ValidCycleCount(), "valid [28, 28]")
	n, src := s.CycleLength()
	assert.Equal(t, 28, n)
	assert.Equal(t, LengthHistory, src)
	assert.Equal(t, Some(0), s.Variability())
	assert.Equal(t, []Day{day("2026-03-26")}, s.Skips())
}

func TestPrepare_Outliers(t *testing.T) {
	s := prep(Input{Periods: history(5, 18, 50)})
	require.Len(t, s.Cycles(), 2)
	assert.True(t, s.Cycles()[0].Short)
	assert.False(t, s.Cycles()[0].Valid)
	assert.True(t, s.Cycles()[1].Long)
	assert.False(t, s.Cycles()[1].Valid)
	assert.Zero(t, s.ValidCycleCount())
	n, src := s.CycleLength()
	assert.Equal(t, 28, n, "outliers never enter the median")
	assert.Equal(t, LengthDefault, src)
	assert.False(t, s.Variability().OK())

	bounds := prep(Input{Periods: history(5, 21, 45)})
	assert.True(t, bounds.Cycles()[0].Valid, "21 is valid")
	assert.True(t, bounds.Cycles()[1].Valid, "45 is valid")
}

func TestPrepare_InvalidAndOpenPeriods(t *testing.T) {
	s := day("2026-01-01")
	st := prep(Input{Periods: []Period{
		{Start: s, End: s},                // 1 day
		{Start: s + 28, End: s + 28 + 10}, // 11 days
	}})
	assert.Zero(t, st.ValidPeriodCount())
	n, src := st.PeriodLength()
	assert.Equal(t, 5, n)
	assert.Equal(t, LengthDefault, src)

	st = prep(Input{Periods: []Period{
		{Start: s, End: s + 5},      // 6 days
		{Start: s + 28, End: NoDay}, // open: never counts
	}})
	assert.Equal(t, 1, st.ValidPeriodCount())
	n, _ = st.PeriodLength()
	assert.Equal(t, 6, n)

	bounds := prep(Input{Periods: []Period{{Start: s, End: s + 1}, {Start: s + 28, End: s + 37}}})
	assert.Equal(t, 2, bounds.ValidPeriodCount(), "2 and 10 days are valid")
}

func TestPrepare_Contradictions(t *testing.T) {
	t.Run("end before start", func(t *testing.T) {
		bad := Period{Start: day("2026-02-10"), End: day("2026-02-08")}
		st := prep(Input{Periods: []Period{{Start: day("2026-01-13"), End: day("2026-01-17")}, bad}})
		assert.True(t, st.Contradictory())
		assert.Equal(t, []Period{bad}, st.Excluded())
		assert.Len(t, st.Periods(), 1)
	})
	t.Run("two overlapping periods: the newer is ignored", func(t *testing.T) {
		old := Period{Start: day("2026-01-01"), End: day("2026-01-05")}
		newer := Period{Start: day("2026-01-03"), End: day("2026-01-11")}
		st := prep(Input{Periods: []Period{newer, old}})
		assert.True(t, st.Contradictory())
		assert.Equal(t, []Period{old}, st.Periods())
		assert.Equal(t, []Period{newer}, st.Excluded())
		n, src := st.PeriodLength()
		assert.Equal(t, 5, n, "the 9-day newer record does not enter the median")
		assert.Equal(t, LengthHistory, src)
	})
	t.Run("a start inside another period", func(t *testing.T) {
		st := prep(Input{Periods: []Period{
			{Start: day("2026-01-01"), End: day("2026-01-07")},
			{Start: day("2026-01-04"), End: day("2026-01-05")},
		}})
		assert.True(t, st.Contradictory())
		assert.Len(t, st.Periods(), 1)
		assert.Empty(t, st.Cycles(), "the excluded start forms no 3-day cycle")
	})
	t.Run("same start twice", func(t *testing.T) {
		st := prep(Input{Periods: []Period{
			{Start: day("2026-01-01"), End: NoDay},
			{Start: day("2026-01-01"), End: day("2026-01-05")},
		}})
		assert.True(t, st.Contradictory())
		assert.Equal(t, []Period{{Start: day("2026-01-01"), End: NoDay}}, st.Periods(), "input order: the first is older")
	})
	t.Run("two open periods: the newer is ignored", func(t *testing.T) {
		st := prep(Input{Periods: []Period{
			{Start: day("2026-01-29"), End: NoDay},
			{Start: day("2026-01-01"), End: NoDay},
		}})
		assert.True(t, st.Contradictory())
		assert.Equal(t, []Period{{Start: day("2026-01-01"), End: NoDay}}, st.Periods())
		assert.Empty(t, st.Cycles())
	})
	t.Run("an older open period followed by later starts is not a conflict", func(t *testing.T) {
		st := prep(Input{Periods: []Period{
			{Start: day("2026-01-01"), End: NoDay},
			{Start: day("2026-01-29"), End: day("2026-02-02")},
		}})
		assert.False(t, st.Contradictory())
		assert.Len(t, st.Cycles(), 1)
	})
	t.Run("adjacent periods do not overlap", func(t *testing.T) {
		st := prep(Input{Periods: []Period{
			{Start: day("2026-01-01"), End: day("2026-01-05")},
			{Start: day("2026-01-06"), End: day("2026-01-07")},
		}})
		assert.False(t, st.Contradictory())
	})
	t.Run("consistent history", func(t *testing.T) {
		st := prep(Input{Periods: history(5, 28, 28)})
		assert.False(t, st.Contradictory())
		assert.Empty(t, st.Excluded())
	})
}

func TestPrepare_ManualLengths(t *testing.T) {
	// E-20: the "use my lengths" switch overrides history, source declared.
	st := prep(Input{
		Periods:  history(5, 28, 28, 28),
		Declared: Declared{CycleLength: Some(32), PeriodLength: Some(6), Manual: true},
	})
	c, cs := st.CycleLength()
	p, ps := st.PeriodLength()
	assert.Equal(t, 32, c)
	assert.Equal(t, 6, p)
	assert.Equal(t, LengthDeclared, cs)
	assert.Equal(t, LengthDeclared, ps)

	// Manual cycle only: the period length still comes from history.
	st = prep(Input{Periods: history(5, 28, 28, 28), Declared: Declared{CycleLength: Some(32), Manual: true}})
	_, ps = st.PeriodLength()
	assert.Equal(t, LengthHistory, ps)
}

func TestPrepare_Variability(t *testing.T) {
	assert.False(t, prep(Input{Periods: history(5, 28)}).Variability().OK(), "one valid → null")
	assert.False(t, prep(Input{}).Variability().OK())
	assert.Equal(t, Some(6), prep(Input{Periods: history(5, 27, 29, 33)}).Variability())
	assert.Equal(t, Some(4), prep(Input{Periods: history(5, 40, 27, 29, 31)}).Variability(), "last 3 only")
	assert.Equal(t, Some(2), prep(Input{Periods: history(5, 27, 50, 29)}).Variability(), "outliers excluded")
}

func TestPrepare_SortsAndCopies(t *testing.T) {
	periods := history(5, 28, 30)
	periods[0], periods[2] = periods[2], periods[0]
	skips := []Day{day("2030-01-01"), NoDay, day("2029-01-01"), day("2029-01-01")}
	in := Input{Periods: periods, Skips: skips, LH: []LHTest{{Date: day("2025-01-14"), Positive: true}}}
	st := prep(in)
	require.Len(t, st.Periods(), 3)
	assert.Less(t, st.Periods()[0].Start, st.Periods()[1].Start)
	assert.Equal(t, []Day{day("2029-01-01"), day("2030-01-01")}, st.Skips(), "sorted, unique, no NoDay")
	n, _ := st.CycleLength()
	assert.Equal(t, 29, n)

	periods[0].Start = 0
	skips[0] = 0
	in.LH[0].Positive = false
	assert.NotEqual(t, Day(0), st.Periods()[0].Start, "Prepare does not alias the caller's slices")
	assert.True(t, st.in.LH[0].Positive)
	assert.Equal(t, DefaultParams(), st.Params())
}

func TestPrepare_DropsRecordsWithoutStart(t *testing.T) {
	st := prep(Input{Periods: []Period{{Start: NoDay, End: day("2026-01-05")}}})
	assert.Empty(t, st.Periods())
	assert.False(t, st.Contradictory())
}

func benchInput() Input {
	lengths := make([]int, 59)
	for i := range lengths {
		lengths[i] = 26 + i%5
	}
	periods := history(5, lengths...)
	periods[len(periods)-1].End = NoDay
	return Input{
		Periods:  periods,
		Skips:    []Day{day("2027-06-01")},
		Declared: Declared{LastPeriodStart: Some(day("2024-12-01")), CycleLength: Some(28), PeriodLength: Some(5)},
	}
}
