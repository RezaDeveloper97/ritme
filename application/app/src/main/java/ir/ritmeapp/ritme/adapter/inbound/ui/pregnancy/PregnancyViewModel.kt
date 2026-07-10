package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyProfile
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.async
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * The pregnancy tracker's single immutable state (§4 MVI). [needsOnboarding] gates the
 * activate/setup flow; once tracking, [selectedWeek] drives the educational-content browser
 * independently of the actual [PregnancyStatus.currentWeek].
 */
@Immutable
data class PregnancyUiState(
    val loading: Boolean = true,
    val isError: Boolean = false,
    val status: PregnancyStatus? = null,
    val profile: PregnancyProfile? = null,
    val activating: Boolean = false,
    val selectedWeek: Int = 1,
    val content: PregnancyWeekContent? = null,
    val contentLoading: Boolean = false,
    val alerts: List<PregnancyAlert> = emptyList(),
) {
    /** True until pregnancy mode is active AND its onboarding answers exist. */
    val needsOnboarding: Boolean
        get() = status?.isActive != true || profile?.onboardingCompleted != true
}

sealed interface PregnancyIntent {
    data object Retry : PregnancyIntent
    data object Activate : PregnancyIntent
    data object PreviousWeek : PregnancyIntent
    data object NextWeek : PregnancyIntent
    data object ResetWeek : PregnancyIntent
    data class MarkAlertRead(val id: Long) : PregnancyIntent
    data class DismissAlert(val id: Long) : PregnancyIntent
    data object MarkAllAlertsRead : PregnancyIntent
}

/** One-shot: activation succeeded, continue into the pregnancy onboarding form. */
sealed interface PregnancyEffect {
    data object GoToOnboarding : PregnancyEffect
}

/** Pregnancy tracker state machine over `/pregnancy/…`. */
class PregnancyViewModel(
    private val tracker: PregnancyTrackerUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(PregnancyUiState())
    val state: StateFlow<PregnancyUiState> = _state.asStateFlow()

    private val _effects = Channel<PregnancyEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    private var contentJob: Job? = null

    init {
        load()
    }

    fun onIntent(intent: PregnancyIntent) {
        when (intent) {
            PregnancyIntent.Retry -> load()
            PregnancyIntent.Activate -> activate()
            PregnancyIntent.PreviousWeek -> selectWeek(_state.value.selectedWeek - 1)
            PregnancyIntent.NextWeek -> selectWeek(_state.value.selectedWeek + 1)
            PregnancyIntent.ResetWeek -> selectWeek(_state.value.status?.currentWeek ?: 1)
            is PregnancyIntent.MarkAlertRead -> alertAction(intent.id) { tracker.markAlertRead(it) }
            is PregnancyIntent.DismissAlert -> dismissAlert(intent.id)
            PregnancyIntent.MarkAllAlertsRead -> markAllAlertsRead()
        }
    }

    private fun load() {
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy:load")
            _state.update { it.copy(loading = true, isError = false) }
            val statusDeferred = async { tracker.status() }
            val profileDeferred = async { tracker.profile() }
            val status = statusDeferred.await()
            val profile = profileDeferred.await()
            if (status is AppResult.Failure && profile is AppResult.Failure) {
                _state.update { it.copy(loading = false, isError = true) }
                return@launch
            }
            val loadedStatus = status.getOrNull()
            _state.update {
                it.copy(
                    loading = false,
                    status = loadedStatus,
                    profile = profile.getOrNull(),
                    selectedWeek = (loadedStatus?.currentWeek ?: 1).coerceIn(1, PregnancyStatus.TOTAL_WEEKS),
                )
            }
            if (!_state.value.needsOnboarding) {
                loadContent(_state.value.selectedWeek)
                loadAlerts()
            }
        }
    }

    private fun activate() {
        if (_state.value.activating) return
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy:activate")
            _state.update { it.copy(activating = true) }
            when (tracker.activate()) {
                is AppResult.Failure -> _state.update { it.copy(activating = false) }
                is AppResult.Success -> {
                    _state.update { it.copy(activating = false) }
                    _effects.send(PregnancyEffect.GoToOnboarding)
                }
            }
        }
    }

    private fun selectWeek(week: Int) {
        val bounded = week.coerceIn(1, PregnancyStatus.TOTAL_WEEKS)
        if (bounded == _state.value.selectedWeek) return
        _state.update { it.copy(selectedWeek = bounded) }
        loadContent(bounded)
    }

    private fun loadContent(week: Int) {
        contentJob?.cancel()
        contentJob = viewModelScope.launch {
            _state.update { it.copy(contentLoading = true) }
            val content = tracker.contentForWeek(week).getOrNull()
            _state.update { current ->
                if (current.selectedWeek != week) return@update current
                current.copy(contentLoading = false, content = content)
            }
        }
    }

    private fun loadAlerts() {
        viewModelScope.launch {
            tracker.alerts().getOrNull()?.let { alerts -> _state.update { it.copy(alerts = alerts) } }
        }
    }

    private fun alertAction(id: Long, action: suspend (Long) -> AppResult<Unit>) {
        _state.update { current ->
            current.copy(alerts = current.alerts.map { if (it.id == id) it.copy(isRead = true) else it })
        }
        viewModelScope.launch {
            if (action(id) is AppResult.Failure) loadAlerts()
        }
    }

    private fun dismissAlert(id: Long) {
        _state.update { current -> current.copy(alerts = current.alerts.filterNot { it.id == id }) }
        viewModelScope.launch {
            if (tracker.dismissAlert(id) is AppResult.Failure) loadAlerts()
        }
    }

    private fun markAllAlertsRead() {
        _state.update { current -> current.copy(alerts = current.alerts.map { it.copy(isRead = true) }) }
        viewModelScope.launch {
            if (tracker.markAllAlertsRead() is AppResult.Failure) loadAlerts()
        }
    }
}
