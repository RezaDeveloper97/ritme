package ir.ritmeapp.ritme.domain.model

/**
 * The personalized daily message from `GET /messages/daily` (its `primary_message` plus a little
 * cycle context). Strings arrive already localized from the backend; the UI shows them verbatim.
 * Home reads [longMessage]/[actionSuggestion] for the smart-tip card and [dos] for recommendations.
 */
data class DailyMessage(
    val phase: CyclePhase?,
    val phaseLabel: String?,
    val cycleDay: Int?,
    val shortMessage: String,
    val longMessage: String,
    val actionSuggestion: String,
    val dos: List<String>,
    val donts: List<String>,
)
