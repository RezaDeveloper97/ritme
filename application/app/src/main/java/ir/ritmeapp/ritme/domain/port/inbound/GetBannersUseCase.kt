package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerSlot

/**
 * Inbound port: the admin-scheduled Home banners, grouped by placement slot.
 */
interface GetBannersUseCase {
    suspend operator fun invoke(): AppResult<Map<BannerSlot, List<Banner>>>
}
