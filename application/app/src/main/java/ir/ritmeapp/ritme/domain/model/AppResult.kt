package ir.ritmeapp.ritme.domain.model

/**
 * Hand-written `Result` type (CLAUDE.md §4). Adapters return this across every port
 * boundary instead of throwing, so failures are part of the type and impossible to
 * forget to handle. Distinct name from `kotlin.Result` to avoid accidental confusion.
 */
sealed interface AppResult<out T> {
    data class Success<out T>(val value: T) : AppResult<T>
    data class Failure(val error: AppError) : AppResult<Nothing>
}

/** Maps the success value, leaving a failure untouched. */
inline fun <T, R> AppResult<T>.map(transform: (T) -> R): AppResult<R> = when (this) {
    is AppResult.Success -> AppResult.Success(transform(value))
    is AppResult.Failure -> this
}

/** Collapses both branches into a single value of type [R]. */
inline fun <T, R> AppResult<T>.fold(
    onSuccess: (T) -> R,
    onFailure: (AppError) -> R,
): R = when (this) {
    is AppResult.Success -> onSuccess(value)
    is AppResult.Failure -> onFailure(error)
}

/** The success value, or `null` on failure. */
fun <T> AppResult<T>.getOrNull(): T? = (this as? AppResult.Success)?.value
