package analysis

// TTC analysis (B-N3-11, Night & Bloom An_Hub_TTC / An_Fertility): GET /api/v1/analysis/ttc (the hub) and
// GET /api/v1/analysis/fertility?cycle= (one cycle's BBT chart). Pure functions of a TTCInput, golden-tested
// like the other reports. Descriptive, never diagnostic: the referral line is general advice, every number
// says where it comes from (BBT, LH test or the calendar estimate).
//
// Where the data is (QUESTIONS #80): BBT and intercourse are taxonomy v2 rows (health_log_entries, synced
// two-way with daily_health_logs); LH tests and cervical mucus are in fertility_logs (the /fertility log)
// *and* in health_log_entries (the v2 log sheet: measurements.lh_test, discharge.consistency). The two are
// not synced, so TTCSignals merges both, keeping the stronger LH result and the more fertile mucus of a day.
//
// Ovulation confirmation — the «سه بالای شش» (three-over-six) thermal-shift rule, the same implementation
// as the /fertility/bbt chart (internal/fertility/bbt, so both screens always agree):
//   - readings are taken in order, days without a reading are skipped (not counted);
//   - the coverline is the highest of the 6 readings before a candidate reading;
//   - the shift is confirmed when the candidate is at least 0.20 °C above the coverline and it and the next
//     two readings (3 in all) all stay above the coverline;
//   - ovulation is the day before the first high reading; it is confirmed only after the third high reading
//     (BBT confirms ovulation after the fact, it never predicts it).
// Source: the WHO / Marshall «three over six» rule (Marshall J. «A field trial of the basal-body-temperature
// method of regulating births.» Lancet 1968;2(7563):8–10; WHO Task Force on Methods for the Determination of
// the Fertile Period, «A prospective multicentre trial of the ovulation method» 1981; the symptothermal
// «higher temperature ≥ 0.2 °C over the coverline of the previous six» of Frank-Herrmann P et al., Hum
// Reprod 2007;22(5):1310–1319).
//
// Referral advice (not a diagnosis): seek an evaluation after 12 months of regular unprotected intercourse
// without pregnancy under age 35, after 6 months from age 35 (ASRM Practice Committee, «Definitions of
// infertility and recurrent pregnancy loss: a committee opinion.» Fertil Steril 2020;113(3):533–535; ACOG
// Committee Opinion 781, «Infertility Workup for the Women's Health Specialist», 2019).
//
// Luteal phase (from a BBT-confirmed ovulation of a completed cycle: cycle length − ovulation day): ≤ 10
// days is short (ASRM Practice Committee, «Diagnosis and treatment of luteal phase deficiency: a committee
// opinion.» Fertil Steril 2021;115(6):1416–1423), 11–17 days the usual range (Lenton EA et al., «Normal
// variation in the length of the luteal phase of the menstrual cycle.» Br J Obstet Gynaecol 1984;91:685–689).

