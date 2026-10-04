package labor_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/pregnancy/labor"
)

var t0 = time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)

// series builds n ended contractions every interval, each lasting dur, from start.
func series(start time.Time, n int, interval, dur time.Duration) []labor.Contraction {
	out := make([]labor.Contraction, n)
	for i := range out {
		s := start.Add(time.Duration(i) * interval)
		out[i] = labor.Contraction{Start: s, End: s.Add(dur)}
	}
	return out
}

func TestFiveOneOne_MetAfterAnHour(t *testing.T) {
	p := labor.DefaultParams()
	// Every 5 minutes, 60 s each: 12 contractions span 55 min + 60 s = 56 min — not yet an hour.
	cs := series(t0, 12, 5*time.Minute, time.Minute)
	r := labor.FiveOneOne(cs, p)
	assert.False(t, r.Met)
	assert.Equal(t, 12, r.Count)
	assert.Equal(t, 56*time.Minute, r.Span)

	// The 13th makes the run span 61 minutes.
	cs = series(t0, 13, 5*time.Minute, time.Minute)
	r = labor.FiveOneOne(cs, p)
	assert.True(t, r.Met)
	assert.Equal(t, 5*time.Minute, r.AvgInterval)
	assert.Equal(t, time.Minute, r.AvgDuration)
	assert.Equal(t, cs[12].End, r.MetAt)
}

func TestFiveOneOne_UsesTheShortestTailSpanningTheRun(t *testing.T) {
	p := labor.DefaultParams()
	// An early slow phase (every 12 min) then 14 contractions every 4 min: only the tail counts.
	cs := append(series(t0, 4, 12*time.Minute, 40*time.Second), series(t0.Add(48*time.Minute), 17, 4*time.Minute, 70*time.Second)...)
	r := labor.FiveOneOne(cs, p)
	assert.True(t, r.Met)
	assert.Equal(t, 4*time.Minute, r.AvgInterval)
	assert.Equal(t, 70*time.Second, r.AvgDuration)
	assert.GreaterOrEqual(t, r.Span, time.Hour)
	assert.Less(t, r.Span, time.Hour+4*time.Minute)
}

func TestFiveOneOne_NotMet(t *testing.T) {
	p := labor.DefaultParams()
	cases := map[string][]labor.Contraction{
		"too far apart": series(t0, 12, 7*time.Minute, time.Minute),
		"too short":     series(t0, 15, 5*time.Minute, 30*time.Second),
		// 70 min of contractions, but a 15 min pause 20 min ago restarts the run.
		"pause restarts the run": append(series(t0, 10, 5*time.Minute, time.Minute), series(t0.Add(60*time.Minute), 5, 5*time.Minute, time.Minute)...),
		"one contraction":        series(t0, 1, 5*time.Minute, time.Minute),
		"none":                   nil,
	}
	for name, cs := range cases {
		assert.False(t, labor.FiveOneOne(cs, p).Met, name)
	}
}

func TestFiveOneOne_IgnoresTheContractionInProgress(t *testing.T) {
	cs := series(t0, 13, 5*time.Minute, time.Minute)
	cs[12].End = time.Time{}
	r := labor.FiveOneOne(cs, labor.DefaultParams())
	assert.False(t, r.Met)
	assert.Equal(t, 12, r.Count)
}

func TestFiveOneOne_AdminThresholds(t *testing.T) {
	// A stricter admin setting (every 4 min, 60 s) rejects the default pattern.
	p := labor.Params{IntervalMax: 4 * time.Minute, DurationMin: time.Minute, Run: time.Hour}
	assert.False(t, labor.FiveOneOne(series(t0, 14, 5*time.Minute, time.Minute), p).Met)
	// A shorter run (30 min) is met sooner.
	p = labor.Params{IntervalMax: 5 * time.Minute, DurationMin: 45 * time.Second, Run: 30 * time.Minute}
	assert.True(t, labor.FiveOneOne(series(t0, 7, 5*time.Minute, time.Minute), p).Met)
}

func TestRecentAndIntervals(t *testing.T) {
	// The artboard: 22:28 (0:47), 22:35 (0:50), 22:41 (0:55) → intervals —, 7:00 → 6:20 on the board is rounded data.
	cs := []labor.Contraction{
		{Start: t0, End: t0.Add(47 * time.Second)},
		{Start: t0.Add(6*time.Minute + 20*time.Second), End: t0.Add(6*time.Minute + 70*time.Second)},
		{Start: t0.Add(12*time.Minute + 25*time.Second), End: t0.Add(12*time.Minute + 80*time.Second)},
		{Start: t0.Add(20 * time.Minute)}, // in progress
	}
	iv := labor.Intervals(cs[:3])
	assert.Equal(t, []time.Duration{0, 6*time.Minute + 20*time.Second, 6*time.Minute + 5*time.Second}, iv)
	s := labor.Recent(cs, time.Hour)
	assert.Equal(t, 3, s.Count)
	assert.Equal(t, (47+50+55)*time.Second/3, s.AvgDuration)
	assert.Equal(t, (12*time.Minute+25*time.Second)/2, s.AvgInterval)
	// Only the last hour counts.
	s = labor.Recent(append(series(t0.Add(-3*time.Hour), 2, time.Minute, 2*time.Minute), cs...), time.Hour)
	assert.Equal(t, 3, s.Count)
	assert.Equal(t, labor.Stats{}, labor.Recent(nil, time.Hour))
}

func TestClock(t *testing.T) {
	assert.Equal(t, "0:48", labor.Clock(48*time.Second))
	assert.Equal(t, "6:10", labor.Clock(6*time.Minute+10*time.Second))
	assert.Equal(t, "1:02:03", labor.Clock(time.Hour+2*time.Minute+3*time.Second))
	assert.Equal(t, "0:00", labor.Clock(-time.Second))
}
