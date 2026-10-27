package shared

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// fakeStates serves one prepared engine state (the real provider needs a database).
type fakeStates struct{ st *engine.State }

func (f fakeStates) ForUser(context.Context, uint64, civildate.Date) (*engine.State, error) {
	return f.st, nil
}

func day(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }

// period is a closed 5-day period starting on s.
func period(s string) engine.Period {
	return engine.Period{Start: day(s), End: day(s) + 4}
}

func viewOn(t *testing.T, in engine.Input, today string) *jsonx.OrderedMap {
	t.Helper()
	r := (&Reader{}).WithEngine(fakeStates{st: engine.Prepare(in, engine.DefaultParams())})
	v, err := r.cycleViewV6(context.Background(), 1, civildate.MustParse(today))
	require.NoError(t, err)
	return v.(*jsonx.OrderedMap)
}

func get(m *jsonx.OrderedMap, k string) any { v, _ := m.Get(k); return v }

// Today's v1.1 view and the v6 view carry exactly the same keys: the engine port never widens what a partner sees.
var viewKeys = []string{"date", "has_data", "cycle_day", "cycle_length", "main_phase", "days_to_period", "predicted_next_period_start", "confidence"}

var history29 = []engine.Period{period("2026-06-01"), period("2026-06-30"), period("2026-07-29"), period("2026-08-27")}

func TestCycleViewV6_SameKeysAsV1(t *testing.T) {
	v := viewOn(t, engine.Input{Periods: history29}, "2026-09-05")
	assert.Equal(t, viewKeys, v.Keys())
	assert.Equal(t, true, get(v, "has_data"))
	assert.Equal(t, 10, get(v, "cycle_day"))
	assert.Equal(t, 29, get(v, "cycle_length"))
	assert.Equal(t, civildate.MustParse("2026-09-25"), get(v, "predicted_next_period_start"))
}

func TestCycleViewV6_ManualLengthsAreHonoured(t *testing.T) {
	v := viewOn(t, engine.Input{
		Periods:  history29,
		Declared: engine.Declared{CycleLength: engine.Some(33), PeriodLength: engine.Some(6), Manual: true},
	}, "2026-09-05")
	assert.Equal(t, 33, get(v, "cycle_length"), "E-20: the manual value wins over the history median")
	assert.Equal(t, civildate.MustParse("2026-09-29"), get(v, "predicted_next_period_start"))
}

func TestCycleViewV6_LatenessIsNeverShown(t *testing.T) {
	// 2026-09-25 is the predicted start; 15 days later is past the 14-day cap.
	for _, today := range []string{"2026-09-27", "2026-10-10"} {
		v := viewOn(t, engine.Input{Periods: history29}, today)
		assert.Nil(t, get(v, "cycle_day"), today)
		assert.Nil(t, get(v, "days_to_period"), today)
		assert.Nil(t, get(v, "predicted_next_period_start"), today)
		assert.Equal(t, viewKeys, v.Keys())
	}
}

func TestCycleViewV6_PillNeverShowsAFertilePhase(t *testing.T) {
	in := engine.Input{Periods: history29}
	assert.Equal(t, "fertile", string(get(viewOn(t, in, "2026-09-09"), "main_phase").(enums.MainPhase)))
	in.Hormonal = true
	for d := day("2026-09-01"); d < day("2026-09-26"); d++ {
		v := viewOn(t, in, d.String())
		assert.NotEqual(t, "fertile", string(get(v, "main_phase").(enums.MainPhase)), d.String())
		assert.Equal(t, viewKeys, v.Keys(), "the suppression flag itself is not a key")
	}
}

func TestCycleViewV6_NoData(t *testing.T) {
	v := viewOn(t, engine.Input{}, "2026-09-05")
	assert.Equal(t, false, get(v, "has_data"))
	assert.Nil(t, get(v, "predicted_next_period_start"))
}
