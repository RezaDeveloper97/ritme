package analysis

import (
	"math"
	"sort"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Association statistics of the correlations screen. Both are effect sizes on the contingency table
// of logged days (no p-value: the screen describes the user's own data, it does not test a hypothesis):
//
//   - two binary variables (short sleep × irritable mood, exercise × severe cramps): the phi
//     coefficient φ = (ad − bc) / √((a+b)(c+d)(a+c)(b+d)), signed;
//   - a binary outcome across the four phases (high energy × phase, good mood × phase): Cramér's V =
//     √(χ² / (n·(min(r, c) − 1))) = √(χ² / n) for a 2 × k table, unsigned.
//
// Strength uses Cohen's conventions for both: |r| ≥ 0.5 strong, ≥ 0.3 medium, ≥ 0.1 weak, below that
// no association (not shown as a finding).
const (
	StrengthStrong = "strong"
	StrengthMedium = "medium"
	StrengthWeak   = "weak"

	strongMin = 0.5
	mediumMin = 0.3
	weakMin   = 0.1
)

// Minimum data for an association: at least CorrMinDays paired days, every compared group (exposed /
// unexposed, or each phase) at least CorrMinGroupDays days, and the outcome logged at least
// CorrMinOutcome times (a table with a handful of hits says nothing).
const (
	CorrMinDays      = 20
	CorrMinGroupDays = 5
	CorrMinOutcome   = 3
)

// Statistic names in the payload.
const (
	StatPhi      = "phi"
	StatCramersV = "cramers_v"
)

// Strength classifies |r| ("" = no association).
func Strength(r float64) string {
	a := math.Abs(r)
	switch {
	case a >= strongMin:
		return StrengthStrong
	case a >= mediumMin:
		return StrengthMedium
	case a >= weakMin:
		return StrengthWeak
	}
	return ""
}

// Phi is the phi coefficient of the 2 × 2 table [[a, b], [c, d]] (rows exposed / unexposed, columns
// outcome / no outcome); 0 when a margin is empty.
func Phi(a, b, c, d int) float64 {
	den := float64(a+b) * float64(c+d) * float64(a+c) * float64(b+d)
	if den == 0 {
		return 0
	}
	return (float64(a)*float64(d) - float64(b)*float64(c)) / math.Sqrt(den)
}

// CramersV of a 2 × k table given per group (hits, total); 0 when degenerate.
func CramersV(hits, totals []int) float64 {
	n, h := 0, 0
	for i := range totals {
		n += totals[i]
		h += hits[i]
	}
	if n == 0 || h == 0 || h == n {
		return 0
	}
	chi := 0.0
	for i := range totals {
		if totals[i] == 0 {
			continue
		}
		for _, cell := range [2][2]float64{
			{float64(hits[i]), float64(totals[i]) * float64(h) / float64(n)},
			{float64(totals[i] - hits[i]), float64(totals[i]) * float64(n-h) / float64(n)},
		} {
			chi += (cell[0] - cell[1]) * (cell[0] - cell[1]) / cell[1]
		}
	}
	return math.Sqrt(chi / float64(n))
}

// round is PHP round() to places.
func round(f float64, places int) float64 { return phpround.Round(f, places) }

func roundPtr(f *float64, places int) *float64 {
	if f == nil {
		return nil
	}
	v := round(*f, places)
	return &v
}

func mean(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	s := 0.0
	for _, v := range values {
		s += v
	}
	return s / float64(len(values)), true
}

func meanPtr(values []float64) *float64 {
	if m, ok := mean(values); ok {
		return &m
	}
	return nil
}

// MovingAverage7 is the trailing 7-day moving average of a daily series at day d: the mean of the
// values logged on d−6 … d (days without a value are skipped, not counted as zero). ok is false when
// none of the 7 days has a value. It damps day-to-day water-weight swings (e.g. before a period).
func MovingAverage7(values map[civildate.Date]float64, d civildate.Date) (float64, bool) {
	var window []float64
	for i := 6; i >= 0; i-- {
		if v, ok := values[d.AddDays(-i)]; ok {
			window = append(window, v)
		}
	}
	return mean(window)
}

// sortedDates returns the keys of a date set in order.
func sortedDates[V any](m map[civildate.Date]V) []civildate.Date {
	out := make([]civildate.Date, 0, len(m))
	for d := range m {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func intPtr(v int) *int { return &v }

func ptrOrNil[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
