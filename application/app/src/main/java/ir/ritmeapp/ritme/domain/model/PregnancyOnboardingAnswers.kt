package ir.ritmeapp.ritme.domain.model

/**
 * What pregnancy onboarding collects (`POST /pregnancy/onboarding`). Exactly one dating source
 * is required — its companion fields are non-null only for that source; everything else is
 * optional history/context the backend uses for risk flags.
 */
data class PregnancyOnboardingAnswers(
    val ageSource: PregnancyAgeSource,
    val lmpDate: JalaliDate? = null,
    val ultrasoundDate: JalaliDate? = null,
    val ultrasoundWeeks: Int? = null,
    val ultrasoundDays: Int? = null,
    val manualWeeks: Int? = null,
    val manualDays: Int? = null,
    val hasMiscarriageHistory: Boolean = false,
    val hasHighRiskHistory: Boolean = false,
    val preExistingConditions: List<String> = emptyList(),
    val bloodType: String? = null,
    val rhFactor: String? = null,
)

/** How the gestational age is anchored. */
enum class PregnancyAgeSource(val apiValue: String) {
    LMP("lmp"),
    ULTRASOUND("ultrasound"),
    MANUAL("manual");

    companion object {
        fun fromApi(value: String?): PregnancyAgeSource? = entries.firstOrNull { it.apiValue == value }
    }
}
