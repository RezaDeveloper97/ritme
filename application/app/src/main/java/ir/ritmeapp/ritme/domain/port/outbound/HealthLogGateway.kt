package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.GregorianDate

/**
 * Outbound port for the daily health log (`/health-logs`). Saving upserts by date server-side,
 * so the caller never distinguishes create from update.
 */
interface HealthLogGateway {

    /** The log recorded on [date], or null when nothing was logged that day (backend 404). */
    suspend fun logFor(date: GregorianDate): AppResult<DailyHealthLog?>

    /** Upserts [log] for its date; triggers a cycle recalculation server-side. */
    suspend fun save(log: DailyHealthLog): AppResult<Unit>
}
