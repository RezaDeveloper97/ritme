package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.JalaliDate

/**
 * Inbound port: the Home «شروع پریود» action — records that a period started on [startDay]
 * so the backend re-anchors the cycle from that date.
 */
interface StartPeriodUseCase {
    suspend operator fun invoke(startDay: JalaliDate): AppResult<Unit>
}
