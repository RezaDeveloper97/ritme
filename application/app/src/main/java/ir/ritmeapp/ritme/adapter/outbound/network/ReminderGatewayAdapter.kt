package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft
import ir.ritmeapp.ritme.domain.model.ReminderRecurrence
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.port.outbound.ReminderGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONArray
import org.json.JSONObject

/**
 * Network [ReminderGateway] over the `/reminders` CRUD endpoints. List responses carry the
 * records directly under `data`; create/update return the single record the same way.
 */
class ReminderGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : ReminderGateway {

    override suspend fun reminders(): AppResult<List<Reminder>> {
        Breadcrumbs.add("api:reminders:list")
        return call(HttpMethod.GET, ApiConfig.REMINDERS_PATH, body = null) { data ->
            when (data) {
                is JSONArray -> (0 until data.length()).mapNotNull { data.optJSONObject(it)?.let(::readReminder) }
                else -> emptyList()
            }
        }
    }

    override suspend fun create(draft: ReminderDraft): AppResult<Reminder> {
        Breadcrumbs.add("api:reminders:create")
        return callForReminder(HttpMethod.POST, ApiConfig.REMINDERS_PATH, draft)
    }

    override suspend fun update(id: Long, draft: ReminderDraft): AppResult<Reminder> {
        Breadcrumbs.add("api:reminders:update")
        return callForReminder(HttpMethod.PUT, ApiConfig.reminderPath(id), draft)
    }

    override suspend fun delete(id: Long): AppResult<Unit> {
        Breadcrumbs.add("api:reminders:delete")
        return call(HttpMethod.DELETE, ApiConfig.reminderPath(id), body = null) { }
    }

    private suspend fun callForReminder(method: HttpMethod, path: String, draft: ReminderDraft): AppResult<Reminder> =
        call(method, path, serialize(draft).toString()) { data ->
            (data as? JSONObject)?.let(::readReminder)
                ?: throw IllegalStateException("Reminder payload missing")
        }

    /** Shared request/envelope handling; [readData] maps the envelope's `data` node. */
    private suspend fun <T> call(
        method: HttpMethod,
        path: String,
        body: String?,
        readData: (Any?) -> T,
    ): AppResult<T> {
        return when (val result = httpClient.authorized(tokenStore, method, path, body)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                val response = result.value
                Breadcrumbs.add("api:reminders:http_${response.statusCode}")
                try {
                    val root = if (response.body.isBlank()) JSONObject() else JSONObject(response.body)
                    if (!root.optBoolean("success", false)) {
                        val message = root.stringOrNull("message") ?: "Reminder request rejected"
                        AppResult.Failure(AppError.Http(response.statusCode, message, response.body))
                    } else {
                        AppResult.Success(readData(root.opt("data")))
                    }
                } catch (e: Exception) {
                    AppResult.Failure(AppError.Parsing("Malformed reminder response (${response.statusCode})", e))
                }
            }
        }
    }

    private fun serialize(draft: ReminderDraft): JSONObject = JSONObject().apply {
        put("type", draft.type.apiValue)
        put("title", draft.title)
        draft.subtitle?.let { put("subtitle", it) }
        draft.notes?.let { put("notes", it) }
        draft.scheduledAt?.let { put("scheduled_at", it) }
        put("recurrence", draft.recurrence.apiValue)
        draft.recurrenceTime?.let { put("recurrence_time", it) }
        put("is_active", draft.isActive)
    }

    private fun readReminder(json: JSONObject): Reminder = Reminder(
        id = json.optLong("id"),
        type = ReminderType.fromApi(json.stringOrNull("type")),
        title = json.stringOrNull("title").orEmpty(),
        subtitle = json.stringOrNull("subtitle"),
        notes = json.stringOrNull("notes"),
        scheduledAt = json.stringOrNull("scheduled_at"),
        recurrence = ReminderRecurrence.fromApi(json.stringOrNull("recurrence")),
        recurrenceTime = json.stringOrNull("recurrence_time"),
        isActive = json.optBoolean("is_active", true),
    )
}
