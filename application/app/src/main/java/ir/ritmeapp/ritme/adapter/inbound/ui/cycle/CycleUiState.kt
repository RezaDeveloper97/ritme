package ir.ritmeapp.ritme.adapter.inbound.ui.cycle

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.CycleCalculation
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.DailyMessage
import ir.ritmeapp.ritme.domain.model.TrackingMode

/**
 * The cycle screen's single immutable state (§4 MVI): today's calculation + derived
 * predictions, the personalized message, and the manual-recalculation progress flag.
 */
@Immutable
data class CycleUiState(
    val loading: Boolean = true,
    val isError: Boolean = false,
    val calculation: CycleCalculation? = null,
    val predictions: CyclePredictions? = null,
    val message: DailyMessage? = null,
    val isRecalculating: Boolean = false,
    val mode: TrackingMode = TrackingMode.CYCLE,
) {
    /** True once there is real cycle data to render (otherwise the empty state shows). */
    val hasCycle: Boolean get() = predictions != null
}

/** Everything the user can do on the cycle screen. */
sealed interface CycleIntent {
    data object Recalculate : CycleIntent
    data object Retry : CycleIntent
}
