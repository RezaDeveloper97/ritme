package ir.ritmeapp.ritme.domain.model

/**
 * One in-app notification (`GET /home/notifications`). [createdAt] stays the raw ISO string;
 * the UI derives the relative «امروز/دیروز/n روز پیش» label at the edge.
 */
data class AppNotification(
    val id: Long,
    val type: String,
    val title: String,
    val body: String,
    val isRead: Boolean,
    val createdAt: String,
)

/** The notifications screen's full payload: the list plus the unread badge count. */
data class NotificationsPage(
    val unreadCount: Int,
    val items: List<AppNotification>,
)
