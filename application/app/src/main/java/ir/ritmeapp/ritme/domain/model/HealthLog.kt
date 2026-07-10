package ir.ritmeapp.ritme.domain.model

/**
 * The daily-health-log vocabulary (`/health-logs`): every field the backend accepts, its input
 * control, its category on the log screen, and its enum options — one closed, data-driven
 * catalog instead of 40+ hand-wired form rows. The Persian labels for fields and options are
 * resolved in the UI layer; this file stays locale-free (§11).
 */

/** The eleven sections of the log screen, in display order (mirrors the web log page). */
enum class HealthLogCategory {
    BLEEDING, PAIN, DIGESTION, MOOD, SLEEP, BODY, DISCHARGE, INTIMATE, SEXUAL, MEASURE, NOTES,
}

/** How a field is edited. Options/ranges live here so the form renders itself from the catalog. */
sealed interface HealthLogControl {

    /** Single-select chips over the backend enum [options]. */
    data class Choice(val options: List<String>) : HealthLogControl

    /** Multi-select chips over the backend enum [options]. */
    data class MultiChoice(val options: List<String>) : HealthLogControl

    /** The three-step intensity segment (`low` / `medium` / `high`). */
    data object Degree : HealthLogControl

    /** A yes/no switch. */
    data object Toggle : HealthLogControl

    /** A numeric wheel between [min] and [max] advancing by [step]. */
    data class Measure(val min: Double, val max: Double, val step: Double) : HealthLogControl

    /** Free multi-line text. */
    data object Note : HealthLogControl
}

/** One recorded value; the variant always matches the field's [HealthLogControl]. */
sealed interface HealthLogValue {
    data class Choice(val option: String) : HealthLogValue
    data class MultiChoice(val options: List<String>) : HealthLogValue
    data class Toggle(val enabled: Boolean) : HealthLogValue
    data class Number(val value: Double) : HealthLogValue
    data class Text(val text: String) : HealthLogValue
}

private val DEGREE_OPTIONS = listOf("low", "medium", "high")

/**
 * Every submittable field, keyed by the backend's snake_case name. Order within a category is
 * the display order on the log screen.
 */
