package analysis

import (
	"sort"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
)

// The pregnancy analysis (B-N3-12, An_Hub_Preg + An_PregWeight): GET /api/v1/analysis/pregnancy. One
// payload serves the hub and the weight-gain screen. Like the cycle reports it is a pure function of a
// PregnancyInput (golden-tested with a fixed clock); the handler loads the input from the pregnancy v1/v2
// logs, the taxonomy v2 day logs and the M3 appointments. Thresholds: pregnancy_thresholds.go.

// BPReading is one blood-pressure reading (mmHg).
type BPReading struct {
	Systolic, Diastolic float64
}

// GlucoseReading is one glucose reading (mg/dL) in a slot (GlucoseFasting / GlucoseOneHour / GlucoseTwoHour).
type GlucoseReading struct {
	Date  civildate.Date
	Slot  string
	Value float64
}

// KickSession is one day's fetal-movement count; Minutes is the time from the first to the last movement
// (nil when the times were not logged).
type KickSession struct {
	Count   int
	Minutes *int
}

// PregnancyVisit is an M3 appointment seen as a pregnancy visit.
type PregnancyVisit struct {
	Date       civildate.Date
	Title      string
	Done       bool // stage done | result
	ResultNote string
}

// PregnancyInput is everything the pregnancy report reads.
type PregnancyInput struct {
	Today civildate.Date
	// Start is day 0 of week 1 (the LMP or its estimate); Due the due date.
	Start, Due civildate.Date
	// HeightCM is user_profiles.height (0 = unknown); ProfileWeight user_profiles.weight (nil = none).
	HeightCM      int
	ProfileWeight *float64
	// Weights (kg), BP readings and symptom days by date, from BaselineLookbackDays before Start to Today.
	Weights  map[civildate.Date]float64
	BP       map[civildate.Date]BPReading
	Glucose  []GlucoseReading
	Symptoms map[civildate.Date]map[string]bool
	// Kicks are the fetal-movement days of the last KicksChartDays.
	Kicks map[civildate.Date]KickSession
	// Visits are the user's (non-cancelled) appointments, soonest first.
	Visits []PregnancyVisit
	// DeepAnalysis is the plus.deep_analysis entitlement (glucose targets, symptoms by trimester).
	DeepAnalysis bool
	// Label names a symptom key (nil → the key).
	Label func(key string) string
	// Copy renders the top finding from the analysis-pregnancy namespace (nil → key + params only).
	Copy *Copy
}

func (in *PregnancyInput) gaDays(d civildate.Date) int { return in.Start.DiffDays(d) }

// CurrentWeek is the 1-based week of today (1..42).
func (in *PregnancyInput) CurrentWeek() int { return min(42, max(1, in.gaDays(in.Today)/7+1)) }

// trimesterOn is the trimester of gaDays completed days on day d, by the pregnancy.trimester version valid on d
// (calc.TrimesterOn, D-169).
func trimesterOn(gaDays int, d civildate.Date) int { return calc.TrimesterOn(max(0, gaDays), d) }

func (in *PregnancyInput) label(key string) string {
	if in.Label == nil {
		return key
	}
	return in.Label(key)
}

// IOMBandFor is the IOM category of a pre-pregnancy BMI.
func IOMBandFor(bmi float64) IOMBand {
	for _, b := range IOMBands {
		if bmi >= b.BMIMin && (b.BMIMax == 0 || bmi < b.BMIMax) {
			return b
		}
	}
	return IOMBands[len(IOMBands)-1]
}

// RecommendedGain is the recommended cumulative gain range at gestational age gaWeeks (fractional): 0 →
// IOMFirstTrimesterGain{Min,Max} linearly over weeks 0–13, then linearly to the band's total at week 40,
// flat afterwards.
func RecommendedGain(b IOMBand, gaWeeks float64) (lo, hi float64) {
	switch {
	case gaWeeks <= 0:
		return 0, 0
	case gaWeeks <= IOMFirstTrimesterWeeks:
		f := gaWeeks / IOMFirstTrimesterWeeks
		return IOMFirstTrimesterGainMin * f, IOMFirstTrimesterGainMax * f
	}
	f := min(1, (gaWeeks-IOMFirstTrimesterWeeks)/(IOMTermWeeks-IOMFirstTrimesterWeeks))
	return IOMFirstTrimesterGainMin + (b.GainMin-IOMFirstTrimesterGainMin)*f,
		IOMFirstTrimesterGainMax + (b.GainMax-IOMFirstTrimesterGainMax)*f
}

