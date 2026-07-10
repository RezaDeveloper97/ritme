package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.UserProfile
import ir.ritmeapp.ritme.domain.port.inbound.GetUserProfileUseCase
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway

/** Default [GetUserProfileUseCase]. */
class GetUserProfileInteractor(
    private val profileGateway: ProfileGateway,
) : GetUserProfileUseCase {

    override suspend fun invoke(): AppResult<UserProfile> = profileGateway.fetchProfile()
}
