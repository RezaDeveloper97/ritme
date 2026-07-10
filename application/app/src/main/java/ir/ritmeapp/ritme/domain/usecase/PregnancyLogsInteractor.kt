package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.PregnancySymptomLog
import ir.ritmeapp.ritme.domain.model.PregnancyWeeklyLog
import ir.ritmeapp.ritme.domain.port.inbound.PregnancyLogsUseCase
import ir.ritmeapp.ritme.domain.port.outbound.PregnancyGateway

/** Default [PregnancyLogsUseCase]. */
class PregnancyLogsInteractor(
    private val pregnancyGateway: PregnancyGateway,
) : PregnancyLogsUseCase {

    override suspend fun symptomLog(date: GregorianDate): AppResult<PregnancySymptomLog?> =
        pregnancyGateway.symptomLog(date)

    override suspend fun saveSymptomLog(log: PregnancySymptomLog): AppResult<Int> =
        pregnancyGateway.saveSymptomLog(log)

    override suspend fun weeklyLog(week: Int): AppResult<PregnancyWeeklyLog?> =
        pregnancyGateway.weeklyLog(week)

    override suspend fun saveWeeklyLog(log: PregnancyWeeklyLog): AppResult<Unit> =
        pregnancyGateway.saveWeeklyLog(log)

    override suspend fun saveFetalMovement(log: FetalMovementLog): AppResult<Unit> =
        pregnancyGateway.saveFetalMovement(log)
}
