package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft
import ir.ritmeapp.ritme.domain.port.inbound.ManageRemindersUseCase
import ir.ritmeapp.ritme.domain.port.outbound.ReminderGateway

/** Default [ManageRemindersUseCase]. */
class ManageRemindersInteractor(
    private val reminderGateway: ReminderGateway,
) : ManageRemindersUseCase {

    override suspend fun list(): AppResult<List<Reminder>> = reminderGateway.reminders()

    override suspend fun create(draft: ReminderDraft): AppResult<Reminder> = reminderGateway.create(draft)

    override suspend fun update(id: Long, draft: ReminderDraft): AppResult<Reminder> =
        reminderGateway.update(id, draft)

    override suspend fun delete(id: Long): AppResult<Unit> = reminderGateway.delete(id)
}
