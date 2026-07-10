package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult

/**
 * Outbound port for ending a session or an account (`POST /auth/logout`, `DELETE /account`).
 * Kept separate from [AuthGateway] (login) so the login flow and the destructive settings
 * actions don't share one interface (§4 I — small, role-specific ports).
 */
interface SessionGateway {

    /** Revokes the current access token server-side. */
    suspend fun logout(): AppResult<Unit>

    /** Irreversibly deletes the account and every record it owns. */
    suspend fun deleteAccount(): AppResult<Unit>

    /** The full account data export as raw JSON text (`GET /profile/export`). */
    suspend fun exportData(): AppResult<String>
}
