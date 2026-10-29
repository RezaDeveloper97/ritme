package analysis

import (
	"slices"
	"sort"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/vitals"
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

// VitalStats is one vitals type of the hub card (§4.1 «فشار و قند», D-175): mean, min–max and the class distribution
// (the vitals classifier, pregnancy- and diabetes-aware through the merged readings), ready with P.VitalsMinLogs
// readings on P.VitalsMinDays different days.
type VitalStats struct {
	Readings, Days int
	Mean, Min, Max []float64 // one value per component (BP: systolic, diastolic; glucose: mg/dL)
	Classes        map[string]int
	values         [][]float64
	days           map[civildate.Date]bool
}

func (v *VitalStats) add(d civildate.Date, class string, values ...float64) {
	if v.days == nil {
		v.days, v.Classes = map[civildate.Date]bool{}, map[string]int{}
	}
	v.Readings++
	v.days[d] = true
	v.Days = len(v.days)
	if class == "" {
		class = "unclassified"
	}
	v.Classes[class]++
	v.values = append(v.values, values)
}

// Ready: the §4.1 minimum.
func (v VitalStats) Ready() bool { return v.Readings >= P.VitalsMinLogs && v.Days >= P.VitalsMinDays }

// ReadingsNeeded: readings (on new days) still missing.
func (v VitalStats) ReadingsNeeded() int {
	return max(0, P.VitalsMinLogs-v.Readings, P.VitalsMinDays-v.Days)
}

func (v *VitalStats) finish() {
	if len(v.values) == 0 {
		return
	}
	k := len(v.values[0])
	v.Mean, v.Min, v.Max = make([]float64, k), make([]float64, k), make([]float64, k)
	for c := range k {
		col := make([]float64, len(v.values))
		for i, r := range v.values {
			col[i] = r[c]
		}
		m, _ := mean(col)
		v.Mean[c], v.Min[c], v.Max[c] = round(m, 0), slices.Min(col), slices.Max(col)
	}
}

// Vitals are the range's blood-pressure and fasting-sugar figures.
type Vitals struct {
	Systolic, Diastolic, Glucose *float64
	BPReadings, GlucoseReadings  int
	BP, Sugar                    VitalStats
}

// vitals summarises the readings of [from, to]: the merged vitals readings when the handler loaded them (bloom
// B-N6-03 — vital_readings + the log sheet, internal/vitals merge rule D-63), else the log sheet's day values.
func (in *Input) vitals(from, to civildate.Date) Vitals {
	var out Vitals
	if in.VitalsLoaded {
		for _, r := range in.VitalReadings {
			if r.Date.Before(from) || r.Date.After(to) {
				continue
			}
			if r.Systolic > 0 && r.Diastolic > 0 {
				out.BP.add(r.Date, r.Class, r.Systolic, r.Diastolic)
			}
			if r.Glucose > 0 && r.SummaryGlucose() {
				out.Sugar.add(r.Date, r.Class, r.Glucose)
			}
		}
	} else {
		for _, d := range sortedDates(in.Days) {
			day := in.Days[d]
			if d.Before(from) || d.After(to) {
				continue
			}
			if day.Systolic != nil && day.Diastolic != nil {
				r := vitals.Reading{Type: vitals.TypeBP, Date: d, Systolic: *day.Systolic, Diastolic: *day.Diastolic}
				out.BP.add(d, r.Class(), *day.Systolic, *day.Diastolic)
			}
			if day.Glucose != nil {
				r := vitals.Reading{Type: vitals.TypeGlucose, Date: d, MgDl: *day.Glucose}
				out.Sugar.add(d, r.Class(), *day.Glucose)
			}
		}
	}
	out.BP.finish()
	out.Sugar.finish()
	out.BPReadings, out.GlucoseReadings = out.BP.Readings, out.Sugar.Readings
	if out.BP.Readings > 0 {
		out.Systolic, out.Diastolic = &out.BP.Mean[0], &out.BP.Mean[1]
	}
	if out.Sugar.Readings > 0 {
		out.Glucose = &out.Sugar.Mean[0]
	}
	return out
}

// Ready: either type meets the §4.1 minimum.
func (v Vitals) Ready() bool { return v.BP.Ready() || v.Sugar.Ready() }

func classesJSON(m map[string]int) *jsonx.OrderedMap {
	out := jsonx.NewObject()
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out.Set(k, m[k])
	}
	return out
}

// JSON is the vitals card body: below the minimum a type keeps its counts and readings_needed, without figures.
func (v Vitals) JSON() *jsonx.OrderedMap {
	var bp, glucose any
	if s := v.BP; s.Readings > 0 {
		o := jsonx.Obj("ready", s.Ready(), "readings", s.Readings, "days", s.Days, "readings_needed", s.ReadingsNeeded(),
			"systolic", nil, "diastolic", nil, "min", nil, "max", nil, "classes", nil, "unit", "mmhg")
		if s.Ready() {
			o.Set("systolic", int(s.Mean[0]))
			o.Set("diastolic", int(s.Mean[1]))
			o.Set("min", jsonx.Obj("systolic", int(s.Min[0]), "diastolic", int(s.Min[1])))
			o.Set("max", jsonx.Obj("systolic", int(s.Max[0]), "diastolic", int(s.Max[1])))
			o.Set("classes", classesJSON(s.Classes))
		}
		bp = o
	}
	if s := v.Sugar; s.Readings > 0 {
		o := jsonx.Obj("ready", s.Ready(), "readings", s.Readings, "days", s.Days, "readings_needed", s.ReadingsNeeded(),
			"avg", nil, "min", nil, "max", nil, "classes", nil, "unit", "mg_dl")
		if s.Ready() {
			o.Set("avg", int(s.Mean[0]))
			o.Set("min", int(s.Min[0]))
			o.Set("max", int(s.Max[0]))
			o.Set("classes", classesJSON(s.Classes))
		}
		glucose = o
	}
	return jsonx.Obj("min_readings", P.VitalsMinLogs, "min_days", P.VitalsMinDays, "blood_pressure", bp, "blood_sugar", glucose)
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
	s.Finding = BuildTopFinding(in, s.Cycle, s.Symptoms, s.Correlations)
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
			"mood_by_phase", plusSection(s.entitled, func() (bool, any) {
				return s.Correlations.MoodByPhase.Ready(), s.Correlations.MoodByPhase.JSON(c)
			}).JSON(),
			"sleep_mood", s.corrCard(CorrSleepMood, func(it Correlation) any { return it.JSON(c) }).JSON(),
			"weight", freeSection(weight.WeightReady(), weight.WeightJSON(hubWeightDays)).JSON(),
			"vitals", freeSection(s.Vitals.Ready(), s.Vitals.JSON()).JSON(),
			// Lab trends (B-N6-06): filled by Handlers.Summary through LabsHub; without it an empty, not-ready section.
			"labs", plusSection(s.entitled, func() (bool, any) { return false, nil }).JSON(),
		),
	)
}