import (
	"slices"

	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// TTC constants.
const (
	// ReferralAge splits the referral advice: under it 12 months of trying, from it 6 months.
	ReferralAge           = 35
	ReferralMonthsUnder35 = 12
	ReferralMonthsFrom35  = 6
	// LutealShortMax: a luteal phase of this many days or fewer is short; LutealLongMin and above is
	// longer than usual (11–17 = usual).
	LutealShortMax = 10
	LutealLongMin  = 18
	// LHOvulationHoursMin / Max: ovulation usually follows a positive LH test by 24–36 hours (copy only).
	LHOvulationHoursMin = 24
	LHOvulationHoursMax = 36
	// TTCMaxCycles caps the trying cycles analysed (about a year; the trying count itself is never capped);
	// TTCFallbackCycles are the recent cycles shown before any TTC log exists.
	TTCMaxCycles      = 13
	TTCFallbackCycles = 6
	// mucusLookback: an egg-white day counts for the mucus pattern when it lies in the 6 days up to the
	// ovulation day (the fertile window).
	mucusLookback = fertileDaysBeforeOvulation
)

// Ovulation sources.
const (
	OvulationBBT      = "bbt"      // confirmed by the three-over-six shift
	OvulationLH       = "lh"       // the day after the first positive LH test
	OvulationEstimate = "estimate" // the calendar estimate (next start − 14)
)

// Timing-dot phases.
const (
	DotPeriod    = "period"
	DotFertile   = "fertile"
	DotOvulation = "ovulation"
	DotOther     = "other"
)

// Luteal statuses.
const (
	LutealShort  = "short"
	LutealNormal = "normal"
	LutealLong   = "long"
)

// Mucus codes, least to most fertile (fertility_logs: dry | sticky | creamy | egg_white; v2
// discharge.consistency: none | sticky | creamy | watery | egg_white, none read as dry).
var mucusRank = map[string]int{"dry": 0, "sticky": 1, "creamy": 2, "watery": 3, "egg_white": 4}

// LH results, weakest first.
var lhRank = map[string]int{"negative": 0, "faint": 1, "positive": 2}

// TTCSignals are the TTC logs by day.
type TTCSignals struct {
	BBT         map[civildate.Date]int // hundredths °C
	LH          map[civildate.Date]string
	Mucus       map[civildate.Date]string
	Intercourse map[civildate.Date]string // protected | unprotected
}

// NewTTCSignals is an empty set.
func NewTTCSignals() *TTCSignals {
	return &TTCSignals{BBT: map[civildate.Date]int{}, LH: map[civildate.Date]string{},
		Mucus: map[civildate.Date]string{}, Intercourse: map[civildate.Date]string{}}
}

func (s *TTCSignals) addLH(d civildate.Date, v string) {
	if _, ok := lhRank[v]; !ok {
		return
	}
	if cur, ok := s.LH[d]; !ok || lhRank[v] > lhRank[cur] {
		s.LH[d] = v
	}
}

func (s *TTCSignals) addMucus(d civildate.Date, v string) {
	if v == "none" {
		v = "dry"
	}
	if _, ok := mucusRank[v]; !ok {
		return
	}
	if cur, ok := s.Mucus[d]; !ok || mucusRank[v] > mucusRank[cur] {
		s.Mucus[d] = v
	}
}

// AddEntries reads the taxonomy v2 rows: measurements.bbt / lh_test, discharge.consistency, sex.intercourse.
func (s *TTCSignals) AddEntries(rows []DayEntries) {
	for _, r := range rows {
		for _, e := range r.Entries {
			switch e.ParamKey() {
			case "measurements.bbt":
				if e.Num.Valid {
					if v, err := bbt.ParseValue(e.Num.String); err == nil {
						s.BBT[r.Date] = v
					}
				}
			case "measurements.lh_test":
				s.addLH(r.Date, e.Code.String)
			case "discharge.consistency":
				s.addMucus(r.Date, e.Code.String)
			case "sex.intercourse":
				if e.Code.Valid && e.Code.String != "" {
					s.Intercourse[r.Date] = e.Code.String
				}
			}
		}
	}
}

// AddFertilityLog merges one fertility_logs day ("" = not logged).
func (s *TTCSignals) AddFertilityLog(d civildate.Date, lh, mucus string) {
	s.addLH(d, lh)
	s.addMucus(d, mucus)
}

// TTCInput is the hub's / detail's input: the shared Input (history, profile, Plus, copy, today) plus the
// TTC logs and the first TTC log day (zero = none yet).
type TTCInput struct {
	*Input
	Signals     *TTCSignals
	FirstSignal civildate.Date
}

// TTCCycle is one analysed cycle.
type TTCCycle struct {
	Cycle
	Index int // 1-based within the analysed list
	// End is the cycle's last day (today for the current one); Days its days so far.
	End  civildate.Date
	Days int
	BBT  bbt.Cycle
	// HighDays are the cycle days of the three high readings of a confirmed shift.
	HighDays []int
	// LHDay is the first positive LH test's cycle day (0 = none).
	LHDay int
	// Tests are the cycle's LH tests by cycle day.
	Tests []DayValue
	// Ovulation day and its source ("" = unknown).
	OvulationDay    int
	OvulationSource string
	// Fertile window (cycle days; 0 = none): the ovulation day and the 5 days before it.
	FertileFrom, FertileTo int
	// Intercourse are the cycle days with unprotected intercourse.
	Intercourse []int
	// Luteal is length − BBT ovulation day of a completed cycle (0 = unknown).
	Luteal int
	// MucusLead is the days from the last egg-white day of the fertile window to the ovulation day (nil = none).
	MucusLead *int
	// PeriodLen is the bleed shown on the chart (logged, else the typical period).
	PeriodLen int
}

// DayValue is a cycle day and a code.
type DayValue struct {
	Day   int
	Value string
}

// tryingStart is the index (into cycles) of the cycle holding the first TTC log; -1 without one.
func tryingStart(cycles []Cycle, first civildate.Date) int {
	if first.IsZero() || len(cycles) == 0 {
		return -1
	}
	idx := 0
	for i, c := range cycles {
		if !c.Start.After(first) {
			idx = i
		}
	}
	return idx
}

// monthsBetween is the whole calendar months from a to b.
func monthsBetween(a, b civildate.Date) int {
	m := (b.Year-a.Year)*12 + int(b.Month) - int(a.Month)
	if b.Day < a.Day {
		m--
	}
	return max(0, m)
}

// TTCReport is the hub before serialisation.
type TTCReport struct {
	in *TTCInput
	// Since is the first trying cycle's start (zero = no TTC log yet).
	Since        civildate.Date
	TryingCycles int
	Months       int
	Confirmed    int // analysed cycles with a BBT-confirmed ovulation
	Cycles       []TTCCycle
	Trying       bool // Cycles are trying cycles (false = the recent-cycles fallback)
}

// analysedCycles picks the cycles of the hub and the detail: the trying cycles (newest TTCMaxCycles) or,
// before any TTC log, the newest TTCFallbackCycles cycles.
func (in *TTCInput) analysedCycles() (all []Cycle, picked []Cycle, start int) {
	all = in.cycles()
	start = tryingStart(all, in.FirstSignal)
	from := max(0, len(all)-TTCFallbackCycles)
	if start >= 0 {
		from = max(start, len(all)-TTCMaxCycles)
	}
	return all, all[from:], start
}

// AnalysedFrom is the first day the TTC logs must cover (zero when there is no cycle).
func (in *TTCInput) AnalysedFrom() civildate.Date {
	_, picked, _ := in.analysedCycles()
	if len(picked) == 0 {
		return civildate.Date{}
	}
	return picked[0].Start
}

// BuildTTC analyses the cycles.
func BuildTTC(in *TTCInput) TTCReport {
	all, picked, start := in.analysedCycles()
	r := TTCReport{in: in, Trying: start >= 0}
	if start >= 0 {
		r.Since = all[start].Start
		r.TryingCycles = len(all) - start
		r.Months = monthsBetween(r.Since, in.Today)
	}
	typCycle, typPeriod := in.typical()
	readings := make([]bbt.Reading, 0, len(in.Signals.BBT))
	for _, d := range sortedDates(in.Signals.BBT) {
		readings = append(readings, bbt.Reading{Date: d, Value: in.Signals.BBT[d]})
	}
	for i, c := range picked {
		tc := in.analyseCycle(c, readings, typCycle, typPeriod)
		tc.Index = i + 1
		if tc.OvulationSource == OvulationBBT {
			r.Confirmed++
		}
		r.Cycles = append(r.Cycles, tc)
	}
	return r
}

func (in *TTCInput) analyseCycle(c Cycle, readings []bbt.Reading, typCycle, typPeriod int) TTCCycle {
	end := c.End(in.Today)
	tc := TTCCycle{Cycle: c, End: end, Days: c.Start.DiffDays(end) + 1, PeriodLen: c.PeriodDays}
	if tc.PeriodLen == 0 {
		tc.PeriodLen = typPeriod
	}
	tc.BBT = bbt.Analyze(bbt.CycleInput{Start: c.Start, End: end}, readings, in.Today)
	if tc.BBT.ShiftDay != nil {
		for i, p := range tc.BBT.Points {
			if p.CycleDay == *tc.BBT.ShiftDay {
				for j := i; j < i+bbt.HighReadings && j < len(tc.BBT.Points); j++ {
					tc.HighDays = append(tc.HighDays, tc.BBT.Points[j].CycleDay)
				}
				break
			}
		}
	}
	tc.Tests = []DayValue{}
	tc.Intercourse = []int{}
	for n := 1; n <= tc.Days; n++ {
		d := c.Start.AddDays(n - 1)
		if v, ok := in.Signals.LH[d]; ok {
			tc.Tests = append(tc.Tests, DayValue{Day: n, Value: v})
			if v == "positive" && tc.LHDay == 0 {
				tc.LHDay = n
			}
		}
		if in.Signals.Intercourse[d] == "unprotected" {
			tc.Intercourse = append(tc.Intercourse, n)
		}
	}
	switch {
	case tc.BBT.OvulationDay() != nil:
		tc.OvulationDay, tc.OvulationSource = *tc.BBT.OvulationDay(), OvulationBBT
	case tc.LHDay > 0 && tc.LHDay < tc.Days:
		tc.OvulationDay, tc.OvulationSource = tc.LHDay+1, OvulationLH
	default:
		length := c.Length
		if c.Current {
			length = typCycle
		}
		if insights.ValidCycleLength(length) {
			tc.OvulationDay, tc.OvulationSource = NewLayout(length, tc.PeriodLen).OvulationDay, OvulationEstimate
		}
	}
	if tc.OvulationDay > 0 {
		tc.FertileTo = tc.OvulationDay
		tc.FertileFrom = max(1, tc.OvulationDay-fertileDaysBeforeOvulation)
		for n := tc.OvulationDay; n >= max(1, tc.OvulationDay-mucusLookback); n-- {
			if n > tc.Days {
				continue
			}
			if in.Signals.Mucus[c.Start.AddDays(n-1)] == "egg_white" {
				lead := tc.OvulationDay - n
				tc.MucusLead = &lead
				break
			}
		}
	}
	if !c.Current && tc.OvulationSource == OvulationBBT {
		tc.Luteal = c.Length - tc.OvulationDay
	}
	return tc
}

// current is the last analysed cycle when it is the running one.
func (r TTCReport) current() *TTCCycle {
	if n := len(r.Cycles); n > 0 && r.Cycles[n-1].Current {
		return &r.Cycles[n-1]
	}
	return nil
}

// LutealStatus of a luteal length.
func LutealStatus(days int) string {
	switch {
	case days <= LutealShortMax:
		return LutealShort
	case days >= LutealLongMin:
		return LutealLong
	}
	return LutealNormal
}

// referral is the advice threshold and band for an age (0 = unknown → the under-35 threshold).
func referral(age int) (months int, band string) {
	switch {
	case age <= 0:
		return ReferralMonthsUnder35, "unknown"
	case age >= ReferralAge:
		return ReferralMonthsFrom35, "from_35"
	}
	return ReferralMonthsUnder35, "under_35"
}

func (r TTCReport) tryingJSON() *jsonx.OrderedMap {
	in := r.in
	age := in.Age()
	threshold, band := referral(age)
	due := r.Trying && r.Months >= threshold
	var summary Phrase
	switch {
	case !r.Trying:
		summary = Phrase{Key: "ttc.summary.none"}
	case r.Confirmed > 0:
		summary = Phrase{Key: "ttc.summary.confirmed", Args: []Arg{Num("cycles", r.Confirmed)}}
	default:
		summary = Phrase{Key: "ttc.summary.not_confirmed"}
	}
	advice := Phrase{Key: "ttc.advice.general", Args: []Arg{
		Num("age", ReferralAge), Num("under", ReferralMonthsUnder35), Num("over", ReferralMonthsFrom35),
	}}
	if due {
		advice = Phrase{Key: "ttc.advice.due_" + band, Args: []Arg{Num("months", r.Months), Num("threshold", threshold), Num("age", ReferralAge)}}
		if band == "unknown" {
			advice.Key = "ttc.advice.due_under_35"
		}
	}
	return jsonx.Obj(
		"since", dateOrNil(r.Since),
		"cycles", r.TryingCycles,
		"months", r.Months,
		"confirmed_cycles", r.Confirmed,
		"age", positiveOrNil(age),
		"referral", jsonx.Obj("age_band", band, "threshold_months", threshold, "due", due),
		"summary", summary.JSON(in.Copy),
		"advice", advice.JSON(in.Copy),
	)
}

func pointsJSON(points []bbt.Point, withDate bool) []any {
	out := make([]any, 0, len(points))
	for _, p := range points {
		if withDate {
			out = append(out, jsonx.Obj("day", p.CycleDay, "date", p.Date.String(), "value", bbt.Format(p.Value)))
		} else {
			out = append(out, jsonx.Obj("day", p.CycleDay, "value", bbt.Format(p.Value)))
		}
	}
	return out
}

func tempOrNil(v *int) any {
	if v == nil {
		return nil
	}
	return bbt.Format(*v)
}

func testsJSON(tests []DayValue) []any {
	out := make([]any, 0, len(tests))
	for _, t := range tests {
		out = append(out, jsonx.Obj("day", t.Day, "value", t.Value))
	}
	return out
}

// bbtCard is the current cycle's BBT line and confirmation (free).
func (r TTCReport) bbtCard() Section {
	c := r.current()
	if c == nil {
		return freeSection(false, nil)
	}
	return freeSection(len(c.BBT.Points) >= MinTrendPoints, jsonx.Obj(
		"cycle_start", c.Start.String(),
		"points", pointsJSON(c.BBT.Points, false),
		"coverline", tempOrNil(c.BBT.Coverline),
		"shift_day", ptrOrNil(c.BBT.ShiftDay),
		"ovulation_day", ptrOrNil(c.BBT.OvulationDay()),
		"confirmed", c.BBT.ShiftDay != nil,
	))
}

// lhPhrase describes a cycle's LH tests.
func lhPhrase(c *TTCCycle) Phrase {
	if c.LHDay > 0 {
		return Phrase{Key: "ttc.lh.positive", Args: []Arg{Num("day", c.LHDay), Num("min", LHOvulationHoursMin), Num("max", LHOvulationHoursMax)}}
	}
	for _, t := range c.Tests {
		if t.Value == "faint" {
			return Phrase{Key: "ttc.lh.faint", Args: []Arg{Num("day", t.Day)}}
		}
	}
	if len(c.Tests) > 0 {
		return Phrase{Key: "ttc.lh.negative", Args: []Arg{Num("count", len(c.Tests))}}
	}
	return Phrase{Key: "ttc.lh.none"}
}

// lhCard is the current cycle's LH tests (free).
func (r TTCReport) lhCard() Section {
	c := r.current()
	if c == nil {
		return freeSection(false, nil)
	}
	return freeSection(len(c.Tests) > 0, jsonx.Obj(
		"cycle_start", c.Start.String(),
		"tests", testsJSON(c.Tests),
		"positive_day", positiveOrNil(c.LHDay),
		"text", lhPhrase(c).JSON(r.in.Copy),
	))
}

// timingCycle is the newest analysed cycle whose fertile window has started.
func (r TTCReport) timingCycle() *TTCCycle {
	for i := len(r.Cycles) - 1; i >= 0; i-- {
		c := &r.Cycles[i]
		if c.FertileFrom > 0 && c.FertileFrom <= c.Days {
			return c
		}
	}
	return nil
}

// dotLength is how many days the timing dots / chart span: the completed length, or for the current
// cycle the larger of the days so far and the typical length.
func (r TTCReport) dotLength(c *TTCCycle) int {
	if !c.Current {
		return c.Length
	}
	typ, _ := r.in.typical()
	return max(c.Days, typ)
}

// timingCard: unprotected-intercourse days in the fertile window (Plus).
func (r TTCReport) timingCard() Section {
	return plusSection(r.in.DeepAnalysis, func() (bool, any) {
		c := r.timingCycle()
		if c == nil {
			return false, nil
		}
		inWindow := 0
		for _, d := range c.Intercourse {
			if d >= c.FertileFrom && d <= c.FertileTo {
				inWindow++
			}
		}
		length := r.dotLength(c)
		days := make([]any, 0, length)
		for n := 1; n <= length; n++ {
			phase := DotOther
			switch {
			case n <= c.PeriodLen:
				phase = DotPeriod
			case n == c.OvulationDay:
				phase = DotOvulation
			case n >= c.FertileFrom && n <= c.FertileTo:
				phase = DotFertile
			}
			days = append(days, jsonx.Obj("day", n, "phase", phase,
				"intercourse", slices.Contains(c.Intercourse, n), "future", n > c.Days))
		}
		return true, jsonx.Obj(
			"cycle_start", c.Start.String(),
			"window_from", c.FertileFrom,
			"window_to", c.FertileTo,
			"window_days", c.FertileTo-c.FertileFrom+1,
			"window_source", c.OvulationSource,
			"in_window", inWindow,
			"days", days,
		)
	})
}

// mucusCard: in how many cycles egg-white mucus came before ovulation, and how many days before (Plus).
func (r TTCReport) mucusCard() Section {
	return plusSection(r.in.DeepAnalysis, func() (bool, any) {
		var leads []int
		analysed := 0
		for _, c := range r.Cycles {
			if c.OvulationDay == 0 || c.OvulationDay > c.Days {
				continue
			}
			analysed++
			if c.MucusLead != nil {
				leads = append(leads, *c.MucusLead)
			}
		}
		var text Phrase
		var lo, hi any
		switch {
		case len(leads) == 0:
			text = Phrase{Key: "ttc.mucus.none"}
		default:
			mn, mx := slices.Min(leads), slices.Max(leads)
			lo, hi = mn, mx
			args := []Arg{Num("cycles", len(leads)), Num("min", mn), Num("max", mx)}
			switch {
			case mx == 0:
				text = Phrase{Key: "ttc.mucus.same_day", Args: args}
			case mn == mx:
				text = Phrase{Key: "ttc.mucus.one", Args: args}
			case mn == 0:
				text = Phrase{Key: "ttc.mucus.up_to", Args: args}
			default:
				text = Phrase{Key: "ttc.mucus.range", Args: args}
			}
		}
		return len(leads) > 0, jsonx.Obj(
			"cycles_with_egg_white", len(leads),
			"cycles_analysed", analysed,
			"lead_min", lo,
			"lead_max", hi,
			"text", text.JSON(r.in.Copy),
		)
	})
}

// lutealDays are the BBT luteal lengths of the analysed completed cycles.
func (r TTCReport) lutealDays() []int {
	var out []int
	for _, c := range r.Cycles {
		if c.Luteal > 0 {
			out = append(out, c.Luteal)
		}
	}
	return out
}

// lutealCard: the median BBT luteal phase (Plus).
func (r TTCReport) lutealCard() Section {
	return plusSection(r.in.DeepAnalysis, func() (bool, any) {
		days := r.lutealDays()
		m := insights.Median(days)
		if m == nil {
			return false, jsonx.Obj("days", nil, "status", nil, "cycles", 0, "source", OvulationBBT)
		}
		return true, jsonx.Obj("days", *m, "status", LutealStatus(*m), "cycles", len(days), "source", OvulationBBT,
			"usual_min", LutealShortMax+1, "usual_max", LutealLongMin-1)
	})
}

// regularityCard: the analysed cycles' lengths (the current one so far) and the FIGO variation (free).
func (r TTCReport) regularityCard() Section {
	if len(r.Cycles) == 0 {
		return freeSection(false, nil)
	}
	bars := make([]any, 0, len(r.Cycles))
	var lens []int
	for _, c := range r.Cycles {
		length := c.Length
		if c.Current {
			length = c.Days
		} else if insights.ValidCycleLength(c.Length) {
			lens = append(lens, c.Length)
		}
		bars = append(bars, jsonx.Obj("index", c.Index, "start", c.Start.String(), "length", length, "current", c.Current))
	}
	limit := FIGOVariationLimit(r.in.Age())
	status := RegularityNotEnoughData
	var variation any
	if len(lens) >= MinTrendPoints {
		v := slices.Max(lens) - slices.Min(lens)
		variation = v
		if len(lens) >= MinPatternCycles {
			status = RegularityRegular
			if v > limit {
				status = RegularityIrregular
			}
		}
	}
	var median any
	if m := insights.Median(lens); m != nil {
		median = *m
	}
	return freeSection(len(lens) >= MinTrendPoints, jsonx.Obj(
		"cycles", bars,
		"median", median,
		"variation", variation,
		"limit", limit,
		"status", status,
	))
}

func (c TTCCycle) summaryJSON() *jsonx.OrderedMap {
	var length, source any
	if !c.Current {
		length = c.Length
	}
	if c.OvulationSource != "" {
		source = c.OvulationSource
	}
	return jsonx.Obj(
		"index", c.Index,
		"start", c.Start.String(),
		"end", c.End.String(),
		"current", c.Current,
		"length", length,
		"ovulation_day", positiveOrNil(c.OvulationDay),
		"ovulation_source", source,
		"lh_day", positiveOrNil(c.LHDay),
		"luteal_days", positiveOrNil(c.Luteal),
	)
}

// JSON is the GET /analysis/ttc body.
func (r TTCReport) JSON() *jsonx.OrderedMap {
	cycles := make([]any, 0, len(r.Cycles))
	for _, c := range r.Cycles {
		cycles = append(cycles, c.summaryJSON())
	}
	return jsonx.Obj(
		"trying", r.tryingJSON(),
		"bbt", r.bbtCard().JSON(),
		"lh", r.lhCard().JSON(),
		"timing", r.timingCard().JSON(),
		"mucus", r.mucusCard().JSON(),
		"luteal", r.lutealCard().JSON(),
		"regularity", r.regularityCard().JSON(),
		"cycles", cycles,
	)
}

// FindCycle is the analysed cycle starting on start (the current / newest one for the zero Date).
func (r TTCReport) FindCycle(start civildate.Date) (int, bool) {
	if len(r.Cycles) == 0 {
		return -1, false
	}
	if start.IsZero() {
		return len(r.Cycles) - 1, true
	}
	for i, c := range r.Cycles {
		if c.Start == start {
			return i, true
		}
	}
	return -1, false
}

// explanation is «چطور تأیید شد؟» for a cycle.
func explanation(c *TTCCycle) Phrase {
	switch {
	case c.OvulationSource == OvulationBBT && len(c.HighDays) > 0:
		return Phrase{Key: "ttc.fertility.how.confirmed", Args: []Arg{
			Num("from", c.HighDays[0]), Num("to", c.HighDays[len(c.HighDays)-1]),
			Num("baseline", bbt.BaselineReadings), Num("day", c.OvulationDay),
		}}
	case len(c.BBT.Points) < bbt.BaselineReadings+bbt.HighReadings:
		return Phrase{Key: "ttc.fertility.how.few", Args: []Arg{
			Num("need", bbt.BaselineReadings+bbt.HighReadings), Num("count", len(c.BBT.Points)),
		}}
	case c.OvulationSource == OvulationLH:
		return Phrase{Key: "ttc.fertility.how.lh", Args: []Arg{Num("day", c.OvulationDay), Num("lh", c.LHDay)}}
	}
	return Phrase{Key: "ttc.fertility.how.waiting", Args: []Arg{Num("baseline", bbt.BaselineReadings)}}
}

// FertilityJSON is the GET /analysis/fertility body for the analysed cycle at index i.
func (r TTCReport) FertilityJSON(i int) *jsonx.OrderedMap {
	c := &r.Cycles[i]
	var prev, next, length, window any
	if i > 0 {
		prev = r.Cycles[i-1].Start.String()
	}
	if i < len(r.Cycles)-1 {
		next = r.Cycles[i+1].Start.String()
	}
	if !c.Current {
		length = c.Length
	}
	if c.FertileFrom > 0 {
		window = jsonx.Obj("from", c.FertileFrom, "to", c.FertileTo)
	}
	var source any
	if c.OvulationSource != "" {
		source = c.OvulationSource
	}
	var lhBefore any
	if c.LHDay > 0 && c.OvulationDay > 0 && c.OvulationSource != OvulationEstimate {
		lhBefore = c.OvulationDay - c.LHDay
	}
	luteal := plusSection(r.in.DeepAnalysis, func() (bool, any) {
		if c.Luteal == 0 {
			return false, jsonx.Obj("days", nil, "status", nil)
		}
		return true, jsonx.Obj("days", c.Luteal, "status", LutealStatus(c.Luteal))
	})
	highDays := c.HighDays
	if highDays == nil {
		highDays = []int{}
	}
	return jsonx.Obj(
		"cycle", jsonx.Obj(
			"index", c.Index,
			"total", len(r.Cycles),
			"start", c.Start.String(),
			"end", c.End.String(),
			"current", c.Current,
			"length", length,
			"days", c.Days,
			"span", r.dotLength(c),
		),
		"prev_start", prev,
		"next_start", next,
		"chart", jsonx.Obj(
			"points", pointsJSON(c.BBT.Points, true),
			"coverline", tempOrNil(c.BBT.Coverline),
			"shift_day", ptrOrNil(c.BBT.ShiftDay),
			"high_days", highDays,
			"confirmed", c.BBT.ShiftDay != nil,
		),
		"period_days", c.PeriodLen,
		"fertile_window", window,
		"ovulation", jsonx.Obj(
			"day", positiveOrNil(c.OvulationDay),
			"source", source,
			"confirmed", c.OvulationSource == OvulationBBT,
		),
		"lh", jsonx.Obj(
			"tests", testsJSON(c.Tests),
			"positive_day", positiveOrNil(c.LHDay),
			"days_before_ovulation", lhBefore,
		),
		"intercourse_days", c.Intercourse,
		"luteal", luteal.JSON(),
		"explanation", explanation(c).JSON(r.in.Copy),
		"tip", Phrase{Key: "ttc.fertility.tip"}.JSON(r.in.Copy),
	)
}
