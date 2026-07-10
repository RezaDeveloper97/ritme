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
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import androidx.compose.ui.res.stringResource
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CyclePredictions
import ir.ritmeapp.ritme.domain.model.DailyMessage
import ir.ritmeapp.ritme.domain.model.JalaliDate

private val WEEKDAY_RES = intArrayOf(
    R.string.week_sat, R.string.week_sun, R.string.week_mon, R.string.week_tue,
    R.string.week_wed, R.string.week_thu, R.string.week_fri,
)

/**
 * The signature Home unit: a Jalali mini-calendar (expandable to the full month) sitting directly on
 * top of the pink cycle-status card, as one rounded, shadowed block (Figma «TodayCalender» +
 * «NextPeriod»). Data comes from the derived [predictions]; the phase description prefers the
 * personalized [message]. When there's no cycle data yet, the card shows a calm onboarding hint.
 */
@Composable
fun HomeCalendarCard(
    today: JalaliDate,
    predictions: CyclePredictions?,
    message: DailyMessage?,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp)),
    ) {
        MiniCalendar(today, colors)
        CycleStatus(predictions, message, colors)
    }
}

@Composable
private fun MiniCalendar(today: JalaliDate, colors: RitmeColors) {
    val weeks = remember(today.year, today.month) { JalaliDate.monthMatrix(today.year, today.month) }
    var open by remember { mutableStateOf(false) }
    var selectedDay by remember { mutableIntStateOf(today.day) }
    val currentWeek = weeks.firstOrNull { week -> week.any { it?.day == selectedDay } } ?: weeks.first()

    Column(Modifier.fillMaxWidth().background(colors.surface).padding(14.dp)) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text("${today.monthName} ${today.year.toPersianDigits()}", style = MaterialTheme.typography.titleSmall, color = colors.pink)
            Text(
                text = stringResource(if (open) R.string.week_close else R.string.week_full_month),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                modifier = Modifier.clip(RoundedCornerShape(8.dp)).clickable { open = !open }.padding(horizontal = 8.dp, vertical = 4.dp),
            )
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
                weeks.forEach { week -> WeekRow(week, today.day, selectedDay, colors) { selectedDay = it } }
            }
        } else {
            WeekRow(currentWeek, today.day, selectedDay, colors) { selectedDay = it }
        }
    }
}

@Composable
private fun WeekRow(week: List<JalaliDate?>, todayDay: Int, selectedDay: Int, colors: RitmeColors, onSelect: (Int) -> Unit) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        week.forEach { cell ->
            Box(Modifier.weight(1f), contentAlignment = Alignment.Center) {
                if (cell != null) DayCell(cell.day, cell.day == todayDay, cell.day == selectedDay, colors) { onSelect(cell.day) }
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
            .clickable(onClick = onClick),
        contentAlignment = Alignment.Center,
    ) {
        Text(day.toPersianDigits(), style = MaterialTheme.typography.labelLarge, color = fg, fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun CycleStatus(predictions: CyclePredictions?, message: DailyMessage?, colors: RitmeColors) {
    Column(
        Modifier
            .fillMaxWidth()
            .background(Brush.linearGradient(listOf(colors.pink, colors.pinkLight)))
            .padding(16.dp),
    ) {
        Text(stringResource(R.string.cycle_today_label), style = MaterialTheme.typography.labelMedium, color = colors.onPink)
        Spacer(Modifier.height(10.dp))

        if (predictions == null) {
            Text(
                text = stringResource(R.string.home_no_cycle),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.onPink,
            )
            return@Column
        }

        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.Top) {
            Column(Modifier.weight(1f)) {
                Text(stringResource(R.string.cycle_next_period_label), style = MaterialTheme.typography.bodyMedium, color = colors.onPink)
                Spacer(Modifier.height(6.dp))
                Text(
                    text = stringResource(R.string.cycle_days, predictions.daysUntilNextPeriod.toPersianDigits()),
                    style = MaterialTheme.typography.headlineSmall,
                    color = colors.onPink,
                    fontWeight = FontWeight.Bold,
                )
            }
            FertilityRing(predictions.fertilityPercent, colors)
        }

        Spacer(Modifier.height(14.dp))
        ProgressBar(predictions.progressPercent, colors)

        val phaseDesc = message?.longMessage?.takeIf { it.isNotBlank() }
        if (phaseDesc != null) {
            Spacer(Modifier.height(12.dp))
            Box(Modifier.fillMaxWidth().clip(RoundedCornerShape(10.dp)).background(colors.pinkLight).padding(10.dp)) {
                Text(phaseDesc, style = MaterialTheme.typography.labelMedium, color = colors.onPink, textAlign = TextAlign.Start)
            }
        }
    }
}

@Composable
private fun FertilityRing(percent: Int, colors: RitmeColors) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(stringResource(R.string.cycle_fertility), style = MaterialTheme.typography.labelSmall, color = colors.onPink)
        Spacer(Modifier.height(6.dp))
        Box(
            modifier = Modifier.size(70.dp).clip(CircleShape).border(3.dp, colors.onPink, CircleShape),
            contentAlignment = Alignment.Center,
        ) {
            Text("${percent.toPersianDigits()}٪", style = MaterialTheme.typography.titleMedium, color = colors.onPink, fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
private fun ProgressBar(percent: Int, colors: RitmeColors) {
    Box(Modifier.fillMaxWidth().height(10.dp).clip(RoundedCornerShape(99.dp)).background(colors.onPink.copy(alpha = 0.3f))) {
        Box(Modifier.fillMaxWidth(percent / 100f).height(10.dp).clip(RoundedCornerShape(99.dp)).background(colors.onPink))
    }
}
