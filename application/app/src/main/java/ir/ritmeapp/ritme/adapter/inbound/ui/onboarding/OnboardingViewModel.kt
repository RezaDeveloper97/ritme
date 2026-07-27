package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.PregnancyIntention
import ir.ritmeapp.ritme.domain.port.inbound.SaveProfileUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.receiveAsFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the signup wizard: accumulates answers across steps, computes the branch (cycle vs
 * pregnancy) from the chosen intention, and — on the pregnant branch — collects the dating basis.
 * The last step hands off to the [OnboardingStep.SETTING_UP] progress-ring screen, which saves the
 * profile once (best-effort, like the web) and navigates home only after both the ring animation
 * and the save have settled. Depends only on the inbound port (§4); no networking/parsing here.
 */
class OnboardingViewModel(
    private val saveProfile: SaveProfileUseCase,
    today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(OnboardingUiState(answers = defaultAnswers(today)).reindex(0))
    val state: StateFlow<OnboardingUiState> = _state.asStateFlow()

    private val _effects = Channel<OnboardingEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    /** Guards the one-shot navigation so a save/ring race can never fire NavigateHome twice. */
    private var finished = false

    fun onIntent(intent: OnboardingIntent) {
        when (intent) {
            is OnboardingIntent.NameChanged -> editAnswers { it.copy(name = intent.name) }
            is OnboardingIntent.BirthdayChanged -> editAnswers { it.copy(birthday = intent.date) }
            is OnboardingIntent.WeightChanged -> editAnswers { it.copy(weightKg = intent.kg) }
            is OnboardingIntent.WeightUnitChanged -> _state.update { it.copy(weightUnit = intent.unit) }
            is OnboardingIntent.HeightChanged -> editAnswers { it.copy(heightCm = intent.cm) }
            is OnboardingIntent.HeightUnitChanged -> _state.update { it.copy(heightUnit = intent.unit) }
            is OnboardingIntent.IntentionSelected -> editAnswers { it.copy(intention = intent.intention) }
            is OnboardingIntent.PeriodChanged -> editAnswers { it.copy(periodDuration = intent.days) }
            is OnboardingIntent.CycleChanged -> editAnswers { it.copy(cycleDuration = intent.days) }
            is OnboardingIntent.LastPeriodChanged -> editAnswers { it.copy(lastPeriod = intent.date) }
            is OnboardingIntent.ConditionToggled -> editAnswers { it.copy(conditions = it.conditions.toggled(intent.condition)) }
            is OnboardingIntent.BasisSourceSelected -> editBasis { it.copy(source = intent.source) }
            is OnboardingIntent.BasisLmpChanged -> editBasis { it.copy(lmp = intent.date) }
            is OnboardingIntent.BasisUltrasoundDateChanged -> editBasis { it.copy(ultrasoundDate = intent.date) }
            is OnboardingIntent.BasisUltrasoundWeeksChanged -> editBasis { it.copy(ultrasoundWeeks = intent.weeks) }
            is OnboardingIntent.BasisUltrasoundDaysChanged -> editBasis { it.copy(ultrasoundDays = intent.days) }
            is OnboardingIntent.BasisManualWeeksChanged -> editBasis { it.copy(manualWeeks = intent.weeks) }
            is OnboardingIntent.BasisManualDaysChanged -> editBasis { it.copy(manualDays = intent.days) }
            OnboardingIntent.SettingUpRingDone -> onRingDone()
            OnboardingIntent.Back -> _state.update { if (it.canGoBack) it.reindex(it.stepNumber - 2) else it }
            OnboardingIntent.Next -> onNext()
            OnboardingIntent.DismissError -> _state.update { it.copy(errorMessage = null, validation = null) }
        }
    }

    private fun onNext() {
        val current = _state.value
        if (current.submitting) return
        if (current.step == OnboardingStep.PREGNANCY_BASIS) {
            val invalid = validateBasis(current.pregnancyBasis)
            if (invalid != null) {
                _state.update { it.copy(validation = invalid) }
                return
            }
        }
        if (current.isLastStep) startSettingUp() else _state.update { it.reindex(it.stepNumber) }
    }

    /** Save once (best-effort — proceed regardless, mirroring the web) and show the progress ring. */
    private fun startSettingUp() {
        Breadcrumbs.add("ui:onboarding:submit")
        _state.update {
            it.copy(
                step = OnboardingStep.SETTING_UP,
                submitting = true,
                saveSettled = false,
                ringDone = false,
                validation = null,
                errorMessage = null,
            )
        }
        viewModelScope.launch {
            saveProfile(_state.value.answers)
            _state.update { it.copy(saveSettled = true) }
            maybeFinish()
        }
    }

    private fun onRingDone() {
        _state.update { it.copy(ringDone = true) }
        maybeFinish()
    }

    /** Leave only once the ring has finished AND the save has settled, so the profile lands first. */
    private fun maybeFinish() {
        val s = _state.value
        if (!finished && s.saveSettled && s.ringDone) {
            finished = true
            viewModelScope.launch { _effects.send(OnboardingEffect.NavigateHome) }
        }
    }

    /** Applies an edit, keeping the current step but recomputing the branch/progress/proceed gate. */
    private inline fun editAnswers(transform: (OnboardingAnswers) -> OnboardingAnswers) {
        _state.update { it.reindex(it.stepNumber - 1, transform(it.answers)) }
    }

    private inline fun editBasis(transform: (PregnancyBasisDraft) -> PregnancyBasisDraft) {
        _state.update { it.copy(pregnancyBasis = transform(it.pregnancyBasis), validation = null) }
    }

    /**
     * Recomputes step/position/proceed for [newIndex] within the current branch, clearing any inline
     * validation. All other fields (units, basis, submitting) are preserved via `copy`.
     */
    private fun OnboardingUiState.reindex(
        newIndex: Int,
        newAnswers: OnboardingAnswers = answers,
    ): OnboardingUiState {
        val steps = stepsFor(newAnswers.intention)
        val idx = newIndex.coerceIn(0, steps.lastIndex)
        val step = steps[idx]
        return copy(
            step = step,
            stepNumber = idx + 1,
            totalSteps = steps.size,
            answers = newAnswers,
            canProceed = canProceed(step, newAnswers),
            validation = null,
        )
    }

    private companion object {
        const val DEFAULT_AGE = 25
        const val DEFAULT_WEIGHT_KG = 60
        const val DEFAULT_HEIGHT_CM = 165
        const val DEFAULT_PERIOD_DAYS = 5
        const val DEFAULT_CYCLE_DAYS = 28

        fun defaultAnswers(today: JalaliDate) = OnboardingAnswers(
            birthday = JalaliDate(today.year - DEFAULT_AGE, 1, 1),
            weightKg = DEFAULT_WEIGHT_KG,
            heightCm = DEFAULT_HEIGHT_CM,
            periodDuration = DEFAULT_PERIOD_DAYS,
            cycleDuration = DEFAULT_CYCLE_DAYS,
            // Left unset so the last-period calendar opens with no day highlighted, like the web
            // (`CycleLenPage` starts `lastPeriod = null`); it fills in only once the user taps a day.
        )

        /**
         * Head is shared; a pregnant intention swaps the cycle questions for the dating-basis step
         * (§ onboarding branch). [OnboardingStep.SETTING_UP] is intentionally excluded — it is the
         * post-submit screen, not a numbered step.
         */
        fun stepsFor(intention: PregnancyIntention?): List<OnboardingStep> {
            val head = listOf(
                OnboardingStep.NAME, OnboardingStep.BIRTHDAY, OnboardingStep.WEIGHT,
                OnboardingStep.HEIGHT, OnboardingStep.INTENTION,
            )
            val tail = if (intention == PregnancyIntention.PREGNANT) {
                listOf(OnboardingStep.PREGNANCY_BASIS, OnboardingStep.CONDITIONS)
            } else {
                listOf(OnboardingStep.PERIOD, OnboardingStep.CYCLE, OnboardingStep.LAST_PERIOD, OnboardingStep.CONDITIONS)
            }
            return head + tail
        }

        fun canProceed(step: OnboardingStep, answers: OnboardingAnswers): Boolean = when (step) {
            OnboardingStep.NAME -> !answers.name.isNullOrBlank()
            OnboardingStep.INTENTION -> answers.intention != null
            else -> true
        }

        /** Mirrors the web dating-step guard: a source is required, and its week input when relevant. */
        fun validateBasis(basis: PregnancyBasisDraft): OnboardingValidation? = when {
            basis.source == null -> OnboardingValidation.SELECT_SOURCE
            basis.source == PregnancyAgeSource.ULTRASOUND && basis.ultrasoundWeeks == null -> OnboardingValidation.FILL_REQUIRED
            basis.source == PregnancyAgeSource.MANUAL && basis.manualWeeks == null -> OnboardingValidation.FILL_REQUIRED
            else -> null
        }

        fun List<ChronicCondition>.toggled(condition: ChronicCondition): List<ChronicCondition> =
            if (contains(condition)) this - condition else this + condition
    }
}
