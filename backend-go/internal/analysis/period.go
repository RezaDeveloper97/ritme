package analysis

import (
	"sort"

	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const (
	// maxOpenBleedDays: an open period's logged flow is looked for this many days from its start.
	maxOpenBleedDays = 10
	// periodListSize is how many periods the intensity grid shows («شهریور · مرداد · تیر»).
	periodListSize = 3
	// topListSize bounds the symptom lists.
	topListSize = 5
)

// PeriodRow is one period of the window.
type PeriodRow struct {
	Start civildate.Date
	// Flows has one code per bleed day ("" = not logged).
	Flows   []string
	Closed  bool
	Current bool
}

// CoSymptom is a symptom logged during the period: the share of periods with it on any bleed day.
type CoSymptom struct {
	Key     string
	Periods int
	Pct     int
}

// PeriodReport is GET /analysis/period.
type PeriodReport struct {
	BasedOn      int
	MedianPeriod *int
	Status       string
	// Average is the mean flow score (1–4) per bleed day across the periods that logged that day.
	Average []*float64
	// PeakDay is the 1-based day with the highest average (0 = no flow logged).
	PeakDay                      int
	SpottingDays, SpottingCycles int
	Cycles                       int
	CoSymptoms                   []CoSymptom
	CoSymptomsReady              bool
	Periods                      []PeriodRow // newest first, at most periodListSize
}

// Ready: the per-day average is a trend over periods (≥ MinTrendPoints).
func (r PeriodReport) Ready() bool { return r.BasedOn >= MinTrendPoints }

// bleedDays is the period's bleed length: a closed plausible period its logged length, an open one the
// last day with flow logged within maxOpenBleedDays (never past today).
func (in *Input) bleedDays(c Cycle) int {
	if c.PeriodDays > 0 {
		return c.PeriodDays
	}
	n := 0
	for i := range maxOpenBleedDays {
		d := c.Start.AddDays(i)
		if d.After(in.Today) || (!c.Current && i >= c.Length) {
			break
		}
		if day := in.day(d); day != nil && day.Flow != "" {
			n = i + 1
		}
	}
	return n
}

// BuildPeriod computes the period report of the range.
func BuildPeriod(in *Input) PeriodReport {
	out := PeriodReport{CoSymptoms: []CoSymptom{}, Periods: []PeriodRow{}}
	var rows []PeriodRow
	var bleeds []int
	cycleStarts := map[civildate.Date]bool{}
	for _, c := range in.cycles() {
		if c.Start.Before(in.Range.From) {
			continue
		}
		cycleStarts[c.Start] = true
		n := in.bleedDays(c)
		row := PeriodRow{Start: c.Start, Flows: make([]string, n), Closed: c.PeriodDays > 0, Current: c.Current}
		for i := range n {
			if day := in.day(c.Start.AddDays(i)); day != nil {
				row.Flows[i] = day.Flow
			}
		}
		if c.PeriodDays > 0 {
			bleeds = append(bleeds, c.PeriodDays)
		}
		rows = append(rows, row)
	}
	out.BasedOn = len(rows)
	out.Cycles = len(rows)
	out.MedianPeriod = insights.Median(bleeds)
	if m := out.MedianPeriod; m != nil {
		out.Status = StatusNormal
		if *m > FIGOPeriodMax {
			out.Status = StatusProlonged
		}
	}

	// Average intensity per bleed day and the peak.
	var sums, counts []int
	for _, r := range rows {
		for i, f := range r.Flows {
			for len(sums) <= i {
				sums, counts = append(sums, 0), append(counts, 0)
			}
			if s := flowScore(f); s > 0 {
				sums[i] += s
				counts[i]++
			}
		}
	}
	best := 0.0
	for i := range sums {
		if counts[i] == 0 {
			out.Average = append(out.Average, nil)
			continue
		}
		v := round(float64(sums[i])/float64(counts[i]), 2)
		out.Average = append(out.Average, &v)
		if v > best {
			best, out.PeakDay = v, i+1
		}
	}

	// Spotting outside the bleed days, and the cycles it showed up in.
	bleedDay := map[civildate.Date]bool{}
	for _, r := range rows {
		for i := range r.Flows {
			bleedDay[r.Start.AddDays(i)] = true
		}
	}
	spotCycles := map[civildate.Date]bool{}
	cycles := in.cycles()
	for _, d := range sortedDates(in.Days) {
		if !in.Range.Contains(d) || !in.Days[d].Spotting || bleedDay[d] {
			continue
		}
		out.SpottingDays++
		for i := len(cycles) - 1; i >= 0; i-- {
			if !cycles[i].Start.After(d) {
				if cycleStarts[cycles[i].Start] {
					spotCycles[cycles[i].Start] = true
				}
				break
			}
		}
	}
	out.SpottingCycles = len(spotCycles)

	// Symptoms during the period: a pattern, so ≥ MinPatternCycles periods.
	out.CoSymptomsReady = len(rows) >= MinPatternCycles
	if out.CoSymptomsReady {
		seen := map[string]int{}
		for _, r := range rows {
			inPeriod := map[string]bool{}
			for i := range r.Flows {
				if day := in.day(r.Start.AddDays(i)); day != nil {
					for k := range day.Symptoms {
						inPeriod[k] = true
					}
				}
			}
			for k := range inPeriod {
				seen[k]++
			}
		}
		for k, n := range seen {
			out.CoSymptoms = append(out.CoSymptoms, CoSymptom{Key: k, Periods: n, Pct: int(round(float64(n)*100/float64(len(rows)), 0))})
		}
		sort.Slice(out.CoSymptoms, func(i, j int) bool {
			a, b := out.CoSymptoms[i], out.CoSymptoms[j]
			if a.Periods != b.Periods {
				return a.Periods > b.Periods
			}
			return a.Key < b.Key
		})
		if len(out.CoSymptoms) > topListSize {
			out.CoSymptoms = out.CoSymptoms[:topListSize]
		}
	}

	for i := len(rows) - 1; i >= 0 && len(out.Periods) < periodListSize; i-- {
		out.Periods = append(out.Periods, rows[i])
	}
	return out
}

func flowOrNil(f string) any {
	if f == "" {
		return nil
	}
	return f
}

// JSON is the `data` of GET /analysis/period.
func (r PeriodReport) JSON(c *Copy) *jsonx.OrderedMap {
	avg := make([]*jsonx.OrderedMap, 0, len(r.Average))
	for i, v := range r.Average {
		var level any
		if v != nil {
			level = flowCodes[int(round(*v, 0))]
		}
		avg = append(avg, jsonx.Obj("day", i+1, "score", ptrOrNil(v), "level", level))
	}
	periods := make([]*jsonx.OrderedMap, 0, len(r.Periods))
	for _, p := range r.Periods {
		days := make([]*jsonx.OrderedMap, 0, len(p.Flows))
		for i, f := range p.Flows {
			days = append(days, jsonx.Obj("day", i+1, "date", p.Start.AddDays(i).String(), "flow", flowOrNil(f)))
		}
		periods = append(periods, jsonx.Obj(
			"start", p.Start.String(), "length", len(p.Flows), "closed", p.Closed, "is_current", p.Current, "days", days,
		))
	}
	co := make([]*jsonx.OrderedMap, 0, len(r.CoSymptoms))
	for _, s := range r.CoSymptoms {
		co = append(co, jsonx.Obj("key", s.Key, "label", c.SymptomLabel(s.Key), "periods", s.Periods, "pct", s.Pct))
	}
	var peak any
	if r.PeakDay > 0 {
		peak = jsonx.Obj("day", r.PeakDay, "level", flowCodes[int(round(*r.Average[r.PeakDay-1], 0))])
	}
	return jsonx.Obj(
		"ready", r.Ready(),
		"based_on_periods", r.BasedOn,
		"figo", jsonx.Obj("period", jsonx.Obj("max", FIGOPeriodMax)),
		"period_length", jsonx.Obj("median", ptrOrNil(r.MedianPeriod), "status", statusOrNil(r.Status)),
		"peak", peak,
		"spotting", jsonx.Obj("days", r.SpottingDays, "cycles", r.SpottingCycles, "of_cycles", r.Cycles),
		"average", avg,
		"periods", periods,
		"co_symptoms", jsonx.Obj("ready", r.CoSymptomsReady, "periods_needed", MinPatternCycles, "items", co),
	)
}
