package enums

import "slices"

// Life-stage modes and the onboarding v2 answers (B-N2-01). Go only — no PHP enum behind them, so they are
// hand-written here and carry no labels: the fa/en (and every admin-added language's) labels are UI strings in
// the translation bundle, `onboarding.enums.<enum>.<value>` (resources/translations/<code>/onboarding.json).

// LifeMode is the user's life stage (Me → «مرحله زندگی»): it picks the tabs, the home and the log tiles.
type LifeMode string

// LifeMode cases, in screen order (nbl_Me_Mode).
const (
	LifeModeCycle      LifeMode = "cycle"
	LifeModeTTC        LifeMode = "ttc"
	LifeModePregnancy  LifeMode = "pregnancy"
	LifeModePostpartum LifeMode = "postpartum"
	LifeModeMenopause  LifeMode = "menopause"
	LifeModeTeen       LifeMode = "teen"
)

var lifeModeCases = []LifeMode{
	LifeModeCycle, LifeModeTTC, LifeModePregnancy, LifeModePostpartum, LifeModeMenopause, LifeModeTeen,
}

// LifeModeValues returns the backed values in screen order.
func LifeModeValues() []string { return valuesOf(lifeModeCases) }

// IsValid reports whether e is one of the declared cases.
func (e LifeMode) IsValid() bool { return slices.Contains(lifeModeCases, e) }

// MessageMode is the message / home engine that serves the mode. Only pregnancy and postpartum have their own
// (postpartum: the PHP enum's engine-less mode); cycle, ttc, menopause and teen run on the cycle engine —
// menopause and teen with the safe defaults of AllowsFertilityContent until roadmap E02-meno / E09-teen.
func (e LifeMode) MessageMode() MessageMode {
	switch e {
	case LifeModePregnancy:
		return MessageModePregnancy
	case LifeModePostpartum:
		return MessageModePostpartum
	}
	return MessageModeCycle
}

// TracksCycle is false for the mode without a menstrual cycle to time things by (menopause, B-N2-11b):
// e.g. a checkup's «روز ۷ تا ۱۰ سیکل» window does not apply.
func (e LifeMode) TracksCycle() bool { return e != LifeModeMenopause }

// AllowsFertilityContent is false for the modes that must never get conception / fertile-window targeted
// content (TTC copy, "best days" nudges): menopause and teen. The engines treat such a user as non-TTC whatever
// user_goal says.
func (e LifeMode) AllowsFertilityContent() bool {
	return e != LifeModeMenopause && e != LifeModeTeen
}

// LegacyUserGoal is the user_profiles.user_goal kept in step with the mode (only ttc is TTC).
func (e LifeMode) LegacyUserGoal() UserGoal {
	if e == LifeModeTTC {
		return UserGoalTtc
	}
	return UserGoalNonTtc
}

// ResolveLifeMode is the effective mode of a user:
//
//  1. an active pregnancy profile (pregnancy_mode = 1) → pregnancy (the pregnancy domain owns that switch);
//  2. a stored postpartum / menopause / teen mode → that mode;
//  3. otherwise the legacy derivation: user_goal ttc → ttc, else cycle (a stored cycle/ttc is kept in step
//     with user_goal; a stored pregnancy without an active pregnancy profile falls back here too).
//
// Users created before B-N2-01 have no stored mode, so they resolve exactly as before (cycle / ttc / pregnancy).
func ResolveLifeMode(stored string, pregnancyActive bool, userGoal string) LifeMode {
	if pregnancyActive {
		return LifeModePregnancy
	}
	switch m := LifeMode(stored); m {
	case LifeModePostpartum, LifeModeMenopause, LifeModeTeen:
		return m
	}
	if userGoal == string(UserGoalTtc) {
		return LifeModeTTC
	}
	return LifeModeCycle
}

// OnboardingGoal is the Goal step («چرا ریتمی را نصب کردی؟», nbl_Onb_Goal): the first life mode.
type OnboardingGoal string

// OnboardingGoal cases, in screen order.
const (
	OnboardingGoalCycle     OnboardingGoal = "cycle"
	OnboardingGoalTTC       OnboardingGoal = "ttc"
	OnboardingGoalPregnancy OnboardingGoal = "pregnancy"
	OnboardingGoalMenopause OnboardingGoal = "menopause"
)

// OnboardingGoalValues returns the backed values in screen order.
func OnboardingGoalValues() []string {
	return []string{string(OnboardingGoalCycle), string(OnboardingGoalTTC), string(OnboardingGoalPregnancy), string(OnboardingGoalMenopause)}
}

// Gender is the Gender step (nbl_Onb_Gender). male → the companion path (B-N4-03).
type Gender string

// Gender cases, in screen order.
const (
	GenderFemale Gender = "female"
	GenderMale   Gender = "male"
)

// GenderValues returns the backed values in screen order.
func GenderValues() []string { return []string{string(GenderFemale), string(GenderMale)} }

// MenopauseStage is the Meno step («کجای مسیر یائسگی هستی؟», nbl_Onb_Meno).
type MenopauseStage string

// MenopauseStage cases, in screen order.
const (
	MenopauseStagePeri   MenopauseStage = "peri"
	MenopauseStageMeno   MenopauseStage = "meno"
	MenopauseStagePost   MenopauseStage = "post"
	MenopauseStageUnsure MenopauseStage = "unsure"
)

// MenopauseStageValues returns the backed values in screen order.
func MenopauseStageValues() []string {
	return []string{string(MenopauseStagePeri), string(MenopauseStageMeno), string(MenopauseStagePost), string(MenopauseStageUnsure)}
}

// ChronicIllnessValues are «بیماری‌های مزمن» of the Conditions step (nbl_Onb_Conditions), screen order; «هیچ‌کدام»
// is the empty list. Separate from the legacy ChronicCondition (POST /profile chronic_conditions).
func ChronicIllnessValues() []string {
	return []string{"diabetes", "hypertension", "thyroid", "asthma", "anemia", "migraine", "other"}
}

// GynConditionValues are «بیماری‌ها و شرایط زنان» of the Conditions step, screen order («هیچ‌کدام» = []).
func GynConditionValues() []string {
	return []string{"pcos", "endometriosis", "fibroids", "recurrent_infections", "other"}
}

// MedicationValues are «داروی دائمی یا روش پیشگیری» of the Conditions step, screen order («ندارم» = []).
func MedicationValues() []string {
	return []string{"contraceptive_pill", "iud", "hormonal_medication"}
}
