package benchfixtures_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/enums"
)

func TestPeriods_DeterministicPrefixAndShape(t *testing.T) {
	for _, h := range benchfixtures.Sizes {
		ps := benchfixtures.Periods(h)
		require.Len(t, ps, h)
		assert.Equal(t, ps, benchfixtures.Periods(h), "same input, same fixture")
		assert.Equal(t, benchfixtures.Today.AddDays(-9), ps[0].Start, "cycle day 10 today")
		assert.Equal(t, 0, ps[0].Length, "current cycle")
		for i := 1; i < h; i++ {
			assert.True(t, ps[i].Start.Before(ps[i-1].Start), "newest first")
			assert.Equal(t, ps[i].Start.DiffDays(ps[i-1].Start), ps[i].Length)
			assert.GreaterOrEqual(t, ps[i].Length, 26)
			assert.LessOrEqual(t, ps[i].Length, 38)
			assert.True(t, ps[i].End().Before(ps[i-1].Start), "closed before the next start")
		}
	}
	assert.Equal(t, benchfixtures.Periods(6), benchfixtures.Periods(24)[:6], "a prefix of the larger history")
	assert.Len(t, benchfixtures.Histories(60), 60)
	assert.Len(t, benchfixtures.HistoryRows(1, 120), 120)
}

func TestDays_CoverTheYearWithABiphasicCurveAndLH(t *testing.T) {
	days := benchfixtures.Days(benchfixtures.LogDays)
	require.Len(t, days, benchfixtures.LogDays)
	assert.Equal(t, benchfixtures.Today, days[len(days)-1].Date)
	assert.Equal(t, 10, days[len(days)-1].CycleDay)
	var low, high, n, positives int
	for _, d := range days {
		assert.NotEmpty(t, d.Entries)
		if d.LH == "positive" {
			positives++
		}
		if d.BBT == 0 {
			continue
		}
		n++
		if d.BBT < 3660 {
			low++
		} else {
			high++
		}
	}
	assert.Greater(t, n, 280, "about six days in seven measured")
	assert.Positive(t, low)
	assert.Positive(t, high)
	assert.GreaterOrEqual(t, positives, 10, "one positive LH per cycle")
	assert.Equal(t, "36.40", benchfixtures.Day{BBT: 3640}.BBTString())
	assert.Equal(t, "36.05", benchfixtures.Day{BBT: 3605}.BBTString())
	assert.Len(t, benchfixtures.DailyLogRows(1, days), benchfixtures.LogDays)
}

func TestUsers_OnePerMode(t *testing.T) {
	seen := map[enums.LifeMode]bool{}
	for _, u := range benchfixtures.Users() {
		assert.False(t, seen[u.Mode], u.Mode)
		seen[u.Mode] = true
		assert.True(t, u.Mode.IsValid(), u.Mode)
	}
	p, ok := benchfixtures.UserFor(enums.LifeModePregnancy)
	require.True(t, ok)
	assert.False(t, p.LMP.IsZero())
}
