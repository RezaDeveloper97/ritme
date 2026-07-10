package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.port.inbound.StartPeriodUseCase
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway

/**
 * Default [StartPeriodUseCase]: a period start is recorded as a partial profile update of
 * `last_period_start` only — the backend re-anchors and recalculates the cycle from it.
 */
class StartPeriodInteractor(
    private val profileGateway: ProfileGateway,
) : StartPeriodUseCase {

    override suspend fun invoke(startDay: JalaliDate): AppResult<Unit> =
        profileGateway.saveProfile(OnboardingAnswers(lastPeriod = startDay))
}
