package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.TrackingMode

/**
 * Inbound port: which tracker (cycle / pregnancy) the account is in — drives the
 * mode-aware bottom navigation and Home routing.
 */
interface GetTrackingModeUseCase {
    suspend operator fun invoke(): AppResult<TrackingMode>
}
