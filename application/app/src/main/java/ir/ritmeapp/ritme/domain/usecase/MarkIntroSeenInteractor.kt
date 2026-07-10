package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.port.inbound.MarkIntroSeenUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AppPreferences

/**
 * Persists that the welcome intro has been seen so it never gates a returning visitor again.
 */
class MarkIntroSeenInteractor(
    private val appPreferences: AppPreferences,
) : MarkIntroSeenUseCase {

    override suspend fun invoke() = appPreferences.markIntroSeen()
}
