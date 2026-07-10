package ir.ritmeapp.ritme.domain.model

/**
 * One admin-scheduled promotional banner from `GET /banners`. [linkUrl] is optional; when
 * present, [linkType] says whether it opens inside the app (a route) or in the browser.
 */
data class Banner(
    val id: Long,
    val title: String,
    val imageUrl: String,
    val slot: BannerSlot,
    val linkUrl: String?,
    val linkType: BannerLinkType?,
)

/** The three Home-screen placements the admin panel can schedule banners into. */
enum class BannerSlot(val apiValue: String) {
    HOME_TOP("home_top"),
    HOME_MIDDLE("home_middle"),
    HOME_BOTTOM("home_bottom");

    companion object {
        fun fromApi(value: String?): BannerSlot? = entries.firstOrNull { it.apiValue == value }
    }
}

/** Where a tapped banner leads: an in-app destination or an external browser page. */
enum class BannerLinkType(val apiValue: String) {
    INTERNAL("internal"),
    EXTERNAL("external");

    companion object {
        fun fromApi(value: String?): BannerLinkType? = entries.firstOrNull { it.apiValue == value }
    }
}
