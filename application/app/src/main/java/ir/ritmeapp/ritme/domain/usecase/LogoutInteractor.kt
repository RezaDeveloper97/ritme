package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.inbound.LogoutUseCase
import ir.ritmeapp.ritme.domain.port.outbound.SessionGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore

/**
 * Default [LogoutUseCase]. Best-effort server revoke, then an unconditional local wipe — a
 * dead network must never trap the user in a signed-in state, so this always succeeds.
 */
class LogoutInteractor(
    private val sessionGateway: SessionGateway,
    private val tokenStore: TokenStore,
) : LogoutUseCase {

    override suspend fun invoke(): AppResult<Unit> {
        sessionGateway.logout() // Result deliberately ignored: local sign-out happens regardless.
        tokenStore.clear()
        return AppResult.Success(Unit)
    }
}
