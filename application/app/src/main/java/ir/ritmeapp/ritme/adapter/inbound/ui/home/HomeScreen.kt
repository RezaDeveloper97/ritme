package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.res.stringResource
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
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** Length of an ISO `yyyy-MM-dd` prefix, used to match a reminder's scheduled day to today. */
private const val ISO_DATE_LENGTH = 10

/**
 * The post-login Home dashboard, mirroring the web layout: a header, the signature calendar + cycle
 * status hero, banner slots, the two-tap start-period action, phase rows, personalized
 * recommendations + smart tip, today's tasks, and the static challenge/reminder/summary blocks,
 * over a soft brand gradient with the mode-aware tab bar pinned below. The long feed is a
 * [LazyColumn] keyed per section (§5). Records the last-safe-screen on entry (§7.2); being the
 * post-auth root it is not swipe-back-wrapped.
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
                .background(Brush.verticalGradient(listOf(colors.pinkContainer, colors.background))),
        ) {
            when {
                state.loading && state.dashboard == null -> Loading(colors)
                state.isError && state.dashboard == null -> ErrorState(state.errorMessage, viewModel::onRetry, colors)
                else -> Content(state, viewModel, onNavigate)
            }
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

    // Today's reminders/to-dos, split into the check-off list and the doctor/medication cards.
    // Same `/reminders` source as the daily-log day planner, so items set there surface here.
    val todayReminders = remember(state.reminders, today) {
        state.reminders.filter { it.scheduledAt?.take(ISO_DATE_LENGTH) == today.toIso() }
    }
    val todayTodos = remember(todayReminders) { todayReminders.filter { it.type == ReminderType.CUSTOM } }
    val careReminders = remember(todayReminders) {
        todayReminders.filter { it.type == ReminderType.DOCTOR || it.type == ReminderType.MEDICATION }
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        item(key = "header") { Header(state.greetingName, colors) }
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
        if (predictions != null) {
            item(key = "phases") { PhaseRowsSection(predictions, today) }
        }
        item(key = "reco") { RecommendationsSection(message) }
        bannerItem(state, BannerSlot.HOME_MIDDLE, onNavigate)
        item(key = "tasks") { TodayTasksSection(todayTodos, viewModel::onToggleTask) }
        item(key = "challenge") { ChallengeSection() }
        if (careReminders.isNotEmpty()) {
            item(key = "reminders") { ReminderCardsSection(careReminders) }
        }
        item(key = "smarttip") { SmartTipSection(message) }
        item(key = "weeksummary") { WeekSummarySection() }
        item(key = "todaystatus") { TodayStatusSection() }
        item(key = "articles") { ArticlesSection() }
        if (predictions != null) {
            item(key = "mycycles") { MyCyclesSection(predictions, today) }
        }
        bannerItem(state, BannerSlot.HOME_BOTTOM, onNavigate)
        if (predictions != null) {
            item(key = "cyclesummary") { CycleSummarySection(predictions) }
        }
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

@Composable
private fun Header(greetingName: String?, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
        Text("ریتمی", style = MaterialTheme.typography.headlineSmall, color = colors.pink)
        Spacer(Modifier.height(2.dp))
        Text(stringResource(R.string.home_tagline), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(10.dp))
        val greeting = greetingName
            ?.let { stringResource(R.string.home_greeting_named, it) }
            ?: stringResource(R.string.home_greeting)
        Text(greeting, style = MaterialTheme.typography.titleMedium, color = colors.ink)
    }
}

@Composable
private fun Loading(colors: RitmeColors) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(color = colors.pink)
    }
}

@Composable
private fun ErrorState(message: String?, onRetry: () -> Unit, colors: RitmeColors) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = message ?: stringResource(R.string.home_error),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = onRetry,
            colors = ButtonDefaults.buttonColors(containerColor = colors.pink, contentColor = colors.onPink),
        ) {
            Text(stringResource(R.string.home_retry))
        }
    }
}
