package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.port.inbound.StartPeriodUseCase
import ir.ritmeapp.ritme.domain.port.outbound.PeriodLogGateway

/**
 * Default [StartPeriodUseCase]: records the start through the dedicated period-log
 * endpoint (idempotent per day, creates a history record the calendar can edit) —
 * not the old partial-profile-update hack.
 */
class StartPeriodInteractor(
    private val periodLogGateway: PeriodLogGateway,
) : StartPeriodUseCase {

    override suspend fun invoke(startDay: JalaliDate): AppResult<Unit> =
        periodLogGateway.start(startDay.toGregorian())
}
