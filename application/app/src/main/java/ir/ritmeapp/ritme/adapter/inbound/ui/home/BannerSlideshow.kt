package ir.ritmeapp.ritme.adapter.inbound.ui.home

import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.NetworkImage
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.Banner
import ir.ritmeapp.ritme.domain.model.BannerLinkType
import ir.ritmeapp.ritme.platform.log.RitmeLog
import kotlinx.coroutines.delay

private const val AUTOPLAY_INTERVAL_MS = 5_000L

/**
 * The admin-scheduled banner carousel (web `banner-slideshow` widget): swipeable 2:1 images
 * with dots and a five-second autoplay that pauses while the user is dragging. Renders nothing
 * when the slot is empty. Internal links map onto app destinations; external ones open the browser.
 */
@Composable
fun BannerSlideshow(
    banners: List<Banner>,
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (banners.isEmpty()) return
    val colors = LocalRitmeColors.current
    val context = LocalContext.current
    val pagerState = rememberPagerState(pageCount = { banners.size })

    LaunchedEffect(banners.size) {
        if (banners.size < 2) return@LaunchedEffect
        while (true) {
            delay(AUTOPLAY_INTERVAL_MS)
            if (!pagerState.isScrollInProgress) {
                pagerState.animateScrollToPage((pagerState.currentPage + 1) % banners.size)
            }
        }
    }

    Column(modifier.fillMaxWidth()) {
        HorizontalPager(
            state = pagerState,
            modifier = Modifier.fillMaxWidth(),
        ) { page ->
            val banner = banners[page]
            Box(
                Modifier
                    .fillMaxWidth()
                    .aspectRatio(2f)
                    .clip(RoundedCornerShape(12.dp))
                    .clickable(enabled = banner.linkUrl != null) { openBanner(banner, onNavigate, context) },
            ) {
                NetworkImage(
                    url = banner.imageUrl,
                    contentDescription = banner.title.ifBlank { stringResource(R.string.banners_region) },
                    modifier = Modifier.fillMaxWidth().aspectRatio(2f),
                )
            }
        }
        if (banners.size > 1) {
            Spacer(Modifier.height(6.dp))
            Row(Modifier.align(Alignment.CenterHorizontally)) {
                repeat(banners.size) { index ->
                    Box(
                        Modifier
                            .padding(horizontal = 3.dp)
                            .size(if (index == pagerState.currentPage) 8.dp else 6.dp)
                            .clip(CircleShape)
                            .background(if (index == pagerState.currentPage) colors.pink else colors.outline),
                    )
                }
            }
        }
    }
}

/** Internal links jump to the matching tab; external ones hand off to the browser. */
private fun openBanner(banner: Banner, onNavigate: (Destination) -> Unit, context: android.content.Context) {
    val url = banner.linkUrl ?: return
    if (banner.linkType == BannerLinkType.EXTERNAL) {
        try {
            context.startActivity(Intent(Intent.ACTION_VIEW, url.toUri()))
        } catch (e: Exception) {
            RitmeLog.w("BannerSlideshow", "No handler for banner link")
        }
        return
    }
    onNavigate(destinationForInternalLink(url))
}

/** Best-effort mapping of a web route path to the equivalent app destination. */
private fun destinationForInternalLink(url: String): Destination = when {
    url.contains("pregnancy") -> Destination.Pregnancy
    url.contains("calendar") -> Destination.Calendar
    url.contains("cycle") -> Destination.Cycle
    url.contains("log") -> Destination.Log()
    url.contains("profile") -> Destination.Profile
    else -> Destination.Home
}
