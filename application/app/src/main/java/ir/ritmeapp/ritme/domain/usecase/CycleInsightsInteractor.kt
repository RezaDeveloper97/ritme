package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CycleCalculation
import ir.ritmeapp.ritme.domain.model.CycleMonth
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.port.inbound.CycleInsightsUseCase
import ir.ritmeapp.ritme.domain.port.outbound.CycleGateway

/** Default [CycleInsightsUseCase]; today a thin delegation to the gateway. */
class CycleInsightsInteractor(
    private val cycleGateway: CycleGateway,
) : CycleInsightsUseCase {

    override suspend fun month(year: Int, month: Int): AppResult<CycleMonth> =
        cycleGateway.month(year, month)

    override suspend fun forDate(date: GregorianDate): AppResult<CycleCalculation> =
        cycleGateway.forDate(date)

    override suspend fun recalculate(): AppResult<Unit> = cycleGateway.recalculate()

    override suspend fun isRecalculating(): AppResult<Boolean> = cycleGateway.isRecalculating()
}
