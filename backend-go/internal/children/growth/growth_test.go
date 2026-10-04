package growth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/seeds/who"
)

// Reference values copied from the official WHO "percentiles expanded tables" (P3 / P50 / P97 columns) and "z-score
// expanded tables" (SD columns), https://www.who.int/tools/child-growth-standards/standards — files
// wfa-girls-percentiles-expanded-tables.xlsx, wfa-boys-percentiles-expanded-tables.xlsx,
// hcfa-girls-percentiles-expanded-tables.xlsx, lhfa-boys-percentiles-expanded-tables.xlsx and
// wfa-girls-zscore-expanded-tables.xlsx. WHO prints them to 3 decimals, so they are matched to ±0.0006 and the
// percentile of a printed value to ±0.05.
var percentileGoldens = []struct {
	ind          who.Indicator
	sex          who.Sex
	day          int
	p3, p50, p97 float64
}{
	{who.Weight, who.Girl, 0, 2.440, 3.232, 4.166},
	{who.Weight, who.Girl, 102, 4.786, 6.062, 7.668},
	{who.Weight, who.Girl, 365, 7.140, 8.946, 11.331},
	{who.Weight, who.Girl, 1095, 10.957, 13.846, 17.830},
	{who.Weight, who.Girl, 1856, 14.079, 18.389, 24.697},
	{who.Weight, who.Boy, 0, 2.507, 3.346, 4.350},
	{who.Weight, who.Boy, 102, 5.299, 6.613, 8.187},
	{who.Weight, who.Boy, 365, 7.844, 9.646, 11.830},
	{who.Weight, who.Boy, 1095, 11.437, 14.339, 18.041},
	{who.Weight, who.Boy, 1856, 14.398, 18.497, 24.002},
	{who.Head, who.Girl, 0, 31.651, 33.879, 36.106},
	{who.Head, who.Girl, 102, 37.572, 39.923, 42.275},
	{who.Head, who.Girl, 365, 42.338, 44.894, 47.450},
	{who.Head, who.Girl, 1856, 47.289, 49.965, 52.642},
	{who.Length, who.Boy, 0, 46.324, 49.884, 53.445},
	{who.Length, who.Boy, 102, 58.479, 62.350, 66.221},
	{who.Length, who.Boy, 365, 71.270, 75.739, 80.208},
	{who.Length, who.Boy, 1095, 89.097, 96.067, 103.038},
	{who.Length, who.Boy, 1856, 101.714, 110.497, 119.279},
}

func TestCurvesMatchWHOPercentileTables(t *testing.T) {
	for _, g := range percentileGoldens {
		p, ok := who.At(g.ind, g.sex, g.day)
		require.True(t, ok)
		assert.InDelta(t, g.p3, ValueAt(g.ind, p, ZP3), 0.0006, "%s %s day %d P3", g.ind, g.sex, g.day)
		assert.InDelta(t, g.p50, ValueAt(g.ind, p, 0), 0.0006, "%s %s day %d P50", g.ind, g.sex, g.day)
		assert.InDelta(t, g.p97, ValueAt(g.ind, p, ZP97), 0.0006, "%s %s day %d P97", g.ind, g.sex, g.day)
	}
}

func TestAssessReturnsWHOPercentiles(t *testing.T) {
	for _, g := range percentileGoldens {
		for _, c := range []struct {
			y, want float64
		}{{g.p3, 3}, {g.p50, 50}, {g.p97, 97}} {
			r, ok := Assess(g.ind, g.sex, g.day, c.y)
			require.True(t, ok)
			assert.InDelta(t, c.want, r.Percentile, 0.05, "%s %s day %d y %.3f", g.ind, g.sex, g.day, c.y)
			assert.True(t, r.InBand || c.want != 50)
		}
	}
}

// wfa-girls-zscore-expanded-tables.xlsx, day 365: SD4neg … SD4. The ±4 SD columns are WHO's restricted extension,
// so they pin the beyond-3-SD rule.
func TestWeightZScoresIncludingRestrictedTails(t *testing.T) {
	sd := []struct{ y, z float64 }{
		{5.505, -4}, {6.273, -3}, {7.041, -2}, {7.925, -1}, {8.946, 0},
		{10.129, 1}, {11.506, 2}, {13.114, 3}, {14.721, 4},
	}
	p, ok := who.At(who.Weight, who.Girl, 365)
	require.True(t, ok)
	for _, c := range sd {
		assert.InDelta(t, c.z, Z(who.Weight, p, c.y), 0.002, "y %.3f", c.y)
		assert.InDelta(t, c.y, ValueAt(who.Weight, p, c.z), 0.0006, "z %.0f", c.z)
	}
}

func TestAssessBandAndRange(t *testing.T) {
	r, ok := Assess(who.Weight, who.Girl, 102, 6.1) // artboard: Ava 3.4 months, 6.1 kg ≈ P52
	require.True(t, ok)
	assert.True(t, r.InBand)
	assert.InDelta(t, 52, r.Percentile, 1)

	r, ok = Assess(who.Weight, who.Girl, 102, 4.0)
	require.True(t, ok)
	assert.False(t, r.InBand)
	assert.Less(t, r.Percentile, 3.0)

	_, ok = Assess(who.Weight, who.Girl, who.MaxDay+1, 20)
	assert.False(t, ok, "beyond 5 years")
	_, ok = Assess(who.Weight, who.Girl, -1, 3)
	assert.False(t, ok)
	_, ok = Assess(who.Weight, who.Sex("x"), 10, 3)
	assert.False(t, ok)
	_, ok = Assess(who.Head, who.Boy, 10, 0)
	assert.False(t, ok)
}

func TestPercentile(t *testing.T) {
	assert.InDelta(t, 50, Percentile(0), 1e-9)
	assert.InDelta(t, 3, Percentile(ZP3), 1e-6)
	assert.InDelta(t, 97, Percentile(ZP97), 1e-6)
	assert.InDelta(t, 15, Percentile(ZP15), 1e-6)
	assert.InDelta(t, 85, Percentile(ZP85), 1e-6)
	assert.InDelta(t, 2.275, Percentile(-2), 0.001)
}
