package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.LoggedPeriod

/**
 * Outbound port for the logged-period endpoints (`/cycle/period/…`). Every mutation
 * re-anchors the cycle server-side and regenerates all downstream predictions.
 */
interface PeriodLogGateway {

    /** Every period the user logged, newest first (`GET /cycle/period/history`). */
    suspend fun history(): AppResult<List<LoggedPeriod>>

    /** Records that a period started on [date] (`POST /cycle/period/start`). */
    suspend fun start(date: GregorianDate): AppResult<Unit>

    /** Moves a logged period to [start]..[end]; a null [end] keeps/leaves it open. */
    suspend fun update(id: Long, start: GregorianDate, end: GregorianDate?): AppResult<Unit>

    /** Deletes a logged period entirely (`DELETE /cycle/period/{id}`). */
    suspend fun delete(id: Long): AppResult<Unit>
}
