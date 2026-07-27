package ir.ritmeapp.ritme.adapter.inbound.ui.calendar

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeBottomSheet
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeSoftButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
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
import ir.ritmeapp.ritme.domain.model.PeriodDayAction
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlin.math.abs
import kotlin.math.min

private val WEEKDAY_RES = intArrayOf(
    R.string.week_sat, R.string.week_sun, R.string.week_mon, R.string.week_tue,
    R.string.week_wed, R.string.week_thu, R.string.week_fri,
)

/** (background, foreground) for a phase/marker — the web `MARKER_STYLE` / `NEUTRAL_STYLE`. */
private data class MarkerStyle(val bg: Color, val fg: Color)

/**
 * The Jalali cycle calendar (web `/calendar`), wired to `GET /cycle/month`: phase-colored
 * rounded-square day cells (period / fertile window / ovulation), an in-card legend, and a
 * tap-to-open bottom sheet holding the day's detail, quick period edits, and the recap of that
 * day's health log.
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

    // The full-screen "Edit Period Date" editor is a UI-only overlay; its open month is kept
    // local (null = closed). VISUAL SHELL — saving a reconciled range needs a new use case.
    var editorMonth by remember { mutableStateOf<Pair<Int, Int>?>(null) }
    // Transient rejected-edit message (web actionError) → red toast, auto-dismissed.
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val actionFailed = stringResource(R.string.calendar_action_failed)

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:calendar:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Calendar.route, null, System.currentTimeMillis()),
        )
        viewModel.effects.collect { effect ->
            when (effect) {
                is CalendarEffect.ShowError ->
                    errorMessage = effect.message.ifBlank { actionFailed }
            }
        }
    }
    LaunchedEffect(errorMessage) {
        if (errorMessage != null) {
            kotlinx.coroutines.delay(ERROR_TOAST_MS)
            errorMessage = null
        }
    }

    Box(modifier.fillMaxSize()) {
        Scaffold(
            modifier = Modifier.fillMaxSize(),
            containerColor = colors.background,
            bottomBar = {
                RitmeBottomBar(active = RitmeTab.CALENDAR, mode = state.mode, onNavigate = onNavigate)
            },
        ) { padding ->
            Box(
                Modifier
                    .fillMaxSize()
                    .padding(padding)
                    .background(
                        Brush.verticalGradient(0f to colors.pinkContainer, 0.35f to colors.background),
                    ),
            ) {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    item(key = "header") {
                        CalendarHeader(
                            editEnabled = !state.isFutureMonth,
                            onEdit = { editorMonth = state.year to state.month },
                            colors = colors,
                        )
                    }
                    if (state.isRecalculating) {
                        item(key = "recalc") { RecalcBanner(colors) }
                    }
                    item(key = "calendar") { CalendarCard(state, viewModel::onIntent, colors) }
                    item(key = "smarttip") { SmartTipCard(colors) }
                    item(key = "tail") { Spacer(Modifier.height(4.dp)) }
                }
            }
        }

        // Day-detail sheet (web `Sheet`) — the tapped day's detail, quick edits, and log recap.
        RitmeBottomSheet(
            visible = state.daySheetOpen,
            onDismiss = { viewModel.onIntent(CalendarIntent.CloseDaySheet) },
        ) {
            DaySheetContent(
                state = state,
                onIntent = viewModel::onIntent,
                onEditPeriod = { start ->
                    viewModel.onIntent(CalendarIntent.CloseDaySheet)
                    editorMonth = start
                },
                onOpenLog = { onNavigate(Destination.Log(state.selected.toIso())) },
                colors = colors,
            )
        }

        // Rejected-edit toast (web actionError banner): fixed near the bottom, tap to dismiss.
        errorMessage?.let { message ->
            ErrorToast(message = message, onDismiss = { errorMessage = null }, colors = colors)
        }

        // Full-screen period-date editor overlay (web PeriodDateEditor) — VISUAL SHELL.
        editorMonth?.let { (y, m) ->
            PeriodDateEditorOverlay(
                openYear = y,
                openMonth = m,
                today = state.today,
                loggedDays = state.loggedPeriodDays,
                onClose = { editorMonth = null },
                colors = colors,
            )
        }
    }
}

@Composable
private fun CalendarHeader(editEnabled: Boolean, onEdit: () -> Unit, colors: RitmeColors) {
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.Top,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f)) {
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
        // "Edit Period Date" chip — disabled for a month entirely in the future.
        Row(
            modifier = Modifier
                .clip(RoundedCornerShape(20.dp))
                .background(colors.pinkContainer)
                .clickable(enabled = editEnabled, onClick = onEdit)
                .padding(horizontal = 14.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_pencil),
                contentDescription = null,
                tint = if (editEnabled) colors.pink else colors.pink.copy(alpha = 0.45f),
                modifier = Modifier.size(14.dp),
            )
            Spacer(Modifier.width(6.dp))
            Text(
                stringResource(R.string.calendar_edit_period_date),
                style = MaterialTheme.typography.labelMedium,
                color = if (editEnabled) colors.pink else colors.pink.copy(alpha = 0.45f),
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

@Composable
private fun RecalcBanner(colors: RitmeColors) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(colors.pinkContainer)
            .padding(horizontal = 12.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_sparkle),
            contentDescription = null,
            tint = colors.pink,
            modifier = Modifier.size(14.dp),
        )
        Spacer(Modifier.width(8.dp))
        Text(
            stringResource(R.string.calendar_recalculating),
            style = MaterialTheme.typography.labelMedium,
            color = colors.pink,
            fontWeight = FontWeight.Bold,
        )
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

    // Tapping the month title opens the quick month/year picker.
    var pickerOpen by remember { mutableStateOf(false) }
    if (pickerOpen) {
        MonthYearPickerDialog(
            year = state.year,
            month = state.month,
            onPick = { y, m ->
                pickerOpen = false
                onIntent(CalendarIntent.PickMonth(y, m))
            },
            onToday = {
                pickerOpen = false
                onIntent(CalendarIntent.JumpToToday)
            },
            onDismiss = { pickerOpen = false },
            colors = colors,
        )
    }

    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            // RTL: the start chevron goes back one month.
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.calendar_prev_month), {
                onIntent(CalendarIntent.PreviousMonth)
            })
            Row(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(10.dp))
                    .clickable { pickerOpen = true }
                    .padding(vertical = 4.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "${JalaliDate.MONTH_NAMES[state.month - 1]} ${state.year.toPersianDigits()}",
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center,
                )
                Spacer(Modifier.width(4.dp))
                Icon(
                    painter = painterResource(R.drawable.ic_chevron_down),
                    contentDescription = null,
                    tint = colors.ink,
                    modifier = Modifier.size(14.dp),
                )
            }
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

        // The grid renders immediately (no blocking spinner); colors fill in as data streams.
        Column(
            verticalArrangement = Arrangement.spacedBy(6.dp),
            // Horizontal swipe on the grid flips months, like the web calendar.
            // RTL: dragging toward the left (dx < 0) advances to the NEXT month.
            modifier = Modifier.pointerInput(Unit) {
                var dragged = 0f
                detectHorizontalDragGestures(
                    onDragStart = { dragged = 0f },
                    onHorizontalDrag = { _, dragAmount -> dragged += dragAmount },
                    onDragEnd = {
                        if (abs(dragged) > SWIPE_THRESHOLD_PX) {
                            onIntent(
                                if (dragged < 0) CalendarIntent.NextMonth else CalendarIntent.PreviousMonth,
                            )
                        }
                    },
                )
            },
        ) {
            visibleWeeks.forEach { week ->
                Row(Modifier.fillMaxWidth()) {
                    week.forEach { cell ->
                        Box(Modifier.weight(1f).padding(horizontal = 2.dp)) {
                            if (cell != null) {
                                DayCell(
                                    day = cell,
                                    snapshot = state.snapshotFor(cell),
                                    isToday = cell == state.today,
                                    isSelected = cell == state.selected,
                                    isLogged = state.isLoggedPeriodDay(cell),
                                    colors = colors,
                                    onClick = { onIntent(CalendarIntent.SelectDay(cell)) },
                                )
                            }
                        }
                    }
                }
            }
        }

        Spacer(Modifier.height(14.dp))
        Box(Modifier.fillMaxWidth().height(1.dp).background(colors.outline))
        Spacer(Modifier.height(14.dp))
        Legend(colors)
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
    isLogged: Boolean,
    colors: RitmeColors,
    onClick: () -> Unit,
) {
    val style = markerStyleFor(snapshot, colors)
    // A period marker the user hasn't actually logged is a prediction → hollow ring, not a fill.
    val isPredicted = snapshot?.phase == CyclePhase.MENSTRUATION && !isLogged
    val shape = RoundedCornerShape(14.dp)
    val fg = style?.fg ?: colors.ink
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(40.dp)
            .clip(shape)
            .background(if (style != null && !isPredicted) style.bg else Color.Transparent)
            .then(
                if (isPredicted && style != null) Modifier.border(1.5.dp, style.fg, shape) else Modifier,
            )
            .then(if (isSelected) Modifier.border(2.dp, colors.pink, shape) else Modifier)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = day.day.toPersianDigits(),
            style = MaterialTheme.typography.labelLarge,
            color = fg,
            fontWeight = FontWeight.Bold,
        )
        if (isToday) {
            Box(
                Modifier
                    .align(Alignment.BottomCenter)
                    .padding(bottom = 4.dp)
                    .size(4.dp)
                    .clip(CircleShape)
                    .background(style?.fg ?: colors.pink),
            )
        }
    }
}

/** Phase/marker → (bg, fg) styling, mirroring the web `MARKER_STYLE`. Null = no marker. */
private fun markerStyleFor(snapshot: CycleDaySnapshot?, colors: RitmeColors): MarkerStyle? = when {
    snapshot == null -> null
    snapshot.phase == CyclePhase.MENSTRUATION -> MarkerStyle(colors.periodContainer, colors.pink)
    snapshot.phase == CyclePhase.OVULATION -> MarkerStyle(colors.ovulationContainer, colors.success)
    snapshot.isFertileWindow -> MarkerStyle(colors.fertileContainer, colors.warning)
    snapshot.isPmsWindow -> MarkerStyle(colors.violetContainer, colors.accent)
    else -> null
}

