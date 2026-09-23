package enums

import "slices"

// TriggerLog is the slice of a daily health log RecommendationTrigger reads. Pointers are nil when
// the column is NULL; Moods returns nil when the column is NULL or not an array.
// Pass a nil TriggerLog (untyped nil, not a typed nil pointer) for "no log today".
type TriggerLog interface {
	HeadacheIntensity() *string
	PelvicPainIntensity() *string
	StomachAcheIntensity() *string
	SleepQuality() *string
	Moods() []string
	BloatingIntensity() *string
	Fatigue() *bool
}

// Matches reports whether today's log shows this symptom. No log means no symptom.
// PHP: backend/app/Enums/RecommendationTrigger.php:46.
func (e RecommendationTrigger) Matches(log TriggerLog) bool {
	if log == nil {
		return false
	}
	switch e {
	case RecommendationTriggerHeadache:
		return log.HeadacheIntensity() != nil
	case RecommendationTriggerCramps:
		return log.PelvicPainIntensity() != nil || log.StomachAcheIntensity() != nil
	case RecommendationTriggerPoorSleep:
		q := log.SleepQuality()
		return q != nil && *q == string(SleepQualityBad)
	case RecommendationTriggerLowMood:
		moods := log.Moods()
		return slices.Contains(moods, string(MoodAnxious)) || slices.Contains(moods, string(MoodSad))
	case RecommendationTriggerBloating:
		return log.BloatingIntensity() != nil
	case RecommendationTriggerFatigue:
		f := log.Fatigue()
		return f != nil && *f // PHP `=== true`: null is not fatigue
	}
	return false
}

// RecommendationTriggerActiveFor returns the trigger values today's log satisfies, in case order
// (an empty, non-nil slice when none). PHP: backend/app/Enums/RecommendationTrigger.php:90.
func RecommendationTriggerActiveFor(log TriggerLog) []string {
	out := []string{}
	for _, c := range recommendationTriggerCases {
		if c.Matches(log) {
			out = append(out, string(c))
		}
	}
	return out
}

// RecommendationTriggerLabelFor labels a stored trigger key; ok is false for an unknown or empty key
// (PHP null). PHP: backend/app/Enums/RecommendationTrigger.php:98.
func RecommendationTriggerLabelFor(value, locale string) (string, bool) {
	return labelFor(RecommendationTriggerFrom, value, locale)
}

// RecommendationTypeLabelFor labels a stored category key, falling back to the generic "general"
// label for an unknown or empty key. PHP: backend/app/Enums/RecommendationType.php:109.
func RecommendationTypeLabelFor(value, locale string) string {
	return recommendationTypeOrGeneral(value).Label(locale)
}

// RecommendationTypeIconFor returns the icon for a stored category key, falling back to the generic
// sparkle. PHP: backend/app/Enums/RecommendationType.php:115.
func RecommendationTypeIconFor(value string) string {
	return recommendationTypeOrGeneral(value).Icon()
}

func recommendationTypeOrGeneral(value string) RecommendationType {
	if e, ok := RecommendationTypeFrom(value); ok {
		return e
	}
	return RecommendationTypeGeneral
}
