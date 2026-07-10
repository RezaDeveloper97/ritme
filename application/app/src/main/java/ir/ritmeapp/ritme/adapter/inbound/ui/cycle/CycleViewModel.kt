package ir.ritmeapp.ritme.adapter.inbound.ui.cycle

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.CycleInsightsUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetHomeDashboardUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Cycle screen state machine. Reuses the Home dashboard use case (today's calculation +
 * daily message are exactly this screen's data). A manual recalculation flips
 * [CycleUiState.isRecalculating] and polls the status every few seconds until the backend
 * finishes, then reloads — mirroring the web's 4-second polling.
 */
class CycleViewModel(
    private val getDashboard: GetHomeDashboardUseCase,
    private val cycleInsights: CycleInsightsUseCase,
    private val getTrackingMode: GetTrackingModeUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(CycleUiState())
    val state: StateFlow<CycleUiState> = _state.asStateFlow()

    private var pollJob: Job? = null

    init {
        load()
        viewModelScope.launch {
            getTrackingMode().getOrNull()?.let { mode -> _state.update { it.copy(mode = mode) } }
        }
    }

    fun onIntent(intent: CycleIntent) {
        when (intent) {
            CycleIntent.Recalculate -> recalculate()
            CycleIntent.Retry -> load()
        }
    }

    private fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, isError = false) }
            when (val result = getDashboard()) {
                is AppResult.Failure -> _state.update { it.copy(loading = false, isError = true) }
                is AppResult.Success -> {
                    val dashboard = result.value
                    _state.update {
                        it.copy(
                            loading = false,
                            calculation = dashboard.calculation,
                            predictions = dashboard.predictions,
                            message = dashboard.message,
                            isRecalculating = dashboard.calculation.isRecalculating,
                        )
                    }
                    if (dashboard.calculation.isRecalculating) pollUntilDone()
                }
            }
        }
    }

    private fun recalculate() {
        if (_state.value.isRecalculating) return
        viewModelScope.launch {
            Breadcrumbs.add("cycle:recalculate")
            when (cycleInsights.recalculate()) {
                is AppResult.Failure -> Unit // Non-fatal; the screen simply stays as-is.
                is AppResult.Success -> {
                    _state.update { it.copy(isRecalculating = true) }
                    pollUntilDone()
                }
            }
        }
    }

    private fun pollUntilDone() {
        pollJob?.cancel()
        pollJob = viewModelScope.launch {
            repeat(MAX_POLLS) {
                delay(POLL_INTERVAL_MS)
                val running = cycleInsights.isRecalculating().getOrNull()
                if (running == false) {
                    _state.update { it.copy(isRecalculating = false) }
                    load()
                    return@launch
                }
            }
            // Backend never reported completion; stop claiming progress but keep the data shown.
            _state.update { it.copy(isRecalculating = false) }
        }
    }

    private companion object {
        const val POLL_INTERVAL_MS = 4_000L
        const val MAX_POLLS = 30
    }
}
