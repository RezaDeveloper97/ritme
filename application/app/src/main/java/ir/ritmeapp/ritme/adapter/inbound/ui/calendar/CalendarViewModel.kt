package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.CycleInsightsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.inbound.HealthLogUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Calendar state machine. A displayed Jalali month maps onto one or two Gregorian months;
 * both are fetched concurrently and merged into one ISO-keyed day map. Selecting a day
 * loads that day's saved health log for the recap card.
 */
class CalendarViewModel(
    private val cycleInsights: CycleInsightsUseCase,
    private val healthLog: HealthLogUseCase,
    private val getTrackingMode: GetTrackingModeUseCase,
    today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(
        CalendarUiState(year = today.year, month = today.month, today = today, selected = today),
    )
    val state: StateFlow<CalendarUiState> = _state.asStateFlow()

    private var dayLogJob: Job? = null

    init {
        loadMonth()
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

    private fun selectDay(day: JalaliDate) {
        _state.update { it.copy(selected = day) }
        loadDayLog(day)
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
}
