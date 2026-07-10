package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatFull
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.PregnancyLogTab
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyAlertLevel
import ir.ritmeapp.ritme.domain.model.PregnancyConfidence
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The pregnancy tracker tab (web `/pregnancy`): the activation gate for accounts not yet in
 * pregnancy mode, and — once tracking — the gestational hero, due-date + confidence cards,
 * quick log actions, the 1..40 week content browser, and the safety alerts feed.
 */
@Composable
fun PregnancyScreen(
    onNavigate: (Destination) -> Unit,
    onOpenLog: (PregnancyLogTab) -> Unit,
    onStartOnboarding: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: PregnancyViewModel = viewModel(factory = container.pregnancyViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:pregnancy:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Pregnancy.route, null, System.currentTimeMillis()),
        )
        viewModel.effects.collect { effect ->
            when (effect) {
                PregnancyEffect.GoToOnboarding -> onStartOnboarding()
            }
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = {
            RitmeBottomBar(active = RitmeTab.PREGNANCY, mode = TrackingMode.PREGNANCY, onNavigate = onNavigate)
        },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .background(Brush.verticalGradient(0f to colors.pinkContainer, 0.35f to colors.background)),
        ) {
            when {
                state.loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = colors.pink)
                }

                state.isError -> ErrorState({ viewModel.onIntent(PregnancyIntent.Retry) }, colors)

                state.needsOnboarding -> ActivateGate(state, viewModel::onIntent, colors)

                else -> Tracker(state, viewModel::onIntent, onOpenLog, colors)
            }
        }
    }
}

