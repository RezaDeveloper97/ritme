package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.PregnancyOnboardingAnswers
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** Which validation message (if any) blocks submission. */
enum class PregnancyOnboardingError { NONE, SELECT_SOURCE, FILL_REQUIRED, SUBMIT_FAILED }

/**
 * The pregnancy onboarding form (§4 MVI): a dating source with its conditional inputs, plus
 * optional history/conditions/blood-type context.
 */
@Immutable
data class PregnancyOnboardingUiState(
    val source: PregnancyAgeSource? = null,
    val lmpDate: JalaliDate,
    val ultrasoundDate: JalaliDate,
    val ultrasoundWeeks: Int = DEFAULT_WEEKS,
    val ultrasoundDays: Int = 0,
    val manualWeeks: Int = DEFAULT_WEEKS,
    val manualDays: Int = 0,
    val hasMiscarriageHistory: Boolean = false,
    val hasHighRiskHistory: Boolean = false,
    val conditions: List<String> = emptyList(),
    val bloodType: String? = null,
    val rhFactor: String? = null,
    val submitting: Boolean = false,
    val error: PregnancyOnboardingError = PregnancyOnboardingError.NONE,
) {
    companion object {
        const val DEFAULT_WEEKS = 8
        const val MAX_WEEKS = 42
        const val MAX_EXTRA_DAYS = 6

        /** The backend's `pre_existing_conditions` enum; `none` is mutually exclusive. */
        val CONDITION_OPTIONS = listOf(
            "chronic_hypertension", "diabetes", "hypothyroidism", "hyperthyroidism", "none",
        )
        val BLOOD_TYPES = listOf("A", "B", "AB", "O")
        val RH_FACTORS = listOf("positive", "negative")
    }
}

sealed interface PregnancyOnboardingIntent {
    data class SourceSelected(val source: PregnancyAgeSource) : PregnancyOnboardingIntent
    data class LmpChanged(val date: JalaliDate) : PregnancyOnboardingIntent
    data class UltrasoundDateChanged(val date: JalaliDate) : PregnancyOnboardingIntent
    data class UltrasoundWeeksChanged(val weeks: Int) : PregnancyOnboardingIntent
    data class UltrasoundDaysChanged(val days: Int) : PregnancyOnboardingIntent
    data class ManualWeeksChanged(val weeks: Int) : PregnancyOnboardingIntent
    data class ManualDaysChanged(val days: Int) : PregnancyOnboardingIntent
    data class MiscarriageToggled(val value: Boolean) : PregnancyOnboardingIntent
    data class HighRiskToggled(val value: Boolean) : PregnancyOnboardingIntent
    data class ConditionToggled(val condition: String) : PregnancyOnboardingIntent
    data class BloodTypeSelected(val type: String) : PregnancyOnboardingIntent
    data class RhSelected(val factor: String) : PregnancyOnboardingIntent
    data object Submit : PregnancyOnboardingIntent
}

/** One-shot: onboarding accepted, return to the tracker. */
sealed interface PregnancyOnboardingEffect {
    data object Completed : PregnancyOnboardingEffect
}

class PregnancyOnboardingViewModel(
    private val tracker: PregnancyTrackerUseCase,
    today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(
        PregnancyOnboardingUiState(lmpDate = today, ultrasoundDate = today),
    )
    val state: StateFlow<PregnancyOnboardingUiState> = _state.asStateFlow()

    private val _effects = Channel<PregnancyOnboardingEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    fun onIntent(intent: PregnancyOnboardingIntent) {
        when (intent) {
            is PregnancyOnboardingIntent.SourceSelected ->
                _state.update { it.copy(source = intent.source, error = PregnancyOnboardingError.NONE) }

            is PregnancyOnboardingIntent.LmpChanged -> _state.update { it.copy(lmpDate = intent.date) }
            is PregnancyOnboardingIntent.UltrasoundDateChanged -> _state.update { it.copy(ultrasoundDate = intent.date) }
            is PregnancyOnboardingIntent.UltrasoundWeeksChanged -> _state.update { it.copy(ultrasoundWeeks = intent.weeks) }
            is PregnancyOnboardingIntent.UltrasoundDaysChanged -> _state.update { it.copy(ultrasoundDays = intent.days) }
            is PregnancyOnboardingIntent.ManualWeeksChanged -> _state.update { it.copy(manualWeeks = intent.weeks) }
            is PregnancyOnboardingIntent.ManualDaysChanged -> _state.update { it.copy(manualDays = intent.days) }
            is PregnancyOnboardingIntent.MiscarriageToggled -> _state.update { it.copy(hasMiscarriageHistory = intent.value) }
            is PregnancyOnboardingIntent.HighRiskToggled -> _state.update { it.copy(hasHighRiskHistory = intent.value) }
            is PregnancyOnboardingIntent.ConditionToggled -> toggleCondition(intent.condition)
            is PregnancyOnboardingIntent.BloodTypeSelected ->
                _state.update { it.copy(bloodType = if (it.bloodType == intent.type) null else intent.type) }

            is PregnancyOnboardingIntent.RhSelected ->
                _state.update { it.copy(rhFactor = if (it.rhFactor == intent.factor) null else intent.factor) }

            PregnancyOnboardingIntent.Submit -> submit()
        }
    }

    /** «هیچ‌کدام» clears everything else; picking a real condition clears «هیچ‌کدام». */
    private fun toggleCondition(condition: String) {
        _state.update { current ->
            val selected = current.conditions
            val updated = when {
                condition == NONE_CONDITION -> if (NONE_CONDITION in selected) emptyList() else listOf(NONE_CONDITION)
                condition in selected -> selected - condition
                else -> selected - NONE_CONDITION + condition
            }
            current.copy(conditions = updated)
        }
    }

    private fun submit() {
        val snapshot = _state.value
        if (snapshot.submitting) return
        val source = snapshot.source
        if (source == null) {
            _state.update { it.copy(error = PregnancyOnboardingError.SELECT_SOURCE) }
            return
        }
        viewModelScope.launch {
            Breadcrumbs.add("pregnancy:onboarding_submit")
            _state.update { it.copy(submitting = true, error = PregnancyOnboardingError.NONE) }
            val answers = PregnancyOnboardingAnswers(
                ageSource = source,
                lmpDate = snapshot.lmpDate.takeIf { source == PregnancyAgeSource.LMP },
                ultrasoundDate = snapshot.ultrasoundDate.takeIf { source == PregnancyAgeSource.ULTRASOUND },
                ultrasoundWeeks = snapshot.ultrasoundWeeks.takeIf { source == PregnancyAgeSource.ULTRASOUND },
                ultrasoundDays = snapshot.ultrasoundDays.takeIf { source == PregnancyAgeSource.ULTRASOUND },
                manualWeeks = snapshot.manualWeeks.takeIf { source == PregnancyAgeSource.MANUAL },
                manualDays = snapshot.manualDays.takeIf { source == PregnancyAgeSource.MANUAL },
                hasMiscarriageHistory = snapshot.hasMiscarriageHistory,
                hasHighRiskHistory = snapshot.hasHighRiskHistory,
                preExistingConditions = snapshot.conditions,
                bloodType = snapshot.bloodType,
                rhFactor = snapshot.rhFactor,
            )
            when (tracker.completeOnboarding(answers)) {
                is AppResult.Failure ->
                    _state.update { it.copy(submitting = false, error = PregnancyOnboardingError.SUBMIT_FAILED) }

                is AppResult.Success -> _effects.send(PregnancyOnboardingEffect.Completed)
            }
        }
    }

    private companion object {
        const val NONE_CONDITION = "none"
    }
}