enum class HealthLogField(
    val apiKey: String,
    val category: HealthLogCategory,
    val control: HealthLogControl,
) {
    // --- Bleeding -----------------------------------------------------------
    BLEEDING_INTENSITY(
        "bleeding_intensity", HealthLogCategory.BLEEDING,
        HealthLogControl.Choice(listOf("low", "medium", "high", "very_high")),
    ),
    BLOOD_COLOR(
        "blood_color", HealthLogCategory.BLEEDING,
        HealthLogControl.Choice(listOf("bright_red", "red", "dark_red", "brown")),
    ),
    BLEEDING_SMELL(
        "bleeding_smell", HealthLogCategory.BLEEDING,
        HealthLogControl.Choice(listOf("normal", "slightly_unusual", "strong_unpleasant")),
    ),
    HAS_CLOTS("has_clots", HealthLogCategory.BLEEDING, HealthLogControl.Toggle),
    SPOTTING("spotting", HealthLogCategory.BLEEDING, HealthLogControl.Toggle),

    // --- Pain ---------------------------------------------------------------
    HEADACHE("headache_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    STOMACH_ACHE("stomach_ache_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    PELVIC_PAIN("pelvic_pain_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    BREAST_PAIN("breast_pain_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    BACK_PAIN("back_pain_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    OVARIAN_PAIN("ovarian_pain_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    NAUSEA("nausea_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    BLOATING("bloating_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),
    BREAST_SENSITIVITY("breast_sensitivity_intensity", HealthLogCategory.PAIN, HealthLogControl.Degree),

    // --- Digestion ------------------------------------------------------------
    DIARRHEA("diarrhea", HealthLogCategory.DIGESTION, HealthLogControl.Toggle),
    CONSTIPATION("constipation", HealthLogCategory.DIGESTION, HealthLogControl.Toggle),
    APPETITE_CHANGE(
        "appetite_change", HealthLogCategory.DIGESTION,
        HealthLogControl.Choice(listOf("loss", "gain", "normal")),
    ),
    FOOD_CRAVING("food_craving", HealthLogCategory.DIGESTION, HealthLogControl.Toggle),

    // --- Mood -------------------------------------------------------------------
    MOODS(
        "moods", HealthLogCategory.MOOD,
        HealthLogControl.MultiChoice(
            listOf("happy", "calm", "angry", "anxious", "sad", "frustrated", "sensitive", "bored"),
        ),
    ),

    // --- Sleep ---------------------------------------------------------------------
    SLEEP_DURATION(
        "sleep_duration", HealthLogCategory.SLEEP,
        HealthLogControl.Choice(listOf("0_3", "3_6", "6_9", "9_plus")),
    ),
    SLEEP_QUALITY(
        "sleep_quality", HealthLogCategory.SLEEP,
        HealthLogControl.Choice(listOf("good", "medium", "bad")),
    ),

    // --- Body / skin ------------------------------------------------------------------
    ACNE("acne", HealthLogCategory.BODY, HealthLogControl.Toggle),
    OILY_SKIN("oily_skin", HealthLogCategory.BODY, HealthLogControl.Toggle),
    HAIR_LOSS("hair_loss", HealthLogCategory.BODY, HealthLogControl.Toggle),
    SWELLING("swelling", HealthLogCategory.BODY, HealthLogControl.Toggle),
    FATIGUE("fatigue", HealthLogCategory.BODY, HealthLogControl.Toggle),
    DIZZINESS("dizziness", HealthLogCategory.BODY, HealthLogControl.Toggle),
    HOT_FLASHES("hot_flashes", HealthLogCategory.BODY, HealthLogControl.Toggle),
    CHILLS("chills", HealthLogCategory.BODY, HealthLogControl.Toggle),

    // --- Discharge ------------------------------------------------------------------------
    DISCHARGE_TEXTURE(
        "discharge_texture", HealthLogCategory.DISCHARGE,
        HealthLogControl.Choice(listOf("watery", "creamy", "egg_white", "thick")),
    ),
    DISCHARGE_AMOUNT(
        "discharge_amount", HealthLogCategory.DISCHARGE,
        HealthLogControl.Choice(listOf("low", "medium", "high")),
    ),
    DISCHARGE_SMELL(
        "discharge_smell", HealthLogCategory.DISCHARGE,
        HealthLogControl.Choice(listOf("normal", "slightly_unusual", "strong_unpleasant")),
    ),
    DISCHARGE_ITCHING("discharge_itching", HealthLogCategory.DISCHARGE, HealthLogControl.Toggle),
    DISCHARGE_BURNING("discharge_burning", HealthLogCategory.DISCHARGE, HealthLogControl.Toggle),

    // --- Urinary / genital -------------------------------------------------------------------
    FREQUENT_URINATION("frequent_urination", HealthLogCategory.INTIMATE, HealthLogControl.Toggle),
    URINATION_BURNING("urination_burning_intensity", HealthLogCategory.INTIMATE, HealthLogControl.Degree),
    URINATION_CHANGE(
        "urination_change", HealthLogCategory.INTIMATE,
        HealthLogControl.Choice(listOf("increase", "decrease", "normal")),
    ),
    VAGINAL_DRYNESS("vaginal_dryness", HealthLogCategory.INTIMATE, HealthLogControl.Toggle),
    VAGINAL_BURNING("vaginal_burning", HealthLogCategory.INTIMATE, HealthLogControl.Toggle),
    VAGINAL_ITCHING("vaginal_itching", HealthLogCategory.INTIMATE, HealthLogControl.Toggle),
    VAGINAL_SMELL_CHANGE("vaginal_smell_change", HealthLogCategory.INTIMATE, HealthLogControl.Toggle),

    // --- Sexual activity -------------------------------------------------------------------------
    SEXUAL_ACTIVITIES(
        "sexual_activities", HealthLogCategory.SEXUAL,
        HealthLogControl.MultiChoice(
            listOf(
                "high_desire", "protected_intercourse", "unprotected_intercourse", "no_desire",
                "dryness", "burning", "pain_during_intercourse", "bleeding_after_intercourse",
                "lubricant_use",
            ),
        ),
    ),

    // --- Measurements ----------------------------------------------------------------------------
    WEIGHT("weight", HealthLogCategory.MEASURE, HealthLogControl.Measure(min = 30.0, max = 150.0, step = 0.5)),
    BASAL_BODY_TEMPERATURE(
        "basal_body_temperature", HealthLogCategory.MEASURE,
        HealthLogControl.Measure(min = 35.0, max = 42.0, step = 0.1),
    ),

    // --- Notes -----------------------------------------------------------------------------------
    NOTES("notes", HealthLogCategory.NOTES, HealthLogControl.Note),
    ;

    companion object {
        /** The catalog grouped for the log screen, preserving declaration order. */
        val byCategory: Map<HealthLogCategory, List<HealthLogField>> = entries.groupBy { it.category }
    }
}

/**
 * One day's log: only the fields the user actually filled. An empty map means "nothing logged".
 * Values are keyed by field so the form can prefill and the adapter can serialize generically.
 */
data class DailyHealthLog(
    val date: GregorianDate,
    val values: Map<HealthLogField, HealthLogValue>,
) {
    /** How many fields in [category] carry a value — the log screen's "{n} مورد" badge. */
    fun countIn(category: HealthLogCategory): Int = values.keys.count { it.category == category }
}
