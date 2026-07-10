package ir.ritmeapp.ritme.domain.port.outbound

/**
 * Outbound port for small, non-sensitive app flags that outlive a launch (currently just whether
 * the first-run intro has been seen). Kept separate from [TokenStore] so session credentials and
 * plain UI flags don't share a lifecycle — signing out must not forget that the intro was shown.
 */
interface AppPreferences {
    /** True once the user has finished or skipped the welcome intro at least once. */
    suspend fun isIntroSeen(): Boolean

    /** Remembers that the intro was shown, so it never gates a returning visitor again. */
    suspend fun markIntroSeen()

    /** The last tracking mode seen from the backend (`cycle`/`pregnancy`), or null if never fetched. */
    suspend fun lastTrackingMode(): String?

    /** Caches the tracking mode so the next launch renders the right navigation instantly. */
    suspend fun saveTrackingMode(apiValue: String)
}
