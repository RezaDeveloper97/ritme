package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.PregnancyIntention

/**
 * The signup wizard's steps, in base order; the branch after [INTENTION] is decided at runtime
 * ([PREGNANCY_BASIS] for a pregnant intention, the cycle questions otherwise). [SETTING_UP] is not
 * a numbered step — it is the post-submit progress-ring screen shown after the last real step.
 */
enum class OnboardingStep {
    NAME, BIRTHDAY, WEIGHT, HEIGHT, INTENTION,
    PREGNANCY_BASIS, PERIOD, CYCLE, LAST_PERIOD, CONDITIONS, SETTING_UP,
}

/** Which unit the weight ruler reads in; the stored answer is always kilograms. */
enum class WeightUnit { KG, LB }

/** Which unit the height ruler reads in; the stored answer is always centimetres. */
enum class HeightUnit { CM, FT }

/** The single inline validation the pregnant-branch dating step can raise (mirrors the web). */
enum class OnboardingValidation { SELECT_SOURCE, FILL_REQUIRED }

/**
 * The dating basis collected only on the pregnant branch (mirrors the web `PregnancyBasis`). Kept in
 * the UI state because the onboarding [OnboardingAnswers] domain model only carries cycle/profile
 * data; turning this into the pregnancy activation call needs the pregnancy use cases (deferred).
 */
@Immutable
data class PregnancyBasisDraft(
    val source: PregnancyAgeSource? = null,
    val lmp: JalaliDate? = null,
    val ultrasoundDate: JalaliDate? = null,
    val ultrasoundWeeks: Int? = null,
    val ultrasoundDays: Int = 0,
    val manualWeeks: Int? = null,
    val manualDays: Int = 0,
)

/**
 * Immutable snapshot the onboarding screen renders from (CLAUDE.md §4 — one state flow, no loose
 * flags). [answers] accumulates across steps; [stepNumber]/[totalSteps] drive the "N / M" header
 * (total shrinks once a pregnant intention swaps the cycle questions for the dating step);
 * [canProceed] gates the Continue button. On [SETTING_UP], [saveSettled] + [ringDone] together gate
 * the one-shot navigation home.
 */
@Immutable
data class OnboardingUiState(
    val step: OnboardingStep = OnboardingStep.NAME,
    val stepNumber: Int = 1,
    val totalSteps: Int = 9,
    val answers: OnboardingAnswers = OnboardingAnswers(),
    val weightUnit: WeightUnit = WeightUnit.KG,
    val heightUnit: HeightUnit = HeightUnit.CM,
    val pregnancyBasis: PregnancyBasisDraft = PregnancyBasisDraft(),
    val canProceed: Boolean = false,
    val submitting: Boolean = false,
    val saveSettled: Boolean = false,
    val ringDone: Boolean = false,
    val validation: OnboardingValidation? = null,
    val errorMessage: String? = null,
) {
    val canGoBack: Boolean get() = step != OnboardingStep.SETTING_UP && stepNumber > 1 && !submitting
    val isSettingUp: Boolean get() = step == OnboardingStep.SETTING_UP
    val isLastStep: Boolean get() = !isSettingUp && stepNumber == totalSteps
}

/** Everything the user can do on the wizard, funneled through one `onIntent` (CLAUDE.md §4). */
sealed interface OnboardingIntent {
    data class NameChanged(val name: String) : OnboardingIntent
    data class BirthdayChanged(val date: JalaliDate) : OnboardingIntent
    data class WeightChanged(val kg: Int) : OnboardingIntent
    data class WeightUnitChanged(val unit: WeightUnit) : OnboardingIntent
    data class HeightChanged(val cm: Int) : OnboardingIntent
    data class HeightUnitChanged(val unit: HeightUnit) : OnboardingIntent
    data class IntentionSelected(val intention: PregnancyIntention) : OnboardingIntent
    data class PeriodChanged(val days: Int) : OnboardingIntent
    data class CycleChanged(val days: Int) : OnboardingIntent
    data class LastPeriodChanged(val date: JalaliDate) : OnboardingIntent
    data class ConditionToggled(val condition: ChronicCondition) : OnboardingIntent
    data class BasisSourceSelected(val source: PregnancyAgeSource) : OnboardingIntent
    data class BasisLmpChanged(val date: JalaliDate) : OnboardingIntent
    data class BasisUltrasoundDateChanged(val date: JalaliDate) : OnboardingIntent
    data class BasisUltrasoundWeeksChanged(val weeks: Int?) : OnboardingIntent
    data class BasisUltrasoundDaysChanged(val days: Int) : OnboardingIntent
    data class BasisManualWeeksChanged(val weeks: Int?) : OnboardingIntent
    data class BasisManualDaysChanged(val days: Int) : OnboardingIntent
    data object SettingUpRingDone : OnboardingIntent
    data object Back : OnboardingIntent
    data object Next : OnboardingIntent
    data object DismissError : OnboardingIntent
}

/** One-shot side effects, delivered off a channel so they don't replay on recomposition (§4). */
sealed interface OnboardingEffect {
    data object NavigateHome : OnboardingEffect
}
