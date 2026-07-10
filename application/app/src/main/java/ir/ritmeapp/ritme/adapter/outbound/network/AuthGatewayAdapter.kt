package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpRequest
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpResponse
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.model.OtpChallenge
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [AuthGateway] over the Ritme OTP endpoints under `/api/v1/auth`. Translates between
 * the domain and the backend's JSON using `org.json` (ships in the Android SDK — not a
 * third-party add, §3). Never throws across the port boundary: every outcome is an [AppResult].
 *
 * The backend wraps every response in a uniform envelope — `{success, message, data:{…}}` —
 * and puts its real (often Persian) reason in `message` even on 4xx/5xx, so a completed
 * exchange is always parsed and `success:false` surfaces that message to the user.
 */
class AuthGatewayAdapter(
    private val httpClient: HttpClient,
) : AuthGateway {

    override suspend fun sendOtp(mobile: PhoneNumber): AppResult<OtpChallenge> {
        Breadcrumbs.add("api:send_otp:start")
        val payload = JSONObject()
            .put("mobile", mobile.national)
            // SMS gateway is currently down — force test mode so the backend uses the
            // fixed OTP 1111 and skips sending a real SMS. Flip back to false once SMS is live.
            .put("is_test", true)
            .toString()
        return post(ApiConfig.SEND_OTP_PATH, payload, "send_otp") { parseChallenge(it) }
    }

    override suspend fun verifyOtp(mobile: PhoneNumber, code: String): AppResult<AuthTokens> {
        Breadcrumbs.add("api:verify_otp:start")
        val payload = JSONObject()
            .put("mobile", mobile.national)
            .put("code", code)
            .toString()
        return post(ApiConfig.VERIFY_OTP_PATH, payload, "verify_otp") { parseTokens(it) }
    }

    /**
     * Runs a POST and hands the completed [HttpResponse] to [parse]. Transport failures are
     * propagated untouched; a completed exchange (even 4xx/5xx) is parsed, because the
     * backend puts its real error message in the body regardless of HTTP status.
     */
    private suspend fun <T> post(
        path: String,
        body: String,
        tag: String,
        parse: (HttpResponse) -> AppResult<T>,
    ): AppResult<T> {
        val request = HttpRequest(method = HttpMethod.POST, path = path, body = body)
        return when (val result = httpClient.execute(request)) {
            is AppResult.Failure -> {
                Breadcrumbs.add("api:$tag:transport_error")
                result
            }

            is AppResult.Success -> {
                Breadcrumbs.add("api:$tag:http_${result.value.statusCode}")
                parse(result.value)
            }
        }
    }

    /** Parses a `send-otp` response into an [OtpChallenge] (`data.new_user`, `data.expires_in`). */
    private fun parseChallenge(response: HttpResponse): AppResult<OtpChallenge> = catching(response) {
        val root = root(response.body)
        if (!root.optBoolean("success", false)) {
            return failure(response, root, "OTP request rejected")
        }
        val data = root.optJSONObject("data") ?: JSONObject()
        AppResult.Success(
            OtpChallenge(
                newUser = data.optBoolean("new_user", false),
                expireSeconds = data.optInt("expires_in", DEFAULT_RESEND_SECONDS),
            ),
        )
    }

    /** Parses a `verify-otp` response (`data.access_token`) into [AuthTokens]. */
    private fun parseTokens(response: HttpResponse): AppResult<AuthTokens> = catching(response) {
        val root = root(response.body)
        if (!root.optBoolean("success", false)) {
            return failure(response, root, "OTP verification failed")
        }
        val data = root.optJSONObject("data") ?: JSONObject()
        val access = data.optString("access_token")
        if (access.isBlank()) {
            return failure(response, root, "No access token in response")
        }
        AppResult.Success(AuthTokens(accessToken = access))
    }

    private fun root(body: String): JSONObject =
        if (body.isBlank()) JSONObject() else JSONObject(body)

    /** Builds a failure, preferring the backend's own [message] when present. */
    private fun failure(
        response: HttpResponse,
        root: JSONObject,
        fallback: String,
    ): AppResult.Failure {
        val serverMessage = root.optString("message").takeIf { it.isNotBlank() }
        Breadcrumbs.add("api:auth:rejected_${response.statusCode}")
        return AppResult.Failure(
            AppError.Http(
                statusCode = response.statusCode,
                message = serverMessage ?: fallback,
                body = response.body,
            ),
        )
    }

    /** Wraps any JSON/parse exception into a domain [AppError.Parsing]. */
    private inline fun <T> catching(
        response: HttpResponse,
        block: () -> AppResult<T>,
    ): AppResult<T> = try {
        block()
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed auth response (${response.statusCode})", e))
    }

    private companion object {
        const val DEFAULT_RESEND_SECONDS = 120
    }
}
