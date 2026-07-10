package ir.ritmeapp.ritme.domain.model

/**
 * A Ritme service entry shown on the home screen. The set mirrors the
 * features the app offers (cycle tracking, pregnancy, logs, calendar, content).
 * Immutable value object with a stable [id] so lists can key on it (CLAUDE.md §5).
 */
data class HomeService(
    val id: String,
    val title: String,
    val tagline: String,
    val accent: ServiceAccent,
)

/**
 * The brand accent a service card is drawn with. Kept as a small closed set here (not a
 * raw color) so the domain owns no Android color type — the UI maps each case to a token.
 */
enum class ServiceAccent { Pink, Accent, Info }
