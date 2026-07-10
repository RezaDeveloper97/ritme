package ir.ritmeapp.ritme.platform.log

import android.util.Log

/**
 * ~20-line logging facade over [android.util.Log] (CLAUDE.md §3 — no Timber).
 * Debug/info/verbose are silenced unless [debugEnabled] is set; warnings and errors
 * always pass through. [RitmeApplication] flips [debugEnabled] from `BuildConfig.DEBUG`
 * at startup so release builds stay quiet.
 */
object RitmeLog {

    @Volatile
    var debugEnabled: Boolean = true

    fun d(tag: String, message: String) {
        if (debugEnabled) Log.d(tag, message)
    }

    fun i(tag: String, message: String) {
        if (debugEnabled) Log.i(tag, message)
    }

    fun w(tag: String, message: String, throwable: Throwable? = null) {
        Log.w(tag, message, throwable)
    }

    fun e(tag: String, message: String, throwable: Throwable? = null) {
        Log.e(tag, message, throwable)
    }
}
