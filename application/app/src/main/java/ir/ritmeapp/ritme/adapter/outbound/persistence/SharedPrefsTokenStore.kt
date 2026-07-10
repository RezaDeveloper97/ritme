package ir.ritmeapp.ritme.adapter.outbound.persistence

import android.content.Context
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * [TokenStore] backed by a private [android.content.SharedPreferences] file. All access
 * runs on [ioDispatcher] so callers stay off the main thread (CLAUDE.md §5). The file lives
 * in app-private storage; hardening to the Android Keystore is a future adapter-only change
 * that leaves the port and every caller untouched.
 */
class SharedPrefsTokenStore(
    context: Context,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : TokenStore {

    private val prefs = context.getSharedPreferences(FILE_NAME, Context.MODE_PRIVATE)

    override suspend fun save(tokens: AuthTokens) = withContext(ioDispatcher) {
        prefs.edit()
            .putString(KEY_ACCESS, tokens.accessToken)
            .apply()
    }

    override suspend fun load(): AuthTokens? = withContext(ioDispatcher) {
        val access = prefs.getString(KEY_ACCESS, null)
        if (access.isNullOrBlank()) null else AuthTokens(accessToken = access)
    }

    override suspend fun clear() = withContext(ioDispatcher) {
        prefs.edit().clear().apply()
    }

    private companion object {
        const val FILE_NAME = "ritme_auth"
        const val KEY_ACCESS = "access_token"
    }
}
