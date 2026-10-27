package healthlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-37 (D-136): under the v6 engine the luteal-spotting warning reads the engine phase of the log day (luteal, at
// least 3 days before the predicted start — v1's cycle_duration − 2 margin) instead of the profile LMP and the
// declared length.
func TestLutealSpottingV6(t *testing.T) {
	today := civildate.MustParse("2026-10-01")
	day := func(s string) engine.Day { return engine.DayOf(civildate.MustParse(s)) }
	// Two logged periods, 29-day cycle: next 2026-09-30, ovulation ≈ 09-16, luteal 09-17 … 09-29.
	in := engine.Input{Periods: []engine.Period{
		{Start: day("2026-08-03"), End: day("2026-08-07")},
		{Start: day("2026-09-01"), End: day("2026-09-05")},
	}}
	st := engine.Prepare(in, engineparams.For(today))
	logOn := func(date string, spotting bool, bleeding string) *model.DailyHealthLog {
		l := model.FromRow(store.DailyHealthLog{LogDate: civildate.MustParse(date)})
		if spotting {
			require.NoError(t, l.Set("spotting", true))
		}
		if bleeding != "" {
			require.NoError(t, l.Set("bleeding_intensity", bleeding))
		}
		return l
	}
	for _, c := range []struct {
		date     string
		spotting bool
		bleeding string
		want     bool
	}{
		{"2026-09-15", true, "", false},    // fertile window (v1's "day 15" rule would warn)
		{"2026-09-17", true, "", true},     // luteal
		{"2026-09-27", false, "low", true}, // 3 days before the predicted start
		{"2026-09-28", true, "", false},    // 2 days before: likely the period itself
		{"2026-09-20", false, "medium", false},
		{"2026-09-03", true, "", false}, // menstrual
	} {
		assert.Equal(t, c.want, IsLikelyLutealSpottingV6(st, logOn(c.date, c.spotting, c.bleeding), today), c.date)
	}

	in.Hormonal = true // the pill: no natural luteal phase
	assert.False(t, IsLikelyLutealSpottingV6(engine.Prepare(in, engineparams.For(today)), logOn("2026-09-20", true, ""), today))
	assert.False(t, IsLikelyLutealSpottingV6(engine.Prepare(engine.Input{}, engineparams.For(today)), logOn("2026-09-20", true, ""), today))
}
