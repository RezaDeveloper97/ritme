package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.NotificationsPage

/**
 * Inbound port: everything the notifications screen does.
 */
interface NotificationsUseCase {

    /** The newest notifications plus the unread count. */
    suspend fun page(perPage: Int): AppResult<NotificationsPage>

    /** Marks one notification read. */
    suspend fun markRead(id: Long): AppResult<Unit>

    /** Marks all notifications read. */
    suspend fun markAllRead(): AppResult<Unit>
}
