package ir.ritmeapp.ritme.domain.port.outbound

/**
 * Outbound port for draining queued crash/error reports to the backend (CLAUDE.md §7.4).
 * Defined as an abstraction so the application layer can trigger uploads without depending
 * on the concrete network adapter, and so tests can substitute a fake.
 */
interface DiagnosticsReportUploader {
    /** Uploads all pending reports; deletes each only after a successful (2xx) response. */
    suspend fun uploadPending()
}
