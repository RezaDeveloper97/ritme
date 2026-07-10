package ir.ritmeapp.ritme.domain.model

/**
 * A snapshot of the last screen that rendered successfully, persisted for crash recovery
 * (CLAUDE.md §7.2). [args] must stay *minimal* — a small identifier like a plan id, never
 * a serialized object — because this record may be written right up to the moment of a crash.
 */
data class SafeScreen(
    val route: String,
    val args: String?,
    val timestampMillis: Long,
)
