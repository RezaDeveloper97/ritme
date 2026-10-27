package challenges

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

func phaseOn(t *testing.T, in engine.Input, today string) string {
	t.Helper()
	s := (&Service{}).WithEngine(fakeStates{st: engine.Prepare(in, engine.DefaultParams())})
	p, err := s.phase(context.Background(), 1, civildate.MustParse(today))
	require.NoError(t, err)
	return p
}

func period(s string) engine.Period {
	st := engine.DayOf(civildate.MustParse(s))
	return engine.Period{Start: st, End: st + 4}
}

func TestPhaseV6(t *testing.T) {
	in := engine.Input{Periods: []engine.Period{period("2026-06-01"), period("2026-06-30"), period("2026-07-29"), period("2026-08-27")}}
	assert.Equal(t, "menstrual", phaseOn(t, in, "2026-08-29"))
	assert.Equal(t, "fertile", phaseOn(t, in, "2026-09-09"))
	assert.Equal(t, "unknown", phaseOn(t, engine.Input{}, "2026-09-09"), "no data: the engine says unknown")

	// Manual 40-day cycle: the same day is still follicular where the 29-day median would call it fertile.
	in.Declared = engine.Declared{CycleLength: engine.Some(40), Manual: true}
	assert.NotEqual(t, "fertile", phaseOn(t, in, "2026-09-09"))

	// 15 days late: beyond the engine's cap the phase is unknown (the v1.1 loader kept predicting).
	in.Declared = engine.Declared{}
	assert.Equal(t, "unknown", phaseOn(t, in, "2026-10-10"))
}
