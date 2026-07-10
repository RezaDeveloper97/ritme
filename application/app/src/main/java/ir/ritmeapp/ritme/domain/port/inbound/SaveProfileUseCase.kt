package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers

/**
 * Inbound port: finish signup by saving the collected [OnboardingAnswers]. The onboarding
 * ViewModel depends on this, never on the concrete gateway.
 */
interface SaveProfileUseCase {
    suspend operator fun invoke(answers: OnboardingAnswers): AppResult<Unit>
}
