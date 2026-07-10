package ir.ritmeapp.ritme.domain.model

/**
 * One month of per-day cycle calculations (`GET /cycle/month/{year}/{month}`) — the calendar
 * screen's data. Days arrive Gregorian (the API's calendar); the UI keys them by ISO date when
 * painting the Jalali grid.
 */
data class CycleMonth(
    val days: List<CycleDaySnapshot>,
    val summary: CycleMonthSummary,
) {
    /** Fast phase lookup for the calendar grid, keyed by ISO `yyyy-MM-dd`. */
    fun byIsoDate(): Map<String, CycleDaySnapshot> = days.associateBy { it.date.toIso() }
}

/** The render-relevant slice of one day's calculation for calendar coloring + the detail card. */
data class CycleDaySnapshot(
    val date: GregorianDate,
    val cycleDay: Int?,
    val phase: CyclePhase?,
    val isFertileWindow: Boolean,
    val isPmsWindow: Boolean,
    /** Conception probability percent for the "chance" tile (backend `final_probability`). */
    val fertilityPercent: Double?,
)

/** The backend's `month_summary` counts. */
data class CycleMonthSummary(
    val fertileDays: Int,
    val periodDays: Int,
    val pmsDays: Int,
)
