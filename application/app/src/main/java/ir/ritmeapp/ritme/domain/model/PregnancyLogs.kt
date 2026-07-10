package ir.ritmeapp.ritme.domain.model

/**
 * The three pregnancy-mode logs (`/pregnancy/symptoms`, `/pregnancy/weekly`,
 * `/pregnancy/fetal-movement`). Like [DailyHealthLog], the daily symptoms are a data-driven
 * catalog so the form and the JSON mapping never drift apart. All logs upsert server-side
 * (symptoms/movement by date, weekly by week).
 */

/** The daily pregnancy symptoms the backend accepts, grouped the way the form shows them. */
enum class PregnancySymptom(val apiKey: String, val group: PregnancySymptomGroup) {
    NAUSEA("nausea", PregnancySymptomGroup.COMMON),
    VOMITING("vomiting", PregnancySymptomGroup.COMMON),
    FATIGUE("fatigue", PregnancySymptomGroup.COMMON),
    HEADACHE("headache", PregnancySymptomGroup.COMMON),
    DIZZINESS("dizziness", PregnancySymptomGroup.COMMON),
    BREAST_PAIN("breast_pain", PregnancySymptomGroup.PAIN),
    LOWER_ABDOMINAL_PAIN("lower_abdominal_pain", PregnancySymptomGroup.PAIN),
    CRAMPING("cramping", PregnancySymptomGroup.PAIN),
    BACK_PAIN("back_pain", PregnancySymptomGroup.PAIN),
    PELVIC_PRESSURE("pelvic_pressure", PregnancySymptomGroup.PAIN),
    SPOTTING("spotting", PregnancySymptomGroup.WARNING),
    BLEEDING("bleeding", PregnancySymptomGroup.WARNING),
    FLUID_LEAKAGE("fluid_leakage", PregnancySymptomGroup.WARNING),
    SEVERE_SUDDEN_PAIN("severe_sudden_pain", PregnancySymptomGroup.WARNING),
    ;

    companion object {
        val byGroup: Map<PregnancySymptomGroup, List<PregnancySymptom>> = entries.groupBy { it.group }
    }
}

/** The form's three sections; WARNING symptoms get the red call-your-doctor treatment. */
enum class PregnancySymptomGroup { COMMON, PAIN, WARNING }

/** Symptom severity (`mild` / `moderate` / `severe`). */
enum class SymptomSeverity(val apiValue: String) {
    MILD("mild"),
    MODERATE("moderate"),
    SEVERE("severe");

    companion object {
        fun fromApi(value: String?): SymptomSeverity? = entries.firstOrNull { it.apiValue == value }
    }
}

/** One day's symptom log: only the symptoms marked present, each with its severity. */
data class PregnancySymptomLog(
    val date: GregorianDate,
    val symptoms: Map<PregnancySymptom, SymptomSeverity>,
    val notes: String?,
)

/** The weekly monitoring check-in (`/pregnancy/weekly`, upserted by [week]). */
data class PregnancyWeeklyLog(
    val week: Int,
    val date: GregorianDate,
    val weightKg: Double? = null,
    val hasSwelling: Boolean = false,
    val swellingLocations: List<String> = emptyList(),
    val hasShortnessOfBreath: Boolean = false,
    val hasBloodPressureDevice: Boolean = false,
    val systolicPressure: Int? = null,
    val diastolicPressure: Int? = null,
    val fastingBloodSugar: Double? = null,
    val postMealBloodSugar: Double? = null,
    val overallMood: String? = null,
    val anxietySeverity: SymptomSeverity? = null,
    val moodSwingsSeverity: SymptomSeverity? = null,
    val depressionSeverity: SymptomSeverity? = null,
    val notes: String? = null,
)

/** A fetal-movement entry (`/pregnancy/fetal-movement`, upserted by [date]). */
data class FetalMovementLog(
    val date: GregorianDate,
    val week: Int,
    val movementStatus: String,
    val movementCount: Int? = null,
    /** `HH:mm`, the API's time-of-day shape. */
    val firstMovementTime: String? = null,
    val lastMovementTime: String? = null,
    val notes: String? = null,
) {
    companion object {
        /** The backend's `movement_status` enum, in the form's display order. */
        val STATUS_OPTIONS = listOf("not_felt_yet", "felt", "normal", "reduced", "increased", "none")

        /** The backend's swelling-location enum for the weekly form. */
        val SWELLING_LOCATIONS = listOf("feet", "hands", "face")

        /** The backend's overall-mood enum for the weekly form. */
        val MOOD_OPTIONS = listOf("good", "moderate", "poor")
    }
}
