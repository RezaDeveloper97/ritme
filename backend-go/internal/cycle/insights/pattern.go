package insights

import (
	"math"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	hlmodel "github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const (
	// PatternMinCycles is how many completed cycles the pattern needs before it says anything.
	PatternMinCycles = 3
	// lutealDays: the days before a period line up one-to-one across cycles (the luteal phase is
	// the steady part); the earlier part of each cycle is stretched onto the typical one.
	lutealDays = 14
	// windowShare: a symptom's window is the run around its peak that stays ≥ this share of it.
	windowShare = 0.5
)

// Window relations (how the frontend words the highlight).
const (
	RelationBeforePeriod = "before_period"
	RelationEarly        = "early"
	RelationMid          = "mid"
)

// PatternLookback is how far back the handler loads daily logs: the window's worth of the longest
// plausible cycles plus the current one.
const PatternLookback = (HistoryWindow + 1) * validCycleMax

// DayLog is one day's logged symptoms: key → weight in (0, 1].
type DayLog map[string]float64

// symptomDef is one row of the catalogue: its group and how a daily log weighs it.
type symptomDef struct {
	group, key string
	weigh      func(l *hlmodel.DailyHealthLog) float64
}

// painWeight maps the pain scale (low / medium / high) onto thirds.
func painWeight(v *string) float64 {
	if v == nil {
		return 0
	}
	switch *v {
	case "low":
		return 1.0 / 3
	case "medium":
		return 2.0 / 3
	case "high":
		return 1
	}
	return 0
}

func pain(fields ...string) func(*hlmodel.DailyHealthLog) float64 {
	return func(l *hlmodel.DailyHealthLog) float64 {
		w := 0.0
		for _, f := range fields {
			w = math.Max(w, painWeight(l.Str(f)))
		}
		return w
	}
}

func flag(field string) func(*hlmodel.DailyHealthLog) float64 {
	return func(l *hlmodel.DailyHealthLog) float64 {
		if b := l.Bool(field); b != nil && *b {
			return 1
		}
		return 0
	}
}

func mood(value string) func(*hlmodel.DailyHealthLog) float64 {
	return func(l *hlmodel.DailyHealthLog) float64 {
		for _, m := range l.Strings("moods") {
			if m == value {
				return 1
			}
		}
		return 0
	}
}

func energetic(l *hlmodel.DailyHealthLog) float64 {
	if v := l.Str("energy_level"); v != nil {
		switch *v {
		case "high":
			return 2.0 / 3
		case "very_high":
			return 1
		}
	}
	return 0
}

// Groups are the screen's tabs, in order.
var Groups = []string{"symptoms", "mood", "pain"}

// catalogue lists every tracked symptom, in display order within its group. Keys are stable codes
// the clients label; they are unique within a group.
var catalogue = []symptomDef{
	{"symptoms", "cramps", pain("pelvic_pain_intensity", "stomach_ache_intensity")},
	{"symptoms", "breast_tenderness", pain("breast_sensitivity_intensity", "breast_pain_intensity")},
	{"symptoms", "bloating", pain("bloating_intensity")},
	{"symptoms", "acne", flag("acne")},
	{"symptoms", "energetic", energetic},
	{"symptoms", "fatigue", flag("fatigue")},
	{"symptoms", "food_craving", flag("food_craving")},
	{"symptoms", "nausea", pain("nausea_intensity")},
	{"mood", "happy", mood("happy")},
	{"mood", "calm", mood("calm")},
	{"mood", "sensitive", mood("sensitive")},
	{"mood", "anxious", mood("anxious")},
	{"mood", "sad", mood("sad")},
	{"mood", "angry", mood("angry")},
	{"mood", "frustrated", mood("frustrated")},
	{"mood", "bored", mood("bored")},
	{"pain", "headache", pain("headache_intensity")},
	{"pain", "stomach_ache", pain("stomach_ache_intensity")},
	{"pain", "pelvic_pain", pain("pelvic_pain_intensity")},
	{"pain", "back_pain", pain("back_pain_intensity")},
	{"pain", "breast_pain", pain("breast_pain_intensity")},
	{"pain", "ovarian_pain", pain("ovarian_pain_intensity")},
}

func catalogueKey(group, key string) string { return group + "." + key }

// Weigh turns one daily log into its symptom weights (only the symptoms present).
func Weigh(l *hlmodel.DailyHealthLog) DayLog {
	out := DayLog{}
	for _, d := range catalogue {
		if w := d.weigh(l); w > 0 {
			out[catalogueKey(d.group, d.key)] = w
		}
	}
	return out
}

// Window is where on the typical cycle a symptom usually shows (1-based days, inclusive).
type Window struct {
	StartDay, EndDay int
	Relation         string
	// Days: for before_period the days before the period it starts; for early the length of the
	// run from day 1; for mid 0 (the frontend names the day range instead).
	Days int
}

// SymptomPattern is one row of the heat strips.
type SymptomPattern struct {
	Key string
	// Cycles counts the cycles in which the symptom was logged at least once.
	Cycles int
	// Strip has one value per typical-cycle day: the mean weight across cycles, 0–1 (2 decimals).
	Strip  []float64
	Window *Window
}

// Pattern is the response of GET /cycle/symptom-pattern.
type Pattern struct {
	Ready         bool
	CyclesCounted int
	CycleLength   int
	PeriodLength  int
	OvulationDay  int
	// Groups maps a group to its symptoms, most consistent first; empty until Ready.
	Groups map[string][]SymptomPattern
}

