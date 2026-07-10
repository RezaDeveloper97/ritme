package ir.ritmeapp.ritme.domain.model

/**
 * The tracker headline from `GET /pregnancy/status`: how far along the pregnancy is and the
 * flags the hero card renders. [currentWeek] drives the week browser's initial position.
 */
data class PregnancyStatus(
    val isActive: Boolean,
    val gestationalWeeks: Int?,
    val gestationalDays: Int?,
    val trimester: Int?,
    val dueDate: GregorianDate?,
    val currentWeek: Int?,
    val isHighRisk: Boolean,
    val fetalMovementTrackingActive: Boolean,
) {
    /** Progress through the standard 40-week term as 0..100 for the hero bar. */
    val progressPercent: Int
        get() {
            val week = currentWeek ?: return 0
            return (week * 100 / TOTAL_WEEKS).coerceIn(0, 100)
        }

    companion object {
        const val TOTAL_WEEKS = 40
    }
}

/**
 * The pregnancy profile record (`GET /pregnancy/profile`) — the slice the app renders:
 * whether onboarding is finished and how confident the dating is.
 */
data class PregnancyProfile(
    val onboardingCompleted: Boolean,
    val confidenceLevel: PregnancyConfidence?,
    val uncertaintyDays: Int?,
    val isLocked: Boolean,
)

/** How trustworthy the gestational-age estimate is, derived server-side from the age source. */
enum class PregnancyConfidence(val apiValue: String) {
    HIGH("high"),
    MEDIUM("medium"),
    LOW("low");

    companion object {
        fun fromApi(value: String?): PregnancyConfidence? = entries.firstOrNull { it.apiValue == value }
    }
}
