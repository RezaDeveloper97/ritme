package ir.ritmeapp.ritme.platform.crash

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.Process
import ir.ritmeapp.ritme.domain.model.SafeScreen
import java.io.PrintWriter
import java.io.StringWriter
import kotlin.system.exitProcess

/**
 * The fatal-error defense (CLAUDE.md §7.3). Installs a process-wide uncaught-exception
 * handler that, **fast and synchronously** (the process is dying — no coroutines, no
 * network), captures a [CrashPayload], writes it locally, drops a recovery flag, schedules
 * a relaunch of the entry activity, then kills the process. If our own capture fails, we
 * fall back to the platform's previous handler so behaviour is never worse than default.
 */
class CrashGuard(
    private val appContext: Context,
    private val store: CrashReportStore,
    private val currentSafeScreen: () -> SafeScreen?,
    private val appVersionName: String,
    private val appVersionCode: Long,
    private val entryActivityClass: Class<*>,
) {

    fun install() {
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            val handled = try {
                captureAndScheduleRestart(thread, throwable)
                true
            } catch (_: Throwable) {
                false
            }
            if (handled) {
                Process.killProcess(Process.myPid())
                exitProcess(0)
            } else {
                previous?.uncaughtException(thread, throwable)
            }
        }
    }

    private fun captureAndScheduleRestart(thread: Thread, throwable: Throwable) {
        val safe = currentSafeScreen()
        val payload = CrashPayload(
            timestampMillis = System.currentTimeMillis(),
            severity = CrashPayload.SEVERITY_FATAL,
            appVersionName = appVersionName,
            appVersionCode = appVersionCode,
            sdkInt = Build.VERSION.SDK_INT,
            osVersion = Build.VERSION.RELEASE ?: "unknown",
            manufacturer = Build.MANUFACTURER ?: "unknown",
            deviceModel = Build.MODEL ?: "unknown",
            freeMemoryBytes = Runtime.getRuntime().freeMemory(),
            threadName = thread.name,
            stackTrace = stackTraceOf(throwable),
            safeRoute = safe?.route,
            safeArgs = safe?.args,
            breadcrumbs = Breadcrumbs.snapshot().map { "${it.timestampMillis} ${it.message}" },
        )
        store.writeReport(payload)
        CrashRecovery.writeFlag(appContext.filesDir, safe?.route, safe?.args)
        scheduleRestart(safe)
    }

    private fun scheduleRestart(safe: SafeScreen?) {
        val intent = Intent(appContext, entryActivityClass).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK)
            putExtra(CrashRecovery.EXTRA_RECOVERED, true)
            putExtra(CrashRecovery.EXTRA_SAFE_ROUTE, safe?.route)
            putExtra(CrashRecovery.EXTRA_SAFE_ARGS, safe?.args)
        }
        val pending = PendingIntent.getActivity(
            appContext,
            RESTART_REQUEST_CODE,
            intent,
            PendingIntent.FLAG_ONE_SHOT or PendingIntent.FLAG_IMMUTABLE,
        )
        val alarm = appContext.getSystemService(Context.ALARM_SERVICE) as? AlarmManager
        alarm?.set(AlarmManager.RTC, System.currentTimeMillis() + RESTART_DELAY_MS, pending)
    }

    private fun stackTraceOf(throwable: Throwable): String {
        val writer = StringWriter()
        throwable.printStackTrace(PrintWriter(writer))
        return writer.toString()
    }

    private companion object {
        const val RESTART_REQUEST_CODE = 0x0DA1
        const val RESTART_DELAY_MS = 300L
    }
}
