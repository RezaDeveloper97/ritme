package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.UserProfile

/**
 * Inbound port: read the account + health profile + BMI the profile screens render.
 */
interface GetUserProfileUseCase {
    suspend operator fun invoke(): AppResult<UserProfile>
}
