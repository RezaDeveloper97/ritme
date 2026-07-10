package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.HomeDashboard

/**
 * Inbound port: load everything the Home screen renders (cycle calculation + derived predictions +
 * daily message). The ViewModel depends on this, never on the gateways directly.
 */
interface GetHomeDashboardUseCase {
    suspend operator fun invoke(): AppResult<HomeDashboard>
}
