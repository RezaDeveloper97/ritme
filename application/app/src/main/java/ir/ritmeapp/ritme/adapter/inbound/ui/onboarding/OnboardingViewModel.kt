package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
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
 * pregnancy) from the chosen intention, and submits once. Depends only on the inbound port (§4);
 * no networking/parsing here. Navigation to home is a one-shot [OnboardingEffect] off a channel so
 * it never replays on recomposition. Kept lean by pushing the profile call into the use case.
 */
class OnboardingViewModel(
    private val saveProfile: SaveProfileUseCase,
    today: JalaliDate,
) : ViewModel() {

    private val _state = MutableStateFlow(recompute(0, defaultAnswers(today)))
    val state: StateFlow<OnboardingUiState> = _state.asStateFlow()

    private val _effects = Channel<OnboardingEffect>(Channel.BUFFERED)
    val effects = _effects.receiveAsFlow()

    fun onIntent(intent: OnboardingIntent) {
        when (intent) {
            is OnboardingIntent.NameChanged -> editAnswers { it.copy(name = intent.name) }
            is OnboardingIntent.BirthdayChanged -> editAnswers { it.copy(birthday = intent.date) }
            is OnboardingIntent.WeightChanged -> editAnswers { it.copy(weightKg = intent.kg) }
            is OnboardingIntent.HeightChanged -> editAnswers { it.copy(heightCm = intent.cm) }
            is OnboardingIntent.IntentionSelected -> editAnswers { it.copy(intention = intent.intention) }
            is OnboardingIntent.PeriodChanged -> editAnswers { it.copy(periodDuration = intent.days) }
            is OnboardingIntent.CycleChanged -> editAnswers { it.copy(cycleDuration = intent.days) }
            is OnboardingIntent.LastPeriodChanged -> editAnswers { it.copy(lastPeriod = intent.date) }
            is OnboardingIntent.ConditionToggled -> editAnswers { it.copy(conditions = it.conditions.toggled(intent.condition)) }
            OnboardingIntent.Back -> _state.update { recompute(it.stepNumber - 2, it.answers, error = it.errorMessage) }
            OnboardingIntent.Next -> onNext()
            OnboardingIntent.DismissError -> _state.update { it.copy(errorMessage = null) }
        }
    }

    private fun onNext() {
        val current = _state.value
        if (current.submitting) return
        if (current.isLastStep) submit() else _state.update { recompute(it.stepNumber, it.answers) }
    }

    private fun submit() {
        Breadcrumbs.add("ui:onboarding:submit")
        _state.update { it.copy(submitting = true, errorMessage = null) }
        viewModelScope.launch {
            when (val result = saveProfile(_state.value.answers)) {
                is AppResult.Success -> _effects.send(OnboardingEffect.NavigateHome)
                is AppResult.Failure ->
                    _state.update { it.copy(submitting = false, errorMessage = serverMessageOf(result.error)) }
            }
        }
    }

    /** Applies an edit, keeping the current step but recomputing the branch/progress/proceed gate. */
    private inline fun editAnswers(transform: (OnboardingAnswers) -> OnboardingAnswers) {
        _state.update { recompute(it.stepNumber - 1, transform(it.answers), error = it.errorMessage) }
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
            lastPeriod = today,
        )

        /** Head is shared; a pregnant intention drops the cycle questions (§ onboarding branch). */
        fun stepsFor(intention: PregnancyIntention?): List<OnboardingStep> {
            val head = listOf(
                OnboardingStep.NAME, OnboardingStep.BIRTHDAY, OnboardingStep.WEIGHT,
                OnboardingStep.HEIGHT, OnboardingStep.INTENTION,
            )
            val tail = if (intention == PregnancyIntention.PREGNANT) {
                listOf(OnboardingStep.CONDITIONS)
            } else {
                listOf(OnboardingStep.PERIOD, OnboardingStep.CYCLE, OnboardingStep.LAST_PERIOD, OnboardingStep.CONDITIONS)
            }
            return head + tail
        }

        fun recompute(
            index: Int,
            answers: OnboardingAnswers,
            submitting: Boolean = false,
            error: String? = null,
        ): OnboardingUiState {
            val steps = stepsFor(answers.intention)
            val idx = index.coerceIn(0, steps.lastIndex)
            val step = steps[idx]
            return OnboardingUiState(
                step = step,
                stepNumber = idx + 1,
                totalSteps = steps.size,
                answers = answers,
                canProceed = canProceed(step, answers),
                submitting = submitting,
                errorMessage = error,
            )
        }

        fun canProceed(step: OnboardingStep, answers: OnboardingAnswers): Boolean = when (step) {
            OnboardingStep.NAME -> !answers.name.isNullOrBlank()
            OnboardingStep.INTENTION -> answers.intention != null
            else -> true
        }

        fun List<ChronicCondition>.toggled(condition: ChronicCondition): List<ChronicCondition> =
            if (contains(condition)) this - condition else this + condition

        /** Prefer the backend's Persian message; connectivity/other errors fall back to a generic string. */
        fun serverMessageOf(error: AppError): String? = (error as? AppError.Http)?.message
    }
}
