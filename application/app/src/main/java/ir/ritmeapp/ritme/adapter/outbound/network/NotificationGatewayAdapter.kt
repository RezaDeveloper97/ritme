package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpRequest
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppNotification
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.model.NotificationsPage
import ir.ritmeapp.ritme.domain.port.outbound.NotificationGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/** Network [NotificationGateway] over `/home/notifications`. */
class NotificationGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : NotificationGateway {

    override suspend fun notifications(perPage: Int): AppResult<NotificationsPage> {
        Breadcrumbs.add("api:notifications:start")
        val token = tokenStore.load()?.accessToken
            ?: return AppResult.Failure(AppError.Http(HTTP_UNAUTHORIZED, "No active session"))
        val result = httpClient.execute(
            HttpRequest(
                method = HttpMethod.GET,
                path = ApiConfig.NOTIFICATIONS_PATH,
                headers = mapOf("Authorization" to "Bearer $token"),
                query = mapOf("per_page" to perPage.toString()),
            ),
        )
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:notifications:http_${result.value.statusCode}")
                when (val envelope = result.value.readEnvelope("Notifications request rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> parsePage(envelope.value.dataObject())
                }
            }
        }
    }

    override suspend fun markRead(id: Long): AppResult<Unit> = confirm(ApiConfig.notificationReadPath(id))

    override suspend fun markAllRead(): AppResult<Unit> = confirm(ApiConfig.NOTIFICATIONS_READ_ALL_PATH)

    private suspend fun confirm(path: String): AppResult<Unit> {
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.POST, path)) {
            is AppResult.Failure -> result
            is AppResult.Success -> result.value.readEnvelope("Notification update rejected").map { }
        }
    }

    private fun parsePage(data: JSONObject): AppResult<NotificationsPage> = try {
        val items = data.optJSONArray("items")
        AppResult.Success(
            NotificationsPage(
                unreadCount = data.optInt("unread_count"),
                items = (0 until (items?.length() ?: 0)).mapNotNull { index ->
                    items?.optJSONObject(index)?.let { json ->
                        AppNotification(
                            id = json.optLong("id"),
                            type = json.stringOrNull("type").orEmpty(),
                            title = json.stringOrNull("title").orEmpty(),
                            body = json.stringOrNull("body").orEmpty(),
                            isRead = json.optBoolean("is_read", false),
                            createdAt = json.stringOrNull("created_at").orEmpty(),
                        )
                    }
                },
            ),
        )
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed notifications page", e))
    }

    private companion object {
        const val HTTP_UNAUTHORIZED = 401
    }
}
