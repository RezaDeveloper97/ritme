package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.InfoTopic
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** One heading + body pair of a static info page. */
private data class InfoSection(@field:StringRes val heading: Int, @field:StringRes val body: Int)

private val PRIVACY_SECTIONS = listOf(
    InfoSection(R.string.info_privacy_h1, R.string.info_privacy_b1),
    InfoSection(R.string.info_privacy_h2, R.string.info_privacy_b2),
    InfoSection(R.string.info_privacy_h3, R.string.info_privacy_b3),
    InfoSection(R.string.info_privacy_h4, R.string.info_privacy_b4),
)

private val TERMS_SECTIONS = listOf(
    InfoSection(R.string.info_terms_h1, R.string.info_terms_b1),
    InfoSection(R.string.info_terms_h2, R.string.info_terms_b2),
    InfoSection(R.string.info_terms_h3, R.string.info_terms_b3),
    InfoSection(R.string.info_terms_h4, R.string.info_terms_b4),
)

private val ABOUT_SECTIONS = listOf(
    InfoSection(R.string.info_about_h1, R.string.info_about_b1),
    InfoSection(R.string.info_about_h2, R.string.info_about_b2),
    InfoSection(R.string.info_about_h3, R.string.info_about_b3),
)

private val HELP_SECTIONS = listOf(
    InfoSection(R.string.info_help_h1, R.string.info_help_b1),
    InfoSection(R.string.info_help_h2, R.string.info_help_b2),
    InfoSection(R.string.info_help_h3, R.string.info_help_b3),
    InfoSection(R.string.info_help_h4, R.string.info_help_b4),
    InfoSection(R.string.info_help_h5, R.string.info_help_b5),
)

/**
 * The static informational pages (web `/profile/info/[topic]`): privacy, terms, about, help —
 * localized copy rendered as heading + body cards.
 */
@Composable
fun InfoScreen(
    topic: InfoTopic,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:info:${topic.name.lowercase()}")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfileInfo.ROUTE, topic.name, System.currentTimeMillis()),
        )
    }

    val title = when (topic) {
        InfoTopic.PRIVACY -> R.string.info_privacy_title
        InfoTopic.TERMS -> R.string.info_terms_title
        InfoTopic.ABOUT -> R.string.info_about_title
        InfoTopic.HELP -> R.string.info_help_title
    }
    val sections = when (topic) {
        InfoTopic.PRIVACY -> PRIVACY_SECTIONS
        InfoTopic.TERMS -> TERMS_SECTIONS
        InfoTopic.ABOUT -> ABOUT_SECTIONS
        InfoTopic.HELP -> HELP_SECTIONS
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = { ScreenHeader(title = stringResource(title), onBack = onBack) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(start = 18.dp, end = 18.dp, top = 4.dp, bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            items(sections, key = { it.heading }) { section ->
                SurfaceCard {
                    Text(
                        text = stringResource(section.heading),
                        style = MaterialTheme.typography.titleSmall.copy(fontSize = 14.5.sp),
                        color = colors.ink,
                        fontWeight = FontWeight.ExtraBold,
                    )
                    Spacer(Modifier.height(8.dp))
                    // Airy paragraph rhythm to match the web's lineHeight:2 on 13.5px body text.
                    Text(
                        text = stringResource(section.body),
                        style = MaterialTheme.typography.bodyMedium.copy(fontSize = 13.5.sp, lineHeight = 27.sp),
                        color = colors.inkMuted,
                    )
                }
            }
        }
    }
}
