package ir.ritmeapp.ritme.domain.model

/**
 * One day's cycle calculation from `GET /cycle/today`. The backend never sends a literally-null
 * object; instead an "empty" calculation (no profile / no last period) arrives with [cycleDay] null
 * and every flag false — [hasData] collapses that to the "no cycle data yet" state the UI shows as
 * unavailable. Internal scoring fields from the API are intentionally dropped (§11).
 */
data class CycleCalculation(
    val cycleDay: Int?,
    val cycleLength: Int?,
    val phase: CyclePhase?,
    val estimatedOvulationDay: Int?,
    /** Conception probability as a percent (backend `final_probability`, capped ~35). */
    val fertilityPercent: Double?,
    val isFertileWindow: Boolean,
    val isPeriodTomorrow: Boolean,
    val isRecalculating: Boolean,
    /** Backend regularity verdict (`regular` / `semi_irregular` / `irregular`), if computed. */
    val variability: String? = null,
) {
    /** True once there is real cycle data to show (a known cycle day). */
    val hasData: Boolean get() = cycleDay != null
}