/**
 * Quick month/year jump, opened by tapping the month title: year arrows over a 3×4 grid
 * of Jalali months, plus a "back to today" action.
 */
@Composable
private fun MonthYearPickerDialog(
    year: Int,
    month: Int,
    onPick: (Int, Int) -> Unit,
    onToday: () -> Unit,
    onDismiss: () -> Unit,
    colors: RitmeColors,
) {
    var browseYear by remember(year) { mutableIntStateOf(year) }

    Dialog(onDismissRequest = onDismiss) {
        Box(Modifier.clip(RoundedCornerShape(20.dp)).background(colors.surface).padding(16.dp)) {
            Column {
                Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    // RTL: the start chevron steps one year back.
                    HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.calendar_prev_year), {
                        browseYear -= 1
                    })
                    Text(
                        text = browseYear.toPersianDigits(),
                        style = MaterialTheme.typography.titleMedium,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.weight(1f),
                    )
                    HeaderIconButton(R.drawable.ic_chevron_left, stringResource(R.string.calendar_next_year), {
                        browseYear += 1
                    })
                }
                Spacer(Modifier.height(12.dp))
                for (row in 0 until 4) {
                    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        for (column in 0 until 3) {
                            val m = row * 3 + column + 1
                            val isCurrent = m == month && browseYear == year
                            Box(
                                modifier = Modifier
                                    .weight(1f)
                                    .clip(RoundedCornerShape(12.dp))
                                    .background(if (isCurrent) colors.pinkContainer else colors.background)
                                    .clickable { onPick(browseYear, m) }
                                    .padding(vertical = 10.dp),
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(
                                    text = JalaliDate.MONTH_NAMES[m - 1],
                                    style = MaterialTheme.typography.labelMedium,
                                    color = if (isCurrent) colors.pink else colors.ink,
                                    fontWeight = FontWeight.Bold,
                                )
                            }
                        }
                    }
                    Spacer(Modifier.height(8.dp))
                }
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(12.dp))
                        .background(colors.pinkContainer)
                        .clickable(onClick = onToday)
                        .padding(vertical = 10.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = stringResource(R.string.calendar_go_today),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
    }
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
        LegendItem(colors.accent, stringResource(R.string.calendar_legend_pms), colors)
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

