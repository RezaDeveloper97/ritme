package analysis

import (
	"sort"

	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Trend buckets: a window of up to dailyBucketMaxDays days is drawn per day, a longer one per week
// (Saturday-start weeks, starting at the first logged week inside the window).
const (
	BucketDay          = "day"
	BucketWeek         = "week"
	dailyBucketMaxDays = 31
)

// SymptomCount is a symptom's frequency in the window.
type SymptomCount struct {
	Key string
	// Days with the symptom logged; Cycles the cycles (completed or current) it showed up in.
	Days, Cycles int
}

// TrendPoint is one bucket of the symptom trend: Values[i] is the summed weight of Keys[i] (a day's
// weight is 0–1; a week sums its days).
type TrendPoint struct {
	Start  civildate.Date
	Values []float64
}

// Highlight is the pattern line the hub and the top finding show («سردرد بیشتر در ۲ روز قبل از پریود»).
type Highlight struct {
	Key            string
	Cycles, Of     int
	Relation       string
	Days, From, To int
}

// SymptomsReport is GET /analysis/symptoms.
type SymptomsReport struct {
	DaysLogged, SymptomDays int
	Top                     []SymptomCount
	Bucket                  string
	Keys                    []string
	Trend                   []TrendPoint
	PatternCycles           int
	Typical                 Layout
	Pattern                 []insights.SymptomPattern
	Highlight               *Highlight
}

// TrendReady: at least MinTrendPoints buckets with a logged symptom.
func (r SymptomsReport) TrendReady() bool {
	n := 0
	for _, p := range r.Trend {
		for _, v := range p.Values {
			if v > 0 {
				n++
				break
			}
		}
	}
	return n >= MinTrendPoints
}

// PatternReady: at least MinPatternCycles completed cycles.
func (r SymptomsReport) PatternReady() bool { return r.PatternCycles >= MinPatternCycles }

// cycleOf is the start of the cycle containing d (zero when before the first start).
func cycleOf(cycles []Cycle, d civildate.Date) civildate.Date {
	for i := len(cycles) - 1; i >= 0; i-- {
		if !cycles[i].Start.After(d) {
			return cycles[i].Start
		}
	}
	return civildate.Date{}
}

// symptomCounts ranks the symptoms logged in the range (days desc, then key).
func (in *Input) symptomCounts() (counts []SymptomCount, logged, withSymptoms int) {
	cycles := in.cycles()
	days := map[string]int{}
	cyc := map[string]map[civildate.Date]bool{}
	for d, day := range in.Days {
		if !in.Range.Contains(d) {
			continue
		}
		logged++
		if len(day.Symptoms) > 0 {
			withSymptoms++
		}
		start := cycleOf(cycles, d)
		for k := range day.Symptoms {
			days[k]++
			if !start.IsZero() {
				if cyc[k] == nil {
					cyc[k] = map[civildate.Date]bool{}
				}
				cyc[k][start] = true
			}
		}
	}
	for k, n := range days {
		counts = append(counts, SymptomCount{Key: k, Days: n, Cycles: len(cyc[k])})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Days != counts[j].Days {
			return counts[i].Days > counts[j].Days
		}
		return counts[i].Key < counts[j].Key
	})
	return counts, logged, withSymptoms
}

