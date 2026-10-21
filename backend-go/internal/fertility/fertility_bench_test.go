package fertility

import (
	"strconv"
	"testing"

	"github.com/ritme/backend-go/internal/benchfixtures"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M9-08 baseline: the pure part of GET /fertility/insights and GET /fertility/bbt over a TTC user with
// H = 6 / 24 / 60 / 120 periods and a year of daily BBT + LH. Both scan up to six cycles day by day through the
// cycle resolver (fertileWindow), which dominates.

func benchInputs(h int) cycleInputs {
	hist, profile := benchfixtures.Histories(h), benchfixtures.Profile()
	return cycleInputs{histories: hist, profile: profile, metrics: metrics.Calculate(hist, profile)}
}

// benchReadings / benchLH are the year's BBT readings and LH tests from `from`, oldest first (what the two
// queries return).
func benchReadings(from civildate.Date) ([]bbt.Reading, []LHTest) {
	var readings []bbt.Reading
	var tests []LHTest
	for _, d := range benchfixtures.Days(benchfixtures.LogDays) {
		if d.Date.Before(from) {
			continue
		}
		if d.BBT != 0 {
			readings = append(readings, bbt.Reading{Date: d.Date, Value: d.BBT})
		}
		if d.LH != "" {
			tests = append(tests, LHTest{Date: d.Date, Value: d.LH})
		}
	}
	return readings, tests
}

func BenchmarkInsights(b *testing.B) {
	today := benchfixtures.Today
	for _, h := range benchfixtures.Sizes {
		b.Run("H="+strconv.Itoa(h), func(b *testing.B) {
			in := benchInputs(h)
			cycles := in.bbtCycles(today)
			readings, tests := benchReadings(cycles[len(cycles)-1].Start)
			b.ReportAllocs()
			for b.Loop() {
				cs := in.bbtCycles(today)
				_ = InsightsJSON(in.insights(cs, readings, tests, today), "fa")
			}
		})
	}
}

func BenchmarkBBTChart(b *testing.B) {
	today := benchfixtures.Today
	for _, rangeSize := range BBTRanges {
		b.Run("range="+strconv.Itoa(rangeSize), func(b *testing.B) {
			in := benchInputs(24)
			cycles := in.bbtCycles(today)
			readings, _ := benchReadings(cycles[len(cycles)-1].Start)
			b.ReportAllocs()
			for b.Loop() {
				cs := in.bbtCycles(today)
				_ = BBTJSON(bbt.Build(cs, readings, today, rangeSize), rangeSize, "fa")
			}
		})
	}
}
