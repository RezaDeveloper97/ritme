package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.model.CycleCalculation
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.CycleMonth
import ir.ritmeapp.ritme.domain.model.CycleMonthSummary
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.port.outbound.CycleGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [CycleGateway] over the `/cycle/…` endpoints. Maps the backend `calculation` object into
 * domain shapes, keeping only render fields and dropping every internal scoring value (§11). The
 * empty-profile case arrives as a calculation whose `cycle_day` is null — preserved so
 * [CycleCalculation.hasData] reports it correctly.
 */
class CycleGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : CycleGateway {

    override suspend fun today(): AppResult<CycleCalculation> = calculationAt(ApiConfig.CYCLE_TODAY_PATH, "cycle_today")

    override suspend fun forDate(date: GregorianDate): AppResult<CycleCalculation> =
        calculationAt(ApiConfig.cycleDatePath(date.toIso()), "cycle_date")

    override suspend fun month(year: Int, month: Int): AppResult<CycleMonth> {
        Breadcrumbs.add("api:cycle_month:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.cycleMonthPath(year, month))
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:cycle_month:http_${result.value.statusCode}")
                when (val envelope = result.value.readEnvelope("Cycle month request rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> parseMonth(envelope.value.dataObject())
                }
            }
        }
    }

    override suspend fun recalculate(): AppResult<Unit> {
        Breadcrumbs.add("api:cycle_recalculate:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.POST, ApiConfig.CYCLE_RECALCULATE_PATH)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> result.value.readEnvelope("Recalculation rejected").map { }
        }
    }

    override suspend fun isRecalculating(): AppResult<Boolean> {
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.CYCLE_STATUS_PATH)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> result.value.readEnvelope("Cycle status rejected")
                .map { it.dataObject().optBoolean("is_processing", false) }
        }
    }

    private suspend fun calculationAt(path: String, crumb: String): AppResult<CycleCalculation> {
        Breadcrumbs.add("api:$crumb:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, path)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:$crumb:http_${result.value.statusCode}")
                when (val envelope = result.value.readEnvelope("Cycle request rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> parseCalculation(envelope.value.dataObject())
                }
            }
        }
    }

    private fun parseCalculation(data: JSONObject): AppResult<CycleCalculation> = try {
        val calc = data.optJSONObject("calculation") ?: JSONObject()
        AppResult.Success(
            CycleCalculation(
                cycleDay = calc.intOrNull("cycle_day"),
                cycleLength = calc.intOrNull("cycle_length_used"),
                phase = CyclePhase.fromApi(calc.stringOrNull("phase")),
                estimatedOvulationDay = calc.intOrNull("estimated_ovulation_day"),
                fertilityPercent = calc.doubleOrNull("final_probability"),
                isFertileWindow = calc.optBoolean("is_fertile_window", false),
                isPeriodTomorrow = calc.optBoolean("is_period_tomorrow", false),
                isRecalculating = data.optBoolean("is_recalculating", false),
                variability = calc.stringOrNull("cycle_variability"),
            ),
        )
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed cycle calculation", e))
    }

    private fun parseMonth(data: JSONObject): AppResult<CycleMonth> = try {
        val calcs = data.optJSONArray("calculations")
        val days = (0 until (calcs?.length() ?: 0)).mapNotNull { index ->
            val calc = calcs?.optJSONObject(index) ?: return@mapNotNull null
            val date = calc.stringOrNull("calculation_date")?.let(GregorianDate::parseIso) ?: return@mapNotNull null
            CycleDaySnapshot(
                date = date,
                cycleDay = calc.intOrNull("cycle_day"),
                phase = CyclePhase.fromApi(calc.stringOrNull("phase")),
                isFertileWindow = calc.optBoolean("is_fertile_window", false),
                isPmsWindow = calc.optBoolean("is_pms_window", false),
                fertilityPercent = calc.doubleOrNull("final_probability"),
            )
        }
        val summary = data.optJSONObject("month_summary") ?: JSONObject()
        AppResult.Success(
            CycleMonth(
                days = days,
                summary = CycleMonthSummary(
                    fertileDays = summary.optInt("fertile_days"),
                    periodDays = summary.optInt("period_days"),
                    pmsDays = summary.optInt("pms_days"),
                ),
            ),
        )
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed cycle month", e))
    }
}