// Weight-gain statuses and missing reasons.
const (
	GainBelow  = "below"
	GainWithin = "within"
	GainAbove  = "above"

	MissingWeights  = "weights"
	MissingBaseline = "baseline"
	MissingHeight   = "height"

	BaselineBeforePregnancy = "before_pregnancy"
	BaselineFirstTrimester  = "first_trimester"
	BaselineProfile         = "profile"
)

type weightPoint struct {
	date         civildate.Date
	weight, gain float64
}

// baseline picks the pre-pregnancy weight: the latest weight logged in the BaselineLookbackDays before
// Start, else the earliest one of the first trimester, else the profile weight.
func (in *PregnancyInput) baseline() (w float64, source string, date *civildate.Date, ok bool) {
	dates := sortedDates(in.Weights)
	for i := len(dates) - 1; i >= 0; i-- {
		d := dates[i]
		if !d.After(in.Start) && !d.Before(in.Start.AddDays(-BaselineLookbackDays)) {
			return in.Weights[d], BaselineBeforePregnancy, &d, true
		}
	}
	for _, d := range dates {
		if g := in.gaDays(d); g > 0 && g < IOMFirstTrimesterWeeks*7 {
			return in.Weights[d], BaselineFirstTrimester, &d, true
		}
	}
	if in.ProfileWeight != nil && *in.ProfileWeight > 0 {
		return *in.ProfileWeight, BaselineProfile, nil, true
	}
	return 0, "", nil, false
}

func iomTableJSON() []any {
	out := make([]any, 0, len(IOMBands))
	for _, b := range IOMBands {
		var lo, hi any
		if b.BMIMin > 0 {
			lo = b.BMIMin
		}
		if b.BMIMax > 0 {
			hi = round(b.BMIMax-0.1, 1) // «۱۸٫۵ تا ۲۴٫۹»
		}
		out = append(out, jsonx.Obj("category", b.Category, "bmi_min", lo, "bmi_max", hi,
			"gain_min", b.GainMin, "gain_max", b.GainMax))
	}
	return out
}

// pregnancyWeightLogs counts the days with a weight from Start to Today (the pre-pregnancy lookback is not counted).
func (in *PregnancyInput) pregnancyWeightLogs() int {
	n := 0
	for d := range in.Weights {
		if !d.Before(in.Start) && !d.After(in.Today) {
			n++
		}
	}
	return n
}

