package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.outbound.SessionGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [SessionGateway] over `POST /auth/logout` and `DELETE /account`. Both are
 * fire-and-confirm calls: the interesting side effects (revoking tokens, wiping data)
 * happen server-side; the caller clears local state on success.
 */
class SessionGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : SessionGateway {

    override suspend fun logout(): AppResult<Unit> {
        Breadcrumbs.add("api:logout:start")
        return confirm(HttpMethod.POST, ApiConfig.LOGOUT_PATH, "Logout rejected")
    }

    override suspend fun deleteAccount(): AppResult<Unit> {
        Breadcrumbs.add("api:delete_account:start")
        return confirm(HttpMethod.DELETE, ApiConfig.ACCOUNT_PATH, "Account deletion rejected")
    }

    override suspend fun exportData(): AppResult<String> {
        Breadcrumbs.add("api:export:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.PROFILE_EXPORT_PATH)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                val response = result.value
                Breadcrumbs.add("api:export:http_${response.statusCode}")
                if (response.isSuccessful) {
                    AppResult.Success(response.body)
                } else {
                    AppResult.Failure(AppError.Http(response.statusCode, "Export rejected", response.body))
                }
            }
        }
    }

    private suspend fun confirm(method: HttpMethod, path: String, rejectionMessage: String): AppResult<Unit> {
        return when (val result = httpClient.authorized(tokenStore, method, path)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                val response = result.value
                Breadcrumbs.add("api:session:http_${response.statusCode}")
                try {
                    val root = if (response.body.isBlank()) JSONObject() else JSONObject(response.body)
                    if (root.optBoolean("success", false)) {
                        AppResult.Success(Unit)
                    } else {
                        val message = root.stringOrNull("message") ?: rejectionMessage
                        AppResult.Failure(AppError.Http(response.statusCode, message, response.body))
                    }
                } catch (e: Exception) {
                    AppResult.Failure(AppError.Parsing("Malformed session response (${response.statusCode})", e))
                }
            }
        }
    }
}
