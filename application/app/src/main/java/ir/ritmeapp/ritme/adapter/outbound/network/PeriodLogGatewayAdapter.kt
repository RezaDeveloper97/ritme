package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.LoggedPeriod
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.port.outbound.PeriodLogGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [PeriodLogGateway] over `/cycle/period/…`. Period dates are sensitive health
 * data: they cross this boundary as ISO strings and are never logged — breadcrumbs carry
 * only the operation name and HTTP status.
 */
class PeriodLogGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : PeriodLogGateway {

    override suspend fun history(): AppResult<List<LoggedPeriod>> {
        Breadcrumbs.add("api:period_history:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.PERIOD_HISTORY_PATH)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:period_history:http_${result.value.statusCode}")
                when (val envelope = result.value.readEnvelope("Period history rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> parseHistory(envelope.value.dataObject())
                }
            }
        }
    }

    override suspend fun start(date: GregorianDate): AppResult<Unit> =
        mutate(HttpMethod.POST, ApiConfig.PERIOD_START_PATH, JSONObject().put("date", date.toIso()), "period_start")

    override suspend fun update(id: Long, start: GregorianDate, end: GregorianDate?): AppResult<Unit> =
        mutate(
            HttpMethod.PUT,
            ApiConfig.periodPath(id),
            JSONObject()
                .put("start_date", start.toIso())
                .put("end_date", end?.toIso() ?: JSONObject.NULL),
            "period_update",
        )

    override suspend fun delete(id: Long): AppResult<Unit> =
        mutate(HttpMethod.DELETE, ApiConfig.periodPath(id), body = null, crumb = "period_delete")

    private suspend fun mutate(
        method: HttpMethod,
        path: String,
        body: JSONObject?,
        crumb: String,
    ): AppResult<Unit> {
        Breadcrumbs.add("api:$crumb:start")
        val result = httpClient.authorized(tokenStore, method, path, body?.toString())
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:$crumb:http_${result.value.statusCode}")
                result.value.readEnvelope("Period edit rejected").map { }
            }
        }
    }

    private fun parseHistory(data: JSONObject): AppResult<List<LoggedPeriod>> = try {
        val array = data.optJSONArray("periods")
        val periods = (0 until (array?.length() ?: 0)).mapNotNull { index ->
            val item = array?.optJSONObject(index) ?: return@mapNotNull null
            val start = item.stringOrNull("period_start_date")?.let(GregorianDate::parseIso)
                ?: return@mapNotNull null
            LoggedPeriod(
                id = item.optLong("id"),
                start = start,
                end = item.stringOrNull("period_end_date")?.let(GregorianDate::parseIso),
            )
        }
        AppResult.Success(periods)
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed period history", e))
    }
}
