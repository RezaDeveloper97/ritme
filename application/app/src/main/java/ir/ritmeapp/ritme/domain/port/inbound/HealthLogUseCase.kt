package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.GregorianDate

/**
 * Inbound port: read and save the daily health log the log screen edits.
 */
interface HealthLogUseCase {

    /** The saved log for [date], or null when that day has no entries. */
    suspend fun logFor(date: GregorianDate): AppResult<DailyHealthLog?>

    /** Upserts [log] for its date. */
    suspend fun save(log: DailyHealthLog): AppResult<Unit>
}
