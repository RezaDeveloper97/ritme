package analysis

import (
	"math"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/cycle/insights"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Top-finding kinds (the first sentence decides).
const (
	FindingCycleFrequent   = "cycle_frequent"
	FindingCycleInfrequent = "cycle_infrequent"
	FindingPeriodProlonged = "period_prolonged"
	FindingCycleIrregular  = "cycle_irregular"
	FindingCycleRegular    = "cycle_regular"
	FindingPattern         = "pattern"
	FindingNotEnoughData   = "not_enough_data" // the «N more days» nudge
	FindingNoData          = "no_data"         // retired by §4.1 (no data = the days nudge); kept for old clients
	FindingKeepLogging     = "keep_logging"    // the not-enough-data nudge of a mode without cycle tracking
)

// TopFinding is «مهم‌ترین یافته»: the best-scoring finding (a correlation adds the not-causal line), or the
// «N more days» nudge when no analysis has enough data yet.
type TopFinding struct {
	Kind            string
	Parts           []Phrase
	Strength, Score float64
	Evidence        civildate.Date
	DaysNeeded      int // the nudge's N (not_enough_data)
}

// cycleFinding is the cycle sentence, in priority order: a median outside the FIGO frequency range,
// a prolonged period, an irregular variation, a regular one. Frequency and duration need ≥ 2 cycles
// (a trend); the variation verdict ≥ 3 (a pattern, BuildCycle).
func cycleFinding(cr CycleReport) (string, *Phrase) {
	if cr.BasedOn >= MinTrendPoints && cr.Applies { // FIGO adult ranges only

		switch {
		case cr.CycleStatus == StatusFrequent:
			return FindingCycleFrequent, &Phrase{Key: "findings.cycle.frequent", Args: []Arg{
				Num("days", *cr.MedianCycle), Num("min", cr.FIGO.CycleMin), Num("max", cr.FIGO.CycleMax)}}
		case cr.CycleStatus == StatusInfrequent:
			return FindingCycleInfrequent, &Phrase{Key: "findings.cycle.infrequent", Args: []Arg{
				Num("days", *cr.MedianCycle), Num("min", cr.FIGO.CycleMin), Num("max", cr.FIGO.CycleMax)}}
		case cr.PeriodStatus == StatusProlonged:
			return FindingPeriodProlonged, &Phrase{Key: "findings.period.prolonged", Args: []Arg{
				Num("days", *cr.MedianPeriod), Num("max", cr.FIGO.PeriodMax)}}
		}
	}
	if !cr.Applies {
		return "", nil
	}
	switch cr.Regularity {
	case RegularityIrregular:
		return FindingCycleIrregular, &Phrase{Key: "findings.cycle.irregular", Args: []Arg{
			Num("days", *cr.Variation), Num("limit", cr.VariationLimit)}}
	case RegularityRegular:
		return FindingCycleRegular, &Phrase{Key: "findings.cycle.regular", Args: []Arg{Num("days", *cr.Variation)}}
	}
	return "", nil
}

// patternFinding is the highlight sentence.
func patternFinding(h *Highlight) *Phrase {
	if h == nil {
		return nil
	}
	base := []Arg{Symptom("symptom", h.Key), Num("cycles", h.Cycles), Num("total", h.Of)}
	switch h.Relation {
	case insights.RelationBeforePeriod:
		return &Phrase{Key: "findings.pattern.before_period", Args: append(base, Num("days", h.Days))}
	case insights.RelationEarly:
		return &Phrase{Key: "findings.pattern.early", Args: append(base, Num("days", h.Days))}
	}
	if h.From == h.To {
		return &Phrase{Key: "findings.pattern.mid_day", Args: append(base, Num("start", h.From))}
	}
	return &Phrase{Key: "findings.pattern.mid", Args: append(base, Num("start", h.From), Num("end", h.To))}
}

// Top finding (§4.1, D-175): every finding the hub can show is a candidate scored strength × recency, recency =
// 0.5^(age / P.TopFindingHalfLifeDays) with age the days from the finding's latest evidence day to today; the hub
// shows the highest (ties: findingOrder). Strength table (0–1, product constants):
//
//	| kind                                   | strength                     |
//	|----------------------------------------|------------------------------|
//	| cycle_frequent, cycle_infrequent       | 1.0                          |
//	| period_prolonged                       | 0.9                          |
//	| cycle_irregular                        | 0.8                          |
//	| correlation (Plus): strong/medium/weak | 0.7 / 0.5 / 0.3              |
//	| pattern                                | 0.6 × cycles ÷ cycles looked |
//	| mood_by_phase (Plus)                   | 0.4                          |
//	| cycle_regular                          | 0.3                          |
//
// Evidence day: the cycle findings — the last day of the newest counted cycle; a pattern — the last day the symptom
// was logged; a correlation / mood by phase — its latest paired day.
const (
	FindingCorrelation = "correlation"
	FindingMoodByPhase = "mood_by_phase"
)

// findingOrder breaks score ties.
// The TTC and pregnancy kinds (finding_hubs.go) slot in by their strength.
var findingOrder = []string{
	FindingPregBPSevere, FindingCycleFrequent, FindingCycleInfrequent, FindingPregBPHigh, FindingPeriodProlonged,
	FindingCycleIrregular, FindingCorrelation, FindingPregGainAbove, FindingPregGainBelow,
	FindingPregGlucoseAbove, FindingPattern, FindingTTCMucus, FindingPregSymptomTop, FindingMoodByPhase,
	FindingCycleRegular, FindingPregGainWithin,
}

var findingStrength = map[string]float64{
	FindingCycleFrequent: 1, FindingCycleInfrequent: 1, FindingPeriodProlonged: 0.9, FindingCycleIrregular: 0.8,
	FindingMoodByPhase: 0.4, FindingCycleRegular: 0.3,
	// TTC / pregnancy hubs (finding_hubs.go)
	FindingPregBPSevere: 1, FindingPregBPHigh: 0.9, FindingPregGainAbove: 0.7,
	FindingPregGainBelow: 0.7, FindingPregGlucoseAbove: 0.6, FindingPregGainWithin: 0.3,
}

var correlationStrength = map[string]float64{StrengthStrong: 0.7, StrengthMedium: 0.5, StrengthWeak: 0.3}

// patternStrengthMax scales a symptom pattern by the share of cycles it showed up in.
const patternStrengthMax = 0.6

// Candidate is one scored finding.
type Candidate struct {
	Kind     string
	Strength float64
	Evidence civildate.Date
	Parts    []Phrase
}

// Recency is 0.5^(age / half-life); age is clamped at 0 (a zero evidence day counts as today).
func Recency(evidence, today civildate.Date) float64 {
	if evidence.IsZero() {
		return 1
	}
	age := max(0, evidence.DiffDays(today))
	return math.Pow(0.5, float64(age)/P.TopFindingHalfLifeDays)
}

// Score is strength × recency.
func (c Candidate) Score(today civildate.Date) float64 {
	return c.Strength * Recency(c.Evidence, today)
}

// PickTopFinding returns the highest-scoring candidate (ties: findingOrder, then the input order); ok false for none.
func PickTopFinding(cands []Candidate, today civildate.Date) (Candidate, bool) {
	best, found := Candidate{}, false
	for _, c := range cands {
		if !found {
			best, found = c, true
			continue
		}
		cs, bs := round(c.Score(today), 6), round(best.Score(today), 6)
		if cs > bs || (cs == bs && slices.Index(findingOrder, c.Kind) < slices.Index(findingOrder, best.Kind)) {
			best = c
		}
	}
	return best, found
}

// lastSymptomDay is the last range day key was logged (zero when never).
func (in *Input) lastSymptomDay(key string) civildate.Date {
	var last civildate.Date
	for d, day := range in.Days {
		if _, ok := day.Symptoms[key]; ok && in.Range.Contains(d) && d.After(last) {
			last = d
		}
	}
	return last
}

// candidates are the hub's findings (corr nil = not entitled).
func (in *Input) candidates(cr CycleReport, sr SymptomsReport, corr *CorrelationsReport) []Candidate {
	var out []Candidate
	if kind, p := cycleFinding(cr); p != nil {
		var ev civildate.Date
		for _, row := range cr.Cycles { // newest first
			if row.Counted {
				ev = row.End
				break
			}
		}
		out = append(out, Candidate{Kind: kind, Strength: findingStrength[kind], Evidence: ev, Parts: []Phrase{*p}})
	}
	if h := sr.Highlight; h != nil && h.Of > 0 {
		out = append(out, Candidate{Kind: FindingPattern, Strength: patternStrengthMax * float64(h.Cycles) / float64(h.Of),
			Evidence: in.lastSymptomDay(h.Key), Parts: []Phrase{*patternFinding(h)}})
	}
	if corr != nil {
		for _, c := range corr.Items {
			if c.Status != CorrReady || c.Phrase == nil || c.Key == CorrPhaseMood { // phase_mood: the mood-by-phase card
				continue
			}
			out = append(out, Candidate{Kind: FindingCorrelation, Strength: correlationStrength[c.Strength], Evidence: c.Last,
				Parts: []Phrase{*c.Phrase, NotCausalPhrase()}})
		}
		if m := corr.MoodByPhase; m.Ready() && m.Phrase != nil {
			out = append(out, Candidate{Kind: FindingMoodByPhase, Strength: findingStrength[FindingMoodByPhase],
				Evidence: m.Last, Parts: []Phrase{*m.Phrase}})
		}
	}
	return out
}

// daysNeeded is the nudge's N (§4.1 «برای دیدن الگوها N روز دیگر ثبت کن»): the days to the nearest unmet minimum among
// the cycle length (P.CycleLengthMinCycles), the symptom pattern (P.SymptomPatternMinCycles) — cycles converted with
// the effective cycle length, skipped without cycle tracking — and, when entitled, the correlations
// (P.CorrMinPairedDays) and mood by phase (P.MoodPhaseMinDays). 0 = no unmet minimum to name.
func (in *Input) daysNeeded(cr CycleReport, sr SymptomsReport, corr *CorrelationsReport) int {
	best := 0
	consider := func(n int) {
		if n > 0 && (best == 0 || n < best) {
			best = n
		}
	}
	if !in.NoCycleNudge {
		cycleLen, _ := in.typical()
		consider(daysForCycles(P.CycleLengthMinCycles-cr.BasedOn, cycleLen))
		consider(daysForCycles(P.SymptomPatternMinCycles-sr.PatternCycles, cycleLen))
	}
	if corr != nil {
		for _, c := range corr.Items {
			if c.Status == CorrNotEnoughData {
				consider(c.DaysNeeded)
			}
		}
		if !corr.MoodByPhase.Ready() {
			consider(corr.MoodByPhase.DaysNeeded)
		}
	}
	return best
}

// BuildTopFinding picks the hub's top finding: the best-scoring candidate, else the days nudge (keep_logging when no
// minimum can be named, e.g. menopause without Plus).
func BuildTopFinding(in *Input, cr CycleReport, sr SymptomsReport, corr *CorrelationsReport) TopFinding {
	if c, ok := PickTopFinding(in.candidates(cr, sr, corr), in.Today); ok {
		return TopFinding{Kind: c.Kind, Parts: c.Parts, Strength: c.Strength, Score: c.Score(in.Today), Evidence: c.Evidence}
	}
	n := in.daysNeeded(cr, sr, corr)
	if n == 0 {
		return TopFinding{Kind: FindingKeepLogging, Parts: []Phrase{{Key: "findings.keep_logging"}}}
	}
	return TopFinding{Kind: FindingNotEnoughData, DaysNeeded: n, Parts: []Phrase{{Key: "findings.not_enough_days",
		Args: []Arg{Num("days", n)}}}}
}

// JSON is {kind, parts: [{key, params, text}], text}.
func (f TopFinding) JSON(c *Copy) *jsonx.OrderedMap {
	parts := make([]*jsonx.OrderedMap, 0, len(f.Parts))
	texts := make([]string, 0, len(f.Parts))
	for _, p := range f.Parts {
		parts = append(parts, p.JSON(c))
		if t := c.Render(p); t != "" {
			texts = append(texts, t)
		}
	}
	var strength, score, days any
	if f.Kind != FindingNotEnoughData && f.Kind != FindingKeepLogging {
		strength, score = round(f.Strength, 2), round(f.Score, 3)
	}
	if f.DaysNeeded > 0 {
		days = f.DaysNeeded
	}
	return jsonx.Obj("kind", f.Kind, "parts", parts, "text", strings.Join(texts, " "),
		"strength", strength, "score", score, "evidence_date", dateOrNil(f.Evidence), "days_needed", days)
}
