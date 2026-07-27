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
import kotlin.math.roundToInt

/** Save progress shared by both profile edit forms. */
enum class EditSaveState { IDLE, SAVING, ERROR }

/** One-shot effects shared by both profile edit forms (MVI §4 — off a channel, not state). */
sealed interface ProfileEditEffect {
    /** A successful save; the screen pops back to the profile, mirroring web `router.back()`. */
    data object NavigateBack : ProfileEditEffect
}

/** The personal-info form (name / birthday / weight / height), prefilled from `GET /profile`. */
@Immutable
data class ProfilePersonalUiState(
    val loading: Boolean = true,
    val name: String = "",
    val birthday: JalaliDate,
    val weightKg: Int = DEFAULT_WEIGHT_KG,
    val heightCm: Int = DEFAULT_HEIGHT_CM,
    val saveState: EditSaveState = EditSaveState.IDLE,
    val birthdayInFuture: Boolean = false,
) {
    companion object {
        const val DEFAULT_WEIGHT_KG = 60
        const val DEFAULT_HEIGHT_CM = 165
        const val MIN_WEIGHT_KG = 30
        const val MAX_WEIGHT_KG = 200
        const val MIN_HEIGHT_CM = 100
        const val MAX_HEIGHT_CM = 220
        const val MIN_BIRTH_YEAR = 1330
    }
}

sealed interface ProfilePersonalIntent {
    data class NameChanged(val value: String) : ProfilePersonalIntent
    data class BirthdayChanged(val value: JalaliDate) : ProfilePersonalIntent
    data class WeightChanged(val kg: Int) : ProfilePersonalIntent
    data class HeightChanged(val cm: Int) : ProfilePersonalIntent
    data object Save : ProfilePersonalIntent
}

class ProfilePersonalViewModel(
    private val getUserProfile: GetUserProfileUseCase,
    private val saveProfile: SaveProfileUseCase,
    private val today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(
        // Mirror web PersonalForm: a user with no saved birthday starts 25 years back.
        ProfilePersonalUiState(birthday = JalaliDate(today.year - DEFAULT_BIRTH_AGE_YEARS, 1, 1)),
    )
    val state: StateFlow<ProfilePersonalUiState> = _state.asStateFlow()

    private val _effects = Channel<ProfileEditEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    init {
        viewModelScope.launch {
            val profile = getUserProfile().getOrNull()
            _state.update { current ->
                current.copy(
                    loading = false,
                    name = profile?.account?.name.orEmpty(),
                    birthday = profile?.health?.birthday?.toJalali() ?: current.birthday,
                    weightKg = profile?.health?.weightKg?.roundToInt() ?: current.weightKg,
                    heightCm = profile?.health?.heightCm ?: current.heightCm,
                )
            }
        }
    }

    fun onIntent(intent: ProfilePersonalIntent) {
        when (intent) {
            is ProfilePersonalIntent.NameChanged ->
                _state.update { it.copy(name = intent.value, saveState = EditSaveState.IDLE) }

            is ProfilePersonalIntent.BirthdayChanged ->
                _state.update { it.copy(birthday = intent.value, saveState = EditSaveState.IDLE) }

            is ProfilePersonalIntent.WeightChanged ->
                _state.update { it.copy(weightKg = intent.kg, saveState = EditSaveState.IDLE) }

            is ProfilePersonalIntent.HeightChanged ->
                _state.update { it.copy(heightCm = intent.cm, saveState = EditSaveState.IDLE) }

            ProfilePersonalIntent.Save -> save()
        }
    }

    private fun save() {
        val snapshot = _state.value
        if (snapshot.saveState == EditSaveState.SAVING) return
        // Web PersonalForm.handleSave runs the future-date check on press (not eagerly),
        // surfacing the localized error only after the user taps Save.
        if (snapshot.birthday.toJdn() >= today.toJdn()) {
            _state.update { it.copy(birthdayInFuture = true, saveState = EditSaveState.IDLE) }
            return
        }
        viewModelScope.launch {
            Breadcrumbs.add("profile:save_personal")
            _state.update { it.copy(birthdayInFuture = false, saveState = EditSaveState.SAVING) }
            val result = saveProfile(
                OnboardingAnswers(
                    name = snapshot.name.trim().takeIf { it.isNotEmpty() },
                    birthday = snapshot.birthday,
                    weightKg = snapshot.weightKg,
                    heightCm = snapshot.heightCm,
                ),
            )
            if (result is AppResult.Success) {
                _effects.send(ProfileEditEffect.NavigateBack)
            } else {
                _state.update { it.copy(saveState = EditSaveState.ERROR) }
            }
        }
    }

    private companion object {
        const val DEFAULT_BIRTH_AGE_YEARS = 25
    }
}
