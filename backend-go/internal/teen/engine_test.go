package teen

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

type fakeStates struct{ st *engine.State }

func (f fakeStates) ForUser(context.Context, uint64, civildate.Date) (*engine.State, error) {
	return f.st, nil
}

func d(s string) civildate.Date { return civildate.MustParse(s) }

func period(s string) engine.Period {
	st := engine.DayOf(d(s))
	return engine.Period{Start: st, End: st + 4}
}

func weekOn(t *testing.T, in engine.Input, today string) PeriodWeek {
	t.Helper()
	s := (&Service{}).WithEngine(fakeStates{st: engine.Prepare(in, engine.DefaultParams())})
	w, err := s.nextPeriodWeekV6(context.Background(), 1, d(today))
	require.NoError(t, err)
	return w
}

var history29 = []engine.Period{period("2026-06-01"), period("2026-06-30"), period("2026-07-29"), period("2026-08-27")}

func TestNextPeriodWeekV6(t *testing.T) {
	// predicted start 2026-09-25 (a Friday); weeks start on Saturday: 09-19..09-25 is this week.
	assert.Equal(t, WeekThis, weekOn(t, engine.Input{Periods: history29}, "2026-09-22"))
	assert.Equal(t, WeekLater, weekOn(t, engine.Input{Periods: history29}, "2026-09-05"))
	assert.Equal(t, WeekUnknown, weekOn(t, engine.Input{}, "2026-09-05"))
}

func TestNextPeriodWeekV6_ManualLengths(t *testing.T) {
	in := engine.Input{
		Periods:  history29,
		Declared: engine.Declared{CycleLength: engine.Some(33), PeriodLength: engine.Some(5), Manual: true},
	}
	// Manual 33 days → 2026-09-29, the week after the 09-22 week of the history prediction.
	assert.Equal(t, WeekNext, weekOn(t, in, "2026-09-22"))
}

func TestNextPeriodWeekV6_LatenessStaysHidden(t *testing.T) {
	// 15 days after the predicted start: never «this week» again, never a hint of lateness.
	w := weekOn(t, engine.Input{Periods: history29}, "2026-10-10")
	assert.Contains(t, []PeriodWeek{WeekNext, WeekLater, WeekUnknown}, w)
	assert.NotEqual(t, WeekThis, w)
}
