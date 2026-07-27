package ir.ritmeapp.ritme.adapter.inbound.ui.home

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CyclePhase
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.DailyMessage
import ir.ritmeapp.ritme.domain.model.JalaliDate

private val WEEKDAY_RES = intArrayOf(
    R.string.week_sat, R.string.week_sun, R.string.week_mon, R.string.week_tue,
    R.string.week_wed, R.string.week_thu, R.string.week_fri,
)

private const val FERTILITY_RING_SIZE_DP = 72
private const val CALENDAR_SHADOW_ELEVATION_DP = 10

/**
 * The signature Home unit: a Jalali mini-calendar (expandable to the full month) sitting above the
 * pink cycle-status hero, as two separate rounded, pink-shadowed cards (Figma «TodayCalender» +
 * «NextPeriod»). Tapping a day updates the hero's "which day" indicator. Data comes from the derived
 * [predictions]; the phase-explanation box prefers the personalized [message]'s short line. When
 * there's no cycle data yet the card still renders its skeleton with placeholder dashes (web parity).
 */
@Composable
fun HomeCalendarCard(
    today: JalaliDate,
    predictions: CyclePredictions?,
    message: DailyMessage?,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    // The tapped day (defaults to today) drives only the hero's day indicator (web selectedDate).
    var selectedDay by remember(today) { mutableIntStateOf(today.day) }
    val selectedDate = remember(today, selectedDay) { JalaliDate(today.year, today.month, selectedDay) }
    val isToday = selectedDay == today.day

    Column(modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        ShadowCard(colors) {
            MiniCalendar(today, selectedDay, colors, onSelect = { selectedDay = it })
        }
        ShadowCard(colors) {
            CycleStatus(predictions, message, isToday, selectedDate, colors)
        }
    }
}

/** A 16dp-rounded card with a soft pink drop shadow (web `.card` + pink boxShadow). */
@Composable
private fun ShadowCard(colors: RitmeColors, content: @Composable () -> Unit) {
    val shape = RoundedCornerShape(16.dp)
    Box(
        Modifier
            .fillMaxWidth()
            .shadow(CALENDAR_SHADOW_ELEVATION_DP.dp, shape, ambientColor = colors.pink, spotColor = colors.pink)
            .clip(shape),
    ) { content() }
}

@Composable
private fun MiniCalendar(today: JalaliDate, selectedDay: Int, colors: RitmeColors, onSelect: (Int) -> Unit) {
    val weeks = remember(today.year, today.month) { JalaliDate.monthMatrix(today.year, today.month) }
    var open by remember { mutableStateOf(false) }
    val currentWeek = weeks.firstOrNull { week -> week.any { it?.day == selectedDay } } ?: weeks.first()

    Column(Modifier.fillMaxWidth().background(colors.surface).padding(14.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text("${today.monthName} ${today.year.toPersianDigits()}", style = MaterialTheme.typography.titleSmall, color = colors.pink)
            Row(
                modifier = Modifier.clip(RoundedCornerShape(8.dp)).clickable { open = !open }.padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Text(
                    text = stringResource(if (open) R.string.week_close else R.string.home_full_month),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                )
                Icon(
                    painter = painterResource(R.drawable.ic_chevron_down),
                    contentDescription = null,
                    tint = colors.inkMuted,
                    modifier = Modifier.size(14.dp).rotate(if (open) 180f else 0f),
                )
            }
        }
        Spacer(Modifier.height(10.dp))
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
        if (open) {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                weeks.forEach { week -> WeekRow(week, today.day, selectedDay, colors, onSelect) }
            }
        } else {
            WeekRow(currentWeek, today.day, selectedDay, colors, onSelect)
        }
    }
}

@Composable
private fun WeekRow(week: List<JalaliDate?>, todayDay: Int, selectedDay: Int, colors: RitmeColors, onSelect: (Int) -> Unit) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        week.forEach { cell ->
            Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                if (cell != null) {
                    DayCell(cell.day, cell.day == todayDay, cell.day == selectedDay && cell.day != todayDay, colors) { onSelect(cell.day) }
                }
            }
        }
    }
}

