package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.port.inbound.IsLoggedInUseCase
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore

/**
 * Default [IsLoggedInUseCase]. A session exists when the [TokenStore] holds an access token —
 * the deliberate seam where token-expiry validation will later live without touching the UI.
 */
class SessionStatusInteractor(
    private val tokenStore: TokenStore,
) : IsLoggedInUseCase {

    override suspend fun invoke(): Boolean = tokenStore.load() != null
}