@Composable
private fun ActivateGate(state: PregnancyUiState, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            Modifier.size(72.dp).clip(CircleShape).background(colors.pinkContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_heart),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(32.dp),
            )
        }
        Spacer(Modifier.height(16.dp))
        Text(
            stringResource(R.string.preg_not_active_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.preg_not_active_body),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(20.dp))
        Button(
            onClick = { onIntent(PregnancyIntent.Activate) },
            enabled = !state.activating,
            colors = ButtonDefaults.buttonColors(containerColor = colors.pink, contentColor = colors.onPink),
            shape = RoundedCornerShape(14.dp),
            modifier = Modifier.fillMaxWidth().height(48.dp),
        ) {
            Text(
                text = if (state.activating) {
                    stringResource(R.string.preg_activating)
                } else {
                    stringResource(R.string.preg_not_active_cta)
                },
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

@Composable
private fun Tracker(
    state: PregnancyUiState,
    onIntent: (PregnancyIntent) -> Unit,
    onOpenLog: (PregnancyLogTab) -> Unit,
    colors: RitmeColors,
) {
    val status = state.status ?: return
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item(key = "hero") { Hero(status, state, colors) }
        item(key = "due") { DueRow(status, state, colors) }
        item(key = "actions") { QuickActions(onOpenLog, colors) }
        item(key = "weeks") { WeekBrowser(state, status, onIntent, colors) }
        item(key = "content") { WeekContent(state, colors) }
        item(key = "alerts") { AlertsCard(state, onIntent, colors) }
        item(key = "tail") { Spacer(Modifier.height(4.dp)) }
    }
}

@Composable
private fun Hero(status: PregnancyStatus, state: PregnancyUiState, colors: RitmeColors) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(Brush.linearGradient(listOf(colors.pink, colors.accent)))
            .padding(16.dp),
    ) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.preg_title),
                style = MaterialTheme.typography.labelMedium,
                color = colors.onPink,
                modifier = Modifier.weight(1f),
            )
            if (status.isHighRisk) {
                Box(
                    Modifier.clip(RoundedCornerShape(10.dp)).background(colors.onPink.copy(alpha = 0.25f))
                        .padding(horizontal = 10.dp, vertical = 4.dp),
                ) {
                    Text(
                        stringResource(R.string.preg_high_risk),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.onPink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        Spacer(Modifier.height(12.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(52.dp).clip(CircleShape).background(colors.onPink.copy(alpha = 0.22f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_heart),
                    contentDescription = null,
                    tint = colors.onPink,
                    modifier = Modifier.size(24.dp),
                )
            }
            Spacer(Modifier.width(12.dp))
            Column {
                Text(
                    text = stringResource(R.string.preg_week_of, (status.currentWeek ?: 1).toPersianDigits()),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.onPink,
                    fontWeight = FontWeight.Bold,
                )
                if (status.gestationalWeeks != null && status.gestationalDays != null) {
                    Text(
                        text = stringResource(
                            R.string.preg_ga_value,
                            status.gestationalWeeks.toPersianDigits(),
                            status.gestationalDays.toPersianDigits(),
                        ),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.onPink,
                    )
                }
            }
        }
        Spacer(Modifier.height(14.dp))
        Box(
            Modifier.fillMaxWidth().height(10.dp).clip(RoundedCornerShape(99.dp))
                .background(colors.onPink.copy(alpha = 0.3f)),
        ) {
            Box(
                Modifier
                    .fillMaxWidth((status.progressPercent / 100f).coerceIn(0.02f, 1f))
                    .height(10.dp)
                    .clip(RoundedCornerShape(99.dp))
                    .background(colors.onPink),
            )
        }
        Spacer(Modifier.height(8.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(
                text = stringResource(R.string.preg_progress, status.progressPercent.toPersianDigits()),
                style = MaterialTheme.typography.labelSmall,
                color = colors.onPink,
            )
            Text(
                text = trimesterLabel(status.trimester),
                style = MaterialTheme.typography.labelSmall,
                color = colors.onPink,
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

@Composable
private fun trimesterLabel(trimester: Int?): String = when (trimester) {
    1 -> stringResource(R.string.preg_trimester_1)
    2 -> stringResource(R.string.preg_trimester_2)
    3 -> stringResource(R.string.preg_trimester_3)
    else -> ""
}

@Composable
private fun DueRow(status: PregnancyStatus, state: PregnancyUiState, colors: RitmeColors) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        SurfaceCard(Modifier.weight(1.4f)) {
            Text(
                stringResource(R.string.preg_due_label),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = status.dueDate?.toJalali()?.formatFull() ?: stringResource(R.string.home_unavailable),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            status.dueDate?.let { due ->
                val daysLeft = due.toJalali().toJdn() -
                    ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali().toJdn()
                if (daysLeft >= 0) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = stringResource(R.string.preg_due_countdown, daysLeft.toPersianDigits()),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        SurfaceCard(Modifier.weight(1f)) {
            Text(
                stringResource(R.string.preg_confidence_label),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                text = when (state.profile?.confidenceLevel) {
                    PregnancyConfidence.HIGH -> stringResource(R.string.preg_confidence_high)
                    PregnancyConfidence.MEDIUM -> stringResource(R.string.preg_confidence_medium)
                    PregnancyConfidence.LOW -> stringResource(R.string.preg_confidence_low)
                    null -> stringResource(R.string.home_unavailable)
                },
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            state.profile?.uncertaintyDays?.let { days ->
                Spacer(Modifier.height(2.dp))
                Text(
                    text = "±${days.toPersianDigits()}",
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                )
            }
        }
    }
}

@Composable
private fun QuickActions(onOpenLog: (PregnancyLogTab) -> Unit, colors: RitmeColors) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        ActionTile(
            icon = R.drawable.ic_plus,
            label = R.string.preg_action_symptoms,
            tint = colors.pink,
            container = colors.pinkContainer,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.SYMPTOMS) }
        ActionTile(
            icon = R.drawable.ic_stetho,
            label = R.string.preg_action_weekly,
            tint = colors.info,
            container = colors.background,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.WEEKLY) }
        ActionTile(
            icon = R.drawable.ic_heart,
            label = R.string.preg_action_movement,
            tint = colors.success,
            container = colors.ovulationContainer,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.MOVEMENT) }
    }
}

@Composable
private fun ActionTile(
    @DrawableRes icon: Int,
    @StringRes label: Int,
    tint: Color,
    container: Color,
    modifier: Modifier,
    colors: RitmeColors,
    onClick: () -> Unit,
) {
    Column(
        modifier = modifier
            .clip(RoundedCornerShape(14.dp))
            .background(colors.surface)
            .clickable(onClick = onClick)
            .padding(vertical = 14.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(Modifier.size(38.dp).clip(CircleShape).background(container), contentAlignment = Alignment.Center) {
            Icon(painter = painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(18.dp))
        }
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(label),
            style = MaterialTheme.typography.labelSmall,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun WeekBrowser(
    state: PregnancyUiState,
    status: PregnancyStatus,
    onIntent: (PregnancyIntent) -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            // RTL: the start chevron moves one week back.
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.preg_week_prev), {
                onIntent(PregnancyIntent.PreviousWeek)
            })
            Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = stringResource(R.string.preg_week_label, state.selectedWeek.toPersianDigits()),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                if (state.selectedWeek != status.currentWeek) {
                    Text(
                        text = stringResource(R.string.preg_week_current),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .clickable { onIntent(PregnancyIntent.ResetWeek) }
                            .padding(horizontal = 6.dp, vertical = 2.dp),
                    )
                }
            }
            HeaderIconButton(R.drawable.ic_chevron_left, stringResource(R.string.preg_week_next), {
                onIntent(PregnancyIntent.NextWeek)
            })
        }
    }
}

/** One module accordion entry: title + icon; expands to the localized body. */
private data class ContentModule(@field:StringRes val title: Int, @field:DrawableRes val icon: Int, val body: String)

@Composable
private fun WeekContent(state: PregnancyUiState, colors: RitmeColors) {
    val content = state.content
    if (state.contentLoading) {
        Box(Modifier.fillMaxWidth().padding(vertical = 20.dp), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(color = colors.pink)
        }
        return
    }
    if (content == null || content.isEmpty) {
        SurfaceCard {
            Text(
                stringResource(R.string.preg_no_content),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        }
        return
    }
    val modules = contentModules(content)
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        modules.forEach { module ->
            if (module.body.isNotBlank()) ModuleAccordion(module, colors)
        }
        if (content.faq.isNotBlank()) {
            ModuleAccordion(ContentModule(R.string.preg_faq_title, R.drawable.ic_info, content.faq), colors)
        }
    }
}

private fun contentModules(content: PregnancyWeekContent): List<ContentModule> = listOf(
    ContentModule(R.string.preg_module_fetal, R.drawable.ic_heart, content.fetalDevelopment),
    ContentModule(R.string.preg_module_mother, R.drawable.ic_user, content.motherBodyChanges),
    ContentModule(R.string.preg_module_adaptation, R.drawable.ic_refresh, content.bodyAdaptation),
    ContentModule(R.string.preg_module_emotional, R.drawable.ic_smile, content.emotionalStatus),
    ContentModule(R.string.preg_module_nutrition, R.drawable.ic_glass, content.keyNutrition),
    ContentModule(R.string.preg_module_activity, R.drawable.ic_walk, content.physicalActivity),
    ContentModule(R.string.preg_module_dos, R.drawable.ic_check_circle, content.dosAndDonts),
    ContentModule(R.string.preg_module_care, R.drawable.ic_shield, content.carePlan),
    ContentModule(R.string.preg_module_tests, R.drawable.ic_stetho, content.testsAndCheckups),
)

@Composable
private fun ModuleAccordion(module: ContentModule, colors: RitmeColors) {
    var expanded by remember(module.title) { mutableStateOf(false) }
    SurfaceCard(Modifier.clickable { expanded = !expanded }) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(34.dp).clip(CircleShape).background(colors.pinkContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(module.icon),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(16.dp),
                )
            }
            Spacer(Modifier.width(10.dp))
            Text(
                stringResource(module.title),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            Icon(
                painter = painterResource(if (expanded) R.drawable.ic_chevron_down else R.drawable.ic_chevron_left),
                contentDescription = null,
                tint = colors.inkMuted,
                modifier = Modifier.size(16.dp),
            )
        }
        AnimatedVisibility(visible = expanded) {
            Text(
                text = module.body,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                modifier = Modifier.padding(top = 10.dp),
            )
        }
    }
}

