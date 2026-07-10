package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.SafeScreen

/**
 * Outbound port for persisting the "last known safe screen" used by crash recovery
 * (CLAUDE.md §7.2). Implementations are best-effort and must never throw across this
 * boundary — a failure to record state must not itself destabilize the app.
 */
interface SafeStateRepository {
    /** Records [screen] as the most recent successful render. Cheap, debounced by the caller. */
    suspend fun save(screen: SafeScreen)

    /** The most recently saved [SafeScreen], or `null` if none has been recorded yet. */
    suspend fun lastSafeScreen(): SafeScreen?
}