@Composable
private fun DayCell(day: Int, isToday: Boolean, isSelected: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    val bg = when {
        isToday -> colors.pink
        isSelected -> colors.pinkContainer
        else -> colors.background
    }
    val fg = when {
        isToday -> colors.onPink
        isSelected -> colors.pink
        else -> colors.ink
    }
    Box(
        modifier = Modifier
            .size(34.dp)
            .clip(CircleShape)
            .background(bg)
            .then(if (isSelected) Modifier.border(2.dp, colors.pink, CircleShape) else Modifier)
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(day.toPersianDigits(), style = MaterialTheme.typography.labelLarge, color = fg, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun CycleStatus(
    predictions: CyclePredictions?,
    message: DailyMessage?,
    isToday: Boolean,
    selectedDate: JalaliDate,
    colors: RitmeColors,
) {
    var expanded by remember { mutableStateOf(true) }
    val dash = stringResource(R.string.home_unavailable)
    val daysValue = predictions?.let { stringResource(R.string.cycle_days, it.daysUntilNextPeriod.toPersianDigits()) } ?: dash
    val fertility = predictions?.let { "${it.fertilityPercent.toPersianDigits()}٪" } ?: dash
    val progress = predictions?.progressPercent ?: 0
    val phaseDesc = message?.shortMessage?.takeIf { it.isNotBlank() } ?: stringResource(R.string.home_phase_desc)

    Column(
        Modifier
            .fillMaxWidth()
            .background(Brush.linearGradient(listOf(colors.pink, colors.pinkLight)))
            .padding(12.dp),
    ) {
        // Which day the info below reflects — updates as the user taps a day.
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Icon(painterResource(R.drawable.ic_calendar), null, tint = colors.onPink, modifier = Modifier.size(14.dp))
            Text(
                text = if (isToday) stringResource(R.string.home_selected_today)
                else stringResource(R.string.home_selected_date, selectedDate.formatDayMonth()),
                style = MaterialTheme.typography.labelMedium,
                color = colors.onPink,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(Modifier.height(8.dp))

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.Top) {
            Column(Modifier.weight(1f)) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Icon(painterResource(R.drawable.ic_calendar), null, tint = colors.onPink, modifier = Modifier.size(18.dp))
                    Text(stringResource(R.string.cycle_next_period_label), style = MaterialTheme.typography.bodyMedium, color = colors.onPink)
                }
                Spacer(Modifier.height(10.dp))
                Text(daysValue, style = MaterialTheme.typography.headlineSmall, color = colors.onPink, fontWeight = FontWeight.Bold)
                if (predictions != null) {
                    Spacer(Modifier.height(10.dp))
                    Text(
                        text = stringResource(R.string.home_next_period_start, selectedDate.addDays(predictions.daysUntilNextPeriod).formatDayMonth()),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.onPink,
                    )
                }
            }
            FertilityRing(fertility, phaseLabelFor(predictions?.phase, message), colors)
        }

        Spacer(Modifier.height(14.dp))
        ProgressBar(progress, colors)

        if (expanded) {
            Spacer(Modifier.height(12.dp))
            Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(8.dp)).background(colors.phasePink).padding(horizontal = 10.dp, vertical = 8.dp)) {
                Text(phaseDesc, style = MaterialTheme.typography.labelMedium, color = colors.onPink, textAlign = TextAlign.Start)
            }
        }

        Spacer(Modifier.height(10.dp))
        Row(
            modifier = Modifier.fillMaxWidth().clickable { expanded = !expanded },
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = if (expanded) stringResource(R.string.cycle_show_less) else stringResource(R.string.home_show_more),
                style = MaterialTheme.typography.labelMedium,
                color = colors.onPink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.size(4.dp))
            Icon(
                painter = painterResource(R.drawable.ic_chevron_down),
                contentDescription = null,
                tint = colors.onPink,
                modifier = Modifier.size(14.dp).rotate(if (expanded) 0f else 180f),
            )
        }
    }
}

/** Resolves the current-phase label line ("فاز فعلی: …"), preferring the enum, then the message. */
@Composable
private fun phaseLabelFor(phase: CyclePhase?, message: DailyMessage?): String {
    val label = when (phase) {
        CyclePhase.MENSTRUATION -> stringResource(R.string.home_phase_label_menstruation)
        CyclePhase.FOLLICULAR -> stringResource(R.string.home_phase_label_follicular)
        CyclePhase.OVULATION -> stringResource(R.string.home_phase_label_ovulation)
        CyclePhase.LUTEAL -> stringResource(R.string.home_phase_label_luteal)
        null -> message?.phaseLabel
    }
    return label?.let { stringResource(R.string.home_current_phase, it) } ?: stringResource(R.string.home_current_phase_default)
}

@Composable
private fun FertilityRing(fertility: String, phaseLabel: String, colors: RitmeColors) {
    Column(horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(phaseLabel, style = MaterialTheme.typography.labelSmall, color = colors.onPink, fontWeight = FontWeight.Bold)
        Box(
            modifier = Modifier.size(FERTILITY_RING_SIZE_DP.dp).clip(CircleShape).border(3.dp, colors.onPink, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(fertility, style = MaterialTheme.typography.titleMedium, color = colors.onPink, fontWeight = FontWeight.Bold)
                Text(stringResource(R.string.home_fertility_short), style = MaterialTheme.typography.labelSmall, color = colors.onPink)
            }
        }
    }
}

@Composable
private fun ProgressBar(percent: Int, colors: RitmeColors) {
    Box(Modifier.fillMaxWidth().height(10.dp).clip(RoundedCornerShape(99.dp)).background(colors.onPink.copy(alpha = 0.3f))) {
        Box(Modifier.fillMaxWidth(percent / 100f).height(10.dp).clip(RoundedCornerShape(99.dp)).background(colors.onPink))
    }
}
