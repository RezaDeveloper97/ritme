package ir.ritmeapp.ritme.adapter.outbound.network.http

/**
 * An immutable description of one HTTP call. [path] is relative to the client's base URL.
 * [body] is an already-serialized payload (JSON string) — serialization is the caller's job,
 * keeping the transport client ignorant of any specific data shape.
 *
 * [cacheTtlMillis] > 0 opts a GET into the two-tier response cache (CLAUDE.md §5) for
 * that long; the default 0 means "always hit the network" so caching stays explicit.
 */
data class HttpRequest(
    val method: HttpMethod,
    val path: String,
    val headers: Map<String, String> = emptyMap(),
    val query: Map<String, String> = emptyMap(),
    val body: String? = null,
    val cacheTtlMillis: Long = 0L,
)
