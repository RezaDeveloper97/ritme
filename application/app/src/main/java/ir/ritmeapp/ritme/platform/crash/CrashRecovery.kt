package ir.ritmeapp.ritme.platform.crash

import android.content.Intent
import java.io.File

/**
 * Shared keys + on-disk flag handling for the crash → relaunch → recover round-trip
 * (CLAUDE.md §7.3). Two recovery signals are supported: the relaunch [Intent] extras (the
 * normal path) and a tiny flag file (fallback when the OS restarts us without our intent).
 */
object CrashRecovery {

    const val EXTRA_RECOVERED = "recovered_from_crash"
    const val EXTRA_SAFE_ROUTE = "recovery_safe_route"
    const val EXTRA_SAFE_ARGS = "recovery_safe_args"

    private const val REPORTS_DIR_NAME = "crash_reports"
    private const val FLAG_FILE_NAME = "needs_recovery.flag"

    /** The directory crash reports are written to / uploaded from. */
    fun reportsDir(filesDir: File): File = File(filesDir, REPORTS_DIR_NAME)

    /** Best-effort write of the recovery flag from inside the dying process. */
    fun writeFlag(filesDir: File, route: String?, args: String?) {
        try {
            File(filesDir, FLAG_FILE_NAME).writeText("${route.orEmpty()}\n${args.orEmpty()}")
        } catch (_: Throwable) {
            // Nothing safe to do while crashing; the intent extra is the primary path anyway.
        }
    }

    fun clearFlag(filesDir: File) {
        try {
            File(filesDir, FLAG_FILE_NAME).delete()
        } catch (_: Throwable) {
        }
    }

    /** Combines the intent extras and the flag file into a single [RecoveryState]. */
    fun resolve(intent: Intent?, filesDir: File): RecoveryState {
        if (intent?.getBooleanExtra(EXTRA_RECOVERED, false) == true) {
            return RecoveryState(
                recovered = true,
                route = intent.getStringExtra(EXTRA_SAFE_ROUTE),
                args = intent.getStringExtra(EXTRA_SAFE_ARGS),
            )
        }
        return readFlag(filesDir) ?: RecoveryState.None
    }

    private fun readFlag(filesDir: File): RecoveryState? = try {
        val file = File(filesDir, FLAG_FILE_NAME)
        if (!file.exists()) {
            null
        } else {
            val lines = file.readText().split('\n')
            val route = lines.getOrNull(0)?.takeIf { it.isNotEmpty() }
            val args = lines.getOrNull(1)?.takeIf { it.isNotEmpty() }
            RecoveryState(recovered = true, route = route, args = args)
        }
    } catch (_: Throwable) {
        null
    }
}
