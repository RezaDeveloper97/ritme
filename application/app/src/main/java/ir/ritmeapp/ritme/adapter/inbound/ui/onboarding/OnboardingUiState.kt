package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyIntention

/** The signup wizard's steps, in base order; the branch after [INTENTION] is decided at runtime. */
enum class OnboardingStep { NAME, BIRTHDAY, WEIGHT, HEIGHT, INTENTION, PERIOD, CYCLE, LAST_PERIOD, CONDITIONS }

/**
 * Immutable snapshot the onboarding screen renders from (CLAUDE.md §4 — one state flow, no loose
 * flags). [answers] accumulates across steps; [stepNumber]/[totalSteps] drive the progress header
 * (total shrinks once a pregnant intention removes the cycle questions); [canProceed] gates the
 * Continue button for the current step.
 */
@Immutable
data class OnboardingUiState(
    val step: OnboardingStep = OnboardingStep.NAME,
    val stepNumber: Int = 1,
    val totalSteps: Int = 9,
    val answers: OnboardingAnswers = OnboardingAnswers(),
    val canProceed: Boolean = false,
    val submitting: Boolean = false,
    val errorMessage: String? = null,
) {
    val canGoBack: Boolean get() = stepNumber > 1 && !submitting
    val isLastStep: Boolean get() = stepNumber == totalSteps
}

/** Everything the user can do on the wizard, funneled through one `onIntent` (CLAUDE.md §4). */
sealed interface OnboardingIntent {
    data class NameChanged(val name: String) : OnboardingIntent
    data class BirthdayChanged(val date: JalaliDate) : OnboardingIntent
    data class WeightChanged(val kg: Int) : OnboardingIntent
    data class HeightChanged(val cm: Int) : OnboardingIntent
    data class IntentionSelected(val intention: PregnancyIntention) : OnboardingIntent
    data class PeriodChanged(val days: Int) : OnboardingIntent
    data class CycleChanged(val days: Int) : OnboardingIntent
    data class LastPeriodChanged(val date: JalaliDate) : OnboardingIntent
    data class ConditionToggled(val condition: ChronicCondition) : OnboardingIntent
    data object Back : OnboardingIntent
    data object Next : OnboardingIntent
    data object DismissError : OnboardingIntent
}

/** One-shot side effects, delivered off a channel so they don't replay on recomposition (§4). */
sealed interface OnboardingEffect {
    data object NavigateHome : OnboardingEffect
}
