package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerSlot
import ir.ritmeapp.ritme.domain.port.inbound.GetBannersUseCase
import ir.ritmeapp.ritme.domain.port.outbound.BannerGateway

/** Default [GetBannersUseCase]. */
class GetBannersInteractor(
    private val bannerGateway: BannerGateway,
) : GetBannersUseCase {

    override suspend fun invoke(): AppResult<Map<BannerSlot, List<Banner>>> = bannerGateway.activeBanners()
}
