package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.BannerSlot
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** Length of an ISO `yyyy-MM-dd` prefix, used to match a reminder's scheduled day to today. */
private const val ISO_DATE_LENGTH = 10

/**
 * The post-login Home dashboard, mirroring the web layout: a bare-icon header, the signature
 * calendar + cycle-status hero, banner slots, the two-tap start-period action, per-phase rows, the
 * interactive day planner, and the recommendation/challenge/summary blocks, over a pink→mint
 * gradient with the mode-aware tab bar pinned below. The long feed is a [LazyColumn] keyed per
 * section (§5). To match the web it never shows a full-screen spinner/error: the layout always
 * renders with placeholder dashes until data arrives. Records the last-safe-screen on entry (§7.2);
 * being the post-auth root it is not swipe-back-wrapped.
 */
@Composable
fun HomeScreen(
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: HomeViewModel = viewModel(factory = container.homeViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:home:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Home.route, null, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = { RitmeBottomBar(active = RitmeTab.TODAY, mode = state.mode, onNavigate = onNavigate) },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .background(Brush.verticalGradient(listOf(colors.homeGradStart, colors.homeGradEnd))),
        ) {
            Content(state, viewModel, onNavigate)
        }
    }
}

@Composable
private fun Content(state: HomeUiState, viewModel: HomeViewModel, onNavigate: (Destination) -> Unit) {
    val colors = LocalRitmeColors.current
    val today = remember { todayJalali() }
    val dashboard = state.dashboard
    val predictions = dashboard?.predictions
    val message = dashboard?.message

    // Today's reminders/to-dos for the day planner. Same `/reminders` source as the daily-log
    // planner, so items set on either surface surface here.
    val todayReminders = remember(state.reminders, today) {
        state.reminders.filter { it.scheduledAt?.take(ISO_DATE_LENGTH) == today.toIso() }
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item(key = "header") { Header(colors) }
        if (dashboard?.calculation?.isRecalculating == true) {
            item(key = "updating") {
                Text(
                    stringResource(R.string.home_updating),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
        }
        item(key = "hero") { HomeCalendarCard(today = today, predictions = predictions, message = message) }
        bannerItem(state, BannerSlot.HOME_TOP, onNavigate)
        item(key = "startperiod") {
            StartPeriodSection(state.startPeriod, onTap = viewModel::onStartPeriodTapped)
        }
        item(key = "phases") { PhaseRowsSection(predictions, today) }
        item(key = "reco") { RecommendationsSection(message) }
        bannerItem(state, BannerSlot.HOME_MIDDLE, onNavigate)
        item(key = "daytasks") {
            DayTasksSection(
                items = todayReminders,
                onAdd = viewModel::onAddTask,
                onToggle = viewModel::onToggleTask,
                onDelete = viewModel::onDeleteTask,
            )
        }
        item(key = "challenge") { ChallengeSection() }
        item(key = "smarttip") { SmartTipSection(message) }
        item(key = "weeksummary") { WeekSummarySection(predictions) }
        item(key = "todaystatus") { TodayStatusSection(predictions, message) }
        item(key = "articles") { ArticlesSection() }
        item(key = "mycycles") { MyCyclesSection(predictions, today) }
        bannerItem(state, BannerSlot.HOME_BOTTOM, onNavigate)
        item(key = "cyclesummary") { CycleSummarySection(predictions, dashboard?.calculation?.variability) }
        item(key = "tail") { Spacer(Modifier.height(8.dp)) }
    }
}

/** Adds one banner slot to the feed when the slot has banners. */
private fun LazyListScope.bannerItem(state: HomeUiState, slot: BannerSlot, onNavigate: (Destination) -> Unit) {
    val banners = state.banners[slot].orEmpty()
    if (banners.isEmpty()) return
    item(key = "banner_${slot.apiValue}") {
        BannerSlideshow(banners = banners, onNavigate = onNavigate)
    }
}

/** Web `HomeHeader`: a start-side sparkle, the centered elongated brand + tagline, an end-side bell. */
@Composable
private fun Header(colors: RitmeColors) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(Modifier.size(36.dp), contentAlignment = Alignment.Center) {
            Icon(painterResource(R.drawable.ic_sparkle), null, tint = colors.pink, modifier = Modifier.size(22.dp))
        }
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(stringResource(R.string.home_header_title), style = MaterialTheme.typography.titleLarge, color = colors.pink, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(2.dp))
            Text(stringResource(R.string.home_header_tagline), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        }
        Box(Modifier.size(36.dp), contentAlignment = Alignment.Center) {
            Icon(painterResource(R.drawable.ic_bell), null, tint = colors.inkMuted, modifier = Modifier.size(21.dp))
        }
    }
}
