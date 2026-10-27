package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func eday(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func prepared(in engine.Input, today civildate.Date) *engine.State {
	return engine.Prepare(in, engineparams.For(today))
}

// T-M9-37: a back-dated one-day bleeding row inside a logged period (the Laravel / D-55 rows) splits the v1 cycle —
// it is the newest cycle_histories start — but the engine excludes it, so the v6 bounds keep the real period.
func TestEngineBounds_BackDatedOneDayRowDoesNotSplit(t *testing.T) {
	today := civildate.MustParse("2026-09-15")
	in := engine.Input{Periods: []engine.Period{
		{Start: eday("2026-08-12"), End: eday("2026-08-16")},
		{Start: eday("2026-09-10"), End: eday("2026-09-14")},
		{Start: eday("2026-09-12"), End: eday("2026-09-12")}, // back-dated 1-day row inside the period
	}}
	b := engineBounds(prepared(in, today), today, today)
	require.Len(t, b.starts, 2)
	assert.Equal(t, civildate.MustParse("2026-09-10"), b.starts[0].Start)
	assert.Equal(t, civildate.MustParse("2026-09-14"), *b.starts[0].End)
	assert.Equal(t, civildate.MustParse("2026-08-12"), b.starts[1].Start)

	// PMDD: one closed cycle 08-12 … 09-09 and the current one from 09-10 (v1 would start it on 09-12).
	cycles := layoutCycles(b.starts, today, b.currentEnd, b.periodLen, nil)
	require.Len(t, cycles, 2)
	assert.Equal(t, civildate.MustParse("2026-09-10"), cycles[0].Start)
	assert.Equal(t, civildate.MustParse("2026-09-09"), cycles[1].End)
	v1 := BuildCycles([]PeriodStart{{Start: civildate.MustParse("2026-09-12")}, {Start: civildate.MustParse("2026-09-10")}},
		today, 28, 5, nil)
	assert.Equal(t, civildate.MustParse("2026-09-12"), v1[0].Start, "the v1 split this test guards against")

	// PBAC: 09-15 is the grace day after the logged end of the 09-10 period; v1's newest row (09-12, ended 09-12)
	// would put it in no period.
	none := func(civildate.Date) bool { return false }
	start, ok := PeriodStartFor(b.starts, today, none)
	require.True(t, ok)
	assert.Equal(t, civildate.MustParse("2026-09-10"), start)
	end := civildate.MustParse("2026-09-12")
	_, ok = PeriodStartFor([]PeriodStart{{Start: end, End: &end}}, today, none)
	assert.False(t, ok)
}

// T-M9-37: a recorded skip «پریود نداشتم» moves the current cycle's expected end to the engine's next prediction
// (skip + effective length) instead of running "until today" from the declared length.
func TestEngineBounds_SkipRespected(t *testing.T) {
	today := civildate.MustParse("2026-10-20")
	in := engine.Input{
		Periods: []engine.Period{
			{Start: eday("2026-07-15"), End: eday("2026-07-19")},
			{Start: eday("2026-08-12"), End: eday("2026-08-16")},
			{Start: eday("2026-09-09"), End: eday("2026-09-13")},
		},
		Skips: []engine.Day{eday("2026-10-07")}, // the predicted start she marked as skipped
	}
	b := engineBounds(prepared(in, today), today, today)
	require.Len(t, b.starts, 3)
	assert.Equal(t, 28, b.cycleLen)
	assert.Equal(t, civildate.MustParse("2026-11-03"), b.currentEnd)
	cycles := layoutCycles(b.starts, today, b.currentEnd, b.periodLen, nil)
	assert.Equal(t, civildate.MustParse("2026-09-09"), cycles[0].Start)
	assert.Equal(t, civildate.MustParse("2026-11-03"), cycles[0].End)
	assert.Equal(t, Window{civildate.MustParse("2026-10-25"), civildate.MustParse("2026-11-03")}, cycles[0].LateLuteal)

	v1 := BuildCycles(b.starts, today, 28, 5, nil)
	assert.Equal(t, today, v1[0].End, "v1 ignores the skip: overdue, the cycle runs until today")
}

// Without a skip a late period keeps the cycle running until today (as v1).
func TestEngineBounds_LateRunsUntilToday(t *testing.T) {
	today := civildate.MustParse("2026-10-20")
	in := engine.Input{Periods: []engine.Period{
		{Start: eday("2026-08-12"), End: eday("2026-08-16")},
		{Start: eday("2026-09-09"), End: eday("2026-09-13")},
	}}
	b := engineBounds(prepared(in, today), today, today)
	assert.Equal(t, civildate.MustParse("2026-10-06"), b.currentEnd)
	cycles := layoutCycles(b.starts, today, b.currentEnd, b.periodLen, nil)
	assert.Equal(t, today, cycles[0].End)
}
