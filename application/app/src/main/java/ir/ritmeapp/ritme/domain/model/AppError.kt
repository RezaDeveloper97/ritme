package ir.ritmeapp.ritme.domain.model

/**
 * The closed set of failures any port can report (CLAUDE.md §4). Keeping this a sealed
 * hierarchy lets every consumer handle errors with an exhaustive `when` instead of
 * guessing at exception types. [message] is safe to surface to logs; user-facing copy is
 * decided in the UI layer, not here.
 *
 * Every variant carries a stable `RIT-XXXX` [code] (§4 Error Handling) so logs, crash
 * reports, and user messages line up across app versions. Ranges: 1xxx network/transport,
 * 2xxx parsing, 3xxx persistence, 4xxx auth/session, 5xxx domain rules, 9xxx unknown.
 */
sealed class AppError(val code: String, val message: String, val cause: Throwable? = null) {

    /** No connectivity / DNS / socket failure before any HTTP status was received. */
    class Network(message: String, cause: Throwable? = null) : AppError(CODE_NETWORK, message, cause)

    /** A request that started but did not complete within the configured timeout. */
    class Timeout(message: String, cause: Throwable? = null) : AppError(CODE_TIMEOUT, message, cause)

    /** The server answered with a non-2xx status. [body] is the raw response, if any. */
    class Http(val statusCode: Int, message: String, val body: String? = null) :
        AppError(codeForStatus(statusCode), message) {

        /** True when the backend rejected the session token — the caller must re-authenticate. */
        val isSessionExpired: Boolean get() = statusCode == STATUS_UNAUTHORIZED
    }

    /** A well-formed response that could not be decoded into the expected shape. */
    class Parsing(message: String, cause: Throwable? = null) : AppError(CODE_PARSING, message, cause)

    /** SQLite / disk-cache failure inside a persistence adapter. */
    class Storage(message: String, cause: Throwable? = null) : AppError(CODE_STORAGE, message, cause)

    /** Input rejected by domain rules before any I/O (e.g. malformed phone number). */
    class Validation(message: String) : AppError(CODE_VALIDATION, message)

    /** Anything not anticipated above; should be rare and worth investigating. */
    class Unexpected(message: String, cause: Throwable? = null) : AppError(CODE_UNEXPECTED, message, cause)

    companion object {
        private const val CODE_NETWORK = "RIT-1001"
        private const val CODE_TIMEOUT = "RIT-1002"
        private const val CODE_HTTP = "RIT-1100"
        private const val CODE_PARSING = "RIT-2001"
        private const val CODE_STORAGE = "RIT-3001"
        private const val CODE_SESSION = "RIT-4001"
        private const val CODE_FORBIDDEN = "RIT-4003"
        private const val CODE_VALIDATION = "RIT-5001"
        private const val CODE_UNEXPECTED = "RIT-9000"

        private const val STATUS_UNAUTHORIZED = 401
        private const val STATUS_FORBIDDEN = 403

        /** Auth statuses get their own 4xxx codes; every other non-2xx is generic transport. */
        private fun codeForStatus(statusCode: Int): String = when (statusCode) {
            STATUS_UNAUTHORIZED -> CODE_SESSION
            STATUS_FORBIDDEN -> CODE_FORBIDDEN
            else -> CODE_HTTP
        }
    }
}
