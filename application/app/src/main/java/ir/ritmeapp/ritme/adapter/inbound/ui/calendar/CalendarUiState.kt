package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.PeriodDayAction
import ir.ritmeapp.ritme.domain.model.TrackingMode

/**
 * The calendar screen's single immutable state (§4 MVI): the displayed Jalali month, the
 * per-day cycle snapshots keyed by ISO date (a Jalali month straddles two Gregorian months,
 * pre-merged by the ViewModel), the selected day, that day's saved health log, and the one
 * period edit action the rules offer for it.
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
    /** The one quick period edit offered for the selected day (start / extend / unmark). */
    val selectedAction: PeriodDayAction = PeriodDayAction.None,
    /** A period mutation is in flight — action buttons disable and the grid shows progress. */
    val periodSaving: Boolean = false,
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

    /** Jump straight to a Jalali month picked from the month/year sheet. */
    data class PickMonth(val year: Int, val month: Int) : CalendarIntent

    /** Return the view (and selection) to today. */
    data object JumpToToday : CalendarIntent

    /** Execute the period edit currently offered for the selected day. */
    data object ApplyPeriodAction : CalendarIntent
}
