package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.TrackingMode

/**
 * The calendar screen's single immutable state (§4 MVI): the displayed Jalali month, the
 * per-day cycle snapshots keyed by ISO date (a Jalali month straddles two Gregorian months,
 * pre-merged by the ViewModel), the selected day, and that day's saved health log.
 */
@Immutable
data class CalendarUiState(
    val year: Int,
    val month: Int,
    val today: JalaliDate,
    val selected: JalaliDate,
    val loading: Boolean = true,
    val isError: Boolean = false,
    val days: Map<String, CycleDaySnapshot> = emptyMap(),
    val monthView: Boolean = true,
    val dayLog: DailyHealthLog? = null,
    val dayLogLoading: Boolean = false,
    val mode: TrackingMode = TrackingMode.CYCLE,
) {
    /** The cycle snapshot behind one Jalali day cell, if the month data covers it. */
    fun snapshotFor(day: JalaliDate): CycleDaySnapshot? = days[day.toIso()]

    /** The selected day's snapshot for the detail card. */
    val selectedSnapshot: CycleDaySnapshot? get() = snapshotFor(selected)
}

/** Everything the user can do on the calendar screen. */
sealed interface CalendarIntent {
    data object PreviousMonth : CalendarIntent
    data object NextMonth : CalendarIntent
    data class SelectDay(val day: JalaliDate) : CalendarIntent
    data object ToggleView : CalendarIntent
    data object Retry : CalendarIntent
}
