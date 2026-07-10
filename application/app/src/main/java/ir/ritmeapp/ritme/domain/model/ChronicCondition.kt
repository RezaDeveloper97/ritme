package ir.ritmeapp.ritme.domain.model

/**
 * A self-reported chronic condition offered as an optional multi-select at onboarding. [apiValue]
 * is the exact token the backend's `chronic_conditions[]` enum accepts; the user-facing Persian
 * label is resolved in the UI layer, not here (§11 — the domain stays locale-free).
 */
enum class ChronicCondition(val apiValue: String) {
    PCOS("pcos"),
    HYPOTHYROIDISM("hypothyroidism"),
    HYPERTHYROIDISM("hyperthyroidism"),
    HYPERTENSION("hypertension"),
    HEART_DISEASE("heart_disease"),
    DIABETES("diabetes"),
    ;

    companion object {
        /** Maps a backend token back to the enum; unknown tokens are dropped, not errors. */
        fun fromApi(value: String?): ChronicCondition? = entries.firstOrNull { it.apiValue == value }
    }
}
