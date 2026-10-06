package healthrecord

import (
	"math"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/phpround"
	"github.com/ritme/backend-go/internal/vitals"
)

// Vitals groups of the record (nbl_Record_Summary «علائم حیاتی (۳۰ روز)», nbl_Record_Preview's table): blood pressure,
// resting heart rate (the contexts the resting range applies to, incl. log-sheet values), fasting glucose, glucose
// after a meal, and every other glucose context (before a meal, bedtime, random — log-sheet values are random).
const (
	VitalsBP             = "blood_pressure"
	VitalsHeartRate      = "heart_rate"
	VitalsGlucoseFasting = "glucose_fasting"
	VitalsGlucoseMeal    = "glucose_after_meal"
	VitalsGlucoseOther   = "glucose_other"
)

func whole(x float64) int { return int(phpround.Round(x, 0)) }

func mean(xs []float64) float64 {
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func percentOf(n, total int) int {
	if total == 0 {
		return 0
	}
	return whole(100 * float64(n) / float64(total))
}

// bpSummary is {systolic, diastolic, min, max, readings, classification, tone} of the BP readings (nil without any).
// min / max are the readings ranked by systolic, then diastolic.
func bpSummary(rs []vitals.Reading) any {
	if len(rs) == 0 {
		return nil
	}
	var sys, dia []float64
	lo, hi := rs[0], rs[0]
	for _, r := range rs {
		sys, dia = append(sys, r.Systolic), append(dia, r.Diastolic)
		if r.Systolic < lo.Systolic || (r.Systolic == lo.Systolic && r.Diastolic < lo.Diastolic) {
			lo = r
		}
		if r.Systolic > hi.Systolic || (r.Systolic == hi.Systolic && r.Diastolic > hi.Diastolic) {
			hi = r
		}
	}
	s, d := whole(mean(sys)), whole(mean(dia))
	class := vitals.ClassifyBP(float64(s), float64(d))
	return jsonx.Obj(
		"systolic", s, "diastolic", d,
		"min", jsonx.Obj("systolic", int(lo.Systolic), "diastolic", int(lo.Diastolic)),
		"max", jsonx.Obj("systolic", int(hi.Systolic), "diastolic", int(hi.Diastolic)),
		"readings", len(rs), "classification", class, "tone", vitals.Tone(vitals.TypeBP, class), "unit", "mmhg",
	)
}

// valueSummary is {avg, min, max, readings[, in_target, in_target_percent], unit} (nil without readings).
func valueSummary(xs []float64, unit string, inTarget *int) any {
	if len(xs) == 0 {
		return nil
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, x := range xs {
		lo, hi = min(lo, x), max(hi, x)
	}
	out := jsonx.Obj("avg", whole(mean(xs)), "min", whole(lo), "max", whole(hi), "readings", len(xs))
	if inTarget != nil {
		out.Set("in_target", *inTarget)
		out.Set("in_target_percent", percentOf(*inTarget, len(xs)))
	}
	out.Set("unit", unit)
	return out
}

// VitalsSummary is the vitals section body over [from, to] of merged readings (timed + log sheet, any order).
func VitalsSummary(rs []vitals.Reading, from, to civildate.Date) (*jsonx.OrderedMap, bool) {
	var bp []vitals.Reading
	var hr []float64
	glucose := map[string][]float64{}
	inTarget := map[string]int{}
	for _, r := range rs {
		if r.Date.Before(from) || r.Date.After(to) {
			continue
		}
		switch r.Type {
		case vitals.TypeBP:
			if r.Systolic > 0 && r.Diastolic > 0 {
				bp = append(bp, r)
			}
		case vitals.TypeHR:
			if r.Pulse > 0 && vitals.HRClassified(r.Context) {
				hr = append(hr, r.Pulse)
			}
		case vitals.TypeGlucose:
			g := VitalsGlucoseOther
			switch r.Context {
			case vitals.ContextFasting:
				g = VitalsGlucoseFasting
			case vitals.ContextAfterMeal:
				g = VitalsGlucoseMeal
			}
			glucose[g] = append(glucose[g], r.MgDl)
			if r.Class() == vitals.ClassInRange {
				inTarget[g]++
			}
		}
	}
	g := func(key string) any {
		n := inTarget[key]
		return valueSummary(glucose[key], "mg_dl", &n)
	}
	has := len(bp) > 0 || len(hr) > 0 || len(glucose) > 0
	return jsonx.Obj(
		"from", from.String(), "to", to.String(), "days", from.DiffDays(to)+1,
		VitalsBP, bpSummary(bp),
		VitalsHeartRate, valueSummary(hr, "bpm", nil),
		VitalsGlucoseFasting, g(VitalsGlucoseFasting),
		VitalsGlucoseMeal, g(VitalsGlucoseMeal),
		VitalsGlucoseOther, g(VitalsGlucoseOther),
	), has
}
