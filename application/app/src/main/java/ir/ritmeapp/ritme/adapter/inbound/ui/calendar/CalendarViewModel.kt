package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.LoggedPeriod
import ir.ritmeapp.ritme.domain.model.PeriodDayAction
import ir.ritmeapp.ritme.domain.model.PeriodDayRules
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.CycleInsightsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.inbound.HealthLogUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ManagePeriodsUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Calendar state machine. A displayed Jalali month maps onto one or two Gregorian months;
 * both are fetched concurrently and merged into one ISO-keyed day map. Selecting a day opens
 * the day-detail sheet and loads that day's saved health log, and the logged-period history
 * decides which quick period edit (start / extend / unmark) the day sheet offers.
 */
class CalendarViewModel(
    private val cycleInsights: CycleInsightsUseCase,
    private val healthLog: HealthLogUseCase,
    private val getTrackingMode: GetTrackingModeUseCase,
    private val managePeriods: ManagePeriodsUseCase,
    today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(
        CalendarUiState(year = today.year, month = today.month, today = today, selected = today),
    )
    val state: StateFlow<CalendarUiState> = _state.asStateFlow()

    private val _effects = Channel<CalendarEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    private var dayLogJob: Job? = null

    /**
     * The logged periods, kept out of UI state (the grid renders backend snapshots; this
     * list decides the selected day's action and which cells are real vs predicted). Null
     * until the first load succeeds — no action is offered while unknown, so a stale empty
     * list can't create duplicates.
     */
    private var periods: List<LoggedPeriod>? = null

    init {
        loadMonth()
        loadPeriods()
        loadDayLog(today)
        viewModelScope.launch {
            getTrackingMode().getOrNull()?.let { mode -> _state.update { it.copy(mode = mode) } }
        }
    }

    fun onIntent(intent: CalendarIntent) {
        when (intent) {
            CalendarIntent.PreviousMonth -> shiftMonth(-1)
            CalendarIntent.NextMonth -> shiftMonth(+1)
            is CalendarIntent.SelectDay -> selectDay(intent.day)
            CalendarIntent.ToggleView -> _state.update { it.copy(monthView = !it.monthView) }
            CalendarIntent.Retry -> loadMonth()
            is CalendarIntent.PickMonth -> jumpTo(intent.year, intent.month)
            CalendarIntent.JumpToToday -> {
                val today = _state.value.today
                _state.update { it.copy(selected = today) }
                jumpTo(today.year, today.month)
                loadDayLog(today)
                refreshSelectedAction()
            }
            CalendarIntent.ApplyPeriodAction -> applyPeriodAction()
            CalendarIntent.CloseDaySheet -> _state.update { it.copy(daySheetOpen = false) }
        }
    }

    private fun shiftMonth(delta: Int) {
        _state.update {
            var y = it.year
            var m = it.month + delta
            if (m < 1) { m = 12; y-- }
            if (m > 12) { m = 1; y++ }
            it.copy(year = y, month = m)
        }
        loadMonth()
    }

    private fun jumpTo(year: Int, month: Int) {
        _state.update { it.copy(year = year, month = month) }
        loadMonth()
    }

    /** Tapping a day selects it, opens the detail sheet, and loads its log + offered action. */
    private fun selectDay(day: JalaliDate) {
        _state.update { it.copy(selected = day, daySheetOpen = true) }
        loadDayLog(day)
        refreshSelectedAction()
    }

    /** Recomputes the quick edit offered for the selected day from the known periods. */
    private fun refreshSelectedAction() {
        val known = periods
        _state.update { current ->
            current.copy(
                selectedAction = if (known == null) {
                    PeriodDayAction.None
                } else {
                    PeriodDayRules.actionFor(
                        day = current.selected.toGregorian(),
                        today = current.today.toGregorian(),
                        periods = known,
                    )
                },
            )
        }
    }

    /** Executes the offered action, then re-syncs history + month from the backend. */
    private fun applyPeriodAction() {
        val current = _state.value
        val action = current.selectedAction
        if (action is PeriodDayAction.None || current.periodSaving) return
        viewModelScope.launch {
            Breadcrumbs.add("calendar:period_action")
            // Mirror the web: close the sheet immediately and show the recalculating banner.
            _state.update { it.copy(periodSaving = true, daySheetOpen = false) }
            when (val result = managePeriods.apply(action, current.selected.toGregorian())) {
                is AppResult.Success -> {
                    _state.update { it.copy(isRecalculating = true) }
                    awaitRecalculation()
                    loadPeriods()
                    reloadMonthInPlace()
                    _state.update { it.copy(isRecalculating = false, periodSaving = false) }
                }

                is AppResult.Failure -> {
                    _state.update { it.copy(periodSaving = false, isRecalculating = false) }
                    _effects.send(CalendarEffect.ShowError(result.error.message))
                }
            }
        }
    }

    /** Gives the backend's async recalculation a moment so the refetched month is fresh. */
    private suspend fun awaitRecalculation() {
        repeat(MAX_RECALC_POLLS) {
            if (cycleInsights.isRecalculating().getOrNull() != true) return
            delay(RECALC_POLL_MS)
        }
    }

    private fun loadPeriods() {
        viewModelScope.launch {
            managePeriods.history().getOrNull()?.let { loaded ->
                periods = loaded
                _state.update { it.copy(loggedPeriodDays = loggedDaysOf(loaded)) }
            }
            refreshSelectedAction()
        }
    }

    /** Every ISO day covered by a real logged period (open periods run the default bleed). */
    private fun loggedDaysOf(periods: List<LoggedPeriod>): Set<String> = buildSet {
        periods.forEach { period ->
            val start = period.start.toJalali()
            val endJdn = period.coveredEnd(PeriodDayRules.DEFAULT_BLEED_DAYS).toJalali().toJdn()
            var cursor = start
            while (cursor.toJdn() <= endJdn) {
                add(cursor.toIso())
                cursor = cursor.addDays(1)
            }
        }
    }

    private fun loadMonth() {
        val snapshot = _state.value
        viewModelScope.launch {
            Breadcrumbs.add("calendar:load:${snapshot.year}-${snapshot.month}")
            _state.update { it.copy(loading = true, isError = false) }
            val days = fetchJalaliMonth(snapshot.year, snapshot.month)
            _state.update { current ->
                if (current.year != snapshot.year || current.month != snapshot.month) return@update current
                current.copy(loading = false, isError = days == null, days = days ?: current.days)
            }
        }
    }

    /** Refetches the shown month's data without flashing the loading state (post-edit). */
    private suspend fun reloadMonthInPlace() {
        val snapshot = _state.value
        val days = fetchJalaliMonth(snapshot.year, snapshot.month) ?: return
        _state.update { current ->
            if (current.year != snapshot.year || current.month != snapshot.month) current else current.copy(days = days)
        }
    }

    /** Fetches the Gregorian month(s) the Jalali month spans; null when every fetch failed. */
    private suspend fun fetchJalaliMonth(year: Int, month: Int): Map<String, CycleDaySnapshot>? = coroutineScope {
        val first = JalaliDate(year, month, 1).toGregorian()
        val last = JalaliDate(year, month, JalaliDate(year, month, 1).daysInMonth()).toGregorian()
        val targets = listOf(first.year to first.month, last.year to last.month).distinct()
        val results = targets
            .map { (gy, gm) -> async { cycleInsights.month(gy, gm) } }
            .map { it.await() }
        if (results.all { it is AppResult.Failure }) return@coroutineScope null
        buildMap {
            results.forEach { result -> result.getOrNull()?.byIsoDate()?.let(::putAll) }
        }
    }

    private fun loadDayLog(day: JalaliDate) {
        dayLogJob?.cancel()
        dayLogJob = viewModelScope.launch {
            _state.update { it.copy(dayLogLoading = true, dayLog = null) }
            val result = healthLog.logFor(day.toGregorian())
            _state.update { current ->
                if (current.selected != day) return@update current
                current.copy(
                    dayLogLoading = false,
                    dayLog = result.getOrNull()?.takeIf { it.values.isNotEmpty() },
                )
            }
        }
    }

    private companion object {
        const val MAX_RECALC_POLLS = 6
        const val RECALC_POLL_MS = 500L
    }
}
