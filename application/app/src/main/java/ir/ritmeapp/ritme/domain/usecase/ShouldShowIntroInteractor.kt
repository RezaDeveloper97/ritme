package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.port.inbound.ShouldShowIntroUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AppPreferences

/**
 * Default first-run intro policy: show the welcome carousel until it has been seen once.
 */
class ShouldShowIntroInteractor(
    private val appPreferences: AppPreferences,
) : ShouldShowIntroUseCase {

    override suspend fun invoke(): Boolean = !appPreferences.isIntroSeen()
}
