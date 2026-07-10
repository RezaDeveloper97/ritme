package ir.ritmeapp.ritme.platform.crash

/**
 * The full diagnostic snapshot written on a fatal (or non-fatal) error. [appVersionCode]
 * and [appVersionName] are first-class so the backend can ask "what's breaking in version
 * X" (CLAUDE.md §7.4). Serialized by [CrashReportStore] with a hand-written JSON writer.
 */
data class CrashPayload(
    val timestampMillis: Long,
    val severity: String,
    val appVersionName: String,
    val appVersionCode: Long,
    val sdkInt: Int,
    val osVersion: String,
    val manufacturer: String,
    val deviceModel: String,
    val freeMemoryBytes: Long,
    val threadName: String,
    val stackTrace: String,
    val safeRoute: String?,
    val safeArgs: String?,
    val breadcrumbs: List<String>,
) {
    companion object {
        const val SEVERITY_FATAL = "fatal"
        const val SEVERITY_NON_FATAL = "non_fatal"
    }
}
