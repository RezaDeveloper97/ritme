package analysis

import (
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

const (
	// hubBars: the hub's cycle card draws the last 6 cycles («فر ار خر تی مر شه»).
	hubBars = 6
	// hubStripCycles: «پریود و تاریخچه سیکل‌ها» shows the last 3 cycles as day dots.
	hubStripCycles = 3
	// hubTopSymptoms: the hub's symptom card lists 3.
	hubTopSymptoms = 3
	// hubWeightDays: the hub's weight sparkline covers the last 30 days («−۰٫۶ در ۳۰ روز»).
	hubWeightDays = 30
)

// Vitals are the range's blood-pressure and fasting-sugar averages.
type Vitals struct {
	Systolic, Diastolic, Glucose *float64
	BPReadings, GlucoseReadings  int
}

// vitals averages the readings of [from, to].
func (in *Input) vitals(from, to civildate.Date) Vitals {
	var sys, dia, glu []float64
	for d, day := range in.Days {
		if d.Before(from) || d.After(to) {
			continue
		}
		if day.Systolic != nil && day.Diastolic != nil {
			sys, dia = append(sys, *day.Systolic), append(dia, *day.Diastolic)
		}
		if day.Glucose != nil {
			glu = append(glu, *day.Glucose)
		}
	}
	return Vitals{
		Systolic: roundPtr(meanPtr(sys), 0), Diastolic: roundPtr(meanPtr(dia), 0), Glucose: roundPtr(meanPtr(glu), 0),
		BPReadings: len(sys), GlucoseReadings: len(glu),
	}
}

// JSON is the vitals card body.
func (v Vitals) JSON() *jsonx.OrderedMap {
	var bp, glucose any
	if v.BPReadings > 0 {
		bp = jsonx.Obj("systolic", int(*v.Systolic), "diastolic", int(*v.Diastolic), "readings", v.BPReadings, "unit", "mmhg")
	}
	if v.GlucoseReadings > 0 {
		glucose = jsonx.Obj("avg", int(*v.Glucose), "readings", v.GlucoseReadings, "unit", "mg_dl")
	}
	return jsonx.Obj("blood_pressure", bp, "blood_sugar", glucose)
}

// Summary is GET /analysis/summary: the hub.
type Summary struct {
	Cycle        CycleReport
	Symptoms     SymptomsReport
	Body         BodyReport
	Vitals       Vitals
	Correlations *CorrelationsReport // nil when locked
	Finding      TopFinding
	recent       []*jsonx.OrderedMap
	entitled     bool
}

// recentCycles are the last hubStripCycles cycles, newest first, as day layouts (the current cycle at
// its typical length).
func (in *Input) recentCycles() []*jsonx.OrderedMap {
	all := in.cycles()
	typCycle, typPeriod := in.typical()
	out := []*jsonx.OrderedMap{}
	for i := len(all) - 1; i >= 0 && len(out) < hubStripCycles; i-- {
		c := all[i]
		length, bleed := c.Length, c.PeriodDays
		if c.Current {
			length = max(typCycle, c.Start.DiffDays(in.Today)+1)
		}
		if bleed == 0 {
			bleed = min(typPeriod, length)
		}
		l := NewLayout(length, bleed)
		var soFar any
		if c.Current {
			soFar = c.Start.DiffDays(in.Today) + 1
		}
		out = append(out, jsonx.Obj(
			"start", c.Start.String(), "length", length, "is_current", c.Current, "days_so_far", soFar,
			"period_days", l.PeriodDays, "fertile_start_day", l.FertileStart, "fertile_end_day", l.FertileEnd,
			"ovulation_day", l.OvulationDay,
		))
	}
	return out
}

// BuildSummary computes the hub. Plus sections are computed only for an entitled user.
func BuildSummary(in *Input) Summary {
	s := Summary{
		Cycle: BuildCycle(in), Symptoms: BuildSymptoms(in), Body: BuildBody(in),
		Vitals: in.vitals(in.Range.From, in.Range.To), recent: in.recentCycles(), entitled: in.DeepAnalysis,
	}
	if in.DeepAnalysis {
		r := BuildCorrelations(in)
		s.Correlations = &r
	}
	s.Finding = BuildTopFinding(s.Cycle, s.Symptoms)
	if in.NoCycleNudge {
		s.Finding = s.Finding.withoutCycleNudge()
	}
	return s
}

func (s Summary) cycleCard() Section {
	r := s.Cycle
	n := min(len(r.Cycles), hubBars)
	bars := make([]*jsonx.OrderedMap, 0, n)
	for i := n - 1; i >= 0; i-- { // oldest → newest, the order the bars read
		c := r.Cycles[i]
		bars = append(bars, jsonx.Obj("start", c.Start.String(), "length", c.Length, "in_figo_range", c.InFIGO))
	}
	return freeSection(r.Ready(), jsonx.Obj(
		"median_cycle", ptrOrNil(r.MedianCycle),
		"median_period", ptrOrNil(r.MedianPeriod),
		"cycle_status", statusOrNil(r.CycleStatus),
		"regularity", r.Regularity,
		"variation_days", ptrOrNil(r.Variation),
		"variation_max", r.VariationLimit,
		"based_on_cycles", r.BasedOn,
		"bars", bars,
	))
}

func (s Summary) symptomsCard(c *Copy) Section {
	r := s.Symptoms
	top := r.Top
	if len(top) > hubTopSymptoms {
		top = top[:hubTopSymptoms]
	}
	return freeSection(len(top) > 0, jsonx.Obj(
		"cycles_counted", r.PatternCycles,
		"cycles_needed", MinPatternCycles,
		"top", topJSON(top, c),
		"highlight", r.Highlight.json(c),
	))
}

func (s Summary) corrCard(key string, body func(Correlation) any) Section {
	return plusSection(s.entitled, func() (bool, any) {
		item, _ := s.Correlations.Item(key)
		return item.Status == CorrReady, body(item)
	})
}

// JSON is the `data` of GET /analysis/summary (range added by the handler).
func (s Summary) JSON(c *Copy) *jsonx.OrderedMap {
	weight := s.Body
	return jsonx.Obj(
		"top_finding", s.Finding.JSON(c),
		"sections", jsonx.Obj(
			"cycle", s.cycleCard().JSON(),
			"recent_cycles", freeSection(len(s.recent) > 0, s.recent).JSON(),
			"symptoms", s.symptomsCard(c).JSON(),
			"mood_by_phase", s.corrCard(CorrPhaseMood, func(it Correlation) any { return it.MoodByPhaseJSON(c) }).JSON(),
			"sleep_mood", s.corrCard(CorrSleepMood, func(it Correlation) any { return it.JSON(c) }).JSON(),
			"weight", freeSection(weight.WeightReady(), weight.WeightJSON(hubWeightDays)).JSON(),
			"vitals", freeSection(s.Vitals.BPReadings+s.Vitals.GlucoseReadings > 0, s.Vitals.JSON()).JSON(),
			// Lab trends arrive with B-N6-06; until then an entitled user gets an empty, not-ready section.
			"labs", plusSection(s.entitled, func() (bool, any) { return false, nil }).JSON(),
		),
	)
}
