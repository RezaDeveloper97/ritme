package labs

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Marker statuses against a reference range.
const (
	StateLow            = "low"
	StateBorderlineLow  = "borderline_low"
	StateNormal         = "normal"
	StateBorderlineHigh = "borderline_high"
	StateHigh           = "high"
	StateUnknown        = "unknown" // no number or no range
)

// Range sources.
const (
	RangeSheet   = "sheet"   // printed on the lab sheet (preferred: labs differ)
	RangeTypical = "typical" // the catalog's typical adult range (sheet had none, units match)
)

// BorderlineMargin: a value outside the range by at most this fraction of the bound is «مرزی» (borderline) —
// nbl_Lab_Result: hemoglobin 11.2 against 12–15.5 is «پایین مرزی», ferritin 9 against 15–150 is «پایین».
const BorderlineMargin = 0.10

// Range is a reference range; either bound may be missing («≥ 20», «< 200»).
type Range struct {
	Low, High *float64
	Text      string
	Source    string // RangeSheet | RangeTypical | ""
}

// Empty reports whether the range has no bound.
func (r Range) Empty() bool { return r.Low == nil && r.High == nil }

// Classify is the status of v against r.
func Classify(v *float64, r Range) string {
	if v == nil || r.Empty() {
		return StateUnknown
	}
	x := *v
	if r.Low != nil && x < *r.Low {
		if *r.Low > 0 && x >= *r.Low*(1-BorderlineMargin) {
			return StateBorderlineLow
		}
		return StateLow
	}
	if r.High != nil && x > *r.High {
		if *r.High > 0 && x <= *r.High*(1+BorderlineMargin) {
			return StateBorderlineHigh
		}
		return StateHigh
	}
	return StateNormal
}

// Attention reports whether a status is outside the range (borderline included).
func Attention(state string) bool { return state != StateNormal && state != StateUnknown }

// lowSide reports whether a status is below the range.
func lowSide(state string) bool { return state == StateLow || state == StateBorderlineLow }

// Trend directions.
const (
	TrendRising  = "rising"
	TrendFalling = "falling"
	TrendStable  = "stable"
)

// TrendWindow is how many recent values the direction looks at; TrendThreshold the relative change that counts.
const (
	TrendWindow    = 3
	TrendThreshold = 0.05
)

// Direction is the trend of the last TrendWindow values (oldest first): rising / falling when the last differs from
// the first by more than TrendThreshold of it, else stable; "" for fewer than 2 values.
func Direction(values []float64) string {
	if len(values) < 2 {
		return ""
	}
	w := values[max(0, len(values)-TrendWindow):]
	first, last := w[0], w[len(w)-1]
	base := math.Abs(first)
	if base == 0 {
		base = 1
	}
	switch d := (last - first) / base; {
	case d > TrendThreshold:
		return TrendRising
	case d < -TrendThreshold:
		return TrendFalling
	}
	return TrendStable
}

// NormalizeUnit makes units comparable: lower case, no spaces, µ/μ/mc → u, powers flattened («×10³/µL», «10^3/uL»,
// «10*3/mm3» → «103/ul»).
func NormalizeUnit(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	u = strings.NewReplacer(" ", "", "µ", "u", "μ", "u", "mcg", "ug", "mcl", "ul", "mm3", "ul", "×", "", "x10", "10",
		"^", "", "*", "", "³", "3", "⁶", "6", "e3", "3", "e6", "6").Replace(u)
	return u
}

// UnitsMatch reports whether two units are the same after NormalizeUnit (empty never matches).
func UnitsMatch(a, b string) bool {
	na, nb := NormalizeUnit(a), NormalizeUnit(b)
	return na != "" && na == nb
}

// normalizeName is the matching form of a printed marker name: lower case, Persian / Arabic digits as ASCII,
// Arabic ي/ك as Persian ی/ک, every character that is not a letter or digit dropped («Vitamin D (25-OH)» →
// «vitamind25oh»).
func normalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == 'ي':
			r = 'ی'
		case r == 'ك':
			r = 'ک'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// parseNumber reads a decimal written with ASCII or Persian digits and . / ٫ / , as the separator.
func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r == '٫' || r == ',':
			b.WriteRune('.')
		default:
			b.WriteRune(r)
		}
	}
	f, err := strconv.ParseFloat(b.String(), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// decimalString is f as a decimal(14,4) column value.
func decimalString(f float64) string { return strconv.FormatFloat(math.Round(f*1e4)/1e4, 'f', 4, 64) }

// number is a decimal column value as a JSON number (trailing zeros dropped), nil when NULL.
func number(s string, valid bool) *float64 {
	if !valid {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
}
