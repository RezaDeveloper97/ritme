package ir.ritmeapp.ritme.domain.model

/**
 * The localized week-by-week educational content (`GET /pregnancy/content/{week}`). Each module
 * is pre-rendered text from the backend; empty strings mean the module has no content that week.
 */
data class PregnancyWeekContent(
    val week: Int,
    val fetalDevelopment: String,
    val motherBodyChanges: String,
    val bodyAdaptation: String,
    val emotionalStatus: String,
    val keyNutrition: String,
    val physicalActivity: String,
    val dosAndDonts: String,
    val carePlan: String,
    val testsAndCheckups: String,
    val faq: String,
) {
    /** True when the backend has nothing for this week — the UI shows the "not ready" note. */
    val isEmpty: Boolean
        get() = listOf(
            fetalDevelopment, motherBodyChanges, bodyAdaptation, emotionalStatus, keyNutrition,
            physicalActivity, dosAndDonts, carePlan, testsAndCheckups, faq,
        ).all { it.isBlank() }
}