// symptomPattern lays the range's plausible completed cycles onto the typical cycle.
func (in *Input) symptomPattern() (n int, typical Layout, items []insights.SymptomPattern) {
	var starts []civildate.Date
	var lens []int
	keys := map[string]bool{}
	for _, c := range in.windowCycles() {
		if !insights.ValidCycleLength(c.Length) {
			continue
		}
		starts, lens = append(starts, c.Start), append(lens, c.Length)
		for i := range c.Length {
			if day := in.day(c.Start.AddDays(i)); day != nil {
				for k := range day.Symptoms {
					keys[k] = true
				}
			}
		}
	}
	typCycle, typPeriod := in.typical()
	typical = NewLayout(typCycle, typPeriod)
	items = []insights.SymptomPattern{}
	if len(starts) < MinPatternCycles {
		return len(starts), typical, items
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, s := range insights.Strips(starts, lens, typCycle, symptomLogs(in.Days), sorted) {
		if s.Cycles >= MinPatternCycles {
			items = append(items, s)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Cycles > items[j].Cycles })
	return len(starts), typical, items
}

// highlightOf picks the pattern line: pre-menstrual windows first (the most actionable pattern), then
// the most concentrated strip (highest peak), then the most cycles, then the key.
func highlightOf(n int, items []insights.SymptomPattern) *Highlight {
	var best *insights.SymptomPattern
	better := func(a, b *insights.SymptomPattern) bool {
		ap, bp := a.Window.Relation == insights.RelationBeforePeriod, b.Window.Relation == insights.RelationBeforePeriod
		if ap != bp {
			return ap
		}
		if pa, pb := peak(a.Strip), peak(b.Strip); pa != pb {
			return pa > pb
		}
		if a.Cycles != b.Cycles {
			return a.Cycles > b.Cycles
		}
		return a.Key < b.Key
	}
	for i := range items {
		if items[i].Window == nil {
			continue
		}
		if best == nil || better(&items[i], best) {
			best = &items[i]
		}
	}
	if best == nil {
		return nil
	}
	w := best.Window
	return &Highlight{
		Key: best.Key, Cycles: best.Cycles, Of: n, Relation: w.Relation, Days: w.Days, From: w.StartDay, To: w.EndDay,
	}
}

func peak(strip []float64) float64 {
	m := 0.0
	for _, v := range strip {
		m = max(m, v)
	}
	return m
}

// BuildSymptoms computes the symptom report of the range.
func BuildSymptoms(in *Input) SymptomsReport {
	out := SymptomsReport{Top: []SymptomCount{}, Keys: []string{}, Trend: []TrendPoint{}}
	counts, logged, withSymptoms := in.symptomCounts()
	out.DaysLogged, out.SymptomDays = logged, withSymptoms
	if len(counts) > topListSize {
		out.Top = counts[:topListSize]
	} else if counts != nil {
		out.Top = counts
	}
	for _, c := range out.Top {
		out.Keys = append(out.Keys, c.Key)
	}

	out.Bucket = BucketDay
	start := in.Range.From
	if in.Range.Days() > dailyBucketMaxDays {
		out.Bucket = BucketWeek
		first := in.Range.To
		for d := range in.Days {
			if in.Range.Contains(d) && len(in.Days[d].Symptoms) > 0 && d.Before(first) {
				first = d
			}
		}
		start = first.StartOfWeek()
	}
	for b := start; !b.After(in.Range.To); {
		next := b.AddDays(1)
		if out.Bucket == BucketWeek {
			next = b.AddDays(7)
		}
		p := TrendPoint{Start: b, Values: make([]float64, len(out.Keys))}
		for d := b; d.Before(next) && !d.After(in.Range.To); d = d.AddDays(1) {
			if day := in.day(d); day != nil && in.Range.Contains(d) {
				for i, k := range out.Keys {
					p.Values[i] += day.Symptoms[k]
				}
			}
		}
		for i := range p.Values {
			p.Values[i] = round(p.Values[i], 2)
		}
		out.Trend = append(out.Trend, p)
		b = next
	}

	out.PatternCycles, out.Typical, out.Pattern = in.symptomPattern()
	out.Highlight = highlightOf(out.PatternCycles, out.Pattern)
	return out
}

func (h *Highlight) json(c *Copy) any {
	if h == nil {
		return nil
	}
	return jsonx.Obj(
		"key", h.Key, "label", c.SymptomLabel(h.Key), "cycles", h.Cycles, "of_cycles", h.Of,
		"relation", h.Relation, "days", h.Days, "start_day", h.From, "end_day", h.To,
	)
}

func topJSON(top []SymptomCount, c *Copy) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, 0, len(top))
	for _, s := range top {
		out = append(out, jsonx.Obj("key", s.Key, "label", c.SymptomLabel(s.Key), "days", s.Days, "cycles", s.Cycles))
	}
	return out
}

// JSON is the `data` of GET /analysis/symptoms.
func (r SymptomsReport) JSON(c *Copy) *jsonx.OrderedMap {
	points := make([]*jsonx.OrderedMap, 0, len(r.Trend))
	for _, p := range r.Trend {
		points = append(points, jsonx.Obj("start", p.Start.String(), "values", p.Values))
	}
	items := make([]*jsonx.OrderedMap, 0, len(r.Pattern))
	for _, s := range r.Pattern {
		var w any
		if s.Window != nil {
			w = jsonx.Obj("start_day", s.Window.StartDay, "end_day", s.Window.EndDay, "relation", s.Window.Relation,
				"days", s.Window.Days)
		}
		items = append(items, jsonx.Obj("key", s.Key, "label", c.SymptomLabel(s.Key), "cycles", s.Cycles, "strip", s.Strip,
			"window", w))
	}
	return jsonx.Obj(
		"days_logged", r.DaysLogged,
		"symptom_days", r.SymptomDays,
		"top", topJSON(r.Top, c),
		"trend", jsonx.Obj("ready", r.TrendReady(), "bucket", r.Bucket, "keys", r.Keys, "points", points),
		"pattern", jsonx.Obj(
			"ready", r.PatternReady(), "cycles_counted", r.PatternCycles, "cycles_needed", MinPatternCycles,
			"typical", r.Typical.JSON(), "items", items,
		),
		"highlight", r.Highlight.json(c),
	)
}
