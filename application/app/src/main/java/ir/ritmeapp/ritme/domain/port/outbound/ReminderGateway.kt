package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft

/**
 * Outbound port for user reminders (`/reminders` CRUD). Scheduling/delivery live server-side;
 * the app only manages the records.
 */
interface ReminderGateway {

    /** All of the user's reminders, newest first. */
    suspend fun reminders(): AppResult<List<Reminder>>

    /** Creates a reminder from [draft]; returns the server record with its id. */
    suspend fun create(draft: ReminderDraft): AppResult<Reminder>

    /** Overwrites reminder [id] with [draft]; returns the updated record. */
    suspend fun update(id: Long, draft: ReminderDraft): AppResult<Reminder>

    /** Deletes reminder [id]. */
    suspend fun delete(id: Long): AppResult<Unit>
}
