package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AuthTokens

/**
 * Outbound port for persisting the session's [AuthTokens] across app launches. The domain
 * states the need; the adapter decides the mechanism (private storage, keystore, …). Kept
 * deliberately tiny (CLAUDE.md §4 — Interface Segregation) so a fake is trivial to write.
 */
interface TokenStore {
    /** Persists [tokens], replacing any previously stored pair. */
    suspend fun save(tokens: AuthTokens)

    /** The stored tokens, or `null` if the user has never signed in / has signed out. */
    suspend fun load(): AuthTokens?

    /** Wipes any stored tokens (sign-out). */
    suspend fun clear()
}
