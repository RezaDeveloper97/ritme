package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult

/**
 * Inbound port: end the current session. Always succeeds locally — the token is cleared even
 * when the revoke call can't reach the server, so the user is never trapped signed-in.
 */
interface LogoutUseCase {
    suspend operator fun invoke(): AppResult<Unit>
}
