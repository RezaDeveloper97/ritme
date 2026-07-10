package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CycleCalculation
import ir.ritmeapp.ritme.domain.model.CycleMonth
import ir.ritmeapp.ritme.domain.model.GregorianDate

/**
 * Outbound port for cycle calculations from the backend (`/cycle/…`). Small and
 * role-specific (§4 ISP); the adapter owns the JSON and the bearer token.
 */
interface CycleGateway {

    /** Today's freshly-computed calculation (the Home hero + cycle screen). */
    suspend fun today(): AppResult<CycleCalculation>

    /** The calculation for one specific [date] (a tapped calendar day). */
    suspend fun forDate(date: GregorianDate): AppResult<CycleCalculation>

    /** Every day of a Gregorian month + the month summary (the calendar screen). */
    suspend fun month(year: Int, month: Int): AppResult<CycleMonth>

    /** Kicks off an async server-side recalculation; progress is polled via [isRecalculating]. */
    suspend fun recalculate(): AppResult<Unit>

    /** Whether the async recalculation is still running (`GET /cycle/status`). */
    suspend fun isRecalculating(): AppResult<Boolean>
}
