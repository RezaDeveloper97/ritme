package ir.ritmeapp.ritme.platform.crash

/** One lightweight navigation/action event captured for diagnostics (CLAUDE.md §7.5). */
data class Breadcrumb(
    val timestampMillis: Long,
    val message: String,
)
