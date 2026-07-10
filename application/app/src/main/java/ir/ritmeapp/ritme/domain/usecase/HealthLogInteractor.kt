package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.port.inbound.HealthLogUseCase
import ir.ritmeapp.ritme.domain.port.outbound.HealthLogGateway

/** Default [HealthLogUseCase]; the seam where cross-field log rules would live. */
class HealthLogInteractor(
    private val healthLogGateway: HealthLogGateway,
) : HealthLogUseCase {

    override suspend fun logFor(date: GregorianDate): AppResult<DailyHealthLog?> =
        healthLogGateway.logFor(date)

    override suspend fun save(log: DailyHealthLog): AppResult<Unit> = healthLogGateway.save(log)
}
