package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.GetUserProfileUseCase
import ir.ritmeapp.ritme.domain.port.inbound.SaveProfileUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** The cycle-settings form (cycle length / period length / last period start). */
@Immutable
data class ProfileHealthUiState(
    val loading: Boolean = true,
    val cycleDuration: Int = DEFAULT_CYCLE_DAYS,
    val periodDuration: Int = DEFAULT_PERIOD_DAYS,
    val lastPeriod: JalaliDate,
    val saveState: EditSaveState = EditSaveState.IDLE,
    val lastPeriodInFuture: Boolean = false,
) {
    companion object {
        const val DEFAULT_CYCLE_DAYS = 28
        const val DEFAULT_PERIOD_DAYS = 6
        const val MIN_CYCLE_DAYS = 15
        const val MAX_CYCLE_DAYS = 60
        const val MIN_PERIOD_DAYS = 1
        const val MAX_PERIOD_DAYS = 15
    }
}

sealed interface ProfileHealthIntent {
    data class CycleChanged(val days: Int) : ProfileHealthIntent
    data class PeriodChanged(val days: Int) : ProfileHealthIntent
    data class LastPeriodChanged(val value: JalaliDate) : ProfileHealthIntent
    data object Save : ProfileHealthIntent
}

class ProfileHealthViewModel(
    private val getUserProfile: GetUserProfileUseCase,
    private val saveProfile: SaveProfileUseCase,
    private val today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(ProfileHealthUiState(lastPeriod = today))
    val state: StateFlow<ProfileHealthUiState> = _state.asStateFlow()

    private val _effects = Channel<ProfileEditEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    init {
        viewModelScope.launch {
            val health = getUserProfile().getOrNull()?.health
            _state.update { current ->
                current.copy(
                    loading = false,
                    cycleDuration = health?.cycleDuration ?: current.cycleDuration,
                    periodDuration = health?.periodDuration ?: current.periodDuration,
                    lastPeriod = health?.lastPeriodStart?.toJalali() ?: current.lastPeriod,
                )
            }
        }
    }

    fun onIntent(intent: ProfileHealthIntent) {
        when (intent) {
            is ProfileHealthIntent.CycleChanged ->
                _state.update { it.copy(cycleDuration = intent.days, saveState = EditSaveState.IDLE) }

            is ProfileHealthIntent.PeriodChanged ->
                _state.update { it.copy(periodDuration = intent.days, saveState = EditSaveState.IDLE) }

            is ProfileHealthIntent.LastPeriodChanged ->
                _state.update { it.copy(lastPeriod = intent.value, saveState = EditSaveState.IDLE) }

            ProfileHealthIntent.Save -> save()
        }
    }

    private fun save() {
        val snapshot = _state.value
        if (snapshot.saveState == EditSaveState.SAVING) return
        // Web HealthForm.handleSave runs the future-date check on press (not eagerly),
        // surfacing the localized error only after the user taps Save.
        if (snapshot.lastPeriod.toJdn() > today.toJdn()) {
            _state.update { it.copy(lastPeriodInFuture = true, saveState = EditSaveState.IDLE) }
            return
        }
        viewModelScope.launch {
            Breadcrumbs.add("profile:save_health")
            _state.update { it.copy(lastPeriodInFuture = false, saveState = EditSaveState.SAVING) }
            val result = saveProfile(
                OnboardingAnswers(
                    periodDuration = snapshot.periodDuration,
                    cycleDuration = snapshot.cycleDuration,
                    lastPeriod = snapshot.lastPeriod,
                ),
            )
            if (result is AppResult.Success) {
                _effects.send(ProfileEditEffect.NavigateBack)
            } else {
                _state.update { it.copy(saveState = EditSaveState.ERROR) }
            }
        }
    }
}
