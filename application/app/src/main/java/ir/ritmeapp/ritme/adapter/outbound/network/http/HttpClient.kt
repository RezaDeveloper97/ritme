package ir.ritmeapp.ritme.adapter.outbound.network.http

import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.IOException
import java.net.HttpURLConnection
import java.net.SocketTimeoutException
import java.net.URL
import java.net.URLEncoder
import java.net.UnknownHostException
import kotlin.coroutines.cancellation.CancellationException

/**
 * Minimal hand-written HTTP client over [HttpURLConnection] (CLAUDE.md §3 — no OkHttp/Retrofit).
 *
 * Design notes:
 *  - All blocking I/O runs on [ioDispatcher] so callers stay off the main thread (§5).
 *  - Response gzip is on transparently: by *not* setting `Accept-Encoding` ourselves,
 *    `HttpURLConnection` advertises gzip and decodes it for us automatically.
 *  - Connection reuse (keep-alive) comes from the platform pool: we fully read and close
 *    each stream and never call `disconnect()`, so the socket returns to the pool.
 *  - Transport failures become [AppResult.Failure]; a completed exchange (even 4xx/5xx)
 *    is an [AppResult.Success] carrying the [HttpResponse].
 */
class HttpClient(
    private val baseUrl: String,
    private val connectTimeoutMs: Int = DEFAULT_CONNECT_TIMEOUT_MS,
    private val readTimeoutMs: Int = DEFAULT_READ_TIMEOUT_MS,
    private val defaultHeaders: Map<String, String> = emptyMap(),
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
    private val responseCache: HttpResponseCache? = null,
) {

    suspend fun execute(request: HttpRequest): AppResult<HttpResponse> = withContext(ioDispatcher) {
        val cacheKey = cacheKeyOrNull(request)
        if (cacheKey != null) {
            responseCache?.get(cacheKey)?.let { return@withContext AppResult.Success(it) }
        }
        try {
            val connection = openConnection(request)
            writeBodyIfPresent(connection, request)
            val status = connection.responseCode
            val stream = if (status in 200..399) connection.inputStream else connection.errorStream
            val text = stream?.bufferedReader(Charsets.UTF_8)?.use { it.readText() }.orEmpty()
            val headers = connection.headerFields
                .filterKeys { it != null }
                .mapKeys { it.key!! }
            val response = HttpResponse(status, headers, text)
            if (cacheKey != null && status in 200..299) {
                responseCache?.put(cacheKey, response, request.cacheTtlMillis)
            }
            AppResult.Success(response)
        } catch (e: CancellationException) {
            throw e // never swallow structured-concurrency cancellation
        } catch (e: SocketTimeoutException) {
            AppResult.Failure(AppError.Timeout("Request timed out: ${request.path}", e))
        } catch (e: UnknownHostException) {
            AppResult.Failure(AppError.Network("No connectivity: ${request.path}", e))
        } catch (e: IOException) {
            AppResult.Failure(AppError.Network("Network error: ${request.path}", e))
        } catch (e: Exception) {
            AppResult.Failure(AppError.Unexpected("Unexpected network failure: ${request.path}", e))
        }
    }

    private fun openConnection(request: HttpRequest): HttpURLConnection =
        (buildUrl(request).openConnection() as HttpURLConnection).apply {
            requestMethod = request.method.name
            connectTimeout = connectTimeoutMs
            readTimeout = readTimeoutMs
            instanceFollowRedirects = true
            setRequestProperty("Accept", "application/json")
            defaultHeaders.forEach { (k, v) -> setRequestProperty(k, v) }
            request.headers.forEach { (k, v) -> setRequestProperty(k, v) }
        }

    private fun writeBodyIfPresent(connection: HttpURLConnection, request: HttpRequest) {
        val body = request.body ?: return
        if (request.method == HttpMethod.GET) return
        connection.doOutput = true
        connection.setRequestProperty("Content-Type", "application/json; charset=utf-8")
        connection.outputStream.use { it.write(body.toByteArray(Charsets.UTF_8)) }
    }

    private fun buildUrl(request: HttpRequest): URL {
        val sb = StringBuilder(baseUrl.trimEnd('/'))
        sb.append('/').append(request.path.trimStart('/'))
        if (request.query.isNotEmpty()) {
            sb.append('?')
            sb.append(
                request.query.entries.joinToString("&") { (k, v) -> "${encode(k)}=${encode(v)}" },
            )
        }
        return URL(sb.toString())
    }

    private fun encode(value: String): String = URLEncoder.encode(value, "UTF-8")

    /** Only explicit-TTL GETs are cacheable; the key is the full resolved URL. */
    private fun cacheKeyOrNull(request: HttpRequest): String? =
        if (request.method == HttpMethod.GET && request.cacheTtlMillis > 0 && responseCache != null) {
            buildUrl(request).toString()
        } else {
            null
        }

    private companion object {
        const val DEFAULT_CONNECT_TIMEOUT_MS = 10_000
        const val DEFAULT_READ_TIMEOUT_MS = 15_000
    }
}
