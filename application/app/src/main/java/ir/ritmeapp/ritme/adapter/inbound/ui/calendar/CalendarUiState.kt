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
    /** True while the tapped day's detail sheet is open (web `daySheetOpen`). */
    val daySheetOpen: Boolean = false,
    /** The backend is re-deriving the cycle after a period edit — drives the top banner. */
    val isRecalculating: Boolean = false,
    /**
     * ISO `yyyy-MM-dd` days covered by a REAL logged period (vs an engine prediction).
     * A period-colored day not in this set is only predicted, so the grid draws it as a
     * hollow ring rather than a solid fill (web spec §12).
     */
    val loggedPeriodDays: Set<String> = emptySet(),
) {
    /** The cycle snapshot behind one Jalali day cell, if the month data covers it. */
    fun snapshotFor(day: JalaliDate): CycleDaySnapshot? = days[day.toIso()]

    /** The selected day's snapshot for the detail card. */
    val selectedSnapshot: CycleDaySnapshot? get() = snapshotFor(selected)

    /** Whether [day] is inside a real logged period (a solid period cell, not a prediction). */
    fun isLoggedPeriodDay(day: JalaliDate): Boolean = day.toIso() in loggedPeriodDays

    /** True when the shown Jalali month is entirely in the future (no day could be bled on). */
    val isFutureMonth: Boolean
        get() = year > today.year || (year == today.year && month > today.month)

    /** True when the selected day is after today — future days can't carry logged data. */
    val selectedIsFuture: Boolean
        get() = selected.toJdn() > today.toJdn()
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

    /** Dismiss the tapped-day detail sheet without acting. */
    data object CloseDaySheet : CalendarIntent
}

/** One-shot side effects (§4 Effect) — surfaced once, never replayed on recomposition. */
sealed interface CalendarEffect {
    /** A period edit the backend rejected → show its Persian message as a toast. */
    data class ShowError(val message: String) : CalendarEffect
}
