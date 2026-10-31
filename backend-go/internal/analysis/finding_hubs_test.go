package analysis

// T-M10-26b (§4.1, D-175): the pregnancy weight minimum and the top finding on the TTC and pregnancy hubs, at their
// boundaries.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/fertility/bbt"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

func topOf(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	f, _ := m["top_finding"].(map[string]any)
	require.NotNil(t, f, "top_finding")
	return f
}

// weighedPregnancy: week 21 with a pre-pregnancy weight and a height (BMI 22.0) and n weights logged in the pregnancy.
func weighedPregnancy(n int) *PregnancyInput {
	start := civildate.MustParse("2026-04-26")
	in := emptyPregnancy(start)
	in.HeightCM = 165
	in.Weights[start.AddDays(-10)] = 60 // before the pregnancy: the baseline, not a pregnancy log
	for i := range n {
		in.Weights[start.AddDays(120+7*i)] = 64 + float64(i)*0.4
	}
	return in
}

func TestPregnancyWeight_MinLogs(t *testing.T) {
	require.Equal(t, 2, P.PregWeightMinLogs)

	one := sec(t, BuildPregnancy(weighedPregnancy(1)), "weight_gain")
	assert.Equal(t, false, one["ready"], "1 log is below the §4.1 minimum")
	d := one["data"].(map[string]any)
	assert.Nil(t, d["missing"], "nothing else is missing")
	assert.InDelta(t, 1, d["logs_needed"], 0)
	assert.InDelta(t, 2, d["min_logs"], 0)
	assert.Nil(t, d["status"], "no IOM verdict below the minimum")
	assert.NotNil(t, d["current"], "the logged weight is still shown")

	two := sec(t, BuildPregnancy(weighedPregnancy(2)), "weight_gain")
	assert.Equal(t, true, two["ready"])
	d = two["data"].(map[string]any)
	assert.InDelta(t, 0, d["logs_needed"], 0)
	assert.NotNil(t, d["status"])

	none := sec(t, BuildPregnancy(weighedPregnancy(0)), "weight_gain")["data"].(map[string]any)
	assert.Equal(t, MissingWeights, none["missing"])
	assert.InDelta(t, 2, none["logs_needed"], 0)

	// The pre-pregnancy lookback weight never counts as a pregnancy log.
	assert.Equal(t, 1, weighedPregnancy(1).pregnancyWeightLogs())
}

func TestPregnancyTopFinding(t *testing.T) {
	// One log: no finding yet, the nudge names the one weight still missing.
	f := topOf(t, render(t, BuildPregnancy(pregCopy(t, weighedPregnancy(1), "fa"))))
	assert.Equal(t, FindingNotEnoughData, f["kind"])
	assert.InDelta(t, 1, f["days_needed"], 0)
	assert.Equal(t, "برای دیدن مهم\u200cترین یافته، ۱ روز دیگر وزنت را ثبت کن.", f["text"])

	// Two logs: the gain verdict is the finding.
	f = topOf(t, render(t, BuildPregnancy(pregCopy(t, weighedPregnancy(2), "en"))))
	assert.Contains(t, []any{FindingPregGainWithin, FindingPregGainAbove, FindingPregGainBelow}, f["kind"])
	assert.NotEmpty(t, f["text"])
	assert.NotNil(t, f["score"])

	// Nothing logged, no baseline: keep logging.
	f = topOf(t, render(t, BuildPregnancy(pregCopy(t, earlyPregnancy(), "fa"))))
	assert.Equal(t, FindingKeepLogging, f["kind"])
	assert.Nil(t, f["days_needed"])

	// A severe reading outranks a high one and the weight; a high reading today outranks a gain verdict.
	in := weighedPregnancy(2)
	in.BP[fixtureToday] = BPReading{Systolic: 165, Diastolic: 100}
	f = topOf(t, render(t, BuildPregnancy(pregCopy(t, in, "en"))))
	assert.Equal(t, FindingPregBPSevere, f["kind"])
	assert.Equal(t, "You logged a blood pressure of 160/110 or higher. Call your doctor or midwife today.", f["text"])
	in.BP[fixtureToday] = BPReading{Systolic: 142, Diastolic: 85}
	f = topOf(t, render(t, BuildPregnancy(pregCopy(t, in, "fa"))))
	assert.Equal(t, FindingPregBPHigh, f["kind"])
	assert.Equal(t, "۱ بار فشار ۱۴۰/۹۰ یا بالاتر ثبت کرده\u200cای. این تشخیص نیست؛ با پزشک یا ماما در میانش بگذار.", f["text"])

	// Recency: a high reading 90 days old (0.9 × 0.125) loses to a gain verdict from this week (0.3+).
	in = weighedPregnancy(2)
	in.BP[fixtureToday.AddDays(-90)] = BPReading{Systolic: 142, Diastolic: 85}
	in.Weights[fixtureToday] = 64.6
	f = topOf(t, render(t, BuildPregnancy(in)))
	assert.NotEqual(t, FindingPregBPHigh, f["kind"])

	// Plus: a glucose average at the target is a finding (target = "under"), one under it is not.
	in = weighedPregnancy(0)
	in.DeepAnalysis = true
	in.Glucose = []GlucoseReading{{Date: fixtureToday, Slot: GlucoseFasting, Value: GDMFastingMax}}
	f = topOf(t, render(t, BuildPregnancy(pregCopy(t, in, "en"))))
	assert.Equal(t, FindingPregGlucoseAbove, f["kind"])
	assert.Equal(t, "Your average fasting sugar is 95, above the target (under 95). Talk it over with your doctor.", f["text"])
	in.Glucose[0].Value = GDMFastingMax - 1
	assert.NotEqual(t, FindingPregGlucoseAbove, topOf(t, render(t, BuildPregnancy(in)))["kind"])
	// … and is never computed for a free user.
	in.DeepAnalysis = false
	in.Glucose[0].Value = 120
	assert.NotEqual(t, FindingPregGlucoseAbove, topOf(t, render(t, BuildPregnancy(in)))["kind"])
}

