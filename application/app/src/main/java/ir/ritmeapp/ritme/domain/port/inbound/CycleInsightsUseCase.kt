package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CycleCalculation
import ir.ritmeapp.ritme.domain.model.CycleMonth
import ir.ritmeapp.ritme.domain.model.GregorianDate

/**
 * Inbound port: the cycle analytics the calendar and cycle screens render — whole-month
 * phase data, single-day drill-down, and the manual recalculation flow.
 */
interface CycleInsightsUseCase {

    /** Every day of a Gregorian month plus its summary counts. */
    suspend fun month(year: Int, month: Int): AppResult<CycleMonth>

    /** The calculation for one tapped day. */
    suspend fun forDate(date: GregorianDate): AppResult<CycleCalculation>

    /** Starts an async server-side recalculation. */
    suspend fun recalculate(): AppResult<Unit>

    /** Polls whether that recalculation is still running. */
    suspend fun isRecalculating(): AppResult<Boolean>
}