// ── Day-detail sheet ───────────────────────────────────────────

@Composable
private fun DaySheetContent(
    state: CalendarUiState,
    onIntent: (CalendarIntent) -> Unit,
    onEditPeriod: (Pair<Int, Int>) -> Unit,
    onOpenLog: () -> Unit,
    colors: RitmeColors,
) {
    Column(
        Modifier
            .fillMaxWidth()
            .heightIn(max = 620.dp)
            .verticalScroll(rememberScrollState()),
    ) {
        DayDetailHeader(state, colors)
        Spacer(Modifier.height(14.dp))
        DetailTiles(state.selectedSnapshot, colors)
        PeriodActionsSection(state, onIntent, onEditPeriod, colors)
        // Future days can't carry logged data, so the log section is hidden (web §).
        if (!state.selectedIsFuture) {
            Spacer(Modifier.height(14.dp))
            DayLogCard(state, onOpenLog, colors)
        }
    }
}

@Composable
private fun DayDetailHeader(state: CalendarUiState, colors: RitmeColors) {
    val snapshot = state.selectedSnapshot
    // The leading drop dot is tinted by the day's marker, falling back to a neutral violet.
    val style = markerStyleFor(snapshot, colors)
        ?: MarkerStyle(colors.violetContainer, NEUTRAL_FG)
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.size(42.dp).clip(CircleShape).background(style.bg),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_drop),
                contentDescription = null,
                tint = style.fg,
                modifier = Modifier.size(20.dp),
            )
        }
        Spacer(Modifier.width(10.dp))
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
                Modifier.clip(RoundedCornerShape(20.dp)).background(colors.pinkContainer)
                    .padding(horizontal = 12.dp, vertical = 4.dp),
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
}

