package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.NotificationsPage
import ir.ritmeapp.ritme.domain.port.inbound.NotificationsUseCase
import ir.ritmeapp.ritme.domain.port.outbound.NotificationGateway

/** Default [NotificationsUseCase]. */
class NotificationsInteractor(
    private val notificationGateway: NotificationGateway,
) : NotificationsUseCase {

    override suspend fun page(perPage: Int): AppResult<NotificationsPage> =
        notificationGateway.notifications(perPage)

    override suspend fun markRead(id: Long): AppResult<Unit> = notificationGateway.markRead(id)

    override suspend fun markAllRead(): AppResult<Unit> = notificationGateway.markAllRead()
}
