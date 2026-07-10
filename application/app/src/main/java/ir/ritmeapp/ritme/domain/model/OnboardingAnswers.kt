package ir.ritmeapp.ritme.domain.model

/**
 * Everything the signup wizard collects, accumulated as the user advances and submitted once to
 * `POST /profile` at the end. Immutable (§4) — each step produces a copy. Fields are nullable
 * because they fill in over the flow; the mapping to the API sends only the ones that are set,
 * and cycle fields ([periodDuration]/[cycleDuration]/[lastPeriod]) stay null in pregnancy mode.
 * Note: subscription tier is deliberately absent — it is never client-set (server-only).
 */
data class OnboardingAnswers(
    val name: String? = null,
    val birthday: JalaliDate? = null,
    val weightKg: Int? = null,
    val heightCm: Int? = null,
    val intention: PregnancyIntention? = null,
    val periodDuration: Int? = null,
    val cycleDuration: Int? = null,
    val lastPeriod: JalaliDate? = null,
    val conditions: List<ChronicCondition> = emptyList(),
)
