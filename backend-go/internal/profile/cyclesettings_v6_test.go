package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func dayOf(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

func per(s string, length int) engine.Period {
	return engine.Period{Start: dayOf(s), End: dayOf(s) + engine.Day(length-1)}
}

func TestEngineMetrics_ManualLengthsAndTheNewestPeriod(t *testing.T) {
	// The newest period comes from the bleeding log (source bleeding_log): the v1.1 metrics skipped it, so the
	// last cycle (09-04 − 08-06 = 29) did not count; the engine counts every real period.
	ps := []engine.Period{per("2026-06-01", 5), per("2026-06-30", 5), per("2026-07-29", 5), per("2026-08-27", 5)}
	ps[3].Source = engine.PeriodBleedingLog
	es := engine.Prepare(engine.Input{
		Periods:  ps,
		Declared: engine.Declared{CycleLength: engine.Some(33), PeriodLength: engine.Some(6), Manual: true},
	}, engine.DefaultParams())
	m := engineMetrics(es)

	assert.Equal(t, 33, m.EffectiveCycleLength, "manual wins")
	assert.Equal(t, 6, m.EffectivePeriodDuration)
	require.NotNil(t, m.CalculatedCycleLength)
	assert.Equal(t, 29, *m.CalculatedCycleLength, "the calculated value ignores the switch")
	assert.Equal(t, 3, m.ValidCyclesCount, "all three gaps count, including the one to the newest bleeding-log period")
	require.NotNil(t, m.ProfileCycleLength)
	assert.Equal(t, 33, *m.ProfileCycleLength)
	require.NotNil(t, m.CalculatedPeriodDuration)
	assert.Equal(t, 5, *m.CalculatedPeriodDuration)

	// PMS window next − 3 .. next − 1: the reminder fires on cycle day length − 2.
	assert.Equal(t, 31, pmsCycleDay(33, es.Params().PMSDays))
	assert.Equal(t, 0, pmsCycleDay(2, 3))
}

func TestEngineMetrics_NoHistoryHasNoCalculatedValues(t *testing.T) {
	m := engineMetrics(engine.Prepare(engine.Input{}, engine.DefaultParams()))
	assert.Nil(t, m.CalculatedCycleLength)
	assert.Nil(t, m.ProfileCycleLength)
	assert.Equal(t, 0, m.ValidCyclesCount)
	assert.Positive(t, m.EffectiveCycleLength, "the engine default")
}
