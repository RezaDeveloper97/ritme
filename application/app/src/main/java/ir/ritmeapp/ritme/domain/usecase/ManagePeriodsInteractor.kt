package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.LoggedPeriod
import ir.ritmeapp.ritme.domain.model.PeriodDayAction
import ir.ritmeapp.ritme.domain.port.inbound.ManagePeriodsUseCase
import ir.ritmeapp.ritme.domain.port.outbound.PeriodLogGateway

/**
 * Default [ManagePeriodsUseCase]: maps each [PeriodDayAction] onto the matching
 * `/cycle/period` gateway call; the backend re-anchors and recalculates from it.
 */
class ManagePeriodsInteractor(
    private val periodLogGateway: PeriodLogGateway,
) : ManagePeriodsUseCase {

    override suspend fun history(): AppResult<List<LoggedPeriod>> = periodLogGateway.history()

    override suspend fun apply(action: PeriodDayAction, day: GregorianDate): AppResult<Unit> = when (action) {
        is PeriodDayAction.StartNew -> periodLogGateway.start(day)
        is PeriodDayAction.Extend -> periodLogGateway.update(action.period.id, action.newStart, action.newEnd)
        is PeriodDayAction.TrimStart -> periodLogGateway.update(action.period.id, action.newStart, action.period.end)
        is PeriodDayAction.TrimEnd -> periodLogGateway.update(action.period.id, action.period.start, action.newEnd)
        is PeriodDayAction.DeletePeriod -> periodLogGateway.delete(action.period.id)
        is PeriodDayAction.None -> AppResult.Success(Unit)
    }
}
