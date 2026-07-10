package ir.ritmeapp.ritme.adapter.inbound.ui.cycle

import androidx.compose.foundation.Canvas
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.runtime.CompositionLocalProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlin.math.max

/**
 * The cycle-analysis tab (web `/cycle`): a conic progress donut of the current cycle day, the
 * phase + fertility summary, an upcoming-events timeline bar, the cycle summary, and a manual
 * recalculation action that polls until fresh numbers arrive.
 */
@Composable
fun CycleScreen(
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: CycleViewModel = viewModel(factory = container.cycleViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:cycle:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Cycle.route, null, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = {
            RitmeBottomBar(active = RitmeTab.CYCLE, mode = state.mode, onNavigate = onNavigate)
        },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .background(Brush.verticalGradient(0f to colors.pinkContainer, 0.4f to colors.background)),
        ) {
            when {
                state.loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator(color = colors.pink)
                }

                state.isError -> ErrorState({ viewModel.onIntent(CycleIntent.Retry) }, colors)

                !state.hasCycle -> EmptyState(onLog = { onNavigate(Destination.Log()) }, colors)

                else -> Content(state, viewModel::onIntent, onNavigate, colors)
            }
        }
    }
}

@Composable
private fun Content(
    state: CycleUiState,
    onIntent: (CycleIntent) -> Unit,
    onNavigate: (Destination) -> Unit,
    colors: RitmeColors,
) {
    val predictions = state.predictions ?: return
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item(key = "header") { Header(state, onIntent, colors) }
        if (state.isRecalculating) {
            item(key = "updating") {
                Text(
                    stringResource(R.string.cycle_updating),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
        }
        item(key = "status") { StatusCard(state, predictions, colors) }
        item(key = "timeline") { TimelineCard(predictions, colors) }
        item(key = "summary") { SummaryCard(state, predictions, onNavigate, colors) }
        item(key = "mycycles") { MyCyclesCard(predictions, onNavigate, colors) }
        item(key = "smarttip") { SmartTip(colors) }
        item(key = "tail") { Spacer(Modifier.height(4.dp)) }
    }
}

@Composable
private fun Header(state: CycleUiState, onIntent: (CycleIntent) -> Unit, colors: RitmeColors) {
    Box(Modifier.fillMaxWidth()) {
        Column(Modifier.align(Alignment.Center), horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                stringResource(R.string.cycle_screen_title),
                style = MaterialTheme.typography.titleLarge,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Text(
                stringResource(R.string.cycle_screen_tagline),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
        }
        Box(
            Modifier
                .align(Alignment.CenterEnd)
                .size(38.dp)
                .clip(CircleShape)
                .background(colors.surface)
                .clickable(enabled = !state.isRecalculating) { onIntent(CycleIntent.Recalculate) },
            contentAlignment = Alignment.Center,
        ) {
            if (state.isRecalculating) {
                CircularProgressIndicator(color = colors.pink, strokeWidth = 2.dp, modifier = Modifier.size(18.dp))
            } else {
                Icon(
                    painter = painterResource(R.drawable.ic_refresh),
                    contentDescription = stringResource(R.string.cycle_recalculate),
                    tint = colors.pink,
                    modifier = Modifier.size(18.dp),
                )
            }
        }
    }
}

@Composable
private fun StatusCard(state: CycleUiState, predictions: CyclePredictions, colors: RitmeColors) {
    val phaseColor = phaseColor(predictions, colors)
    SurfaceCard {
        Text(
            stringResource(R.string.cycle_based_on),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(14.dp))
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            CycleDonut(predictions, phaseColor, colors)
            Spacer(Modifier.width(16.dp))
            Column(Modifier.weight(1f)) {
                PhaseChip(predictions, phaseColor, colors)
                Spacer(Modifier.height(10.dp))
                Column(
                    Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.background).padding(10.dp),
                ) {
                    Text(
                        stringResource(R.string.cycle_fertility_today),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                    Text(
                        text = "${predictions.fertilityPercent.toPersianDigits()}٪",
                        style = MaterialTheme.typography.titleMedium,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        val statusLine = when {
            predictions.isFertileWindow -> stringResource(R.string.cycle_fertile_window_now)
            predictions.daysUntilPmsStart == 0 -> stringResource(R.string.cycle_pms_window)
            else -> null
        }
        if (statusLine != null) {
            Spacer(Modifier.height(10.dp))
            Text(
                text = statusLine,
                style = MaterialTheme.typography.labelMedium,
                color = phaseColor,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(Modifier.height(10.dp))
        Text(
            text = state.message?.longMessage?.takeIf { it.isNotBlank() } ?: phaseDescription(predictions),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
        )
    }
}

@Composable
private fun CycleDonut(predictions: CyclePredictions, phaseColor: Color, colors: RitmeColors) {
    val progress = predictions.progressPercent / 100f
    val track = colors.background
    Box(Modifier.size(104.dp), contentAlignment = Alignment.Center) {
        Canvas(Modifier.fillMaxSize()) {
            val stroke = Stroke(width = 12.dp.toPx(), cap = StrokeCap.Round)
            val inset = 6.dp.toPx()
            val arcSize = Size(size.width - inset * 2, size.height - inset * 2)
            drawArc(
                color = track,
                startAngle = 0f,
                sweepAngle = 360f,
                useCenter = false,
                topLeft = Offset(inset, inset),
                size = arcSize,
                style = stroke,
            )
            drawArc(
                color = phaseColor,
                startAngle = -90f,
                sweepAngle = 360f * progress,
                useCenter = false,
                topLeft = Offset(inset, inset),
                size = arcSize,
                style = stroke,
            )
        }
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                text = predictions.cycleDay.toPersianDigits(),
                style = MaterialTheme.typography.headlineSmall,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Text(
                text = stringResource(R.string.cycle_of_n, predictions.cycleLength.toPersianDigits()),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
        }
    }
}

@Composable
private fun PhaseChip(predictions: CyclePredictions, phaseColor: Color, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(20.dp))
            .background(phaseContainer(predictions, colors))
            .padding(horizontal = 12.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_drop_solid),
            contentDescription = null,
            tint = phaseColor,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.width(6.dp))
        Text(
            text = phaseLabel(predictions),
            style = MaterialTheme.typography.labelMedium,
            color = phaseColor,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun TimelineCard(predictions: CyclePredictions, colors: RitmeColors) {
    val today = todayJalali()
    SurfaceCard {
        Text(
            stringResource(R.string.cycle_timeline_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(14.dp))
        // Physical day-1→day-N bar; forced LTR like the web so progress grows one way.
        CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
            CycleBar(predictions, colors)
        }
        Spacer(Modifier.height(14.dp))
        TimelineRow(
            color = colors.warning,
            container = colors.fertileContainer,
            label = stringResource(R.string.cycle_timeline_window),
            value = today.addDays(max(0, predictions.daysUntilFertileWindow)).formatDayMonth(),
            colors = colors,
        )
        Spacer(Modifier.height(10.dp))
        TimelineRow(
            color = colors.success,
            container = colors.ovulationContainer,
            label = stringResource(R.string.cycle_timeline_ovulation),
            value = today.addDays(max(0, predictions.daysUntilOvulation)).formatDayMonth(),
            colors = colors,
        )
        Spacer(Modifier.height(10.dp))
        TimelineRow(
            color = colors.pink,
            container = colors.periodContainer,
            label = stringResource(R.string.cycle_timeline_next_period),
            value = today.addDays(predictions.daysUntilNextPeriod).formatDayMonth(),
            colors = colors,
        )
    }
}

@Composable
private fun CycleBar(predictions: CyclePredictions, colors: RitmeColors) {
    val length = max(1, predictions.cycleLength)
    val dayFraction = (predictions.cycleDay - 1).coerceAtLeast(0) / length.toFloat()
    val ovulationDay = predictions.cycleDay + predictions.daysUntilOvulation
    val fertileStart = ((ovulationDay - FERTILE_LEAD_DAYS - 1).coerceAtLeast(0)) / length.toFloat()
    val fertileEnd = (ovulationDay.coerceAtMost(length)) / length.toFloat()
    val barBrush = Brush.horizontalGradient(listOf(colors.pinkLight, colors.pink))

    Box(Modifier.fillMaxWidth().height(14.dp)) {
        Box(
            Modifier.fillMaxSize().clip(RoundedCornerShape(99.dp)).background(colors.background),
        )
        // Fertile band.
        Row(Modifier.fillMaxSize()) {
            Spacer(Modifier.weight(fertileStart.coerceAtLeast(0.001f)))
            Box(
                Modifier
                    .weight((fertileEnd - fertileStart).coerceAtLeast(0.001f))
                    .height(14.dp)
                    .clip(RoundedCornerShape(99.dp))
                    .background(colors.fertileContainer),
            )
            Spacer(Modifier.weight((1f - fertileEnd).coerceAtLeast(0.001f)))
        }
        // Progress to today.
        Box(
            Modifier
                .fillMaxWidth(dayFraction.coerceIn(0.02f, 1f))
                .height(14.dp)
                .clip(RoundedCornerShape(99.dp))
                .background(barBrush),
        )
    }
}

@Composable
private fun TimelineRow(
    color: Color,
    container: Color,
    label: String,
    value: String,
    colors: RitmeColors,
) {
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(28.dp).clip(CircleShape).background(container), contentAlignment = Alignment.Center) {
            Icon(
                painter = painterResource(R.drawable.ic_drop_solid),
                contentDescription = null,
                tint = color,
                modifier = Modifier.size(12.dp),
            )
        }
        Spacer(Modifier.width(10.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            modifier = Modifier.weight(1f),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun SummaryCard(
    state: CycleUiState,
    predictions: CyclePredictions,
    onNavigate: (Destination) -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard {
        Text(
            stringResource(R.string.cycle_summary_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(12.dp))
        SummaryRow(
            stringResource(R.string.cycle_summary_length),
            stringResource(R.string.cycle_days, predictions.cycleLength.toPersianDigits()),
            colors,
        )
        SummaryRow(
            stringResource(R.string.cycle_summary_ovulation_day),
            stringResource(
                R.string.cycle_summary_day_n,
                (predictions.cycleDay + predictions.daysUntilOvulation).toPersianDigits(),
            ),
            colors,
        )
        SummaryRow(stringResource(R.string.cycle_summary_phase), phaseLabel(predictions), colors)
        SummaryRow(stringResource(R.string.cycle_variability_label), variabilityLabel(state), colors)
        Spacer(Modifier.height(12.dp))
        Button(
            onClick = { onNavigate(Destination.Calendar) },
            colors = ButtonDefaults.buttonColors(containerColor = colors.pinkContainer, contentColor = colors.pink),
            shape = RoundedCornerShape(14.dp),
            modifier = Modifier.fillMaxWidth().height(44.dp),
        ) {
            Text(stringResource(R.string.cycle_summary_view_calendar), fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
private fun SummaryRow(label: String, value: String, colors: RitmeColors) {
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted)
        Text(value, style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun MyCyclesCard(predictions: CyclePredictions, onNavigate: (Destination) -> Unit, colors: RitmeColors) {
    val today = todayJalali()
    SurfaceCard {
        Text(
            stringResource(R.string.cycles_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            text = stringResource(R.string.cycles_current_day, predictions.cycleDay.toPersianDigits()),
            style = MaterialTheme.typography.bodyLarge,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = stringResource(R.string.cycles_started, today.addDays(-(predictions.cycleDay - 1)).formatDayMonth()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(10.dp))
        Text(
            text = stringResource(R.string.cycles_add_previous),
            style = MaterialTheme.typography.labelMedium,
            color = colors.pink,
            fontWeight = FontWeight.Bold,
            modifier = Modifier
                .clip(RoundedCornerShape(12.dp))
                .clickable { onNavigate(Destination.Log()) }
                .padding(vertical = 4.dp),
        )
    }
}

@Composable
private fun SmartTip(colors: RitmeColors) {
    SurfaceCard {
        Text(
            stringResource(R.string.smarttip_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.cycle_smarttip_body),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(10.dp))
        Row(
            Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.pinkContainer).padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_sparkle),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(18.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                stringResource(R.string.cycle_smarttip_quote),
                style = MaterialTheme.typography.labelMedium,
                color = colors.ink,
            )
        }
    }
}

@Composable
private fun EmptyState(onLog: () -> Unit, colors: RitmeColors) {
    Column(
        Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            Modifier.size(64.dp).clip(CircleShape).background(colors.pinkContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_drop),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(28.dp),
            )
        }
        Spacer(Modifier.height(16.dp))
        Text(
            stringResource(R.string.cycle_empty_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.cycle_empty_body),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(16.dp))
        Button(
            onClick = onLog,
            colors = ButtonDefaults.buttonColors(containerColor = colors.pink, contentColor = colors.onPink),
            shape = RoundedCornerShape(14.dp),
        ) {
            Text(stringResource(R.string.cycle_empty_cta), fontWeight = FontWeight.Bold)
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

// --- Phase presentation helpers ---------------------------------------------

private fun phaseColor(predictions: CyclePredictions, colors: RitmeColors): Color = when {
    predictions.phase == CyclePhase.MENSTRUATION -> colors.pink
    predictions.phase == CyclePhase.OVULATION -> colors.success
    predictions.isFertileWindow -> colors.warning
    else -> colors.accent
}

private fun phaseContainer(predictions: CyclePredictions, colors: RitmeColors): Color = when {
    predictions.phase == CyclePhase.MENSTRUATION -> colors.periodContainer
    predictions.phase == CyclePhase.OVULATION -> colors.ovulationContainer
    predictions.isFertileWindow -> colors.fertileContainer
    else -> colors.violetContainer
}

@Composable
private fun phaseLabel(predictions: CyclePredictions): String = when {
    predictions.phase == CyclePhase.MENSTRUATION -> stringResource(R.string.phase_menstruation)
    predictions.phase == CyclePhase.OVULATION -> stringResource(R.string.phase_ovulation)
    predictions.isFertileWindow -> stringResource(R.string.calendar_phase_fertile)
    predictions.phase == CyclePhase.FOLLICULAR -> stringResource(R.string.phase_follicular)
    else -> stringResource(R.string.phase_luteal)
}

@Composable
private fun phaseDescription(predictions: CyclePredictions): String = when {
    predictions.phase == CyclePhase.MENSTRUATION -> stringResource(R.string.cycle_desc_period)
    predictions.phase == CyclePhase.OVULATION -> stringResource(R.string.cycle_desc_ovulation)
    predictions.phase == CyclePhase.FOLLICULAR -> stringResource(R.string.cycle_desc_follicular)
    else -> stringResource(R.string.cycle_desc_luteal)
}

@Composable
private fun variabilityLabel(state: CycleUiState): String = when (state.calculation?.variability) {
    "regular" -> stringResource(R.string.cycle_variability_regular)
    "semi_irregular", "irregular" -> stringResource(R.string.cycle_variability_irregular)
    else -> stringResource(R.string.cycle_variability_unknown)
}

private const val FERTILE_LEAD_DAYS = 4