@Composable
private fun DetailTiles(snapshot: CycleDaySnapshot?, colors: RitmeColors) {
    val style = markerStyleFor(snapshot, colors) ?: MarkerStyle(colors.violetContainer, NEUTRAL_FG)
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        // Phase tile — filled/tinted by the phase, start-aligned colored value.
        Column(
            Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).background(style.bg).padding(horizontal = 12.dp, vertical = 10.dp),
        ) {
            Text(
                stringResource(R.string.calendar_phase_label),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(3.dp))
            Text(
                phaseLabel(snapshot),
                style = MaterialTheme.typography.bodyMedium,
                color = style.fg,
                fontWeight = FontWeight.Bold,
            )
        }
        // Chance tile — neutral, start-aligned steel value.
        Column(
            Modifier.weight(1f).clip(RoundedCornerShape(12.dp)).background(colors.background).padding(horizontal = 12.dp, vertical = 10.dp),
        ) {
            Text(
                stringResource(R.string.calendar_chance_label),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(3.dp))
            Text(
                chanceLabel(snapshot),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.steel,
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

/**
 * The quick period edits offered for the selected day (web day sheet). Inside a logged period
 * → "edit this period" + a remove/end action; just outside one → extend; far from any → start.
 */
@Composable
private fun PeriodActionsSection(
    state: CalendarUiState,
    onIntent: (CalendarIntent) -> Unit,
    onEditPeriod: (Pair<Int, Int>) -> Unit,
    colors: RitmeColors,
) {
    when (val action = state.selectedAction) {
        is PeriodDayAction.RemoveDay -> {
            val removeLabel = if (action is PeriodDayAction.TrimEnd) {
                stringResource(R.string.calendar_end_period_here)
            } else {
                stringResource(R.string.calendar_remove_period_day)
            }
            val startJalali = action.period.start.toJalali()
            Spacer(Modifier.height(14.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                // "Edit this period" — opens the paint editor (VISUAL SHELL) on the period's month.
                Row(
                    modifier = Modifier
                        .weight(1f)
                        .height(44.dp)
                        .clip(RoundedCornerShape(14.dp))
                        .background(colors.periodContainer)
                        .clickable { onEditPeriod(startJalali.year to startJalali.month) },
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_drop),
                        contentDescription = null,
                        tint = colors.pink,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        stringResource(R.string.calendar_edit_this_period),
                        style = MaterialTheme.typography.labelLarge,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
                RitmeSoftButton(
                    text = removeLabel,
                    onClick = { onIntent(CalendarIntent.ApplyPeriodAction) },
                    enabled = !state.periodSaving,
                    modifier = Modifier.weight(1f),
                    leadingIcon = {
                        Icon(
                            painter = painterResource(R.drawable.ic_x),
                            contentDescription = null,
                            tint = colors.steel,
                            modifier = Modifier.size(16.dp),
                        )
                    },
                )
            }
        }

        is PeriodDayAction.Extend -> {
            Spacer(Modifier.height(14.dp))
            RitmePrimaryButton(
                text = stringResource(R.string.calendar_add_to_period),
                onClick = { onIntent(CalendarIntent.ApplyPeriodAction) },
                enabled = !state.periodSaving,
                loading = state.periodSaving,
                leadingIcon = { DropIconOnPink(colors) },
            )
        }

        is PeriodDayAction.StartNew -> {
            Spacer(Modifier.height(14.dp))
            RitmePrimaryButton(
                text = stringResource(R.string.calendar_start_period_here),
                onClick = { onIntent(CalendarIntent.ApplyPeriodAction) },
                enabled = !state.periodSaving,
                loading = state.periodSaving,
                leadingIcon = { DropIconOnPink(colors) },
            )
        }

        PeriodDayAction.None -> Unit
    }
}

@Composable
private fun DropIconOnPink(colors: RitmeColors) {
    Icon(
        painter = painterResource(R.drawable.ic_drop),
        contentDescription = null,
        tint = colors.onPink,
        modifier = Modifier.size(16.dp),
    )
}

@Composable
private fun phaseLabel(snapshot: CycleDaySnapshot?): String = when {
    snapshot?.phase == CyclePhase.MENSTRUATION -> stringResource(R.string.calendar_phase_period)
    snapshot?.phase == CyclePhase.OVULATION -> stringResource(R.string.calendar_phase_ovulation)
    snapshot?.isFertileWindow == true -> stringResource(R.string.calendar_phase_fertile)
    snapshot?.isPmsWindow == true -> stringResource(R.string.calendar_phase_pms)
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
    val hasEntries = log != null
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(R.drawable.ic_pencil),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                stringResource(R.string.calendar_daylog_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            if (hasEntries) {
                Row(
                    modifier = Modifier
                        .clip(RoundedCornerShape(20.dp))
                        .background(colors.background)
                        .clickable(onClick = onEdit)
                        .padding(horizontal = 12.dp, vertical = 5.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_pencil),
                        contentDescription = null,
                        tint = colors.inkMuted,
                        modifier = Modifier.size(13.dp),
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(
                        stringResource(R.string.action_edit),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
        if (hasEntries) {
            Spacer(Modifier.height(14.dp))
            LogRecap(log, colors)
        } else {
            // Empty day → a start-aligned gradient prompt with a trailing FAB (web empty state).
            Spacer(Modifier.height(14.dp))
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(16.dp))
                    .background(Brush.horizontalGradient(listOf(colors.pinkContainer, colors.violetContainer)))
                    .clickable(onClick = onEdit)
                    .padding(14.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(
                        stringResource(R.string.calendar_daylog_empty_title),
                        style = MaterialTheme.typography.bodyMedium,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                    )
                    Spacer(Modifier.height(3.dp))
                    Text(
                        stringResource(R.string.calendar_daylog_empty_subtitle, state.selected.formatDayMonth()),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                }
                Spacer(Modifier.width(12.dp))
                Box(
                    Modifier
                        .size(42.dp)
                        .clip(CircleShape)
                        .background(Brush.verticalGradient(listOf(colors.pink, colors.accent))),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(R.drawable.ic_plus),
                        contentDescription = stringResource(R.string.action_edit),
                        tint = colors.onPink,
                        modifier = Modifier.size(20.dp),
                    )
                }
            }
        }
    }
}

/** Read-only grouped recap of a saved day log: per-category colored header + two-column rows. */
@Composable
private fun LogRecap(log: DailyHealthLog, colors: RitmeColors) {
    Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
        HealthLogCategory.entries.forEach { category ->
            val fields = log.values.keys
                .filter { it.category == category && hasRecapValue(log.values.getValue(it)) }
                .sortedBy { it.ordinal }
            if (fields.isEmpty()) return@forEach
            Column(Modifier.fillMaxWidth()) {
                Text(
                    stringResource(category.labelRes()),
                    style = MaterialTheme.typography.labelMedium,
                    color = categoryAccent(category, colors),
                    fontWeight = FontWeight.Bold,
                )
                Spacer(Modifier.height(8.dp))
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    fields.forEach { field ->
                        Row(
                            Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalAlignment = Alignment.Top,
                        ) {
                            Text(
                                stringResource(field.labelRes()),
                                style = MaterialTheme.typography.labelMedium,
                                color = colors.inkMuted,
                                modifier = Modifier.weight(1f),
                            )
                            Text(
                                text = recapValue(field, log.values.getValue(field)),
                                style = MaterialTheme.typography.labelMedium,
                                color = colors.ink,
                                fontWeight = FontWeight.Bold,
                                textAlign = TextAlign.End,
                            )
                        }
                    }
                }
            }
        }
    }
}

/** Presentation accent per log category (mirrors the web `CATEGORY_ACCENT`). */
private fun categoryAccent(category: HealthLogCategory, colors: RitmeColors): Color = when (category) {
    HealthLogCategory.BLEEDING -> colors.pink
    HealthLogCategory.PAIN -> colors.warning
    HealthLogCategory.DIGESTION -> colors.success
    HealthLogCategory.MOOD -> colors.accent
    HealthLogCategory.SLEEP -> Color(0xFF5B6BE1) // TODO token
    HealthLogCategory.BODY -> Color(0xFFE9662E) // TODO token
    HealthLogCategory.DISCHARGE -> Color(0xFF2E9BE9) // TODO token
    HealthLogCategory.INTIMATE -> Color(0xFF0E9C8A) // TODO token
    HealthLogCategory.SEXUAL -> Color(0xFFE9276E) // TODO token
    HealthLogCategory.MEASURE -> Color(0xFF6D7A87) // TODO token
    HealthLogCategory.NOTES -> colors.inkMuted
}

/** A recorded value worth showing — drops false toggles, empty multis and blank notes (web `fieldLine`). */
private fun hasRecapValue(value: HealthLogValue): Boolean = when (value) {
    is HealthLogValue.Toggle -> value.enabled
    is HealthLogValue.MultiChoice -> value.options.isNotEmpty()
    is HealthLogValue.Text -> value.text.isNotBlank()
    else -> true
}

/** The value half of a recap row (label is rendered separately, web `fieldLine`). */
@Composable
private fun recapValue(field: HealthLogField, value: HealthLogValue): String {
    // Capture the context so per-option labels can be resolved inside the non-composable
    // joinToString lambda (stringResource may only be called directly in a composable).
    val context = LocalContext.current
    return when (value) {
        is HealthLogValue.Choice ->
            optionLabelRes(field, value.option)?.let { stringResource(it) } ?: value.option

        is HealthLogValue.MultiChoice -> value.options.joinToString("، ") { option ->
            optionLabelRes(field, option)?.let { context.getString(it) } ?: option
        }

        is HealthLogValue.Toggle -> stringResource(R.string.log_yes)
        is HealthLogValue.Number -> value.value.toString().toPersianDigits()
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

// ── Rejected-edit toast ────────────────────────────────────────

@Composable
private fun androidx.compose.foundation.layout.BoxScope.ErrorToast(
    message: String,
    onDismiss: () -> Unit,
    colors: RitmeColors,
) {
    Box(
        Modifier
            .align(Alignment.BottomCenter)
            .fillMaxWidth()
            .padding(start = 16.dp, end = 16.dp, bottom = 88.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(colors.error)
            .clickable(onClick = onDismiss)
            .padding(horizontal = 16.dp, vertical = 12.dp),
    ) {
        Text(
            message,
            style = MaterialTheme.typography.labelLarge,
            color = colors.onPink,
            fontWeight = FontWeight.Bold,
        )
    }
}

// ── Full-screen "Edit Period Date" editor (VISUAL SHELL) ───────

/**
 * The web `PeriodDateEditor` as a visual shell: a pink-header, vertically-stacked multi-month
 * grid of independent day toggles pre-filled from the logged periods, with an order badge per
 * contiguous run and a cancel/save footer. Save is DEFERRED — reconciling the painted selection
 * into logged periods needs a new use case (`ManagePeriodsUseCase` only exposes `apply`), so
 * this closes without persisting; see the summary.
 */
@Composable
private fun PeriodDateEditorOverlay(
    openYear: Int,
    openMonth: Int,
    today: JalaliDate,
    loggedDays: Set<String>,
    onClose: () -> Unit,
    colors: RitmeColors,
) {
    val painted = remember { mutableStateOf(loggedDays) }
    val todayJdn = remember(today) { today.toJdn() }
    val months = remember(openYear, openMonth, today) { editorMonths(today, openYear, openMonth) }
    val orderMap = remember(painted.value) { runOrders(painted.value) }

    Box(Modifier.fillMaxSize().background(colors.surface)) {
        Column(Modifier.fillMaxSize()) {
            // Pink gradient header: close + centered title + weekday row.
            Column(
                Modifier
                    .fillMaxWidth()
                    .background(Brush.verticalGradient(listOf(colors.pinkDark, colors.pink)))
                    .padding(horizontal = 14.dp, vertical = 12.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        Modifier
                            .size(38.dp)
                            .clip(CircleShape)
                            .background(colors.ink.copy(alpha = 0.18f))
                            .clickable(onClick = onClose),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            painter = painterResource(R.drawable.ic_x),
                            contentDescription = stringResource(R.string.week_close),
                            tint = colors.onPink,
                            modifier = Modifier.size(20.dp),
                        )
                    }
                    Text(
                        stringResource(R.string.calendar_edit_period_date),
                        style = MaterialTheme.typography.titleMedium,
                        color = colors.onPink,
                        fontWeight = FontWeight.Bold,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.weight(1f).padding(end = 38.dp),
                    )
                }
                Spacer(Modifier.height(10.dp))
                Row(Modifier.fillMaxWidth()) {
                    WEEKDAY_RES.forEach { res ->
                        Text(
                            text = stringResource(res),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.onPink.copy(alpha = 0.85f),
                            textAlign = TextAlign.Center,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.weight(1f),
                        )
                    }
                }
            }

            // Continuous vertical scroll of months (newest at the bottom).
            LazyColumn(Modifier.weight(1f).fillMaxWidth()) {
                items(months, key = { "${it.first}-${it.second}" }) { (y, m) ->
                    EditorMonth(
                        year = y,
                        month = m,
                        painted = painted.value,
                        orderMap = orderMap,
                        todayJdn = todayJdn,
                        onToggle = { iso, jdn ->
                            if (jdn <= todayJdn) {
                                painted.value = painted.value.toMutableSet().apply {
                                    if (!add(iso)) remove(iso)
                                }
                            }
                        },
                        colors = colors,
                    )
                }
            }

            // Footer: hint + Cancel / Save.
            Column(Modifier.fillMaxWidth().background(colors.surface)) {
                Box(Modifier.fillMaxWidth().height(1.dp).background(colors.outline))
                Text(
                    stringResource(R.string.calendar_editor_hint),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.pink,
                    fontWeight = FontWeight.Bold,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth().background(colors.background).padding(vertical = 11.dp),
                )
                Row(
                    Modifier.fillMaxWidth().padding(14.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Box(
                        Modifier
                            .weight(1f)
                            .height(48.dp)
                            .clip(RoundedCornerShape(24.dp))
                            .border(1.5.dp, colors.pink, RoundedCornerShape(24.dp))
                            .clickable(onClick = onClose),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            stringResource(R.string.action_cancel),
                            style = MaterialTheme.typography.labelLarge,
                            color = colors.pink,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                    Box(
                        Modifier
                            .weight(1f)
                            .height(48.dp)
                            .clip(RoundedCornerShape(24.dp))
                            .background(colors.success)
                            // DEFERRED: reconcile-save needs a new use case; close for now.
                            .clickable(onClick = onClose),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            stringResource(R.string.action_save),
                            style = MaterialTheme.typography.labelLarge,
                            color = colors.onPink,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun EditorMonth(
    year: Int,
    month: Int,
    painted: Set<String>,
    orderMap: Map<String, Int>,
    todayJdn: Int,
    onToggle: (String, Int) -> Unit,
    colors: RitmeColors,
) {
    val weeks = remember(year, month) { JalaliDate.monthMatrix(year, month) }
    Column(Modifier.fillMaxWidth()) {
        Text(
            "${JalaliDate.MONTH_NAMES[month - 1]} ${year.toPersianDigits()}",
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.fillMaxWidth().background(colors.background).padding(horizontal = 16.dp, vertical = 12.dp),
        )
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 14.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            weeks.forEach { week ->
                Row(Modifier.fillMaxWidth()) {
                    week.forEach { cell ->
                        Box(Modifier.weight(1f).padding(horizontal = 2.dp), contentAlignment = Alignment.Center) {
                            if (cell != null) {
                                EditorDayCell(
                                    cell = cell,
                                    isSelected = cell.toIso() in painted,
                                    order = orderMap[cell.toIso()],
                                    isFuture = cell.toJdn() > todayJdn,
                                    isToday = cell.toJdn() == todayJdn,
                                    onToggle = { onToggle(cell.toIso(), cell.toJdn()) },
                                    colors = colors,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun EditorDayCell(
    cell: JalaliDate,
    isSelected: Boolean,
    order: Int?,
    isFuture: Boolean,
    isToday: Boolean,
    onToggle: () -> Unit,
    colors: RitmeColors,
) {
    val border = when {
        isSelected -> colors.pink
        isToday -> colors.ink
        else -> colors.outline
    }
    val fg = when {
        isFuture -> colors.outline
        isSelected -> colors.pink
        else -> colors.ink
    }
    Box(contentAlignment = Alignment.TopEnd) {
        Box(
            Modifier
                .size(42.dp)
                .clip(CircleShape)
                .border(1.5.dp, border, CircleShape)
                .clickable(enabled = !isFuture, onClick = onToggle),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                cell.day.toPersianDigits(),
                style = MaterialTheme.typography.labelLarge,
                color = fg,
                fontWeight = FontWeight.Bold,
            )
        }
        if (!isFuture) {
            Box(
                Modifier
                    .size(18.dp)
                    .clip(CircleShape)
                    .background(if (isSelected) colors.pink else colors.surface)
                    .then(if (isSelected) Modifier else Modifier.border(1.5.dp, colors.outline, CircleShape)),
                contentAlignment = Alignment.Center,
            ) {
                if (isSelected && order != null) {
                    Text(
                        order.toPersianDigits(),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.onPink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
    }
}

/** The Jalali months to stack in the editor: a little before the opened month, up to today. */
private fun editorMonths(today: JalaliDate, openYear: Int, openMonth: Int): List<Pair<Int, Int>> {
    fun ord(y: Int, m: Int) = y * 12 + (m - 1)
    val todayOrd = ord(today.year, today.month)
    val startOrd = min(ord(openYear, openMonth) - 1, todayOrd - EDITOR_MONTHS_BACK)
    return (startOrd..todayOrd).map { o -> (o / 12) to (o % 12 + 1) }
}

/** Per-day 1-based position within its own contiguous run (a one-day gap restarts at 1). */
private fun runOrders(selected: Set<String>): Map<String, Int> {
    val dates = selected.mapNotNull { iso ->
        val g = ir.ritmeapp.ritme.domain.model.GregorianDate.parseIso(iso) ?: return@mapNotNull null
        iso to g.toJalali().toJdn()
    }.sortedBy { it.second }
    val map = HashMap<String, Int>(dates.size)
    var n = 0
    var prevJdn: Int? = null
    for ((iso, jdn) in dates) {
        n = if (prevJdn != null && jdn == prevJdn + 1) n + 1 else 1
        map[iso] = n
        prevJdn = jdn
    }
    return map
}

private const val LOW_CHANCE_MAX = 10.0
private const val MEDIUM_CHANCE_MAX = 22.0

/** Horizontal drag past this many px flips the month (matches the web swipe feel). */
private const val SWIPE_THRESHOLD_PX = 90f

/** Rejected-edit toast lifetime, matching the web 5s auto-dismiss. */
private const val ERROR_TOAST_MS = 5000L

/** Months of history stacked above the opened month in the date editor (web MONTHS_BACK). */
private const val EDITOR_MONTHS_BACK = 12

/** Neutral drop-dot / phase-tile foreground for a day with no marker (web NEUTRAL_STYLE). */
private val NEUTRAL_FG = Color(0xFF7C7CF0) // TODO token