@Composable
private fun AlertsCard(state: PregnancyUiState, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.preg_alerts_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (state.alerts.any { !it.isRead }) {
                Text(
                    text = stringResource(R.string.preg_alerts_mark_all),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.pink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier
                        .clip(RoundedCornerShape(8.dp))
                        .clickable { onIntent(PregnancyIntent.MarkAllAlertsRead) }
                        .padding(4.dp),
                )
            }
        }
        Spacer(Modifier.height(10.dp))
        if (state.alerts.isEmpty()) {
            Text(
                stringResource(R.string.preg_alerts_empty),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
            )
            return@SurfaceCard
        }
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            state.alerts.forEach { alert -> AlertRow(alert, onIntent, colors) }
        }
    }
}

@Composable
private fun AlertRow(alert: PregnancyAlert, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    val accent = when (alert.level) {
        PregnancyAlertLevel.INFO -> colors.info
        PregnancyAlertLevel.WARNING -> colors.warning
        PregnancyAlertLevel.EMERGENCY -> colors.error
    }
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(colors.background)
            .padding(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(Modifier.size(10.dp).clip(CircleShape).background(accent))
            Spacer(Modifier.width(8.dp))
            Text(
                text = alert.title,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (!alert.isRead) {
                Text(
                    text = stringResource(R.string.preg_alerts_unread),
                    style = MaterialTheme.typography.labelSmall,
                    color = accent,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
        if (alert.message.isNotBlank()) {
            Spacer(Modifier.height(6.dp))
            Text(alert.message, style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        }
        if (alert.recommendedActions.isNotEmpty()) {
            Spacer(Modifier.height(6.dp))
            Text(
                stringResource(R.string.preg_alerts_recommended),
                style = MaterialTheme.typography.labelSmall,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            alert.recommendedActions.forEach { action ->
                Text("• $action", style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
            }
        }
        Spacer(Modifier.height(8.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            if (!alert.isRead) {
                Text(
                    text = stringResource(R.string.action_done),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.success,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier
                        .clip(RoundedCornerShape(8.dp))
                        .clickable { onIntent(PregnancyIntent.MarkAlertRead(alert.id)) }
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                )
            }
            Text(
                text = stringResource(R.string.preg_alerts_dismiss),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
                modifier = Modifier
                    .clip(RoundedCornerShape(8.dp))
                    .clickable { onIntent(PregnancyIntent.DismissAlert(alert.id)) }
                    .padding(horizontal = 8.dp, vertical = 4.dp),
            )
        }
    }
}

@Composable
private fun ErrorState(onRetry: () -> Unit, colors: RitmeColors) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            stringResource(R.string.home_error),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = onRetry,
            colors = ButtonDefaults.buttonColors(containerColor = colors.pink, contentColor = colors.onPink),
        ) {
            Text(stringResource(R.string.action_retry))
        }
    }
}
