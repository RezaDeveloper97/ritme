package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.material3.CircularProgressIndicator
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
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.log.hintRes
import ir.ritmeapp.ritme.adapter.inbound.ui.log.labelRes
import ir.ritmeapp.ritme.adapter.inbound.ui.log.optionLabelRes
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CycleDaySnapshot
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.HealthLogCategory
import ir.ritmeapp.ritme.domain.model.HealthLogField
import ir.ritmeapp.ritme.domain.model.HealthLogValue
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

private val WEEKDAY_RES = intArrayOf(
    R.string.week_sat, R.string.week_sun, R.string.week_mon, R.string.week_tue,
    R.string.week_wed, R.string.week_thu, R.string.week_fri,
)

/**
 * The Jalali cycle calendar (web `/calendar`), wired to `GET /cycle/month`: phase-colored day
 * cells (period / fertile window / ovulation), a legend, the selected day's detail card, and a
 * recap of that day's health log with a jump into the log editor.
 */
@Composable
fun CalendarScreen(
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: CalendarViewModel = viewModel(factory = container.calendarViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:calendar:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Calendar.route, null, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = {
            RitmeBottomBar(active = RitmeTab.CALENDAR, mode = state.mode, onNavigate = onNavigate)
        },
    ) { padding ->
        Box(
            Modifier
                .fillMaxSize()
                .padding(padding)
                .background(Brush.verticalGradient(0f to colors.pinkContainer, 0.35f to colors.background)),
        ) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                item(key = "header") {
                    Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(
                            stringResource(R.string.calendar_title),
                            style = MaterialTheme.typography.titleLarge,
                            color = colors.ink,
                            fontWeight = FontWeight.Bold,
                        )
                        Spacer(Modifier.height(4.dp))
                        Text(
                            stringResource(R.string.calendar_subtitle),
                            style = MaterialTheme.typography.labelMedium,
                            color = colors.inkMuted,
                        )
                    }
                }
                item(key = "calendar") { CalendarCard(state, viewModel::onIntent, colors) }
                item(key = "legend") { Legend(colors) }
                item(key = "detail") { DayDetailCard(state, colors) }
                item(key = "daylog") {
                    DayLogCard(
                        state = state,
                        onEdit = { onNavigate(Destination.Log(state.selected.toIso())) },
                        colors = colors,
                    )
                }
                item(key = "smarttip") { SmartTipCard(colors) }
                item(key = "tail") { Spacer(Modifier.height(4.dp)) }
            }
        }
    }
}

