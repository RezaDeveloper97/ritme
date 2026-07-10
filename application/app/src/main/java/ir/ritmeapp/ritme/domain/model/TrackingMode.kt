package ir.ritmeapp.ritme.domain.model

/**
 * Which tracker the account is running (`GET /messages/mode`). Mode is first-class in Ritme:
 * the bottom navigation, Home feed, and log screens all reshape around it.
 */
enum class TrackingMode(val apiValue: String) {
    CYCLE("cycle"),
    PREGNANCY("pregnancy");

    companion object {
        /** Maps the backend token; anything unknown falls back to cycle mode. */
        fun fromApi(value: String?): TrackingMode = entries.firstOrNull { it.apiValue == value } ?: CYCLE
    }
}
