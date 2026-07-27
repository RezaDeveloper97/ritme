package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.CycleWellbeing
import ir.ritmeapp.ritme.domain.model.DailyMessage
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderType
import kotlin.math.max

/** A white rounded surface card — the shared container for every dashboard section. */
@Composable
internal fun HomeCard(colors: RitmeColors, content: @Composable ColumnScope.() -> Unit) {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .padding(16.dp),
        content = content,
    )
}

/** Section title with an optional trailing "view all" action (web `SectionHead`). */
@Composable
private fun SectionHeader(title: String, colors: RitmeColors, action: String? = null) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
        Text(title, style = MaterialTheme.typography.titleMedium, color = colors.ink)
        if (action != null) {
            Text(action, style = MaterialTheme.typography.labelMedium, color = colors.pink, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
internal fun PhaseRowsSection(predictions: CyclePredictions?, today: JalaliDate) {
    val colors = LocalRitmeColors.current
    val dash = stringResource(R.string.home_unavailable)
    val rows = listOf(
        PhaseRow(stringResource(R.string.home_phase_window), predictions?.let { offsetDate(today, max(0, it.daysUntilFertileWindow)) } ?: dash, colors.warning, colors.fertileContainer),
        PhaseRow(stringResource(R.string.home_phase_ovulation), predictions?.let { offsetDate(today, max(0, it.daysUntilOvulation)) } ?: dash, colors.greenDot, colors.mintContainer),
        PhaseRow(
            stringResource(R.string.home_phase_pms),
            predictions?.let { "${offsetDate(today, it.daysUntilPmsStart)} - ${offsetDate(today, it.daysUntilPmsEnd)}" } ?: dash,
            colors.accent,
            colors.violetContainer,
        ),
        PhaseRow(stringResource(R.string.home_phase_next_period), predictions?.let { offsetDate(today, it.daysUntilNextPeriod) } ?: dash, colors.pinkLight, colors.periodContainer),
    )
    HomeCard(colors) {
        rows.forEachIndexed { i, row ->
            if (i > 0) Spacer(Modifier.height(14.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Box(Modifier.size(30.dp).clip(CircleShape).background(row.tint), contentAlignment = Alignment.Center) {
                        Icon(painterResource(R.drawable.ic_drop_solid), null, tint = row.color, modifier = Modifier.size(16.dp))
                    }
                    Text(row.label, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted, fontWeight = FontWeight.Bold)
                }
                Box(Modifier.clip(RoundedCornerShape(12.dp)).background(row.tint).padding(horizontal = 14.dp, vertical = 8.dp)) {
                    Text(row.date, style = MaterialTheme.typography.labelMedium, color = colors.steel, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

private data class PhaseRow(val label: String, val date: String, val color: Color, val tint: Color)

@Composable
internal fun RecommendationsSection(message: DailyMessage?) {
    val colors = LocalRitmeColors.current
    val items: List<Pair<String, String?>> = message?.dos?.take(3)?.takeIf { it.isNotEmpty() }?.map { it to null }
        ?: listOf(
            stringResource(R.string.home_reco_iron) to stringResource(R.string.home_reco_iron_desc),
            stringResource(R.string.home_reco_omega) to stringResource(R.string.home_reco_omega_desc),
        )
    HomeCard(colors) {
        Text(stringResource(R.string.reco_title), style = MaterialTheme.typography.titleMedium, color = colors.pink)
        Spacer(Modifier.height(12.dp))
        items.forEachIndexed { i, (title, desc) ->
            if (i > 0) Spacer(Modifier.height(8.dp))
            Row(
                Modifier
                    .fillMaxWidth()
                    .clip(RoundedCornerShape(8.dp))
                    .background(Brush.horizontalGradient(listOf(colors.pinkContainer, colors.mintContainer)))
                    .padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Box(Modifier.size(36.dp).clip(CircleShape).background(colors.surface), contentAlignment = Alignment.Center) {
                    Icon(painterResource(R.drawable.ic_check), null, tint = colors.greenDot, modifier = Modifier.size(18.dp))
                }
                Column(Modifier.weight(1f)) {
                    Text(title, style = MaterialTheme.typography.bodyMedium, color = colors.ink, textAlign = TextAlign.Start)
                    if (desc != null) {
                        Spacer(Modifier.height(4.dp))
                        Text(desc, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted, textAlign = TextAlign.Start)
                    }
                }
            }
        }
    }
}

@Composable
internal fun SmartTipSection(message: DailyMessage?) {
    val colors = LocalRitmeColors.current
    val body = message?.longMessage?.takeIf { it.isNotBlank() } ?: stringResource(R.string.home_smarttip_body)
    val quote = message?.actionSuggestion?.takeIf { it.isNotBlank() } ?: stringResource(R.string.home_smarttip_quote)
    HomeCard(colors) {
        Text(stringResource(R.string.smarttip_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(10.dp))
        Text(body, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted, textAlign = TextAlign.Start)
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .background(Brush.horizontalGradient(listOf(colors.pinkContainer, colors.violetContainer)))
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(painterResource(R.drawable.ic_sparkle), null, tint = colors.pink, modifier = Modifier.size(20.dp))
            Text(quote, style = MaterialTheme.typography.labelLarge, color = colors.ink, modifier = Modifier.weight(1f), textAlign = TextAlign.Start)
        }
    }
}

/** The kinds of item the add row offers, mirroring the web `DayTasks` type picker. */
private val ADD_TYPES = listOf(
    ReminderType.CUSTOM to R.string.daytasks_type_custom,
    ReminderType.DOCTOR to R.string.home_daytasks_type_doctor,
    ReminderType.MEDICATION to R.string.home_daytasks_type_medication,
)

/**
 * «کارها و یادآورهای این روز» — the interactive day-planner (web `DayTasks`): a type picker, a text
 * field + add button, and per-item rows with an icon bubble, subtitle, done-toggle and delete. Same
 * `/reminders` source as the daily-log planner, so items set on either surface appear on the other.
 * "Done" maps to an inactive reminder (the API has no separate completed flag).
 */
@Composable
internal fun DayTasksSection(
    items: List<Reminder>,
    onAdd: (ReminderType, String) -> Unit,
    onToggle: (Reminder) -> Unit,
    onDelete: (Reminder) -> Unit,
) {
    val colors = LocalRitmeColors.current
    var draft by remember { mutableStateOf("") }
    var type by remember { mutableStateOf(ReminderType.CUSTOM) }
    val done = items.count { !it.isActive }
    val submit = {
        if (draft.isNotBlank()) {
            onAdd(type, draft)
            draft = ""
        }
    }
    HomeCard(colors) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(stringResource(R.string.daytasks_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
            if (items.isNotEmpty()) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        stringResource(R.string.daytasks_progress, done.toPersianDigits(), items.size.toPersianDigits()),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.steel,
                        fontWeight = FontWeight.Bold,
                    )
                    Icon(painterResource(R.drawable.ic_check_circle), null, tint = colors.greenDot, modifier = Modifier.size(16.dp))
                }
            }
        }
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.daytasks_subtitle), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted, textAlign = TextAlign.Start)
        Spacer(Modifier.height(12.dp))

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            ADD_TYPES.forEach { (value, labelRes) ->
                TypeChip(value == type, value.chipIconRes(), stringResource(labelRes), colors) { type = value }
            }
        }
        Spacer(Modifier.height(8.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier
                    .weight(1f)
                    .height(42.dp)
                    .clip(RoundedCornerShape(12.dp))
                    .background(colors.surface)
                    .border(1.dp, colors.outline, RoundedCornerShape(12.dp))
                    .padding(horizontal = 12.dp),
                contentAlignment = Alignment.CenterStart,
            ) {
                BasicTextField(
                    value = draft,
                    onValueChange = { draft = it.take(TASK_MAX_LENGTH) },
                    singleLine = true,
                    textStyle = MaterialTheme.typography.bodyMedium.copy(color = colors.ink),
                    cursorBrush = SolidColor(colors.pink),
                    keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                    keyboardActions = KeyboardActions(onDone = { submit() }),
                    decorationBox = { inner ->
                        if (draft.isEmpty()) {
                            Text(stringResource(R.string.daytasks_placeholder), style = MaterialTheme.typography.bodyMedium, color = colors.placeholder)
                        }
                        inner()
                    },
                )
            }
            Box(Modifier.width(84.dp)) {
                RitmePrimaryButton(text = stringResource(R.string.daytasks_add), onClick = submit, enabled = draft.isNotBlank())
            }
        }
        Spacer(Modifier.height(if (items.isEmpty()) 8.dp else 12.dp))

        if (items.isEmpty()) {
            Text(
                stringResource(R.string.daytasks_empty),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        } else {
            items.forEach { item -> TaskRow(item, colors, onToggle, onDelete) }
        }
    }
}

@Composable
private fun TypeChip(selected: Boolean, @DrawableRes icon: Int, label: String, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        Modifier
            .clip(RoundedCornerShape(10.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(1.dp, if (selected) colors.pink else colors.outline, RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Icon(painterResource(icon), null, tint = if (selected) colors.pink else colors.steel, modifier = Modifier.size(14.dp))
        Text(label, style = MaterialTheme.typography.labelMedium, color = if (selected) colors.pink else colors.steel, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun TaskRow(item: Reminder, colors: RitmeColors, onToggle: (Reminder) -> Unit, onDelete: (Reminder) -> Unit) {
    val done = !item.isActive
    Row(
        Modifier.fillMaxWidth().padding(vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Box(Modifier.size(32.dp).clip(CircleShape).background(colors.periodContainer), contentAlignment = Alignment.Center) {
            Icon(painterResource(item.type.chipIconRes()), null, tint = colors.pink, modifier = Modifier.size(16.dp))
        }
        Column(Modifier.weight(1f)) {
            Text(
                item.title,
                style = MaterialTheme.typography.bodyMedium,
                color = if (done) colors.inkMuted else colors.ink,
                textDecoration = if (done) TextDecoration.LineThrough else null,
                textAlign = TextAlign.Start,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            val subtitle = item.subtitle
            if (!subtitle.isNullOrBlank()) {
                Text(subtitle, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted, textAlign = TextAlign.Start)
            }
        }
        Box(
            Modifier
                .size(22.dp)
                .clip(RoundedCornerShape(7.dp))
                .background(if (done) colors.pink else colors.surface)
                .border(1.5.dp, if (done) colors.pink else colors.outline, RoundedCornerShape(7.dp))
                .clickable { onToggle(item) },
            contentAlignment = Alignment.Center,
        ) {
            if (done) Icon(painterResource(R.drawable.ic_check), null, tint = colors.onPink, modifier = Modifier.size(14.dp))
        }
        Icon(
            painterResource(R.drawable.ic_trash),
            contentDescription = stringResource(R.string.home_daytasks_delete),
            tint = colors.inkMuted,
            modifier = Modifier.size(20.dp).clip(CircleShape).clickable { onDelete(item) }.padding(2.dp),
        )
    }
}

@DrawableRes
private fun ReminderType.chipIconRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.drawable.ic_stetho
    ReminderType.MEDICATION -> R.drawable.ic_pill
    ReminderType.APPOINTMENT -> R.drawable.ic_calendar
    ReminderType.CUSTOM -> R.drawable.ic_pencil
}

@Composable
internal fun MyCyclesSection(predictions: CyclePredictions?, today: JalaliDate) {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        SectionHeader(stringResource(R.string.cycles_title), colors)
        Spacer(Modifier.height(12.dp))
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Box(Modifier.size(40.dp).clip(CircleShape).background(colors.periodContainer), contentAlignment = Alignment.Center) {
                Icon(painterResource(R.drawable.ic_drop_solid), null, tint = colors.pink, modifier = Modifier.size(18.dp))
            }
            Column {
                Text(
                    text = predictions?.let { stringResource(R.string.home_cycles_current_day, it.cycleDay.toPersianDigits()) }
                        ?: stringResource(R.string.home_cycles_current_day_default),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                if (predictions != null) {
                    Spacer(Modifier.height(3.dp))
                    Text(
                        text = stringResource(R.string.home_cycles_started_on, offsetDate(today, -(predictions.cycleDay - 1))),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                }
            }
        }
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth().height(1.dp).background(colors.outline))
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.Center, verticalAlignment = Alignment.CenterVertically) {
            Icon(painterResource(R.drawable.ic_plus), null, tint = colors.pink, modifier = Modifier.size(18.dp))
            Spacer(Modifier.size(8.dp))
            Text(stringResource(R.string.home_cycles_add_previous), style = MaterialTheme.typography.bodyMedium, color = colors.pink, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
internal fun CycleSummarySection(predictions: CyclePredictions?, variability: String?) {
    val colors = LocalRitmeColors.current
    val dash = stringResource(R.string.home_unavailable)
    // Same real cycle facts the /cycle screen's summary card shows — length, ovulation day, regularity.
    val rows = listOf(
        stringResource(R.string.cyclesummary_length) to
            (predictions?.let { stringResource(R.string.cycle_days, it.cycleLength.toPersianDigits()) } ?: dash),
        stringResource(R.string.cyclesummary_ovulation) to
            (predictions?.let {
                stringResource(R.string.cycle_summary_day_n, (it.cycleDay + it.daysUntilOvulation).toPersianDigits())
            } ?: dash),
        stringResource(R.string.cycle_variability_label) to variabilityLabel(variability),
    )
    HomeCard(colors) {
        SectionHeader(stringResource(R.string.cyclesummary_title), colors)
        Spacer(Modifier.height(12.dp))
        Column(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .border(1.dp, colors.outline, RoundedCornerShape(12.dp))
                .padding(horizontal = 14.dp),
        ) {
            rows.forEachIndexed { i, (label, value) ->
                Row(
                    Modifier.fillMaxWidth().padding(vertical = 12.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(label, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted, fontWeight = FontWeight.Bold)
                    Text(value, style = MaterialTheme.typography.bodyMedium, color = colors.steel)
                }
                if (i < rows.lastIndex) Box(Modifier.fillMaxWidth().height(1.dp).background(colors.outline))
            }
        }
        Spacer(Modifier.height(14.dp))
        RitmePrimaryButton(text = stringResource(R.string.cyclesummary_view_more), onClick = {})
    }
}

@Composable
internal fun StartPeriodSection(status: StartPeriodStatus, onTap: () -> Unit) {
    val colors = LocalRitmeColors.current
    val confirming = status == StartPeriodStatus.CONFIRMING
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(if (confirming) colors.pink else colors.surface)
            .clickable(enabled = status != StartPeriodStatus.PENDING, onClick = onTap)
            .padding(vertical = 14.dp),
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(
            painter = painterResource(R.drawable.ic_drop_solid),
            contentDescription = null,
            tint = if (confirming) colors.onPink else colors.pink,
            modifier = Modifier.size(18.dp),
        )
        Spacer(Modifier.size(8.dp))
        Text(
            text = when (status) {
                StartPeriodStatus.CONFIRMING -> stringResource(R.string.start_period_confirm)
                StartPeriodStatus.PENDING -> stringResource(R.string.start_period_pending)
                StartPeriodStatus.ERROR -> stringResource(R.string.start_period_error)
                StartPeriodStatus.IDLE -> stringResource(R.string.start_period)
            },
            style = MaterialTheme.typography.bodyMedium,
            color = when (status) {
                StartPeriodStatus.CONFIRMING -> colors.onPink
                StartPeriodStatus.ERROR -> colors.error
                else -> colors.pink
            },
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
internal fun ChallengeSection() {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Icon(painterResource(R.drawable.ic_flame), null, tint = colors.warning, modifier = Modifier.size(18.dp))
            Text(stringResource(R.string.challenge_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        }
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(8.dp))
                .border(2.dp, colors.success, RoundedCornerShape(8.dp))
                .padding(12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Icon(painterResource(R.drawable.ic_thermo), null, tint = colors.pink, modifier = Modifier.size(18.dp))
                Text(stringResource(R.string.challenge_item), style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
            }
            Box(Modifier.size(22.dp).clip(RoundedCornerShape(7.dp)).border(2.dp, colors.outline, RoundedCornerShape(7.dp)))
        }
        Spacer(Modifier.height(10.dp))
        GradientCaption(stringResource(R.string.challenge_desc), colors)
    }
}

/**
 * «خلاصه هفته» — mood/sleep/energy projected from the current cycle phase via the [CycleWellbeing]
 * engine (there is no per-day wellbeing signal in the API, so these are typical-for-this-phase
 * estimates). Renders placeholder dashes until the phase is known.
 */
@Composable
internal fun WeekSummarySection(predictions: CyclePredictions?) {
    val colors = LocalRitmeColors.current
    val known = predictions?.phase != null
    val wellbeing = CycleWellbeing.from(predictions?.phase)
    val dash = stringResource(R.string.home_unavailable)
    val mood = if (known) stringResource(R.string.home_percent, wellbeing.moodPercent.toPersianDigits()) else dash
    val sleep = if (known) stringResource(R.string.home_percent, wellbeing.sleepPercent.toPersianDigits()) else dash
    val energy = if (known) stringResource(R.string.home_percent, wellbeing.energyPercent.toPersianDigits()) else dash
    HomeCard(colors) {
        SectionHeader(stringResource(R.string.weeksummary_title), colors, action = stringResource(R.string.home_view_all))
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.weeksummary_subtitle), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted, textAlign = TextAlign.Start)
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            WeekSummaryTile(R.drawable.ic_smile, R.string.weeksummary_mood, mood, colors.pinkLight, colors, Modifier.weight(1f))
            WeekSummaryTile(R.drawable.ic_moon, R.string.weeksummary_sleep, sleep, colors.info, colors, Modifier.weight(1f))
            WeekSummaryTile(R.drawable.ic_zap, R.string.weeksummary_energy, energy, colors.warning, colors, Modifier.weight(1f))
        }
        Spacer(Modifier.height(12.dp))
        GradientCaption(stringResource(R.string.challenge_desc), colors)
    }
}

@Composable
private fun WeekSummaryTile(
    @DrawableRes icon: Int,
    @StringRes label: Int,
    value: String,
    tint: Color,
    colors: RitmeColors,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.clip(RoundedCornerShape(12.dp)).background(colors.background).padding(vertical = 12.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Box(Modifier.size(36.dp).clip(CircleShape).background(colors.surface), contentAlignment = Alignment.Center) {
            Icon(painterResource(icon), null, tint = tint, modifier = Modifier.size(20.dp))
        }
        Text(stringResource(label), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted, fontWeight = FontWeight.Bold)
        Text(value, style = MaterialTheme.typography.labelMedium, color = colors.ink, fontWeight = FontWeight.Bold)
    }
}

/**
 * «وضعیت امروز» — the same headline cycle facts the /cycle screen shows (current phase, today's
 * fertility, cycle day), in the Home card's three-tile design. Falls back to dashes with no cycle.
 */
@Composable
internal fun TodayStatusSection(predictions: CyclePredictions?, message: DailyMessage?) {
    val colors = LocalRitmeColors.current
    val dash = stringResource(R.string.home_unavailable)
    val phaseValue = phaseLabelOf(predictions?.phase, message) ?: dash
    val fertilityValue = predictions?.let { stringResource(R.string.home_percent, it.fertilityPercent.toPersianDigits()) } ?: dash
    val cycleDayValue = predictions?.let { stringResource(R.string.cycle_summary_day_n, it.cycleDay.toPersianDigits()) } ?: dash
    HomeCard(colors) {
        SectionHeader(stringResource(R.string.todaystatus_title), colors, action = stringResource(R.string.home_view_all))
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            TodayStatusTile(R.drawable.ic_drop_solid, stringResource(R.string.todaystatus_phase), phaseValue, colors.pinkLight, colors, Modifier.weight(1f))
            TodayStatusTile(R.drawable.ic_heart, stringResource(R.string.todaystatus_fertility), fertilityValue, colors.warning, colors, Modifier.weight(1f))
            TodayStatusTile(R.drawable.ic_calendar, stringResource(R.string.todaystatus_cycle_day), cycleDayValue, colors.info, colors, Modifier.weight(1f))
        }
    }
}

@Composable
private fun TodayStatusTile(
    @DrawableRes icon: Int,
    label: String,
    value: String,
    tint: Color,
    colors: RitmeColors,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(12.dp))
            .padding(vertical = 10.dp, horizontal = 8.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(painterResource(icon), null, tint = tint, modifier = Modifier.size(22.dp))
        Text(label, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted, fontWeight = FontWeight.Bold, textAlign = TextAlign.Center)
        Box(Modifier.clip(RoundedCornerShape(4.dp)).background(colors.cyanContainer).padding(horizontal = 8.dp, vertical = 2.dp)) {
            Text(value, style = MaterialTheme.typography.labelSmall, color = colors.cyanInk, fontWeight = FontWeight.Bold, maxLines = 1, overflow = TextOverflow.Ellipsis)
        }
    }
}

/** Resolves the phase display label, preferring the enum then the personalized message's label. */
@Composable
private fun phaseLabelOf(phase: CyclePhase?, message: DailyMessage?): String? = when (phase) {
    CyclePhase.MENSTRUATION -> stringResource(R.string.phase_menstruation)
    CyclePhase.FOLLICULAR -> stringResource(R.string.phase_follicular)
    CyclePhase.OVULATION -> stringResource(R.string.phase_ovulation)
    CyclePhase.LUTEAL -> stringResource(R.string.phase_luteal)
    null -> message?.phaseLabel
}

/** Maps the backend regularity verdict to its Persian label (regular / irregular / assessing). */
@Composable
private fun variabilityLabel(variability: String?): String = when (variability) {
    "regular" -> stringResource(R.string.cycle_variability_regular)
    "irregular" -> stringResource(R.string.cycle_variability_irregular)
    else -> stringResource(R.string.cycle_variability_unknown)
}

@Composable
internal fun ArticlesSection() {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Text(stringResource(R.string.articles_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(16.dp))
        Row(
            Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            repeat(ARTICLE_COUNT) {
                Column(Modifier.width(162.dp)) {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .height(130.dp)
                            .clip(RoundedCornerShape(12.dp))
                            .background(Brush.linearGradient(listOf(colors.pinkContainer, colors.violetContainer))),
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(painterResource(R.drawable.ic_book_open), null, tint = colors.pink, modifier = Modifier.size(30.dp))
                    }
                    Spacer(Modifier.height(8.dp))
                    Text(
                        stringResource(R.string.articles_article1),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                        maxLines = 2,
                        textAlign = TextAlign.Start,
                    )
                    Spacer(Modifier.height(6.dp))
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(5.dp)) {
                        Icon(painterResource(R.drawable.ic_book_open), null, tint = colors.ink, modifier = Modifier.size(16.dp))
                        Text(stringResource(R.string.articles_min, ARTICLE_MINUTES.toPersianDigits()), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
                    }
                }
            }
        }
        Spacer(Modifier.height(18.dp))
        RitmePrimaryButton(text = stringResource(R.string.articles_read_more), onClick = {})
    }
}

/** The soft pink→violet "✨ …" caption strip shared by the challenge and week-summary cards. */
@Composable
private fun GradientCaption(text: String, colors: RitmeColors) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .background(Brush.horizontalGradient(listOf(colors.pinkContainer, colors.violetContainer)))
            .padding(horizontal = 12.dp, vertical = 10.dp),
        horizontalArrangement = Arrangement.Center,
    ) {
        Text("✨ $text", style = MaterialTheme.typography.labelSmall, color = colors.ink, textAlign = TextAlign.Center)
    }
}

/** Formats "today + n days" as a Jalali "day month" label. */
private fun offsetDate(today: JalaliDate, days: Int): String = today.addDays(days).formatDayMonth()

private const val TASK_MAX_LENGTH = 120
private const val ARTICLE_COUNT = 3
private const val ARTICLE_MINUTES = 9
