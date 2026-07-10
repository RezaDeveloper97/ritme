package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.inbound.ExportDataUseCase
import ir.ritmeapp.ritme.domain.port.outbound.SessionGateway

/** Default [ExportDataUseCase]. */
class ExportDataInteractor(
    private val sessionGateway: SessionGateway,
) : ExportDataUseCase {

    override suspend fun invoke(): AppResult<String> = sessionGateway.exportData()
}
