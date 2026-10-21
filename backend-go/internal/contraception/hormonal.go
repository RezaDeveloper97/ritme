package contraception

import "slices"

// Hormonal contraception for the cycle engine (DECISIONS.md I-03, T-M9-25): the engine suppresses fertility and
// reads the cycle as withdrawal bleeds while it is on (engine.Input.Hormonal).

// Onboarding «داروی دائمی یا روش پیشگیری» answers (enums.MedicationValues) that I-03 maps.
const (
	MedicationContraceptivePill = "contraceptive_pill" // hormonal
	MedicationIUD               = "iud"                // type unknown → not hormonal (the user picks copper/hormonal later)
)

// IsHormonal reports whether a saved contraception method is hormonal: combined_pill, progestin_pill, hormonal_iud,
// injection and implant. copper_iud, condom, other and unknown spellings are not.
func IsHormonal(method string) bool {
	switch method {
	case MethodCombinedPill, MethodProgestinPill, MethodHormonalIUD, MethodInjection, MethodImplant:
		return true
	}
	return false
}

// HormonalFromOnboarding reports whether the onboarding medications say hormonal contraception: only
// contraceptive_pill does; iud (type unknown) and hormonal_medication (not contraception) do not.
func HormonalFromOnboarding(medications []string) bool {
	return slices.Contains(medications, MedicationContraceptivePill)
}

// HormonalActive is the engine's Hormonal input (I-03). A saved method (a contraception_methods row exists only
// while the user tracks one) decides alone, so a user who moved from the pill to a copper IUD is not hormonal any
// more; without one the onboarding answer decides.
func HormonalActive(method string, hasMethod bool, medications []string) bool {
	if hasMethod {
		return IsHormonal(method)
	}
	return HormonalFromOnboarding(medications)
}
