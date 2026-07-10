package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.port.inbound.DeleteAccountUseCase
import ir.ritmeapp.ritme.domain.port.outbound.SessionGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore

/**
 * Default [DeleteAccountUseCase]. Unlike logout, the server call must succeed first — the
 * local session is only wiped once the account is actually gone.
 */
class DeleteAccountInteractor(
    private val sessionGateway: SessionGateway,
    private val tokenStore: TokenStore,
) : DeleteAccountUseCase {

    override suspend fun invoke(): AppResult<Unit> =
        when (val result = sessionGateway.deleteAccount()) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                tokenStore.clear()
                result
            }
        }
}
