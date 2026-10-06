package vitals

import (
	"math"
	"sort"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// Report ranges (nbl_Vitals_BPReport «۷ روز | ۳۰ روز | ۳ ماه», nbl_Vitals_GlucoseReport «۱۴ روز اخیر»).
var RangeDays = map[string]int{"7d": 7, "14d": 14, "30d": 30, "90d": 90}

// RangeKeys is the accepted order.
var RangeKeys = []string{"7d", "14d", "30d", "90d"}

// DefaultRange per type.
var DefaultRange = map[string]string{TypeBP: "7d", TypeGlucose: "14d", TypeHR: "7d"}

// GlucoseFilterAll is the «همه» glucose filter.
const GlucoseFilterAll = "all"

// Range is a closed day range [From, To].
type Range struct {
	Key      string
	From, To civildate.Date
}

// NewRange is the range key ending today.
func NewRange(key string, today civildate.Date) Range {
	return Range{Key: key, From: today.AddDays(1 - RangeDays[key]), To: today}
}

// Days is the number of days in the range.
func (r Range) Days() int { return r.From.DiffDays(r.To) + 1 }

// Contains reports whether d is in the range.
func (r Range) Contains(d civildate.Date) bool { return !d.Before(r.From) && !d.After(r.To) }

// JSON is {key, from, to, days}.
func (r Range) JSON() *jsonx.OrderedMap {
	return jsonx.Obj("key", r.Key, "from", r.From.String(), "to", r.To.String(), "days", r.Days())
}

func avg(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func whole(x float64) int { return int(phpround.Round(x, 0)) }

func percent(n, total int) int {
	if total == 0 {
		return 0
	}
	return whole(100 * float64(n) / float64(total))
}

// distribution counts classes in order: [{code, tone, count, percent}] over the classified readings.
func distribution(typ string, classes []string, rs []Reading) []any {
	counts := map[string]int{}
	total := 0
	for _, r := range rs {
		if c := r.Class(); c != "" {
			counts[c]++
			total++
		}
	}
	out := make([]any, 0, len(classes))
	for _, c := range classes {
		out = append(out, jsonx.Obj("code", c, "tone", Tone(typ, c), "count", counts[c], "percent", percent(counts[c], total)))
	}
	return out
}

// value is the number a reading is ranked by (BP: systolic, then diastolic as a tie-break).
func value(r Reading) (float64, float64) {
	switch r.Type {
	case TypeBP:
		return r.Systolic, r.Diastolic
	case TypeGlucose:
		return r.MgDl, 0
	}
	return r.Pulse, 0
}

// extremes are the lowest and highest readings.
func extremes(rs []Reading) (lo, hi *Reading) {
	sorted := append([]Reading{}, rs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a1, a2 := value(sorted[i])
		b1, b2 := value(sorted[j])
		if a1 != b1 {
			return a1 < b1
		}
		return a2 < b2
	})
	if len(sorted) == 0 {
		return nil, nil
	}
	first, last := sorted[0], sorted[len(sorted)-1]
	return &first, &last
}

func readingOrNull(r *Reading) any {
	if r == nil {
		return nil
	}
	return ReadingJSON(*r)
}

// bpAverage is {systolic, diastolic, readings, classification} or null.
func bpAverage(typ string, rs []Reading) any {
	if len(rs) == 0 {
		return nil
	}
	var sys, dia []float64
	for _, r := range rs {
		sys, dia = append(sys, r.Systolic), append(dia, r.Diastolic)
	}
	s, d := whole(avg(sys)), whole(avg(dia))
	return jsonx.Obj("systolic", s, "diastolic", d, "readings", len(rs),
		"classification", ClassJSON(typ, ClassifyBP(float64(s), float64(d))))
}

func glucoseAverage(rs []Reading) any {
	if len(rs) == 0 {
		return nil
	}
	var xs []float64
	for _, r := range rs {
		xs = append(xs, r.MgDl)
	}
	m := avg(xs)
	return jsonx.Obj("mg_dl", MgDlRounded(m), "mmol_l", MmolL(m), "readings", len(rs))
}

func hrAverage(rs []Reading) any {
	if len(rs) == 0 {
		return nil
	}
	var xs []float64
	for _, r := range rs {
		xs = append(xs, r.Pulse)
	}
	return jsonx.Obj("bpm", whole(avg(xs)), "readings", len(rs))
}

func average(typ string, rs []Reading) any {
	switch typ {
	case TypeBP:
		return bpAverage(typ, rs)
	case TypeGlucose:
		return glucoseAverage(rs)
	}
	return hrAverage(rs)
}

// morningVsNight compares timed morning (04–12) and night (18–04) readings; above_normal counts the night readings
// outside the normal / in-range class («۲ ثبت شبانه بالاتر از طبیعی بوده»).
func morningVsNight(typ string, rs []Reading) *jsonx.OrderedMap {
	var morning, night []Reading
	above := 0
	for _, r := range rs {
		switch r.Period() {
		case PeriodMorning:
			morning = append(morning, r)
		case PeriodNight:
			night = append(night, r)
			if c := r.Class(); c != "" && c != ClassNormal && c != ClassInRange {
				above++
			}
		}
	}
	return jsonx.Obj("morning", average(typ, morning), "night", average(typ, night), "night_out_of_range", above)
}

// series is one point per day (ascending) with the day's average.
func series(typ string, rs []Reading, glucoseSplit bool) []any {
	byDay := map[civildate.Date][]Reading{}
	var days []civildate.Date
	for _, r := range rs {
		if _, ok := byDay[r.Date]; !ok {
			days = append(days, r.Date)
		}
		byDay[r.Date] = append(byDay[r.Date], r)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	out := make([]any, 0, len(days))
	for _, d := range days {
		day := byDay[d]
		p := jsonx.Obj("date", d.String(), "readings", len(day))
		switch typ {
		case TypeBP:
			a := bpAverage(typ, day).(*jsonx.OrderedMap)
			s, _ := a.Get("systolic")
			di, _ := a.Get("diastolic")
			p.Set("systolic", s)
			p.Set("diastolic", di)
		case TypeGlucose:
			if glucoseSplit {
				groups := map[string][]float64{}
				for _, r := range day {
					g := "other"
					if r.Context == ContextFasting || r.Context == ContextAfterMeal {
						g = r.Context
					}
					groups[g] = append(groups[g], r.MgDl)
				}
				for _, g := range []string{ContextFasting, ContextAfterMeal, "other"} {
					if xs := groups[g]; len(xs) > 0 {
						p.Set(g, MgDlRounded(avg(xs)))
					} else {
						p.Set(g, nil)
					}
				}
			} else {
				var xs []float64
				for _, r := range day {
					xs = append(xs, r.MgDl)
				}
				p.Set("mg_dl", MgDlRounded(avg(xs)))
			}
		case TypeHR:
			var xs []float64
			for _, r := range day {
				xs = append(xs, r.Pulse)
			}
			p.Set("bpm", whole(avg(xs)))
		}
		out = append(out, p)
	}
	return out
}

// Report is GET /vitals/reports/{type}: averages, extremes, class distribution, morning vs night, time in range and
// the daily series of rs (one type, already limited to rng; any order). filter narrows glucose by context ("all").
func Report(typ string, rs []Reading, rng Range, filter string) *jsonx.OrderedMap {
	var in []Reading
	for _, r := range rs {
		if r.Type != typ || !rng.Contains(r.Date) {
			continue
		}
		if typ == TypeGlucose && filter != "" && filter != GlucoseFilterAll && r.Context != filter {
			continue
		}
		in = append(in, r)
	}
	SortNewestFirst(in)
	lo, hi := extremes(in)
	out := jsonx.Obj("type", typ, "range", rng.JSON())
	switch typ {
	case TypeBP:
		normal := 0
		for _, r := range in {
			if r.Class() == ClassNormal {
				normal++
			}
		}
		var pulses []float64
		for _, r := range in {
			if r.Pulse > 0 {
				pulses = append(pulses, r.Pulse)
			}
		}
		var pulse any
		if len(pulses) > 0 {
			pulse = jsonx.Obj("bpm", whole(avg(pulses)), "readings", len(pulses))
		}
		out.Set("unit", "mmhg")
		out.Set("readings", len(in))
		out.Set("average", bpAverage(typ, in))
		out.Set("pulse", pulse)
		out.Set("min", readingOrNull(lo))
		out.Set("max", readingOrNull(hi))
		out.Set("distribution", distribution(typ, BPClasses, in))
		out.Set("morning_vs_night", morningVsNight(typ, in))
		out.Set("time_in_range", jsonx.Obj("in_range", normal, "readings", len(in), "percent", percent(normal, len(in))))
		out.Set("target", jsonx.Obj("systolic_max", BPElevatedSystolic, "diastolic_max", BPStage1Diastolic))
		out.Set("series", series(typ, in, false))
	case TypeGlucose:
		if filter == "" {
			filter = GlucoseFilterAll
		}
		inRange, below, above := 0, 0, 0
		for _, r := range in {
			switch r.Class() {
			case ClassInRange:
				inRange++
			case ClassLow, ClassUrgentLow:
				below++
			default:
				above++
			}
		}
		byContext := []any{}
		for _, c := range GlucoseContexts {
			var cs []Reading
			ab := 0
			for _, r := range in {
				if r.Context == c {
					cs = append(cs, r)
					if k := r.Class(); k == ClassHigh || k == ClassVeryHigh {
						ab++
					}
				}
			}
			if len(cs) == 0 {
				continue
			}
			b := GlucoseBands[c]
			byContext = append(byContext, jsonx.Obj("context", c, "readings", len(cs), "average", glucoseAverage(cs),
				"above_target", ab, "target", jsonx.Obj("min", b.TargetMin, "max", b.TargetMax)))
		}
		out.Set("filter", filter)
		out.Set("unit", UnitMgDl)
		out.Set("readings", len(in))
		out.Set("average", glucoseAverage(in))
		out.Set("by_context", byContext)
		out.Set("min", readingOrNull(lo))
		out.Set("max", readingOrNull(hi))
		out.Set("distribution", distribution(typ, GlucoseClasses, in))
		out.Set("morning_vs_night", morningVsNight(typ, in))
		out.Set("time_in_range", jsonx.Obj("in_range", inRange, "below", below, "above", above, "readings", len(in),
			"percent", percent(inRange, len(in))))
		out.Set("series", series(typ, in, true))
	case TypeHR:
		var classified []Reading
		normal := 0
		for _, r := range in {
			if c := r.Class(); c != "" {
				classified = append(classified, r)
				if c == ClassNormal {
					normal++
				}
			}
		}
		byContext := []any{}
		for _, c := range HRContexts {
			var cs []Reading
			for _, r := range in {
				if r.Context == c {
					cs = append(cs, r)
				}
			}
			if len(cs) > 0 {
				byContext = append(byContext, jsonx.Obj("context", c, "readings", len(cs), "average", hrAverage(cs)))
			}
		}
		out.Set("unit", "bpm")
		out.Set("readings", len(in))
		out.Set("average", hrAverage(in))
		out.Set("resting_average", hrAverage(classified))
		out.Set("by_context", byContext)
		out.Set("min", readingOrNull(lo))
		out.Set("max", readingOrNull(hi))
		out.Set("distribution", distribution(typ, HRClasses, in))
		out.Set("morning_vs_night", morningVsNight(typ, in))
		out.Set("time_in_range", jsonx.Obj("in_range", normal, "readings", len(classified),
			"percent", percent(normal, len(classified))))
		out.Set("target", jsonx.Obj("min", HRRestingMin, "max", HRRestingMax))
		out.Set("series", series(typ, in, false))
	}
	return out
}
