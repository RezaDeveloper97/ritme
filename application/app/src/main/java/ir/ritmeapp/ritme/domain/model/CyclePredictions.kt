package ir.ritmeapp.ritme.domain.model

import kotlin.math.max
import kotlin.math.roundToInt

/**
 * Display-ready cycle predictions derived purely from a [CycleCalculation] — the Android port of the
 * web's `deriveCyclePredictions`, kept locale- and framework-free so it is unit-testable (§7). The
 * `daysUntil*` values are offsets from today; the UI turns them into Jalali dates at the boundary.
 */
data class CyclePredictions(
    val cycleDay: Int,
    val cycleLength: Int,
    val phase: CyclePhase?,
    val fertilityPercent: Int,
    val daysUntilNextPeriod: Int,
    val daysUntilOvulation: Int,
    val daysUntilFertileWindow: Int,
    val daysUntilPmsStart: Int,
    val daysUntilPmsEnd: Int,
    val isFertileWindow: Boolean,
    val isPeriodTomorrow: Boolean,
) {
    /** Cycle progress as a 0..100 percentage for the hero card's bar. */
    val progressPercent: Int get() = if (cycleLength <= 0) 0 else ((cycleDay.toDouble() / cycleLength) * 100).roundToInt().coerceIn(0, 100)

    companion object {
        private const val FERTILE_WINDOW_LEAD_DAYS = 4
        private const val PMS_WINDOW_DAYS = 4

        /** Derives predictions, or null when the calculation has no cycle data yet. */
        fun from(calc: CycleCalculation): CyclePredictions? {
            val day = calc.cycleDay ?: return null
            val length = calc.cycleLength ?: return null
            val daysUntilNextPeriod = max(0, length - day)
            val daysUntilOvulation = calc.estimatedOvulationDay?.let { it - day } ?: 0
            return CyclePredictions(
                cycleDay = day,
                cycleLength = length,
                phase = calc.phase,
                fertilityPercent = (calc.fertilityPercent ?: 0.0).roundToInt().coerceIn(0, 100),
                daysUntilNextPeriod = daysUntilNextPeriod,
                daysUntilOvulation = daysUntilOvulation,
                daysUntilFertileWindow = daysUntilOvulation - FERTILE_WINDOW_LEAD_DAYS,
                daysUntilPmsStart = max(0, daysUntilNextPeriod - PMS_WINDOW_DAYS),
                daysUntilPmsEnd = max(0, daysUntilNextPeriod - 1),
                isFertileWindow = calc.isFertileWindow,
                isPeriodTomorrow = calc.isPeriodTomorrow,
            )
        }
    }
}
