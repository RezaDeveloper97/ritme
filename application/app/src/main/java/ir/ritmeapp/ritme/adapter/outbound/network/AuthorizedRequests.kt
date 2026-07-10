package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpRequest
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpResponse
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore

private const val HTTP_UNAUTHORIZED = 401

/**
 * Runs an [HttpClient] request with the stored session's bearer token attached — the shared seam for
 * every authenticated endpoint adapter (profile, cycle, message). A missing token short-circuits to
 * a 401 [AppError.Http] rather than firing a doomed request.
 */
internal suspend fun HttpClient.authorized(
    tokenStore: TokenStore,
    method: HttpMethod,
    path: String,
    body: String? = null,
): AppResult<HttpResponse> {
    val token = tokenStore.load()?.accessToken
    if (token.isNullOrBlank()) {
        return AppResult.Failure(AppError.Http(HTTP_UNAUTHORIZED, "No active session"))
    }
    return execute(
        HttpRequest(
            method = method,
            path = path,
            headers = mapOf("Authorization" to "Bearer $token"),
            body = body,
        ),
    )
}
