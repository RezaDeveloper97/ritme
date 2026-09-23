package legacy

import (
	"slices"

	"github.com/ritme/backend-go/internal/enums"
)

// The multipliers are variables, not constants: PHP computes `1.30 - 1` in float64
// (0.30000000000000004); a Go constant expression would fold it exactly to 0.3.
var (
	// baseProbabilityMap is BASE_PROBABILITY_MAP: base chance by day relative to ovulation.
	baseProbabilityMap = map[int]float64{-5: 0.10, -4: 0.15, -3: 0.18, -2: 0.25, -1: 0.30, 0: 0.33, 1: 0.12}

	// ageFactors is AGE_FACTORS: [min, max] inclusive → factor.
	ageFactors = []struct {
		min, max int
		factor   float64
	}{{20, 29, 1.0}, {30, 34, 0.85}, {35, 37, 0.70}, {38, 40, 0.55}, {41, 100, 0.35}}

	// POSITIVE_SYMPTOMS
	ewcmFactor               = 1.30
	ovulationCrampsFactor    = 1.20
	highLibidoFactor         = 1.15
	wateryDischargeFactor    = 1.10
	abdominalHeavinessFactor = 1.05

	// NEGATIVE_SYMPTOMS
	pmsFactor            = 0.8
	dryMucusFactor       = 0.75
	lutealSpottingFactor = 0.7

	cycleScoreBase      = 0.60
	cycleScoreRegular   = 0.10
	cycleScoreSemi      = 0.05
	cycleScoreSignals   = 0.10
	cycleScoreIrregCap  = 0.40
	cycleScoreCap       = 0.85
	positiveScoreCap    = 1.4
	finalProbabilityCap = 0.35
	ageFactorOutOfRange = 0.35
	neutralSymptomScore = 1.0
	noBirthdayAgeFactor = 1.0
)

// CycleScore is calculateCycleScore (HealthDataEngine.php:528): 0.60 + variability bonus + strong
// fertile signals bonus, capped at 0.40 for irregular cycles and at 0.85 overall.
func CycleScore(variability enums.CycleVariability, log *DailyLog) float64 {
	score := cycleScoreBase
	switch variability {
	case enums.CycleVariabilityRegular:
		score += cycleScoreRegular
	case enums.CycleVariabilitySemiIrregular:
		score += cycleScoreSemi
	}
	if log != nil && strongFertileSignals(log) {
		score += cycleScoreSignals
	}
	if variability == enums.CycleVariabilityIrregular {
		score = min(score, cycleScoreIrregCap)
	}
	return min(score, cycleScoreCap)
}

// strongFertileSignals is hasStrongFertileSignals: EWCM, ovulation cramps, high libido or watery
// discharge.
func strongFertileSignals(log *DailyLog) bool {
	return eq(log.DischargeTexture, "egg_white") ||
		log.OvarianPainIntensity != nil ||
		highLibido(log) ||
		eq(log.DischargeTexture, "watery")
}

// highLibido is hasHighLibido: sexual_desire "higher", or the legacy "high_desire" entry of
// sexual_activities.
func highLibido(log *DailyLog) bool {
	if eq(log.SexualDesire, string(enums.SexualDesireHigher)) {
		return true
	}
	return slices.Contains(log.SexualActivities, "high_desire")
}

// AgeFactor maps an age (Carbon `->age`) to its factor; outside every band (under 20, over 100) 0.35.
func AgeFactor(age int) float64 {
	for _, band := range ageFactors {
		if age >= band.min && age <= band.max {
			return band.factor
		}
	}
	return ageFactorOutOfRange
}

// BaseProbability is getBaseProbability: the map value, 0.0 outside O−5..O+1.
func BaseProbability(dayFromOvulation int) float64 {
	return baseProbabilityMap[dayFromOvulation]
}

// SymptomScore is calculateSymptomScore (HealthDataEngine.php:632): the larger of the positive
// boost (capped at 1.4) and the strongest negative multiplier. No log → 1.0.
func SymptomScore(log *DailyLog, isPmsWindow, isLutealSpotting bool) float64 {
	if log == nil {
		return neutralSymptomScore
	}
	positive := 1.0
	negative := 1.0

	if eq(log.DischargeTexture, "egg_white") {
		positive += ewcmFactor - 1
	}
	if log.OvarianPainIntensity != nil {
		positive += ovulationCrampsFactor - 1
	}
	if highLibido(log) {
		positive += highLibidoFactor - 1
	}
	if eq(log.DischargeTexture, "watery") {
		positive += wateryDischargeFactor - 1
	}
	if log.BloatingIntensity != nil {
		positive += abdominalHeavinessFactor - 1
	}
	positive = min(positive, positiveScoreCap)

	if isPmsWindow {
		negative = min(negative, pmsFactor)
	}
	if isTrue(log.VaginalDryness) {
		negative = min(negative, dryMucusFactor)
	}
	if isLutealSpotting {
		negative = min(negative, lutealSpottingFactor)
	}
	return max(positive, negative)
}

// FinalProbability is calculateFinalProbability: base × age × cycle × symptom, capped at 0.35.
func FinalProbability(base, age, cycle, symptom float64) float64 {
	return min(base*age*cycle*symptom, finalProbabilityCap)
}
