package ir.ritmeapp.ritme.domain.model

/**
 * One safety alert raised by the backend from logged symptoms (`GET /pregnancy/alerts`).
 * Emergency-level alerts are rendered loudest and first.
 */
data class PregnancyAlert(
    val id: Long,
    val level: PregnancyAlertLevel,
    val title: String,
    val message: String,
    val recommendedActions: List<String>,
    val isRead: Boolean,
)

/** Alert severity, ordered from calm to act-now. */
enum class PregnancyAlertLevel(val apiValue: String) {
    INFO("info"),
    WARNING("warning"),
    EMERGENCY("emergency");

    companion object {
        fun fromApi(value: String?): PregnancyAlertLevel = entries.firstOrNull { it.apiValue == value } ?: INFO
    }
}
