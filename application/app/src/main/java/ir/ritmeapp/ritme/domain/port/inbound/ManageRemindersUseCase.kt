package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft

/**
 * Inbound port: everything the reminders screen can do. One cohesive port rather than four
 * single-method ones because the screen always needs the full set together and the
 * operations share one resource.
 */
interface ManageRemindersUseCase {

    /** All reminders, newest first. */
    suspend fun list(): AppResult<List<Reminder>>

    /** Creates from [draft]; returns the stored record. */
    suspend fun create(draft: ReminderDraft): AppResult<Reminder>

    /** Overwrites reminder [id] with [draft]. */
    suspend fun update(id: Long, draft: ReminderDraft): AppResult<Reminder>

    /** Deletes reminder [id]. */
    suspend fun delete(id: Long): AppResult<Unit>
}
