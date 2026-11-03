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
// Strength is not read from r: §4.1 grades an association by the group-rate ratio (RatioStrength, A-02); φ and V
// stay in the payload as `value` (φ's sign gives the direction).
const (
	StrengthStrong = "strong"
	StrengthMedium = "medium"
	StrengthWeak   = "weak"
)

// Statistic names in the payload.
const (
	StatPhi      = "phi"
	StatCramersV = "cramers_v"
)

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
	var buf [7]float64 // on the stack: called once per day of the range (T-M9-49c)
	n := 0
	for i := 6; i >= 0; i-- {
		if v, ok := values[d.AddDays(-i)]; ok {
			buf[n] = v
			n++
		}
	}
	return mean(buf[:n])
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
