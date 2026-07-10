package ir.ritmeapp.ritme.platform.crash

/**
 * Whether this launch is a recovery from a previous crash, and where to land if so
 * (CLAUDE.md §7.3). Resolved once at entry by [CrashRecovery.resolve].
 */
data class RecoveryState(
    val recovered: Boolean,
    val route: String?,
    val args: String?,
) {
    companion object {
        val None = RecoveryState(recovered = false, route = null, args = null)
    }
}
