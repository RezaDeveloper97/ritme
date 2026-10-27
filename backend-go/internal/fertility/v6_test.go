package fertility

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-35 (D-128): the v6 insights and chart over a prepared engine state (no DB).

var v6Today = civildate.MustParse("2026-09-23")

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

// v6Biphasic is a 28-day series rising on shiftDay (cycle day).
func v6Biphasic(start string, shiftDay, n int) []engine.BBTReading {
	from := day(start)
	out := make([]engine.BBTReading, 0, n)
	for i := 0; i < n; i++ {
		c := int32(3620 + (i%3)*5)
		if i+1 >= shiftDay {
			c = 3680
		}
		out = append(out, engine.BBTReading{Date: from + engine.Day(i), Centi: c})
	}
	return out
}

// v6State: three logged periods (07-16, 08-13, 09-10; 5-day bleeds), BBT shifts on cycle day 15 and 16 in the two
// finished cycles; lh adds LH tests.
func v6State(lh ...engine.LHTest) *engine.State {
	var periods []engine.Period
	for _, s := range []string{"2026-07-16", "2026-08-13", "2026-09-10"} {
		periods = append(periods, engine.Period{Start: day(s), End: day(s) + 4})
	}
	bbt := append(v6Biphasic("2026-07-16", 15, 28), v6Biphasic("2026-08-13", 16, 28)...)
	return engine.Prepare(engine.Input{
		Periods: periods, BBT: bbt, LH: lh,
		Declared: engine.Declared{CycleLength: engine.Some(28), PeriodLength: engine.Some(5)},
	}, engine.DefaultParams())
}

func TestInsightsV6_OneRangePerCycle(t *testing.T) {
	st := v6State()
	v := newView(st, v6Today)
	now, cycles := chartCyclesV6(v, v6Today)
	require.Len(t, cycles, 3)
	ins := insightsV6(v, now, cycles, nil, v6Today)
	assert.LessOrEqual(t, v.ranges, len(cycles), "≤ 1 Range per cycle")
	assert.Equal(t, 1, v.resolves, "one Resolve (today)")

	require.Len(t, ins.History, 2)
	assert.Equal(t, SourceBBT, ins.History[0].Source)
	assert.Equal(t, 15, *ins.History[0].OvulationDay, "BBT shift on day 16 → ovulation day 15")
	assert.Equal(t, "2026-08-27", ins.History[0].Ovulation.String())
	assert.Equal(t, 5, *ins.History[0].PeriodDays)
	assert.Equal(t, SourceBBT, ins.History[1].Source)
	assert.Equal(t, 14, *ins.History[1].OvulationDay)

	assert.Equal(t, []int{16, 15}, ins.Evidence[1].ShiftDays)
	assert.Equal(t, StrengthStrong, ins.Evidence[1].Strength)
	assert.Equal(t, 2, ins.CyclesUsed)
	assert.Equal(t, StrengthMedium, ins.Evidence[0].Strength, "2 cycles: regularity insufficient_data → medium")
	require.NotNil(t, ins.Window)
	assert.False(t, ins.Window.End.Before(ins.Window.Start))
	assert.Equal(t, ins.Window.End, ins.Window.Ovulation)
}

// The LH/BBT conflict is shown with BBT (§2.17): LH positive on cycle day 5 of the first cycle, the BBT ovulation on
// day 14 → the history keeps the BBT day and source.
func TestInsightsV6_LHBBTConflictShowsBBT(t *testing.T) {
	st := v6State(engine.LHTest{Date: day("2026-07-20"), Positive: true})
	require.True(t, st.Ovulations()[0].Conflict)
	v := newView(st, v6Today)
	now, cycles := chartCyclesV6(v, v6Today)
	ins := insightsV6(v, now, cycles, nil, v6Today)
	row := ins.History[1]
	assert.Equal(t, SourceBBT, row.Source)
	assert.Equal(t, 14, *row.OvulationDay)
}

// LH only (no BBT in the cycle): the engine's LH ovulation (first positive + 1).
func TestInsightsV6_LHOnly(t *testing.T) {
	periods := []engine.Period{{Start: day("2026-07-16"), End: day("2026-07-20")}, {Start: day("2026-08-13"), End: day("2026-08-17")},
		{Start: day("2026-09-10"), End: day("2026-09-14")}}
	st := engine.Prepare(engine.Input{Periods: periods, LH: []engine.LHTest{{Date: day("2026-08-26"), Positive: true}}},
		engine.DefaultParams())
	v := newView(st, v6Today)
	now, cycles := chartCyclesV6(v, v6Today)
	ins := insightsV6(v, now, cycles, nil, v6Today)
	assert.Equal(t, SourceLH, ins.History[0].Source)
	assert.Equal(t, 15, *ins.History[0].OvulationDay)
	assert.Equal(t, SourceEstimate, ins.History[1].Source)
}

// The chart's shift is the engine's: nine readings of the current cycle never confirm (MinReadings 10).
func TestBBTV6_NineReadingsUnconfirmed(t *testing.T) {
	periods := []engine.Period{{Start: day("2026-09-10"), End: day("2026-09-14")}}
	rs := []int32{3620, 3630, 3610, 3640, 3625, 3635, 3665, 3670, 3675}
	var bbt []engine.BBTReading
	for i, c := range rs {
		bbt = append(bbt, engine.BBTReading{Date: day("2026-09-12") + engine.Day(i), Centi: c})
	}
	st := engine.Prepare(engine.Input{Periods: periods, BBT: bbt}, engine.DefaultParams())
	res := BBTV6(st, v6Today, 1)
	require.Len(t, res.Cycles, 1)
	assert.Nil(t, res.Cycles[0].ShiftDay)
	assert.Len(t, res.Cycles[0].Points, 9)
	assert.False(t, st.Ovulations()[0].BBT.IsSet(), "the engine agrees")
}

// No anchor: no chart cycle, no window, the unknown chance.
func TestV6_NoAnchor(t *testing.T) {
	st := engine.Prepare(engine.Input{}, engine.DefaultParams())
	res := BBTV6(st, v6Today, 3)
	assert.Empty(t, res.Cycles)
	v := newView(st, v6Today)
	now, cycles := chartCyclesV6(v, v6Today)
	ins := insightsV6(v, now, cycles, nil, v6Today)
	assert.Nil(t, ins.Window)
	assert.Equal(t, ConfidenceLow, ins.Confidence)
	cx := contextV6(newView(st, v6Today), v6Today)
	assert.Nil(t, cx.Chance.Level)
	assert.Nil(t, cx.CycleDay)
}

// Hormonal contraception suppresses fertility: no window, no chance level.
func TestV6_SuppressedFertility(t *testing.T) {
	periods := []engine.Period{{Start: day("2026-09-10"), End: day("2026-09-14")}}
	st := engine.Prepare(engine.Input{Periods: periods, Hormonal: true}, engine.DefaultParams())
	v := newView(st, v6Today)
	now, cycles := chartCyclesV6(v, v6Today)
	assert.Nil(t, insightsV6(v, now, cycles, nil, v6Today).Window)
	assert.Nil(t, contextV6(newView(st, v6Today), civildate.MustParse("2026-09-23")).Chance.Level)
}

func BenchmarkInsightsV6(b *testing.B) {
	st := v6State()
	b.ReportAllocs()
	for b.Loop() {
		v := newView(st, v6Today)
		now, cycles := chartCyclesV6(v, v6Today)
		_ = InsightsJSON(insightsV6(v, now, cycles, nil, v6Today), "fa")
	}
}
