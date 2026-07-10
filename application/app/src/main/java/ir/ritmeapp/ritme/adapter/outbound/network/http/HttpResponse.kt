package ir.ritmeapp.ritme.adapter.outbound.network.http

/**
 * The outcome of a completed HTTP exchange. A non-2xx status is still a *response*, not a
 * transport failure, so it arrives here (with its [statusCode]) rather than as an
 * `AppResult.Failure`; the calling adapter decides how to treat it.
 */
data class HttpResponse(
    val statusCode: Int,
    val headers: Map<String, List<String>>,
    val body: String,
) {
    val isSuccessful: Boolean get() = statusCode in 200..299
}
