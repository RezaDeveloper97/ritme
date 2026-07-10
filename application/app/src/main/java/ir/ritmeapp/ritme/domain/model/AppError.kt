package ir.ritmeapp.ritme.domain.model

/**
 * The closed set of failures any port can report (CLAUDE.md §4). Keeping this a sealed
 * hierarchy lets every consumer handle errors with an exhaustive `when` instead of
 * guessing at exception types. [message] is safe to surface to logs; user-facing copy is
 * decided in the UI layer, not here.
 */
sealed class AppError(val message: String, val cause: Throwable? = null) {

    /** No connectivity / DNS / socket failure before any HTTP status was received. */
    class Network(message: String, cause: Throwable? = null) : AppError(message, cause)

    /** A request that started but did not complete within the configured timeout. */
    class Timeout(message: String, cause: Throwable? = null) : AppError(message, cause)

    /** The server answered with a non-2xx status. [body] is the raw response, if any. */
    class Http(val statusCode: Int, message: String, val body: String? = null) : AppError(message)

    /** A well-formed response that could not be decoded into the expected shape. */
    class Parsing(message: String, cause: Throwable? = null) : AppError(message, cause)

    /** Input rejected by domain rules before any I/O (e.g. malformed phone number). */
    class Validation(message: String) : AppError(message)

    /** Anything not anticipated above; should be rare and worth investigating. */
    class Unexpected(message: String, cause: Throwable? = null) : AppError(message, cause)
}
