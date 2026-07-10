package ir.ritmeapp.ritme.adapter.outbound.persistence

import android.content.Context
import ir.ritmeapp.ritme.domain.port.outbound.AppPreferences
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * [AppPreferences] backed by a private [android.content.SharedPreferences] file, separate from the
 * auth store so sign-out never clears it. Reads/writes run on [ioDispatcher] to keep callers off
 * the main thread (§5).
 */
class SharedPrefsAppPreferences(
    context: Context,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : AppPreferences {

    private val prefs = context.getSharedPreferences(FILE_NAME, Context.MODE_PRIVATE)

    override suspend fun isIntroSeen(): Boolean = withContext(ioDispatcher) {
        prefs.getBoolean(KEY_INTRO_SEEN, false)
    }

    override suspend fun markIntroSeen() = withContext(ioDispatcher) {
        prefs.edit().putBoolean(KEY_INTRO_SEEN, true).apply()
    }

    override suspend fun lastTrackingMode(): String? = withContext(ioDispatcher) {
        prefs.getString(KEY_TRACKING_MODE, null)
    }

    override suspend fun saveTrackingMode(apiValue: String) = withContext(ioDispatcher) {
        prefs.edit().putString(KEY_TRACKING_MODE, apiValue).apply()
    }

    private companion object {
        const val FILE_NAME = "ritme_prefs"
        const val KEY_INTRO_SEEN = "intro_seen"
        const val KEY_TRACKING_MODE = "tracking_mode"
    }
}