// BuildPattern maps each of the last ≤ HistoryWindow completed, plausible cycles onto a typical
// cycle and averages the logged symptoms per day. logs is keyed by log date.
func BuildPattern(histories []model.History, profile *model.Profile, logs map[civildate.Date]DayLog, today civildate.Date) Pattern {
	m := metrics.Calculate(histories, profile)
	periods := confirmedStarts(histories, today)

	type span struct {
		start  civildate.Date
		length int
	}
	var cycles []span
	var lengths, bleeds []int
	for i := 0; i+1 < len(periods); i++ {
		n := periods[i].PeriodStart.DiffDays(periods[i+1].PeriodStart)
		if !validLength(n) {
			continue
		}
		cycles = append(cycles, span{periods[i].PeriodStart, n})
		if d := closedPeriodDays(periods[i]); d >= validPeriodMin && d <= validPeriodMax {
			bleeds = append(bleeds, d)
		}
	}
	if len(cycles) > HistoryWindow {
		cycles = cycles[len(cycles)-HistoryWindow:]
	}
	for _, c := range cycles {
		lengths = append(lengths, c.length)
	}

	out := Pattern{
		CyclesCounted: len(cycles),
		CycleLength:   m.EffectiveCycleLength,
		PeriodLength:  max(1, m.EffectivePeriodDuration),
		Groups:        map[string][]SymptomPattern{},
	}
	if v := median(lengths); v != nil {
		out.CycleLength = *v
	}
	if v := median(bleeds); v != nil {
		out.PeriodLength = *v
	}
	out.OvulationDay = max(1, out.CycleLength-lutealDays)
	for _, g := range Groups {
		out.Groups[g] = []SymptomPattern{}
	}
	if len(cycles) < PatternMinCycles {
		return out
	}
	out.Ready = true

	typical := out.CycleLength
	for _, d := range catalogue {
		ck := catalogueKey(d.group, d.key)
		sum := make([]float64, typical)
		seen := 0
		for _, c := range cycles {
			logged := false
			for t := range typical {
				w := logs[c.start.AddDays(sourceDay(t, typical, c.length))][ck]
				sum[t] += w
			}
			for day := range c.length {
				if logs[c.start.AddDays(day)][ck] > 0 {
					logged = true
					break
				}
			}
			if logged {
				seen++
			}
		}
		if seen == 0 {
			continue
		}
		strip := make([]float64, typical)
		for t := range sum {
			strip[t] = math.Round(sum[t]/float64(len(cycles))*100) / 100
		}
		out.Groups[d.group] = append(out.Groups[d.group], SymptomPattern{
			Key: d.key, Cycles: seen, Strip: strip, Window: window(strip),
		})
	}
	for g, items := range out.Groups {
		sortByCycles(items)
		out.Groups[g] = items
	}
	return out
}

// sourceDay maps typical-cycle day t (0-based, of typical days) onto the 0-based day of a cycle of
// n days: the last lutealDays line up with the end, the rest scales proportionally from day 0.
func sourceDay(t, typical, n int) int {
	if tail := typical - t; tail <= lutealDays {
		return max(0, n-tail)
	}
	head, srcHead := typical-lutealDays-1, n-lutealDays-1
	if head <= 0 || srcHead <= 0 {
		return min(t, n-1)
	}
	return int(math.Round(float64(t) * float64(srcHead) / float64(head)))
}

// window finds the run around the strip's peak that stays ≥ windowShare of it.
func window(strip []float64) *Window {
	peak, at := 0.0, -1
	for i, v := range strip {
		if v > peak {
			peak, at = v, i
		}
	}
	if at < 0 {
		return nil
	}
	lo, hi := at, at
	for lo > 0 && strip[lo-1] >= peak*windowShare {
		lo--
	}
	for hi < len(strip)-1 && strip[hi+1] >= peak*windowShare {
		hi++
	}
	n := len(strip)
	w := &Window{StartDay: lo + 1, EndDay: hi + 1, Relation: RelationMid}
	switch {
	case w.EndDay >= n-1 && w.StartDay > 1:
		w.Relation, w.Days = RelationBeforePeriod, n-w.StartDay+1
	case w.StartDay <= 2:
		w.Relation, w.Days = RelationEarly, w.EndDay
	}
	return w
}

// sortByCycles orders by Cycles descending, keeping catalogue order for ties (insertion sort: the
// lists are a handful long).
func sortByCycles(items []SymptomPattern) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Cycles > items[j-1].Cycles; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

// JSON is the `data` object of GET /cycle/symptom-pattern.
func (p Pattern) JSON() *jsonx.OrderedMap {
	groups := make([]*jsonx.OrderedMap, 0, len(Groups))
	for _, g := range Groups {
		items := make([]*jsonx.OrderedMap, 0, len(p.Groups[g]))
		for _, s := range p.Groups[g] {
			var w any
			if s.Window != nil {
				w = jsonx.Obj("start_day", s.Window.StartDay, "end_day", s.Window.EndDay,
					"relation", s.Window.Relation, "days", s.Window.Days)
			}
			items = append(items, jsonx.Obj("key", s.Key, "cycles", s.Cycles, "strip", s.Strip, "window", w))
		}
		groups = append(groups, jsonx.Obj("key", g, "items", items))
	}
	return jsonx.Obj(
		"ready", p.Ready,
		"cycles_counted", p.CyclesCounted,
		"cycles_needed", PatternMinCycles,
		"typical", jsonx.Obj(
			"cycle_length", p.CycleLength,
			"period_length", p.PeriodLength,
			"ovulation_day", p.OvulationDay,
		),
		"groups", groups,
	)
}
