package ir.ritmeapp.ritme.platform.crash

import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.port.outbound.SafeStateRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * Keeps the last safe screen both **in memory** (read synchronously by the Crash Guard
 * while the process is dying) and **persisted** (survives an OS-initiated restart). The UI
 * calls [record] from each screen's success `LaunchedEffect`; the durable write is fired
 * off the main thread and never blocks rendering (CLAUDE.md §7.2).
 */
class SafeScreenTracker(
    private val repository: SafeStateRepository,
    private val scope: CoroutineScope,
) {

    @Volatile
    var current: SafeScreen? = null
        private set

    fun record(screen: SafeScreen) {
        current = screen
        scope.launch { repository.save(screen) }
    }
}
