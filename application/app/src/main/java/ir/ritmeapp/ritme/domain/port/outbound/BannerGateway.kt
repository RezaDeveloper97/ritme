package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerSlot

/**
 * Outbound port for the admin-scheduled Home banners (`GET /banners`). Returns every active
 * banner grouped by placement; the Home screen picks the slots it renders.
 */
interface BannerGateway {
    suspend fun activeBanners(): AppResult<Map<BannerSlot, List<Banner>>>
}
