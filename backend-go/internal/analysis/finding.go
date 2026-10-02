package analysis

import (
	"strings"

	"github.com/ritme/backend-go/internal/cycle/insights"
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
	FindingNotEnoughData   = "not_enough_data"
	FindingNoData          = "no_data"
	FindingKeepLogging     = "keep_logging" // the not-enough-data nudge of a mode without cycle tracking
)

// TopFinding is «مهم‌ترین یافته»: up to two sentences — the cycle verdict and the strongest symptom
// pattern — or a nudge when there is not enough data yet.
type TopFinding struct {
	Kind  string
	Parts []Phrase
}

// cycleFinding is the cycle sentence, in priority order: a median outside the FIGO frequency range,
// a prolonged period, an irregular variation, a regular one. Frequency and duration need ≥ 2 cycles
// (a trend); the variation verdict ≥ 3 (a pattern, BuildCycle).
func cycleFinding(cr CycleReport) (string, *Phrase) {
	if cr.BasedOn >= MinTrendPoints && cr.Applies { // FIGO adult ranges only

		switch {
		case cr.CycleStatus == StatusFrequent:
			return FindingCycleFrequent, &Phrase{Key: "findings.cycle.frequent", Args: []Arg{
				Num("days", *cr.MedianCycle), Num("min", FIGOCycleMin), Num("max", FIGOCycleMax)}}
		case cr.CycleStatus == StatusInfrequent:
			return FindingCycleInfrequent, &Phrase{Key: "findings.cycle.infrequent", Args: []Arg{
				Num("days", *cr.MedianCycle), Num("min", FIGOCycleMin), Num("max", FIGOCycleMax)}}
		case cr.PeriodStatus == StatusProlonged:
			return FindingPeriodProlonged, &Phrase{Key: "findings.period.prolonged", Args: []Arg{
				Num("days", *cr.MedianPeriod), Num("max", FIGOPeriodMax)}}
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

// BuildTopFinding picks the top finding from the cycle and symptom reports.
func BuildTopFinding(cr CycleReport, sr SymptomsReport) TopFinding {
	var out TopFinding
	if kind, p := cycleFinding(cr); p != nil {
		out.Kind = kind
		out.Parts = append(out.Parts, *p)
	}
	if p := patternFinding(sr.Highlight); p != nil {
		if out.Kind == "" {
			out.Kind = FindingPattern
		}
		out.Parts = append(out.Parts, *p)
	}
	if len(out.Parts) > 0 {
		return out
	}
	if cr.BasedOn == 0 && cr.Current == nil && sr.DaysLogged == 0 {
		return TopFinding{Kind: FindingNoData, Parts: []Phrase{{Key: "findings.no_data"}}}
	}
	return TopFinding{Kind: FindingNotEnoughData, Parts: []Phrase{{Key: "findings.not_enough_data", Args: []Arg{
		Num("needed", max(1, MinPatternCycles-cr.BasedOn))}}}}
}

// withoutCycleNudge swaps the «log N more cycles» nudge for a plain keep-logging one (menopause).
func (f TopFinding) withoutCycleNudge() TopFinding {
	if f.Kind != FindingNotEnoughData {
		return f
	}
	return TopFinding{Kind: FindingKeepLogging, Parts: []Phrase{{Key: "findings.keep_logging"}}}
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
	return jsonx.Obj("kind", f.Kind, "parts", parts, "text", strings.Join(texts, " "))
}
