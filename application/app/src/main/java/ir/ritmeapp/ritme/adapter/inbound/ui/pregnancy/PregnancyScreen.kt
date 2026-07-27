package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
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
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
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

// One-off accent hexes mirroring the web MODULE_STYLE / LEVEL_STYLE tables (not in the shared
// token set). TODO token: promote to colors.xml if these gain reuse beyond this screen.
private val ModuleFetalColor = Color(0xFFE9276E) // TODO token
private val ModuleAdaptationColor = Color(0xFF5B6BE1) // TODO token
private val ModuleEmotionalColor = Color(0xFFE9662E) // TODO token
private val ModuleActivityColor = Color(0xFF0E9C8A) // TODO token
private val ModuleDosColor = Color(0xFF2E9BE9) // TODO token
private val ModuleTestsColor = Color(0xFF6D7A87) // TODO token
private val ActionWeeklyColor = Color(0xFF5B6BE1) // TODO token

private val AlertInfoColor = Color(0xFF2E9BE9) // TODO token
private val AlertInfoSoft = Color(0xFFE3F1FD) // TODO token
private val AlertWarningColor = Color(0xFFE9662E) // TODO token
private val AlertWarningSoft = Color(0xFFFDEBE2) // TODO token
private val AlertEmergencyColor = Color(0xFFE5484D) // TODO token
private val AlertEmergencySoft = Color(0xFFFCE7E7) // TODO token

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
        // Flat `var(--page)` background like the web — no page-level pink wash (Scaffold paints it).
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            when {
                state.loading -> LoadingState(colors)

                state.isError -> ErrorState({ viewModel.onIntent(PregnancyIntent.Retry) }, colors)

                state.needsOnboarding -> ActivateGate(state, viewModel::onIntent, colors)

                else -> Tracker(state, viewModel::onIntent, onOpenLog, colors)
            }
        }
    }
}

/** Full-screen load: centered localized text near the top, matching the web loading shell. */
@Composable
private fun LoadingState(colors: RitmeColors) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.TopCenter) {
        Text(
            stringResource(R.string.loading),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            modifier = Modifier.padding(top = 60.dp),
        )
    }
}

@Composable
private fun ActivateGate(state: PregnancyUiState, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    // Top-anchored (not vertically centered) to mirror the web ActivateGate (`padding: 40px 22px 0`).
    Column(
        Modifier
            .fillMaxSize()
            .padding(start = 22.dp, end = 22.dp, top = 40.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(72.dp).clip(CircleShape).background(colors.pinkContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_heart),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(34.dp),
            )
        }
        Spacer(Modifier.height(18.dp))
        Text(
            stringResource(R.string.preg_not_active_title),
            style = MaterialTheme.typography.titleLarge.copy(fontSize = 20.sp),
            color = colors.ink,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            stringResource(R.string.preg_not_active_body),
            style = MaterialTheme.typography.bodyMedium.copy(lineHeight = 28.sp),
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))
        RitmePrimaryButton(
            text = if (state.activating) {
                stringResource(R.string.preg_activating)
            } else {
                stringResource(R.string.preg_not_active_cta)
            },
            onClick = { onIntent(PregnancyIntent.Activate) },
            enabled = !state.activating,
        )
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
    // The hero is full-bleed (no side inset); every other section carries its own 16dp side
    // padding and web-matched top gap, so the LazyColumn itself adds no padding/spacing.
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(0.dp),
    ) {
        item(key = "hero") { Hero(status, colors) }
        item(key = "due") {
            Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 14.dp)) { DueRow(status, state, colors) }
        }
        item(key = "actions") {
            Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 12.dp)) { QuickActions(onOpenLog, colors) }
        }
        item(key = "weeks") {
            Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 18.dp)) { WeekBrowser(state, status, onIntent, colors) }
        }
        item(key = "content") {
            Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 12.dp)) { WeekContent(state, colors) }
        }
        item(key = "alerts") {
            Box(Modifier.padding(start = 16.dp, end = 16.dp, top = 20.dp)) { AlertsCard(state, onIntent, colors) }
        }
        item(key = "tail") { Spacer(Modifier.height(24.dp)) }
    }
}