// weightGain is the An_PregWeight body (and the hub's weight card); the candidate is the gain against the band.
func (in *PregnancyInput) weightGain() (Section, *Candidate) {
	base, source, baseDate, hasBase := in.baseline()
	// The pregnancy weights: one point per gestational week (its last weight), oldest first.
	byWeek := map[int]civildate.Date{}
	for d := range in.Weights {
		g := in.gaDays(d)
		if g < 0 || d.After(in.Today) {
			continue
		}
		if prev, ok := byWeek[g/7]; !ok || d.After(prev) {
			byWeek[g/7] = d
		}
	}
	weeks := make([]int, 0, len(byWeek))
	for w := range byWeek {
		weeks = append(weeks, w)
	}
	sort.Ints(weeks)
	var points []weightPoint
	for _, w := range weeks {
		d := byWeek[w]
		points = append(points, weightPoint{date: d, weight: in.Weights[d], gain: in.Weights[d] - base})
	}

	var bmi, category, band, target, status, current, recent, baseline any
	missing := any(nil)
	switch {
	case !hasBase:
		missing = MissingBaseline
	case in.HeightCM <= 0:
		missing = MissingHeight
	case len(points) == 0:
		missing = MissingWeights
	}
	// §4.1 (D-175, T-M10-26b): the IOM comparison needs P.PregWeightMinLogs weights logged in the pregnancy (days
	// from Start to Today) besides the pre-pregnancy BMI; below it ready false + logs_needed.
	logsNeeded := max(0, P.PregWeightMinLogs-in.pregnancyWeightLogs())
	if hasBase {
		var bd any
		if baseDate != nil {
			bd = baseDate.String()
		}
		baseline = jsonx.Obj("weight", round(base, 1), "source", source, "date", bd)
	}
	var b IOMBand
	if hasBase && in.HeightCM > 0 {
		m := float64(in.HeightCM) / 100
		v := round(base/(m*m), 1)
		b = IOMBandFor(v)
		bmi, category = v, b.Category
		target = jsonx.Obj("min", b.GainMin, "max", b.GainMax)
		band = []any{
			jsonx.Obj("ga_weeks", 0, "min", 0.0, "max", 0.0),
			jsonx.Obj("ga_weeks", IOMFirstTrimesterWeeks, "min", IOMFirstTrimesterGainMin, "max", IOMFirstTrimesterGainMax),
			jsonx.Obj("ga_weeks", IOMTermWeeks, "min", b.GainMin, "max", b.GainMax),
		}
	}
	pts := []any{}
	if hasBase {
		for _, p := range points {
			g := in.gaDays(p.date)
			pts = append(pts, jsonx.Obj("date", p.date.String(), "ga_days", g, "week", min(42, g/7+1),
				"weight", round(p.weight, 1), "gain", round(p.gain, 1)))
		}
	}
	ready := missing == nil && logsNeeded == 0
	var cand *Candidate
	if hasBase && len(points) > 0 {
		last := points[len(points)-1]
		g := in.gaDays(last.date)
		var recommended any
		if ready {
			lo, hi := RecommendedGain(b, float64(g)/7)
			lo, hi = round(lo, 1), round(hi, 1)
			gain := round(last.gain, 1)
			switch {
			case gain < lo:
				status = GainBelow
			case gain > hi:
				status = GainAbove
			default:
				status = GainWithin
			}
			recommended = jsonx.Obj("min", lo, "max", hi)
			kind := "preg_gain_" + status.(string)
			cand = &Candidate{Kind: kind, Strength: findingStrength[kind], Evidence: last.date, Parts: []Phrase{{
				Key: "findings.gain." + status.(string), Args: []Arg{
					Num("week", min(42, g/7+1)), Num("gain", gain), Num("min", lo), Num("max", hi)}}}}
		}
		current = jsonx.Obj("date", last.date.String(), "ga_days", g, "week", min(42, g/7+1),
			"weight", round(last.weight, 1), "gain", round(last.gain, 1), "recommended", recommended)
		// «۴ هفته اخیر»: the latest weight against the last one logged ≥ 28 days before it.
		dates := sortedDates(in.Weights)
		for i := len(dates) - 1; i >= 0; i-- {
			d := dates[i]
			gap := d.DiffDays(last.date)
			if gap >= IOMRecentDays && gap <= IOMRecentDays+IOMRecentSlackDays && !d.Before(in.Start) {
				recent = round(last.weight-in.Weights[d], 1)
				break
			}
			if gap > IOMRecentDays+IOMRecentSlackDays {
				break
			}
		}
	}
	return freeSection(ready, jsonx.Obj(
		"missing", missing,
		"min_logs", P.PregWeightMinLogs,
		"logs_needed", logsNeeded,
		"baseline", baseline,
		"height_cm", nullInt(in.HeightCM),
		"bmi", bmi,
		"bmi_category", category,
		"target", target,
		"status", status,
		"current", current,
		"recent_4w", recent,
		"points", pts,
		"band", band,
		"iom_table", iomTableJSON(),
		"singleton_only", true,
		"source", "IOM 2009",
	)), cand
}

func nullInt(n int) any {
	if n <= 0 {
		return nil
	}
	return n
}

// Blood-pressure statuses.
const (
	BPNormal = "below_threshold"
	BPHigh   = "high"
	BPSevere = "severe"
)

