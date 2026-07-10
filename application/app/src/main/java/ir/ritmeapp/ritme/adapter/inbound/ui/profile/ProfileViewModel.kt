package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.DeleteAccountUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ExportDataUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetTrackingModeUseCase
import ir.ritmeapp.ritme.domain.port.inbound.GetUserProfileUseCase
import ir.ritmeapp.ritme.domain.port.inbound.LogoutUseCase
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Profile hub state machine: loads the profile + mode, and runs logout / mode switch /
 * export / account deletion. Navigation after sign-out and the export hand-off are one-shot
 * [ProfileEffect]s, never state (§4).
 */
class ProfileViewModel(
    private val getUserProfile: GetUserProfileUseCase,
    private val getTrackingMode: GetTrackingModeUseCase,
    private val logout: LogoutUseCase,
    private val deleteAccount: DeleteAccountUseCase,
    private val exportData: ExportDataUseCase,
    private val pregnancyTracker: PregnancyTrackerUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(ProfileUiState())
    val state: StateFlow<ProfileUiState> = _state.asStateFlow()

    private val _effects = Channel<ProfileEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    init {
        load()
    }

    fun onIntent(intent: ProfileIntent) {
        when (intent) {
            ProfileIntent.Retry -> load()
            ProfileIntent.Logout -> signOut()
            ProfileIntent.SwitchToCycleMode -> switchToCycle()
            ProfileIntent.Export -> export()
            ProfileIntent.OpenDeleteSheet -> _state.update { it.copy(deleteSheetOpen = true, deleteError = false) }
            ProfileIntent.CloseDeleteSheet -> _state.update { it.copy(deleteSheetOpen = false) }
            ProfileIntent.ConfirmDelete -> confirmDelete()
        }
    }

    private fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, isError = false) }
            val profile = getUserProfile()
            val mode = getTrackingMode().getOrNull() ?: TrackingMode.CYCLE
            _state.update {
                it.copy(
                    loading = false,
                    isError = profile is AppResult.Failure,
                    profile = profile.getOrNull() ?: it.profile,
                    mode = mode,
                )
            }
        }
    }

    private fun signOut() {
        if (_state.value.loggingOut) return
        viewModelScope.launch {
            Breadcrumbs.add("profile:logout")
            _state.update { it.copy(loggingOut = true) }
            logout()
            _effects.send(ProfileEffect.SignedOut)
        }
    }

    private fun switchToCycle() {
        if (_state.value.switchingMode) return
        viewModelScope.launch {
            Breadcrumbs.add("profile:deactivate_pregnancy")
            _state.update { it.copy(switchingMode = true) }
            if (pregnancyTracker.deactivate() is AppResult.Success) {
                _state.update { it.copy(switchingMode = false, mode = TrackingMode.CYCLE) }
            } else {
                _state.update { it.copy(switchingMode = false) }
            }
        }
    }

    private fun export() {
        if (_state.value.exporting) return
        viewModelScope.launch {
            Breadcrumbs.add("profile:export")
            _state.update { it.copy(exporting = true, exportError = false) }
            when (val result = exportData()) {
                is AppResult.Failure -> _state.update { it.copy(exporting = false, exportError = true) }
                is AppResult.Success -> {
                    _state.update { it.copy(exporting = false) }
                    _effects.send(ProfileEffect.ShareExport(result.value))
                }
            }
        }
    }

    private fun confirmDelete() {
        if (_state.value.deleting) return
        viewModelScope.launch {
            Breadcrumbs.add("profile:delete_account")
            _state.update { it.copy(deleting = true, deleteError = false) }
            when (deleteAccount()) {
                is AppResult.Failure -> _state.update { it.copy(deleting = false, deleteError = true) }
                is AppResult.Success -> _effects.send(ProfileEffect.SignedOut)
            }
        }
    }
}
