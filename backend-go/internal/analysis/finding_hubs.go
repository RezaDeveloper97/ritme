package analysis

import "github.com/ritme/backend-go/internal/platform/civildate"

// Top finding on the TTC and pregnancy hubs (§4.1 «مهم‌ترین یافته بالای هر هاب», D-175, T-M10-26b): the same scorer
// as the cycle hub — every candidate scored strength × recency (PickTopFinding, half-life P.TopFindingHalfLifeDays),
// ties by findingOrder — over each hub's own findings. Strengths (0–1, product constants, next to the cycle hub's
// table in finding.go):
//
//	| hub       | kind                         | strength                                  | evidence day                |
//	|-----------|------------------------------|-------------------------------------------|-----------------------------|
//	| TTC       | cycle_* (the cycle report)   | as on the cycle hub                       | newest counted cycle's end  |
//	| TTC       | ttc_mucus (Plus)             | 0.4 × cycles with egg-white ÷ analysed    | the newest egg-white day    |
//	| pregnancy | preg_bp_severe               | 1.0                                       | the latest severe reading   |
//	| pregnancy | preg_bp_high                 | 0.9                                       | the latest high reading     |
//	| pregnancy | preg_gain_above / _below     | 0.7                                       | the latest weight           |
//	| pregnancy | preg_glucose_above (Plus)    | 0.6 (a slot's average at/above its target)| the slot's latest reading   |
//	| pregnancy | preg_symptom_top (Plus)      | 0.4 × days ÷ logged days (this trimester) | the symptom's last day      |
//	| pregnancy | preg_gain_within             | 0.3                                       | the latest weight           |
//
// The LH line (ttc.lh.positive) is a card caption, not a sentence, so a positive test is not a candidate until the
// analysis copy has a finding sentence for it. None is a correlation, so none carries the not-causal line; every sentence describes her own logs, never a cause.
// Without a candidate: the days nudge (the nearest unmet minimum, in days) or keep_logging.
const (
	FindingTTCMucus         = "ttc_mucus"
	FindingPregBPSevere     = "preg_bp_severe"
	FindingPregBPHigh       = "preg_bp_high"
	FindingPregGainAbove    = "preg_gain_above"
	FindingPregGainBelow    = "preg_gain_below"
	FindingPregGainWithin   = "preg_gain_within"
	FindingPregGlucoseAbove = "preg_glucose_above"
	FindingPregSymptomTop   = "preg_symptom_top"
)

// ttcMucusStrengthMax / pregSymptomStrengthMax scale a share like patternStrengthMax.
const (
	ttcMucusStrengthMax    = 0.4
	pregSymptomStrengthMax = 0.4
)

// pickOrNudge is the hub's top finding: the best candidate, else the days nudge for n > 0, else keep_logging.
func pickOrNudge(cands []Candidate, today civildate.Date, daysNeeded int) TopFinding {
	if c, ok := PickTopFinding(cands, today); ok {
		return TopFinding{Kind: c.Kind, Parts: c.Parts, Strength: c.Strength, Score: c.Score(today), Evidence: c.Evidence}
	}
	if daysNeeded == 0 {
		return TopFinding{Kind: FindingKeepLogging, Parts: []Phrase{{Key: "findings.keep_logging"}}}
	}
	return TopFinding{Kind: FindingNotEnoughData, DaysNeeded: daysNeeded, Parts: []Phrase{{Key: "findings.not_enough_days",
		Args: []Arg{Num("days", daysNeeded)}}}}
}

// ttcCandidates are the TTC hub's findings: the cycle sentence of the cycle report and, for Plus, the egg-white
// mucus pattern.
func (r TTCReport) ttcCandidates(cr CycleReport) []Candidate {
	out := r.in.candidates(cr, SymptomsReport{}, nil) // the cycle finding only: the TTC hub reads no day logs
	if r.in.DeepAnalysis {
		with, analysed := 0, 0
		var last civildate.Date
		for _, c := range r.Cycles {
			if c.OvulationDay == 0 || c.OvulationDay > c.Days {
				continue
			}
			analysed++
			if c.MucusLead != nil {
				with++
				if d := c.Start.AddDays(c.OvulationDay - *c.MucusLead - 1); d.After(last) {
					last = d
				}
			}
		}
		if with > 0 {
			out = append(out, Candidate{Kind: FindingTTCMucus,
				Strength: ttcMucusStrengthMax * float64(with) / float64(analysed), Evidence: last,
				Parts: []Phrase{r.mucusPhrase()}})
		}
	}
	return out
}

// ttcDaysNeeded is the TTC nudge's N: the nearest of the cycle-length minimum (cycles in days) and the current
// cycle's BBT minimum (one temperature a day).
func (r TTCReport) ttcDaysNeeded(cr CycleReport) int {
	best := 0
	consider := func(n int) {
		if n > 0 && (best == 0 || n < best) {
			best = n
		}
	}
	cycleLen, _ := r.in.typical()
	consider(daysForCycles(P.CycleLengthMinCycles-cr.BasedOn, cycleLen))
	if c := r.current(); c != nil {
		consider(P.BBTMinTemps - len(c.BBT.Points))
	}
	return best
}

// TopFinding is the TTC hub's top finding.
func (r TTCReport) TopFinding() TopFinding {
	cr := BuildCycle(r.in.Input)
	return pickOrNudge(r.ttcCandidates(cr), r.in.Today, r.ttcDaysNeeded(cr))
}

// topFinding is the pregnancy hub's top finding; without a candidate the nudge names the weights still missing
// for the weight card (P.PregWeightMinLogs, one a day) when only those block it, else keep_logging.
func (in *PregnancyInput) topFinding(cands []Candidate) TopFinding {
	n := 0
	if base, _, _, ok := in.baseline(); ok && base > 0 && in.HeightCM > 0 {
		n = max(0, P.PregWeightMinLogs-in.pregnancyWeightLogs())
	}
	return pickOrNudge(cands, in.Today, n)
}
