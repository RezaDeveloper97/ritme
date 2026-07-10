package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult

/**
 * Inbound port: fetch the account's full data export (raw JSON text) so the user can keep a
 * copy of everything Ritme stores about them.
 */
interface ExportDataUseCase {
    suspend operator fun invoke(): AppResult<String>
}
