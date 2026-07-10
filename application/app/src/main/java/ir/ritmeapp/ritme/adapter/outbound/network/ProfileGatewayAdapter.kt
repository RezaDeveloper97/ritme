package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpRequest
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.BmiInfo
import ir.ritmeapp.ritme.domain.model.ChronicCondition
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.HealthProfile
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyIntention
import ir.ritmeapp.ritme.domain.model.UserAccount
import ir.ritmeapp.ritme.domain.model.UserProfile
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONArray
import org.json.JSONObject

/**
 * Network [ProfileGateway] over `POST /api/v1/profile`. Serializes [OnboardingAnswers] into the
 * backend's allow-listed field set (dates converted to ISO Gregorian), attaches the stored bearer
 * token, and reads the uniform `{success,message,data}` envelope. Health inputs are NEVER logged
 * or breadcrumbed (§11) — only coarse lifecycle markers are.
 *
 * Two fields are deliberately omitted: `user_goal` (the backend derives it from the intention) and
 * `subscription_type` (server-only; a client must never grant itself premium).
 */
class ProfileGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : ProfileGateway {

    override suspend fun fetchProfile(): AppResult<UserProfile> {
        Breadcrumbs.add("api:fetch_profile:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.PROFILE_PATH)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:fetch_profile:http_${result.value.statusCode}")
                parseProfile(result.value.statusCode, result.value.body)
            }
        }
    }

    private fun parseProfile(statusCode: Int, body: String): AppResult<UserProfile> = try {
        val root = if (body.isBlank()) JSONObject() else JSONObject(body)
        if (!root.optBoolean("success", false)) {
            val message = root.stringOrNull("message") ?: "Profile request rejected"
            AppResult.Failure(AppError.Http(statusCode, message, body))
        } else {
            val data = root.optJSONObject("data") ?: JSONObject()
            val user = data.optJSONObject("user") ?: JSONObject()
            AppResult.Success(
                UserProfile(
                    account = UserAccount(
                        id = user.optLong("id"),
                        name = user.stringOrNull("name"),
                        mobile = user.stringOrNull("mobile").orEmpty(),
                    ),
                    health = data.optJSONObject("profile")?.let(::readHealthProfile),
                    bmi = data.optJSONObject("bmi")?.let(::readBmi),
                ),
            )
        }
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed profile response ($statusCode)", e))
    }

    private fun readHealthProfile(profile: JSONObject): HealthProfile = HealthProfile(
        birthday = profile.stringOrNull("birthday")?.let(::parseIsoDate),
        weightKg = profile.doubleOrNull("weight"),
        heightCm = profile.intOrNull("height"),
        periodDuration = profile.intOrNull("period_duration"),
        cycleDuration = profile.intOrNull("cycle_duration"),
        lastPeriodStart = profile.stringOrNull("last_period_start")?.let(::parseIsoDate),
        intention = PregnancyIntention.fromApi(profile.stringOrNull("pregnancy_intention")),
        conditions = profile.stringList("chronic_conditions").mapNotNull(ChronicCondition::fromApi),
    )

    private fun readBmi(bmi: JSONObject): BmiInfo? {
        val value = bmi.doubleOrNull("value") ?: return null
        return BmiInfo(
            value = value,
            category = bmi.stringOrNull("category").orEmpty(),
            categoryLabel = bmi.stringOrNull("category_label").orEmpty(),
            message = bmi.stringOrNull("message").orEmpty(),
        )
    }

    /** Date fields may arrive as bare `yyyy-MM-dd` or a full ISO datetime; keep the date part. */
    private fun parseIsoDate(value: String): GregorianDate? =
        GregorianDate.parseIso(value.substringBefore('T').substringBefore(' '))

    override suspend fun saveProfile(answers: OnboardingAnswers): AppResult<Unit> {
        Breadcrumbs.add("api:save_profile:start")
        val token = tokenStore.load()?.accessToken
        if (token.isNullOrBlank()) {
            Breadcrumbs.add("api:save_profile:no_session")
            return AppResult.Failure(AppError.Http(HTTP_UNAUTHORIZED, "No active session"))
        }

        val request = HttpRequest(
            method = HttpMethod.POST,
            path = ApiConfig.PROFILE_PATH,
            headers = mapOf("Authorization" to "Bearer $token"),
            body = buildBody(answers).toString(),
        )

        return when (val result = httpClient.execute(request)) {
            is AppResult.Failure -> {
                Breadcrumbs.add("api:save_profile:transport_error")
                result
            }

            is AppResult.Success -> {
                val response = result.value
                Breadcrumbs.add("api:save_profile:http_${response.statusCode}")
                parse(response.statusCode, response.body)
            }
        }
    }

    /** Only the fields that are set are sent; cycle fields are skipped entirely in pregnancy mode. */
    private fun buildBody(answers: OnboardingAnswers): JSONObject {
        val json = JSONObject()
        answers.name?.trim()?.takeIf { it.isNotEmpty() }?.let { json.put("name", it) }
        answers.birthday?.let { json.put("birthday", it.toIso()) }
        answers.weightKg?.let { json.put("weight", it) }
        answers.heightCm?.let { json.put("height", it) }
        answers.intention?.let { json.put("pregnancy_intention", it.apiValue) }

        if (answers.intention?.isCycleMode != false) {
            answers.periodDuration?.let { json.put("period_duration", it) }
            answers.cycleDuration?.let { json.put("cycle_duration", it) }
            answers.lastPeriod?.let { json.put("last_period_start", it.toIso()) }
        }

        // Only sent when actually chosen — a partial update (e.g. start-period) must never
        // overwrite the stored conditions with an empty list.
        if (answers.conditions.isNotEmpty()) {
            val conditions = JSONArray()
            answers.conditions.forEach { conditions.put(it.apiValue) }
            json.put("chronic_conditions", conditions)
        }
        return json
    }

    /** A completed exchange is parsed regardless of status: the backend puts its reason in `message`. */
    private fun parse(statusCode: Int, body: String): AppResult<Unit> = try {
        val root = if (body.isBlank()) JSONObject() else JSONObject(body)
        if (root.optBoolean("success", false)) {
            AppResult.Success(Unit)
        } else {
            val serverMessage = root.optString("message").takeIf { it.isNotBlank() }
            AppResult.Failure(AppError.Http(statusCode, serverMessage ?: "Profile save rejected", body))
        }
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed profile response ($statusCode)", e))
    }

    private companion object {
        const val HTTP_UNAUTHORIZED = 401
    }
}
