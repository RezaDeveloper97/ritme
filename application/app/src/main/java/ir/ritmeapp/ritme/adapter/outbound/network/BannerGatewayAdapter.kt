package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerLinkType
import ir.ritmeapp.ritme.domain.model.BannerSlot
import ir.ritmeapp.ritme.domain.port.outbound.BannerGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONObject

/**
 * Network [BannerGateway] over `GET /banners`. The backend groups banners under
 * `data.positions.{home_top,home_middle,home_bottom}`; unknown slots and malformed entries
 * are skipped so one bad banner never blanks the Home screen.
 */
class BannerGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : BannerGateway {

    override suspend fun activeBanners(): AppResult<Map<BannerSlot, List<Banner>>> {
        Breadcrumbs.add("api:banners:start")
        return when (val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.BANNERS_PATH)) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:banners:http_${result.value.statusCode}")
                parse(result.value.statusCode, result.value.body)
            }
        }
    }

    private fun parse(statusCode: Int, body: String): AppResult<Map<BannerSlot, List<Banner>>> = try {
        val root = if (body.isBlank()) JSONObject() else JSONObject(body)
        if (!root.optBoolean("success", false)) {
            val message = root.stringOrNull("message") ?: "Banner request rejected"
            AppResult.Failure(AppError.Http(statusCode, message, body))
        } else {
            val positions = root.optJSONObject("data")?.optJSONObject("positions") ?: JSONObject()
            val grouped = BannerSlot.entries.associateWith { slot ->
                val array = positions.optJSONArray(slot.apiValue)
                (0 until (array?.length() ?: 0)).mapNotNull { index ->
                    array?.optJSONObject(index)?.let { readBanner(it, slot) }
                }
            }
            AppResult.Success(grouped)
        }
    } catch (e: Exception) {
        AppResult.Failure(AppError.Parsing("Malformed banners response ($statusCode)", e))
    }

    private fun readBanner(json: JSONObject, slot: BannerSlot): Banner? {
        val imageUrl = json.stringOrNull("image_url") ?: return null
        return Banner(
            id = json.optLong("id"),
            title = json.stringOrNull("title").orEmpty(),
            imageUrl = imageUrl,
            slot = slot,
            linkUrl = json.stringOrNull("link_url"),
            linkType = BannerLinkType.fromApi(json.stringOrNull("link_type")),
        )
    }
}
