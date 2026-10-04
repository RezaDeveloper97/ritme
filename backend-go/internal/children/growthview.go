package children

import (
	"database/sql"
	"math"
	"strconv"

	"github.com/ritme/backend-go/internal/children/growth"
	"github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/seeds/who"
)

// Growth verdicts.
const (
	GrowthNormal  = "normal"  // every latest value within P3–P97
	GrowthCheck   = "check"   // a latest value outside P3–P97
	GrowthUnknown = "unknown" // nothing to place (no measurement, sex not told, older than 5 years)
)

// Point is one measurement (or the birth values) on a day.
type Point struct {
	ID         uint64 // 0 for the birth point
	MeasuredOn civildate.Date
	Values     map[who.Indicator]float64
}

func parseDec(ns sql.NullString) (float64, bool) {
	if !ns.Valid {
		return 0, false
	}
	f, err := strconv.ParseFloat(ns.String, 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return f, true
}

func pointOf(id uint64, on civildate.Date, w, l, h sql.NullString) Point {
	p := Point{ID: id, MeasuredOn: on, Values: map[who.Indicator]float64{}}
	if v, ok := parseDec(w); ok {
		p.Values[who.Weight] = v
	}
	if v, ok := parseDec(l); ok {
		p.Values[who.Length] = v
	}
	if v, ok := parseDec(h); ok {
		p.Values[who.Head] = v
	}
	return p
}

// BirthPoint is the child's birth values as a point (nil when none were given).
func BirthPoint(c store.Child) *Point {
	p := pointOf(0, c.BirthDate, c.BirthWeightKg, c.BirthLengthCm, c.BirthHeadCm)
	if len(p.Values) == 0 {
		return nil
	}
	return &p
}

// Points are the measurements (newest first) followed by the birth point when the birth day has no measurement row.
func Points(c store.Child, ms []store.ChildMeasurement) []Point {
	out := make([]Point, 0, len(ms)+1)
	birthMeasured := false
	for _, m := range ms {
		out = append(out, pointOf(m.ID, m.MeasuredOn, m.WeightKg, m.LengthCm, m.HeadCm))
		if m.MeasuredOn == c.BirthDate {
			birthMeasured = true
		}
	}
	if b := BirthPoint(c); b != nil && !birthMeasured {
		out = append(out, *b)
	}
	return out
}

func whoSex(c store.Child) (who.Sex, bool) {
	if !c.Sex.Valid {
		return "", false
	}
	switch c.Sex.String {
	case string(who.Girl):
		return who.Girl, true
	case string(who.Boy):
		return who.Boy, true
	}
	return "", false
}

// assess places value on the standard for the child's sex and age on day (nil percentile when it cannot).
func assess(c store.Child, ind who.Indicator, on civildate.Date, value float64) *jsonx.OrderedMap {
	out := jsonx.Obj("value", round(value, decimalsOf(ind)), "percentile", nil, "z", nil, "in_band", nil)
	sex, ok := whoSex(c)
	if !ok {
		return out
	}
	r, ok := growth.Assess(ind, sex, c.BirthDate.DiffDays(on), value)
	if !ok {
		return out
	}
	out.Set("percentile", growth.Round(r.Percentile, 1))
	out.Set("z", growth.Round(r.Z, 2))
	out.Set("in_band", r.InBand)
	return out
}

func decimalsOf(ind who.Indicator) int {
	if ind == who.Weight {
		return 3
	}
	return 1
}

func round(v float64, n int) float64 { return growth.Round(v, n) }

// ageMonths is the age in months with one decimal (WHO months: days / 30.4375), as the chart's x axis.
func ageMonths(days int) float64 { return growth.Round(float64(days)/30.4375, 1) }

// PointJSON is a point with every indicator placed on the standard.
func PointJSON(c store.Child, p Point, locale string) *jsonx.OrderedMap {
	var id any
	source := "birth"
	if p.ID != 0 {
		id, source = p.ID, "measurement"
	}
	age := AgeOn(c.BirthDate, p.MeasuredOn)
	out := jsonx.Obj(
		"id", id,
		"source", source,
		"measured_on", p.MeasuredOn.String(),
		"age", jsonx.Obj("days", age.Days, "months", ageMonths(age.Days), "label", age.Label(locale)),
	)
	for _, ind := range who.Indicators {
		if v, ok := p.Values[ind]; ok {
			out.Set(string(ind), assess(c, ind, p.MeasuredOn, v))
		} else {
			out.Set(string(ind), nil)
		}
	}
	return out
}

// Verdict is the growth status from the latest value of each indicator.
func Verdict(c store.Child, points []Point) string {
	sex, ok := whoSex(c)
	if !ok {
		return GrowthUnknown
	}
	placed, out := 0, false
	for _, ind := range who.Indicators {
		for _, p := range points { // newest first
			v, has := p.Values[ind]
			if !has {
				continue
			}
			if r, ok := growth.Assess(ind, sex, c.BirthDate.DiffDays(p.MeasuredOn), v); ok {
				placed++
				if !r.InBand {
					out = true
				}
			}
			break
		}
	}
	switch {
	case placed == 0:
		return GrowthUnknown
	case out:
		return GrowthCheck
	}
	return GrowthNormal
}

// VerdictJSON is {status, label}.
func VerdictJSON(status, locale string) *jsonx.OrderedMap {
	return jsonx.Obj("status", status, "label", T("growth."+status, locale))
}

// Indicator units.
func unitOf(ind who.Indicator) string {
	if ind == who.Weight {
		return "kg"
	}
	return "cm"
}

// Chart range: at least the first year, the child's age plus 3 months, at most the standard's 60 months.
const (
	ChartMinMonths   = 12
	ChartAheadMonths = 3
	ChartMaxMonths   = 60
)

// SeriesJSON is GET /children/{id}/growth: the WHO reference curves (P3, P15, P50, P85, P97 per month) for the
// child's sex and the child's points with percentiles.
func SeriesJSON(c store.Child, ind who.Indicator, points []Point, today civildate.Date, locale string) *jsonx.OrderedMap {
	age := AgeOn(c.BirthDate, today)
	to := min(ChartMaxMonths, max(ChartMinMonths, age.Months+ChartAheadMonths))
	sex, known := whoSex(c)
	reference := make([]*jsonx.OrderedMap, 0, to+1)
	var reason, median any
	if known {
		median = T("growth.median."+string(sex), locale)
		for m := 0; m <= to; m++ {
			day := min(who.MaxDay, int(math.Round(float64(m)*30.4375)))
			p, ok := who.At(ind, sex, day)
			if !ok {
				continue
			}
			n := decimalsOf(ind)
			reference = append(reference, jsonx.Obj(
				"month", m, "day", day,
				"p3", round(growth.ValueAt(ind, p, growth.ZP3), n),
				"p15", round(growth.ValueAt(ind, p, growth.ZP15), n),
				"p50", round(growth.ValueAt(ind, p, 0), n),
				"p85", round(growth.ValueAt(ind, p, growth.ZP85), n),
				"p97", round(growth.ValueAt(ind, p, growth.ZP97), n),
			))
		}
	} else {
		reason = "sex_unknown"
	}
	series := make([]*jsonx.OrderedMap, 0, len(points))
	for i := len(points) - 1; i >= 0; i-- { // oldest first for the line
		p := points[i]
		v, ok := p.Values[ind]
		if !ok {
			continue
		}
		var id any
		if p.ID != 0 {
			id = p.ID
		}
		a := assess(c, ind, p.MeasuredOn, v)
		pc, _ := a.Get("percentile")
		inBand, _ := a.Get("in_band")
		days := c.BirthDate.DiffDays(p.MeasuredOn)
		series = append(series, jsonx.Obj(
			"id", id, "measured_on", p.MeasuredOn.String(), "age_days", days, "age_months", ageMonths(days),
			"value", round(v, decimalsOf(ind)), "percentile", pc, "in_band", inBand,
		))
	}
	var latest any
	if len(series) > 0 {
		latest = series[len(series)-1]
	}
	return jsonx.Obj(
		"indicator", string(ind),
		"label", T("indicators."+string(ind), locale),
		"unit", unitOf(ind),
		"unit_label", T("units."+unitOf(ind), locale),
		"sex", nullStr(c.Sex),
		"available", known,
		"reason", reason,
		"range", jsonx.Obj("from_month", 0, "to_month", to),
		"band", jsonx.Obj("low_percentile", growth.BandLowPercentile, "high_percentile", growth.BandHighPercentile, "label", T("growth.band", locale)),
		"median_label", median,
		"reference", reference,
		"points", series,
		"latest", latest,
		"verdict", VerdictJSON(Verdict(c, points), locale),
		"disclaimer", T("growth.disclaimer", locale),
		"source", who.Source,
	)
}

func nullStr(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}
