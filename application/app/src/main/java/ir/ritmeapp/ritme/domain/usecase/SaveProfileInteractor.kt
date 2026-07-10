package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.port.inbound.SaveProfileUseCase
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway

/**
 * Default [SaveProfileUseCase]. Thin today, but the deliberate seam where cross-field onboarding
 * rules (e.g. don't submit cycle data in pregnancy mode) can live without touching UI or network.
 */
class SaveProfileInteractor(
    private val profileGateway: ProfileGateway,
) : SaveProfileUseCase {

    override suspend fun invoke(answers: OnboardingAnswers): AppResult<Unit> =
        profileGateway.saveProfile(answers)
}