@Composable
private fun Hero(status: PregnancyStatus, colors: RitmeColors) {
    // Web `.home-grad`: pale pink → mint vertical wash, full-bleed, white content on top.
    Column(
        Modifier
            .fillMaxWidth()
            .background(Brush.verticalGradient(listOf(colors.homeGradStart, colors.homeGradEnd)))
            .padding(start = 18.dp, end = 18.dp, top = 18.dp, bottom = 22.dp),
    ) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.preg_title),
                style = MaterialTheme.typography.labelMedium,
                color = colors.onPink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (status.isHighRisk) {
                Box(
                    Modifier.clip(RoundedCornerShape(20.dp)).background(colors.onPink.copy(alpha = 0.22f))
                        .padding(horizontal = 10.dp, vertical = 3.dp),
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
        Spacer(Modifier.height(14.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(62.dp).clip(CircleShape).background(colors.onPink.copy(alpha = 0.2f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_heart),
                    contentDescription = null,
                    tint = colors.onPink,
                    modifier = Modifier.size(30.dp),
                )
            }
            Spacer(Modifier.width(14.dp))
            Column {
                Text(
                    text = stringResource(R.string.preg_week_of, (status.currentWeek ?: 1).toPersianDigits()),
                    style = MaterialTheme.typography.titleLarge.copy(fontSize = 22.sp),
                    color = colors.onPink,
                    fontWeight = FontWeight.Black,
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
                        modifier = Modifier.padding(top = 3.dp),
                    )
                }
            }
        }
        Spacer(Modifier.height(16.dp))
        Box(
            Modifier.fillMaxWidth().height(8.dp).clip(RoundedCornerShape(20.dp))
                .background(colors.onPink.copy(alpha = 0.28f)),
        ) {
            Box(
                Modifier
                    .fillMaxWidth((status.progressPercent / 100f).coerceIn(0.02f, 1f))
                    .height(8.dp)
                    .clip(RoundedCornerShape(20.dp))
                    .background(colors.onPink),
            )
        }
        Spacer(Modifier.height(7.dp))
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
    val due = status.dueDate
    val confidence = state.profile?.confidenceLevel
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        // Only render each card when its data exists (web gates on estimatedDueDate / confidenceLevel).
        if (due != null) {
            SurfaceCard(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Icon(
                        painter = painterResource(R.drawable.ic_calendar),
                        contentDescription = null,
                        tint = colors.inkMuted,
                        modifier = Modifier.size(13.dp),
                    )
                    Text(
                        stringResource(R.string.preg_due_label),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                        fontWeight = FontWeight.Bold,
                    )
                }
                Spacer(Modifier.height(5.dp))
                Text(
                    text = due.toJalali().formatFull(),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                val daysLeft = due.toJalali().toJdn() -
                    ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali().toJdn()
                if (daysLeft >= 0) {
                    Spacer(Modifier.height(3.dp))
                    Text(
                        text = stringResource(R.string.preg_due_countdown, daysLeft.toPersianDigits()),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        if (confidence != null) {
            SurfaceCard(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Icon(
                        painter = painterResource(R.drawable.ic_info),
                        contentDescription = null,
                        tint = colors.inkMuted,
                        modifier = Modifier.size(13.dp),
                    )
                    Text(
                        stringResource(R.string.preg_confidence_label),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                        fontWeight = FontWeight.Bold,
                    )
                }
                Spacer(Modifier.height(5.dp))
                Text(
                    text = when (confidence) {
                        PregnancyConfidence.HIGH -> stringResource(R.string.preg_confidence_high)
                        PregnancyConfidence.MEDIUM -> stringResource(R.string.preg_confidence_medium)
                        PregnancyConfidence.LOW -> stringResource(R.string.preg_confidence_low)
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                state.profile?.uncertaintyDays?.let { days ->
                    Spacer(Modifier.height(3.dp))
                    Text(
                        text = "±${days.toPersianDigits()}",
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                }
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
            color = colors.pink,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.SYMPTOMS) }
        ActionTile(
            icon = R.drawable.ic_stetho,
            label = R.string.preg_action_weekly,
            color = ActionWeeklyColor,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.WEEKLY) }
        ActionTile(
            icon = R.drawable.ic_heart,
            label = R.string.preg_action_movement,
            color = colors.success,
            modifier = Modifier.weight(1f),
            colors = colors,
        ) { onOpenLog(PregnancyLogTab.MOVEMENT) }
    }
}

@Composable
private fun ActionTile(
    @DrawableRes icon: Int,
    @StringRes label: Int,
    color: Color,
    modifier: Modifier,
    colors: RitmeColors,
    onClick: () -> Unit,
) {
    Column(
        modifier = modifier
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(16.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 14.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier.size(40.dp).clip(CircleShape).background(color.copy(alpha = 0.1f)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painter = painterResource(icon), contentDescription = null, tint = color, modifier = Modifier.size(20.dp))
        }
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(label),
            style = MaterialTheme.typography.labelSmall.copy(lineHeight = 17.sp),
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
    val atMin = state.selectedWeek <= 1
    val atMax = state.selectedWeek >= PregnancyStatus.TOTAL_WEEKS
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            // RTL: the start (right) chevron moves one week back; dimmed + inert at the bounds.
            HeaderIconButton(
                icon = R.drawable.ic_chevron_right,
                contentDescription = stringResource(R.string.preg_week_prev),
                onClick = { if (!atMin) onIntent(PregnancyIntent.PreviousWeek) },
                modifier = Modifier.alpha(if (atMin) 0.3f else 1f),
            )
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
            HeaderIconButton(
                icon = R.drawable.ic_chevron_left,
                contentDescription = stringResource(R.string.preg_week_next),
                onClick = { if (!atMax) onIntent(PregnancyIntent.NextWeek) },
                modifier = Modifier.alpha(if (atMax) 0.3f else 1f),
            )
        }
    }
}

/** One module accordion entry: presentation-only icon + accent per module (web MODULE_STYLE). */
private data class ContentModule(
    val key: String,
    @field:StringRes val title: Int,
    @field:DrawableRes val icon: Int,
    val color: Color,
    val body: String,
)

@Composable
private fun WeekContent(state: PregnancyUiState, colors: RitmeColors) {
    if (state.contentLoading) {
        SurfaceCard {
            Text(
                stringResource(R.string.loading),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        }
        return
    }
    val content = state.content
    val modules = content?.let { contentModules(it, colors).filter { module -> module.body.isNotBlank() } } ?: emptyList()
    if (modules.isEmpty()) {
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
    // Single-open accordion, first (fetalDevelopment) module expanded; resets per week like the web.
    var openKey by remember(content?.week) { mutableStateOf<String?>("fetalDevelopment") }
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        modules.forEach { module ->
            ModuleAccordion(
                module = module,
                open = openKey == module.key,
                onToggle = { openKey = if (openKey == module.key) null else module.key },
                colors = colors,
            )
        }
        // NOTE: the FAQ section (web WeekContent) is intentionally omitted here — see final summary:
        // the backend serves faq as a JSON array but the network/domain layer models it as a flat
        // String (always empty), so structured Q&A data isn't available to this screen yet.
    }
}

private fun contentModules(content: PregnancyWeekContent, colors: RitmeColors): List<ContentModule> = listOf(
    ContentModule("fetalDevelopment", R.string.preg_module_fetal, R.drawable.ic_heart, ModuleFetalColor, content.fetalDevelopment),
    ContentModule("motherBodyChanges", R.string.preg_module_mother, R.drawable.ic_sparkle, colors.accent, content.motherBodyChanges),
    ContentModule("bodyAdaptation", R.string.preg_module_adaptation, R.drawable.ic_refresh, ModuleAdaptationColor, content.bodyAdaptation),
    ContentModule("emotionalStatus", R.string.preg_module_emotional, R.drawable.ic_smile, ModuleEmotionalColor, content.emotionalStatus),
    ContentModule("keyNutrition", R.string.preg_module_nutrition, R.drawable.ic_glass, colors.success, content.keyNutrition),
    ContentModule("physicalActivity", R.string.preg_module_activity, R.drawable.ic_walk, ModuleActivityColor, content.physicalActivity),
    ContentModule("dosAndDonts", R.string.preg_module_dos, R.drawable.ic_check_circle, ModuleDosColor, content.dosAndDonts),
    ContentModule("carePlan", R.string.preg_module_care, R.drawable.ic_stetho, colors.pink, content.carePlan),
    ContentModule("testsAndCheckups", R.string.preg_module_tests, R.drawable.ic_chart, ModuleTestsColor, content.testsAndCheckups),
)

@Composable
private fun ModuleAccordion(module: ContentModule, open: Boolean, onToggle: () -> Unit, colors: RitmeColors) {
    val chevronRotation by animateFloatAsState(if (open) 180f else 0f, label = "moduleChevron")
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(16.dp))
            .clickable(onClick = onToggle),
    ) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                Modifier.size(34.dp).clip(CircleShape).background(module.color.copy(alpha = 0.1f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(module.icon),
                    contentDescription = null,
                    tint = module.color,
                    modifier = Modifier.size(17.dp),
                )
            }
            Spacer(Modifier.width(11.dp))
            Text(
                stringResource(module.title),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            Icon(
                painter = painterResource(R.drawable.ic_chevron_down),
                contentDescription = null,
                tint = colors.inkMuted,
                modifier = Modifier.size(18.dp).rotate(chevronRotation),
            )
        }
        AnimatedVisibility(visible = open) {
            Text(
                text = module.body,
                style = MaterialTheme.typography.bodyMedium.copy(lineHeight = 27.sp),
                color = colors.ink,
                modifier = Modifier.padding(start = 14.dp, end = 14.dp, bottom = 14.dp),
            )
        }
    }
}

@Composable
private fun AlertsCard(state: PregnancyUiState, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    // Title lives OUTSIDE any card; each alert is its own card (web AlertsCard).
    Column(Modifier.fillMaxWidth()) {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 2.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
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
                    style = MaterialTheme.typography.labelMedium,
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
            SurfaceCard {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_check_circle),
                        contentDescription = null,
                        tint = colors.success,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        stringResource(R.string.preg_alerts_empty),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.inkMuted,
                    )
                }
            }
            return@Column
        }
        Column(verticalArrangement = Arrangement.spacedBy(9.dp)) {
            state.alerts.forEach { alert -> AlertRow(alert, onIntent, colors) }
        }
    }
}

