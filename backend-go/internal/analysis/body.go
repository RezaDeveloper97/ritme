package analysis

import (
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// WeightPoint is one day of the weight line: the logged value (nil when none that day) and the trailing
// 7-day moving average.
type WeightPoint struct {
	Date  civildate.Date
	Value *float64
	Avg7  float64
}

// WeekBucket is one 7-day bucket of the activity bars (the last one may be shorter).
type WeekBucket struct {
	Start            civildate.Date
	Days, ActiveDays int
}

// BodyReport is GET /analysis/body.
type BodyReport struct {
	Weight                    []WeightPoint
	Current                   *float64 // the latest 7-day average in the range
	AsOf                      civildate.Date
	Delta7, Delta30, DeltaAll *float64
	BMI                       *float64
	SleepNights               int
	SleepAvg, SleepLuteal     *float64
	SleepByWeekday            [7]*float64 // Saturday first
	ActiveDays, RangeDays     int
	Weeks                     []WeekBucket
}

// WeightReady / SleepReady: trends need ≥ MinTrendPoints points.
func (r BodyReport) WeightReady() bool { return len(r.Weight) >= MinTrendPoints }

// SleepReady reports whether at least two nights were logged.
func (r BodyReport) SleepReady() bool { return r.SleepNights >= MinTrendPoints }

// weights are every logged weight of the loaded days (the range plus the lookback).
func (in *Input) weights() map[civildate.Date]float64 {
	out := map[civildate.Date]float64{}
	for d, day := range in.Days {
		if day.Weight != nil && !d.After(in.Today) {
			out[d] = *day.Weight
		}
	}
	return out
}

func delta(values map[civildate.Date]float64, d civildate.Date, back int, current float64) *float64 {
	prev, ok := MovingAverage7(values, d.AddDays(-back))
	if !ok {
		return nil
	}
	v := round(current-prev, 1)
	return &v
}

// BuildBody computes the body & lifestyle report of the range.
func BuildBody(in *Input) BodyReport {
	out := BodyReport{Weight: []WeightPoint{}, Weeks: []WeekBucket{}, RangeDays: in.Range.Days()}
	values := in.weights()
	for d := in.Range.From; !d.After(in.Range.To); d = d.AddDays(1) {
		avg, ok := MovingAverage7(values, d)
		if !ok {
			continue
		}
		p := WeightPoint{Date: d, Avg7: round(avg, 1)}
		if v, has := values[d]; has {
			p.Value = &v
		}
		out.Weight = append(out.Weight, p)
	}
	if n := len(out.Weight); n > 0 {
		last := out.Weight[n-1]
		raw, _ := MovingAverage7(values, last.Date)
		out.Current, out.AsOf = &last.Avg7, last.Date
		out.Delta7 = delta(values, last.Date, 7, raw)
		out.Delta30 = delta(values, last.Date, 30, raw)
		if n >= MinTrendPoints {
			first, _ := MovingAverage7(values, out.Weight[0].Date)
			v := round(raw-first, 1)
			out.DeltaAll = &v
		}
		if in.HeightCM > 0 {
			m := float64(in.HeightCM) / 100
			bmi := round(raw/(m*m), 1)
			out.BMI = &bmi
		}
	}

	// Sleep: average hours (bucket midpoints), the luteal-phase average and the weekday profile.
	phases := in.phaseDays()
	var all, luteal []float64
	var byWeekday [7][]float64
	first := in.Range.To
	for d, day := range in.Days {
		if !in.Range.Contains(d) {
			continue
		}
		if d.Before(first) {
			first = d
		}
		if day.Active {
			out.ActiveDays++
		}
		h, ok := day.SleepHours()
		if !ok {
			continue
		}
		all = append(all, h)
		if phases[d] == PhaseLuteal {
			luteal = append(luteal, h)
		}
		i := (int(d.Weekday()) + 1) % 7 // Saturday = 0
		byWeekday[i] = append(byWeekday[i], h)
	}
	out.SleepNights = len(all)
	out.SleepAvg = roundPtr(meanPtr(all), 1)
	out.SleepLuteal = roundPtr(meanPtr(luteal), 1)
	for i := range byWeekday {
		out.SleepByWeekday[i] = roundPtr(meanPtr(byWeekday[i]), 1)
	}

	// Activity: 7-day buckets from the window start (for `all`, from the first logged day).
	start := in.Range.From
	if in.Range.Key == RangeAll {
		start = first
	}
	for b := start; !b.After(in.Range.To); b = b.AddDays(7) {
		w := WeekBucket{Start: b}
		for d := b; d.Before(b.AddDays(7)) && !d.After(in.Range.To); d = d.AddDays(1) {
			w.Days++
			if day := in.day(d); day != nil && day.Active {
				w.ActiveDays++
			}
		}
		out.Weeks = append(out.Weeks, w)
	}
	return out
}

// WeightJSON is the weight block; lastDays > 0 keeps only the points of the last lastDays days (the
// hub's sparkline), 0 keeps all.
func (r BodyReport) WeightJSON(lastDays int) *jsonx.OrderedMap {
	points := make([]*jsonx.OrderedMap, 0, len(r.Weight))
	for _, p := range r.Weight {
		if lastDays > 0 && p.Date.DiffDays(r.AsOf) >= lastDays {
			continue
		}
		points = append(points, jsonx.Obj("date", p.Date.String(), "value", ptrOrNil(p.Value), "avg7", p.Avg7))
	}
	return jsonx.Obj(
		"ready", r.WeightReady(),
		"unit", "kg",
		"current", ptrOrNil(r.Current),
		"as_of", dateOrNil(r.AsOf),
		"delta_7d", ptrOrNil(r.Delta7),
		"delta_30d", ptrOrNil(r.Delta30),
		"delta_range", ptrOrNil(r.DeltaAll),
		"bmi", ptrOrNil(r.BMI),
		"moving_average_days", 7,
		"points", points,
	)
}

// JSON is the `data` of GET /analysis/body.
func (r BodyReport) JSON() *jsonx.OrderedMap {
	weekdays := make([]any, 7)
	for i, v := range r.SleepByWeekday {
		weekdays[i] = ptrOrNil(v)
	}
	weeks := make([]*jsonx.OrderedMap, 0, len(r.Weeks))
	for _, w := range r.Weeks {
		weeks = append(weeks, jsonx.Obj("start", w.Start.String(), "days", w.Days, "active_days", w.ActiveDays))
	}
	return jsonx.Obj(
		"weight", r.WeightJSON(0),
		"sleep", jsonx.Obj(
			"ready", r.SleepReady(),
			"nights", r.SleepNights,
			"avg_hours", ptrOrNil(r.SleepAvg),
			"luteal_avg_hours", ptrOrNil(r.SleepLuteal),
			"by_weekday", weekdays,
		),
		"activity", jsonx.Obj("active_days", r.ActiveDays, "days", r.RangeDays, "weeks", weeks),
	)
}
