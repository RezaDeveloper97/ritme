package ir.ritmeapp.ritme.application.service

import ir.ritmeapp.ritme.domain.port.outbound.DiagnosticsReportUploader
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * Coordinates deferred, non-UI work at app start (CLAUDE.md §5 — keep cold start minimal).
 * Today it kicks off draining any queued crash reports off the main thread; future startup
 * chores (session refresh, catalog prefetch) belong here too. Depends only on outbound
 * ports, never on concrete adapters.
 */
class AppStartupCoordinator(
    private val diagnosticsReportUploader: DiagnosticsReportUploader,
    private val scope: CoroutineScope,
) {
    fun onAppStart() {
        scope.launch { diagnosticsReportUploader.uploadPending() }
    }
}