@Composable
private fun AlertRow(alert: PregnancyAlert, onIntent: (PregnancyIntent) -> Unit, colors: RitmeColors) {
    val (levelColor, levelSoft, levelIcon) = when (alert.level) {
        PregnancyAlertLevel.INFO -> Triple(AlertInfoColor, AlertInfoSoft, R.drawable.ic_info)
        PregnancyAlertLevel.WARNING -> Triple(AlertWarningColor, AlertWarningSoft, R.drawable.ic_flame)
        PregnancyAlertLevel.EMERGENCY -> Triple(AlertEmergencyColor, AlertEmergencySoft, R.drawable.ic_shield)
    }
    val shape = RoundedCornerShape(16.dp)
    Row(
        Modifier
            .fillMaxWidth()
            .alpha(if (alert.isRead) 0.7f else 1f)
            .clip(shape)
            .background(colors.surface)
            .border(1.dp, colors.outline, shape),
    ) {
        // Leading-edge accent bar (web borderInlineStart: 3px solid level-color).
        Box(Modifier.width(3.dp).fillMaxHeight().background(levelColor))
        Column(Modifier.weight(1f).padding(horizontal = 13.dp, vertical = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    Modifier.size(28.dp).clip(CircleShape).background(levelSoft),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(levelIcon),
                        contentDescription = null,
                        tint = levelColor,
                        modifier = Modifier.size(15.dp),
                    )
                }
                Spacer(Modifier.width(9.dp))
                Text(
                    text = alert.title,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                if (!alert.isRead) {
                    Box(
                        Modifier.clip(RoundedCornerShape(20.dp)).background(levelSoft)
                            .padding(horizontal = 8.dp, vertical = 2.dp),
                    ) {
                        Text(
                            text = stringResource(R.string.preg_alerts_unread),
                            style = MaterialTheme.typography.labelSmall,
                            color = levelColor,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
            }
            if (alert.message.isNotBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(
                    alert.message,
                    style = MaterialTheme.typography.labelMedium.copy(lineHeight = 22.sp),
                    color = colors.inkMuted,
                )
            }
            if (alert.recommendedActions.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                // Bulleted list with NO heading (web renders a plain <ul>).
                alert.recommendedActions.forEach { action ->
                    Text(
                        "•  $action",
                        style = MaterialTheme.typography.labelSmall.copy(lineHeight = 23.sp),
                        color = colors.ink,
                    )
                }
            }
            Spacer(Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // Compact inline actions matching the web .btn-soft / .btn-ghost (32dp, width auto).
                // The shared Ritme{Soft,Ghost}Button primitives force fillMaxWidth / a 40-44dp height,
                // so they can't render this side-by-side compact pair — hence these local pills.
                if (!alert.isRead) {
                    AlertSoftPill(colors) { onIntent(PregnancyIntent.MarkAlertRead(alert.id)) }
                }
                AlertGhostPill(colors) { onIntent(PregnancyIntent.DismissAlert(alert.id)) }
            }
        }
    }
}

/** Compact web `.btn-soft` — an icon-only "mark read" pill. */
@Composable
private fun AlertSoftPill(colors: RitmeColors, onClick: () -> Unit) {
    Box(
        Modifier
            .height(32.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(colors.background)
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp),
        contentAlignment = Alignment.Center,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_check),
            contentDescription = stringResource(R.string.action_done),
            tint = colors.steel,
            modifier = Modifier.size(14.dp),
        )
    }
}

/** Compact web `.btn-ghost` — the "dismiss" pill. */
@Composable
private fun AlertGhostPill(colors: RitmeColors, onClick: () -> Unit) {
    Box(
        Modifier
            .height(32.dp)
            .clip(RoundedCornerShape(24.dp))
            .background(colors.surface)
            .border(1.5.dp, colors.pinkGhostBorder, RoundedCornerShape(24.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = stringResource(R.string.preg_alerts_dismiss),
            style = MaterialTheme.typography.labelMedium.copy(fontSize = 12.sp),
            color = colors.pink,
            fontWeight = FontWeight.Bold,
        )
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
        RitmePrimaryButton(
            text = stringResource(R.string.action_retry),
            onClick = onRetry,
        )
    }
}
