package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.TrackingMode

/**
 * Outbound port for the account's tracking mode (`GET /messages/mode`) — the switch the
 * bottom navigation and Home feed reshape around.
 */
interface ModeGateway {
    suspend fun currentMode(): AppResult<TrackingMode>
}
