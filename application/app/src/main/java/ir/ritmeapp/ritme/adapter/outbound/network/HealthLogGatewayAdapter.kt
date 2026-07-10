package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.HealthLogControl
import ir.ritmeapp.ritme.domain.model.HealthLogField
import ir.ritmeapp.ritme.domain.model.HealthLogValue
import ir.ritmeapp.ritme.domain.port.outbound.HealthLogGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONArray
import org.json.JSONObject

private const val HTTP_NOT_FOUND = 404

/**
 * Network [HealthLogGateway] over `/health-logs`. Reading and writing are both driven by the
 * [HealthLogField] catalog: each field is (de)serialized according to its control kind, so a new
 * field needs one enum entry, not new mapping code. Health values are NEVER breadcrumbed (§11);
 * only coarse lifecycle markers are.
 */
class HealthLogGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : HealthLogGateway {

    override suspend fun logFor(date: GregorianDate): AppResult<DailyHealthLog?> {
        Breadcrumbs.add("api:health_log_get:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.healthLogPath(date.toIso()))
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                val response = result.value
                Breadcrumbs.add("api:health_log_get:http_${response.statusCode}")
                // A day with no log is a normal state, not an error.
                if (response.statusCode == HTTP_NOT_FOUND) return AppResult.Success(null)
                when (val envelope = response.readEnvelope("Health-log request rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> parseLog(date, envelope.value.dataObject())
                }
            }
        }
    }

    override suspend fun save(log: DailyHealthLog): AppResult<Unit> {
        Breadcrumbs.add("api:health_log_save:start")
        val result = httpClient.authorized(
            tokenStore, HttpMethod.POST, ApiConfig.HEALTH_LOGS_PATH,
            body = serialize(log).toString(),
        )
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:health_log_save:http_${result.value.statusCode}")
                result.value.readEnvelope("Health-log save rejected").map { }
            }
        }
    }

    private fun parseLog(date: GregorianDate, data: JSONObject): AppResult<DailyHealthLog?> = try {
        val values = buildMap {
            for (field in HealthLogField.entries) {
                readValue(data, field)?.let { put(field, it) }
            }
        }
        AppResult.Success(DailyHealthLog(date, values))
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed health log", e))
    }

    /** Reads one field per its control kind; absent/null/off values return null (not recorded). */
    private fun readValue(json: JSONObject, field: HealthLogField): HealthLogValue? = when (field.control) {
        is HealthLogControl.Choice, HealthLogControl.Degree ->
            json.stringOrNull(field.apiKey)?.let { HealthLogValue.Choice(it) }

        is HealthLogControl.MultiChoice ->
            json.stringList(field.apiKey).takeIf { it.isNotEmpty() }?.let { HealthLogValue.MultiChoice(it) }

        HealthLogControl.Toggle ->
            if (json.has(field.apiKey) && !json.isNull(field.apiKey) && json.readBoolean(field.apiKey)) {
                HealthLogValue.Toggle(true)
            } else {
                null
            }

        is HealthLogControl.Measure ->
            json.doubleOrNull(field.apiKey)?.let { HealthLogValue.Number(it) }

        HealthLogControl.Note ->
            json.stringOrNull(field.apiKey)?.takeIf { it.isNotBlank() }?.let { HealthLogValue.Text(it) }
    }

    private fun serialize(log: DailyHealthLog): JSONObject {
        val json = JSONObject()
        json.put("log_date", log.date.toIso())
        log.values.forEach { (field, value) ->
            when (value) {
                is HealthLogValue.Choice -> json.put(field.apiKey, value.option)
                is HealthLogValue.MultiChoice -> json.put(field.apiKey, JSONArray(value.options))
                is HealthLogValue.Toggle -> json.put(field.apiKey, value.enabled)
                is HealthLogValue.Number -> json.put(field.apiKey, value.value)
                is HealthLogValue.Text -> json.put(field.apiKey, value.text)
            }
        }
        return json
    }
}

/** SQLite-backed booleans may arrive as `true`/`1`/`"1"`; normalize all of them. */
private fun JSONObject.readBoolean(key: String): Boolean = when (val raw = opt(key)) {
    is Boolean -> raw
    is Int -> raw != 0
    is String -> raw == "1" || raw.equals("true", ignoreCase = true)
    else -> false
}
