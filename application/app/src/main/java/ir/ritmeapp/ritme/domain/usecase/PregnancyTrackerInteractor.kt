package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyOnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyProfile
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyTrackerUseCase
import ir.ritmeapp.ritme.domain.port.outbound.PregnancyGateway

/** Default [PregnancyTrackerUseCase]; thin delegation to the pregnancy gateway. */
class PregnancyTrackerInteractor(
    private val pregnancyGateway: PregnancyGateway,
) : PregnancyTrackerUseCase {

    override suspend fun activate(): AppResult<Unit> = pregnancyGateway.activate()

    override suspend fun deactivate(): AppResult<Unit> = pregnancyGateway.deactivate()

    override suspend fun completeOnboarding(answers: PregnancyOnboardingAnswers): AppResult<Unit> =
        pregnancyGateway.completeOnboarding(answers)

    override suspend fun status(): AppResult<PregnancyStatus> = pregnancyGateway.status()

    override suspend fun profile(): AppResult<PregnancyProfile?> = pregnancyGateway.profile()

    override suspend fun contentForWeek(week: Int): AppResult<PregnancyWeekContent> =
        pregnancyGateway.contentForWeek(week)

    override suspend fun alerts(): AppResult<List<PregnancyAlert>> = pregnancyGateway.alerts()

    override suspend fun markAlertRead(id: Long): AppResult<Unit> = pregnancyGateway.markAlertRead(id)

    override suspend fun dismissAlert(id: Long): AppResult<Unit> = pregnancyGateway.dismissAlert(id)

    override suspend fun markAllAlertsRead(): AppResult<Unit> = pregnancyGateway.markAllAlertsRead()
}
