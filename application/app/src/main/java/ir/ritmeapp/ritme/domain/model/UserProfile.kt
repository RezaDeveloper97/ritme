package ir.ritmeapp.ritme.domain.model

/**
 * The complete answer of `GET /profile`: the account record, the health profile (null until
 * onboarding has been submitted), and the server-computed BMI. This is the single source the
 * profile screens render from — the client never recomputes BMI or re-derives goals locally.
 */
data class UserProfile(
    val account: UserAccount,
    val health: HealthProfile?,
    val bmi: BmiInfo?,
)

/** The bare account identity (`data.user`): who is logged in, independent of health data. */
data class UserAccount(
    val id: Long,
    val name: String?,
    val mobile: String,
)

/**
 * The editable health profile (`data.profile`). Dates stay Gregorian here (the API's shape);
 * the UI converts to Jalali at the edge. [pregnancyIntention] drives cycle-vs-pregnancy mode.
 */
data class HealthProfile(
    val birthday: GregorianDate?,
    val weightKg: Double?,
    val heightCm: Int?,
    val periodDuration: Int?,
    val cycleDuration: Int?,
    val lastPeriodStart: GregorianDate?,
    val intention: PregnancyIntention?,
    val conditions: List<ChronicCondition>,
)

/** Server-computed body-mass index (`data.bmi`) with its localized category text. */
data class BmiInfo(
    val value: Double,
    val category: String,
    val categoryLabel: String,
    val message: String,
)