@Composable
private fun CalendarCard(state: CalendarUiState, onIntent: (CalendarIntent) -> Unit, colors: RitmeColors) {
    val weeks = remember(state.year, state.month) { JalaliDate.monthMatrix(state.year, state.month) }
    val visibleWeeks = if (state.monthView) {
        weeks
    } else {
        listOf(weeks.firstOrNull { week -> week.any { it == state.selected } } ?: weeks.first())
    }

    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            // RTL: the start chevron goes back one month.
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.calendar_prev_month), {
                onIntent(CalendarIntent.PreviousMonth)
            })
            Text(
                text = "${JalaliDate.MONTH_NAMES[state.month - 1]} ${state.year.toPersianDigits()}",
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                textAlign = TextAlign.Center,
                modifier = Modifier.weight(1f),
            )
            ViewToggle(state.monthView, { onIntent(CalendarIntent.ToggleView) }, colors)
            Spacer(Modifier.width(6.dp))
            HeaderIconButton(R.drawable.ic_chevron_left, stringResource(R.string.calendar_next_month), {
                onIntent(CalendarIntent.NextMonth)
            })
        }

        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth()) {
            WEEKDAY_RES.forEach { res ->
                Text(
                    text = stringResource(res),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        Spacer(Modifier.height(8.dp))

        if (state.loading && state.days.isEmpty()) {
            Box(Modifier.fillMaxWidth().padding(vertical = 24.dp), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.pink)
            }
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                visibleWeeks.forEach { week ->
                    Row(Modifier.fillMaxWidth()) {
                        week.forEach { cell ->
                            Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                                if (cell != null) {
                                    DayCell(
                                        day = cell,
                                        snapshot = state.snapshotFor(cell),
                                        isToday = cell == state.today,
                                        isSelected = cell == state.selected,
                                        colors = colors,
                                        onClick = { onIntent(CalendarIntent.SelectDay(cell)) },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ViewToggle(monthView: Boolean, onToggle: () -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(20.dp))
            .background(colors.background)
            .clickable(onClick = onToggle)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(if (monthView) R.drawable.ic_grid else R.drawable.ic_calendar),
            contentDescription = null,
            tint = colors.inkMuted,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.width(4.dp))
        Text(
            text = stringResource(if (monthView) R.string.calendar_week_view else R.string.calendar_full_month),
            style = MaterialTheme.typography.labelSmall,
            color = colors.inkMuted,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun DayCell(
    day: JalaliDate,
    snapshot: CycleDaySnapshot?,
    isToday: Boolean,
    isSelected: Boolean,
    colors: RitmeColors,
    onClick: () -> Unit,
) {
    val (bg, fg) = dayColors(snapshot, colors)
    Box(
        modifier = Modifier
            .size(38.dp)
            .clip(CircleShape)
            .background(bg)
            .then(if (isSelected) Modifier.border(2.dp, colors.pink, CircleShape) else Modifier)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                text = day.day.toPersianDigits(),
                style = MaterialTheme.typography.labelLarge,
                color = fg,
                fontWeight = FontWeight.Bold,
            )
            if (isToday) {
                Box(Modifier.size(4.dp).clip(CircleShape).background(colors.pink))
            }
        }
    }
}

/** Phase → (cell background, text) colors, matching the web calendar palette. */
private fun dayColors(snapshot: CycleDaySnapshot?, colors: RitmeColors): Pair<Color, Color> = when {
    snapshot == null -> colors.background to colors.ink
    snapshot.phase == CyclePhase.MENSTRUATION -> colors.periodContainer to colors.pink
    snapshot.phase == CyclePhase.OVULATION -> colors.ovulationContainer to colors.success
    snapshot.isFertileWindow -> colors.fertileContainer to colors.warning
    else -> colors.background to colors.ink
}

@Composable
private fun Legend(colors: RitmeColors) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        LegendItem(colors.pink, stringResource(R.string.calendar_legend_period), colors)
        LegendItem(colors.warning, stringResource(R.string.calendar_legend_fertile), colors)
        LegendItem(colors.success, stringResource(R.string.calendar_legend_ovulation), colors)
    }
}

@Composable
private fun LegendItem(dot: Color, label: String, colors: RitmeColors) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Box(Modifier.size(10.dp).clip(CircleShape).background(dot))
        Spacer(Modifier.width(6.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
    }
}

@Composable
private fun DayDetailCard(state: CalendarUiState, colors: RitmeColors) {
    val snapshot = state.selectedSnapshot
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = state.selected.formatDayMonth(),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                snapshot?.cycleDay?.let { day ->
                    Text(
                        text = stringResource(R.string.calendar_cycle_day, day.toPersianDigits()),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                }
            }
            if (state.selected == state.today) {
                Box(
                    Modifier.clip(RoundedCornerShape(10.dp)).background(colors.pinkContainer)
                        .padding(horizontal = 10.dp, vertical = 4.dp),
                ) {
                    Text(
                        stringResource(R.string.calendar_today),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            DetailTile(
                label = stringResource(R.string.calendar_phase_label),
                value = phaseLabel(snapshot),
                colors = colors,
                modifier = Modifier.weight(1f),
            )
            DetailTile(
                label = stringResource(R.string.calendar_chance_label),
                value = chanceLabel(snapshot),
                colors = colors,
                modifier = Modifier.weight(1f),
            )
        }
    }
}

@Composable
private fun DetailTile(label: String, value: String, colors: RitmeColors, modifier: Modifier = Modifier) {
    Column(
        modifier = modifier.clip(RoundedCornerShape(12.dp)).background(colors.background).padding(12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text(label, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        Spacer(Modifier.height(4.dp))
        Text(value, style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun phaseLabel(snapshot: CycleDaySnapshot?): String = when {
    snapshot?.phase == CyclePhase.MENSTRUATION -> stringResource(R.string.calendar_phase_period)
    snapshot?.phase == CyclePhase.OVULATION -> stringResource(R.string.calendar_phase_ovulation)
    snapshot?.isFertileWindow == true -> stringResource(R.string.calendar_phase_fertile)
    snapshot?.phase == CyclePhase.FOLLICULAR -> stringResource(R.string.calendar_phase_follicular)
    snapshot?.phase == CyclePhase.LUTEAL -> stringResource(R.string.calendar_phase_luteal)
    else -> stringResource(R.string.home_unavailable)
}

@Composable
private fun chanceLabel(snapshot: CycleDaySnapshot?): String {
    val percent = snapshot?.fertilityPercent ?: return stringResource(R.string.home_unavailable)
    return when {
        percent < LOW_CHANCE_MAX -> stringResource(R.string.calendar_chance_low)
        percent < MEDIUM_CHANCE_MAX -> stringResource(R.string.calendar_chance_medium)
        else -> stringResource(R.string.calendar_chance_high)
    }
}

@Composable
private fun DayLogCard(state: CalendarUiState, onEdit: () -> Unit, colors: RitmeColors) {
    val log = state.dayLog
    SurfaceCard {
        if (log == null) {
            Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.fillMaxWidth()) {
                Text(
                    stringResource(R.string.calendar_daylog_empty_title),
                    style = MaterialTheme.typography.bodyLarge,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    stringResource(R.string.calendar_daylog_empty_subtitle, state.selected.formatDayMonth()),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                    textAlign = TextAlign.Center,
                )
                Spacer(Modifier.height(12.dp))
                Box(
                    Modifier
                        .size(46.dp)
                        .clip(CircleShape)
                        .background(Brush.verticalGradient(listOf(colors.pink, colors.accent)))
                        .clickable(onClick = onEdit),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_plus),
                        contentDescription = stringResource(R.string.action_edit),
                        tint = colors.onPink,
                        modifier = Modifier.size(22.dp),
                    )
                }
            }
        } else {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.calendar_daylog_title),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = stringResource(R.string.action_edit),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.pink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier
                        .clip(RoundedCornerShape(12.dp))
                        .background(colors.pinkContainer)
                        .clickable(onClick = onEdit)
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                )
            }
            Spacer(Modifier.height(10.dp))
            LogRecap(log, colors)
        }
    }
}

/** Read-only grouped recap of a saved day log (category → localized value labels). */
@Composable
private fun LogRecap(log: DailyHealthLog, colors: RitmeColors) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        HealthLogCategory.entries.forEach { category ->
            val fields = log.values.keys.filter { it.category == category }.sortedBy { it.ordinal }
            if (fields.isEmpty()) return@forEach
            Column(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.background).padding(10.dp),
            ) {
                Text(
                    stringResource(category.labelRes()),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.pink,
                    fontWeight = FontWeight.Bold,
                )
                Spacer(Modifier.height(4.dp))
                fields.forEach { field ->
                    Text(
                        text = recapLine(field, log.values.getValue(field)),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.ink,
                    )
                }
            }
        }
    }
}

@Composable
private fun recapLine(field: HealthLogField, value: HealthLogValue): String {
    val label = stringResource(field.labelRes())
    return when (value) {
        is HealthLogValue.Choice ->
            "$label: ${optionLabelRes(field, value.option)?.let { stringResource(it) } ?: value.option}"

        is HealthLogValue.MultiChoice -> {
            val options = value.options.map { option ->
                optionLabelRes(field, option)?.let { stringResource(it) } ?: option
            }
            "$label: ${options.joinToString("، ")}"
        }

        is HealthLogValue.Toggle -> label
        is HealthLogValue.Number -> "$label: ${value.value.toString().toPersianDigits()}"
        is HealthLogValue.Text -> value.text
    }
}

@Composable
private fun SmartTipCard(colors: RitmeColors) {
    SurfaceCard {
        Text(
            stringResource(R.string.smarttip_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.calendar_smarttip_body),
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
                stringResource(R.string.calendar_smarttip_quote),
                style = MaterialTheme.typography.labelMedium,
                color = colors.ink,
            )
        }
    }
}

private const val LOW_CHANCE_MAX = 10.0
private const val MEDIUM_CHANCE_MAX = 22.0
