package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.NotificationsPage

/**
 * Outbound port for in-app notifications (`/home/notifications`). Read-marking is
 * fire-and-confirm; the caller refreshes the list afterwards.
 */
interface NotificationGateway {

    /** The newest notifications plus the unread count. */
    suspend fun notifications(perPage: Int): AppResult<NotificationsPage>

    /** Marks one notification read. */
    suspend fun markRead(id: Long): AppResult<Unit>

    /** Marks every notification read. */
    suspend fun markAllRead(): AppResult<Unit>
}