// bloodPressure is the BP card; the candidate names a high / severe reading (the latest one is the evidence).
func (in *PregnancyInput) bloodPressure() (Section, *Candidate) {
	dates := []civildate.Date{}
	for d := range in.BP {
		if !d.Before(in.Start) && !d.After(in.Today) {
			dates = append(dates, d)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	high, severe := 0, 0
	var lastHigh, lastSevere civildate.Date
	readings := []any{}
	for i, d := range dates {
		r := in.BP[d]
		isHigh := r.Systolic >= BPHighSystolic || r.Diastolic >= BPHighDiastolic
		if isHigh {
			high++
			lastHigh = d
		}
		if r.Systolic >= BPSevereSystolic || r.Diastolic >= BPSevereDiastolic {
			severe++
			lastSevere = d
		}
		if i >= len(dates)-BPChartReadings {
			readings = append(readings, jsonx.Obj("date", d.String(), "systolic", int(r.Systolic),
				"diastolic", int(r.Diastolic), "high", isHigh))
		}
	}
	var status, latest any
	if n := len(dates); n > 0 {
		switch {
		case severe > 0:
			status = BPSevere
		case high > 0:
			status = BPHigh
		default:
			status = BPNormal
		}
		r := in.BP[dates[n-1]]
		latest = jsonx.Obj("date", dates[n-1].String(), "systolic", int(r.Systolic), "diastolic", int(r.Diastolic))
	}
	var cand *Candidate
	switch {
	case severe > 0:
		cand = &Candidate{Kind: FindingPregBPSevere, Strength: findingStrength[FindingPregBPSevere], Evidence: lastSevere,
			Parts: []Phrase{{Key: "findings.bp.severe", Args: []Arg{Num("sys", BPSevereSystolic), Num("dia", BPSevereDiastolic)}}}}
	case high > 0:
		cand = &Candidate{Kind: FindingPregBPHigh, Strength: findingStrength[FindingPregBPHigh], Evidence: lastHigh,
			Parts: []Phrase{{Key: "findings.bp.high", Args: []Arg{
				Num("n", high), Num("sys", BPHighSystolic), Num("dia", BPHighDiastolic)}}}}
	}
	return freeSection(len(dates) > 0, jsonx.Obj(
		"readings_count", len(dates),
		"high_count", high,
		"severe_count", severe,
		"status", status,
		"latest", latest,
		"readings", readings,
		"threshold", jsonx.Obj("systolic", BPHighSystolic, "diastolic", BPHighDiastolic),
		"severe_threshold", jsonx.Obj("systolic", BPSevereSystolic, "diastolic", BPSevereDiastolic),
		"source", "ACOG",
	)), cand
}

// glucose is the Plus glucose card; a candidate per slot whose average is at or above its target.
func (in *PregnancyInput) glucose() (Section, []Candidate) {
	var cands []Candidate
	sec := plusSection(in.DeepAnalysis, func() (bool, any) {
		slots := []any{}
		ready := false
		for _, t := range GlucoseTargets {
			var vals []float64
			inTarget := 0
			var last *GlucoseReading
			for i := range in.Glucose {
				g := in.Glucose[i]
				if g.Slot != t.Slot || g.Date.Before(in.Start) || g.Date.After(in.Today) {
					continue
				}
				vals = append(vals, g.Value)
				if g.Value < float64(t.Max) {
					inTarget++
				}
				if last == nil || g.Date.After(last.Date) {
					last = &in.Glucose[i]
				}
			}
			var avg, latest, within any
			if m, ok := mean(vals); ok {
				ready = true
				a := int(round(m, 0))
				avg, within = a, a < t.Max
				latest = jsonx.Obj("date", last.Date.String(), "value", round(last.Value, 1))
				if a >= t.Max {
					cands = append(cands, Candidate{Kind: FindingPregGlucoseAbove,
						Strength: findingStrength[FindingPregGlucoseAbove], Evidence: last.Date,
						Parts: []Phrase{{Key: "findings.glucose_above." + t.Slot, Args: []Arg{Num("avg", a), Num("target", t.Max)}}}})
				}
			}
			slots = append(slots, jsonx.Obj("slot", t.Slot, "target_max", t.Max, "avg", avg,
				"within_target", within, "readings", len(vals), "in_target", inTarget, "latest", latest))
		}
		return ready, jsonx.Obj("slots", slots, "unit", "mg_dl", "source", "ADA 2024")
	})
	return sec, cands
}

func (in *PregnancyInput) kicks() Section {
	days := []any{}
	sessions, timed, low := 0, 0, 0
	allWithin := true
	for i := KicksChartDays - 1; i >= 0; i-- {
		d := in.Today.AddDays(-i)
		k, ok := in.Kicks[d]
		var count, minutes any
		if ok {
			sessions++
			count = k.Count
			if k.Count < KicksTarget {
				low++
			}
			if k.Minutes != nil && *k.Minutes <= KicksSessionMaxMinutes {
				timed++
				if k.Count >= KicksTarget {
					minutes = *k.Minutes
				}
				if k.Count < KicksTarget || *k.Minutes > KicksWindowMinutes {
					allWithin = false
				}
			}
		}
		days = append(days, jsonx.Obj("date", d.String(), "count", count, "minutes_to_target", minutes))
	}
	var within any
	if timed > 0 {
		within = allWithin
	}
	return freeSection(sessions > 0, jsonx.Obj(
		"days", days,
		"sessions", sessions,
		"timed_sessions", timed,
		"low_count_days", low,
		"all_within_window", within,
		"target", KicksTarget,
		"window_minutes", KicksWindowMinutes,
		"from_week", KicksFromWeek,
		"current_week", in.CurrentWeek(),
	))
}

// symptomsByTrimester is the Plus symptoms card; the candidate is the current trimester's most logged symptom.
func (in *PregnancyInput) symptomsByTrimester() (Section, *Candidate) {
	var cand *Candidate
	sec := plusSection(in.DeepAnalysis, func() (bool, any) {
		counts := [4]map[string]int{{}, {}, {}, {}}
		lastDay := [4]map[string]civildate.Date{{}, {}, {}, {}}
		logged := [4]int{}
		for d, set := range in.Symptoms {
			g := in.gaDays(d)
			if g < 0 || d.After(in.Today) || len(set) == 0 {
				continue
			}
			t := trimesterOn(g, d)
			logged[t]++
			for k := range set {
				counts[t][k]++
				if d.After(lastDay[t][k]) {
					lastDay[t][k] = d
				}
			}
		}
		cur := trimesterOn(in.gaDays(in.Today), in.Today)
		ready := false
		out := []any{}
		for t := 1; t <= 3; t++ {
			keys := make([]string, 0, len(counts[t]))
			for k := range counts[t] {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool {
				if counts[t][keys[i]] != counts[t][keys[j]] {
					return counts[t][keys[i]] > counts[t][keys[j]]
				}
				return keys[i] < keys[j]
			})
			items := []any{}
			for _, k := range keys[:min(len(keys), PregnancyTopSymptoms)] {
				items = append(items, jsonx.Obj("key", k, "label", in.label(k), "days", counts[t][k]))
			}
			ready = ready || len(items) > 0
			if t == cur && len(keys) > 0 && logged[t] > 0 {
				k := keys[0]
				cand = &Candidate{Kind: FindingPregSymptomTop,
					Strength: pregSymptomStrengthMax * float64(counts[t][k]) / float64(logged[t]), Evidence: lastDay[t][k],
					Parts: []Phrase{{Key: "findings.symptom_top", Args: []Arg{Symptom("symptom", k), Num("days", counts[t][k])}}}}
			}
			out = append(out, jsonx.Obj("trimester", t, "is_current", t == cur, "is_future", t > cur,
				"logged_days", logged[t], "items", items))
		}
		return ready, jsonx.Obj("trimesters", out)
	})
	return sec, cand
}

func (in *PregnancyInput) visits() Section {
	done, upcoming := 0, 0
	var next, result any
	var resultDate civildate.Date
	for _, v := range in.Visits {
		past := v.Date.Before(in.Today)
		if v.Done || past {
			done++
		} else {
			upcoming++
			if next == nil {
				g := in.gaDays(v.Date)
				next = jsonx.Obj("title", v.Title, "date", v.Date.String(), "week", min(42, max(1, g/7+1)),
					"days_until", in.Today.DiffDays(v.Date))
			}
		}
		if v.ResultNote != "" && !v.Date.After(in.Today) && (result == nil || !v.Date.Before(resultDate)) {
			resultDate = v.Date
			result = jsonx.Obj("title", v.Title, "date", v.Date.String(), "note", v.ResultNote)
		}
	}
	return freeSection(done+upcoming > 0, jsonx.Obj(
		"done", done, "upcoming", upcoming, "next", next, "latest_result", result,
	))
}

// BuildPregnancy is GET /analysis/pregnancy: the sections and the hub's top finding (§4.1, T-M10-26b).
func BuildPregnancy(in *PregnancyInput) *jsonx.OrderedMap {
	g := in.gaDays(in.Today)
	weight, gainCand := in.weightGain()
	bp, bpCand := in.bloodPressure()
	glucose, glucoseCands := in.glucose()
	symptoms, symptomCand := in.symptomsByTrimester()
	var cands []Candidate
	for _, c := range []*Candidate{bpCand, gainCand} {
		if c != nil {
			cands = append(cands, *c)
		}
	}
	cands = append(cands, glucoseCands...)
	if symptomCand != nil {
		cands = append(cands, *symptomCand)
	}
	return jsonx.Obj(
		"pregnancy", jsonx.Obj(
			"week", in.CurrentWeek(), "weeks", max(0, g)/7, "days", max(0, g)%7,
			"trimester", trimesterOn(g, in.Today), "due_date", in.Due.String(), "start_date", in.Start.String(),
		),
		"top_finding", in.topFinding(cands).JSON(in.Copy),
		"sections", jsonx.Obj(
			"weight_gain", weight.JSON(),
			"blood_pressure", bp.JSON(),
			"glucose", glucose.JSON(),
			"kicks", in.kicks().JSON(),
			"symptoms", symptoms.JSON(),
			"visits", in.visits().JSON(),
		),
	)
}
