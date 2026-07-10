package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.DailyMessage

/**
 * Outbound port for the personalized daily message (`GET /messages/daily`). The backend returns 400
 * when the profile is incomplete; the adapter maps that to a `null` success so Home can still render
 * without the message rather than treating it as an error.
 */
interface MessageGateway {
    suspend fun daily(): AppResult<DailyMessage?>
}
