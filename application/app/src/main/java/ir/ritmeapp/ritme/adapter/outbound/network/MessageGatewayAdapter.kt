package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.DailyMessage
import ir.ritmeapp.ritme.domain.port.outbound.MessageGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [MessageGateway] over `GET /messages/daily`. The backend answers `success:false` (HTTP
 * 400) when the profile is incomplete and sends `primary_message` as an empty array in error/empty
 * results; both map to a `null` success here so the Home screen simply renders without a message
 * rather than surfacing an error. Strings are already localized server-side and passed through.
 */
class MessageGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : MessageGateway {

    override suspend fun daily(): AppResult<DailyMessage?> {
        Breadcrumbs.add("api:daily_message:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.MESSAGES_DAILY_PATH)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:daily_message:http_${result.value.statusCode}")
                parse(result.value.body)
            }
        }
    }

    private fun parse(body: String): AppResult<DailyMessage?> = try {
        val root = if (body.isBlank()) JSONObject() else JSONObject(body)
        // A rejected/incomplete-profile response or a missing primary object → no message, not an error.
        val data = root.takeIf { it.optBoolean("success", false) }?.optJSONObject("data")
        val primary = data?.optJSONObject("primary_message")
        if (primary == null) {
            AppResult.Success(null)
        } else {
            val context = data.optJSONObject("context_info")
            AppResult.Success(
                DailyMessage(
                    phase = CyclePhase.fromApi(context?.stringOrNull("phase") ?: primary.stringOrNull("phase")),
                    phaseLabel = context?.stringOrNull("phase_label") ?: primary.stringOrNull("phase_label"),
                    cycleDay = context?.intOrNull("cycle_day") ?: primary.intOrNull("cycle_day"),
                    shortMessage = primary.optString("short_message"),
                    longMessage = primary.optString("long_message"),
                    actionSuggestion = primary.optString("action_suggestion"),
                    dos = primary.stringList("dos"),
                    donts = primary.stringList("donts"),
                ),
            )
        }
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed daily-message response", e))
    }
}