func TestPregnancyTopFinding_SymptomStrength(t *testing.T) {
	in := emptyPregnancy(civildate.MustParse("2026-04-26")) // week 22: trimester 2
	in.DeepAnalysis = true
	for i := range 10 {
		d := fixtureToday.AddDays(-i)
		key := "symptoms.general.fatigue"
		if i < 6 {
			key = "pain.location.back"
		}
		in.Symptoms[d] = map[string]bool{key: true}
	}
	f := topOf(t, render(t, BuildPregnancy(pregCopy(t, in, "fa"))))
	assert.Equal(t, FindingPregSymptomTop, f["kind"])
	assert.InDelta(t, 0.24, f["strength"], 1e-9, "0.4 × 6 of 10 logged days")
	assert.Contains(t, f["text"], "پرتکرارترین علامتی")
}

func TestTTCTopFinding(t *testing.T) {
	// The fixture: six regular cycles and egg-white mucus before each ovulation.
	in := ttcFixture(true, "1995-02-10")
	in.Copy = copyFor(t, "fa")
	r := BuildTTC(in)
	f := r.TopFinding()
	require.NotEqual(t, FindingNotEnoughData, f.Kind)
	cands := r.ttcCandidates(BuildCycle(in.Input))
	kinds := map[string]bool{}
	for _, c := range cands {
		kinds[c.Kind] = true
	}
	assert.True(t, kinds[FindingCycleRegular], "the cycle hub's cycle sentence")
	assert.True(t, kinds[FindingTTCMucus], "Plus: the egg-white pattern")
	best, _ := PickTopFinding(cands, in.Today)
	assert.Equal(t, best.Kind, f.Kind, "the same scorer as the cycle hub")

	// Free: no mucus candidate.
	free := ttcFixture(false, "1995-02-10")
	for _, c := range BuildTTC(free).ttcCandidates(BuildCycle(free.Input)) {
		assert.NotEqual(t, FindingTTCMucus, c.Kind)
	}

	// No cycles at all: the days nudge (2 cycles at the default length) — never a weak finding.
	empty := ttcEmptyFixture()
	empty.Copy = copyFor(t, "fa")
	e := BuildTTC(empty).TopFinding()
	assert.Equal(t, FindingNotEnoughData, e.Kind)
	assert.Positive(t, e.DaysNeeded)
	assert.Equal(t, "findings.not_enough_days", e.Parts[0].Key)
}

func TestTTCDaysNeeded_BBT(t *testing.T) {
	r := TTCReport{in: ttcEmptyFixture()}
	c := TTCCycle{Cycle: Cycle{Current: true}}
	for range P.BBTMinTemps - 3 {
		c.BBT.Points = append(c.BBT.Points, bbtPoint())
	}
	r.Cycles = []TTCCycle{c}
	assert.Equal(t, 3, r.ttcDaysNeeded(CycleReport{BasedOn: P.CycleLengthMinCycles}), "3 temperatures short")
	c.BBT.Points = append(c.BBT.Points, bbtPoint(), bbtPoint(), bbtPoint())
	r.Cycles = []TTCCycle{c}
	assert.Equal(t, 0, r.ttcDaysNeeded(CycleReport{BasedOn: P.CycleLengthMinCycles}), "both minimums met")
}

func bbtPoint() bbt.Point { return bbt.Point{Value: 3640} }
