package ir.ritmeapp.ritme.adapter.inbound.ui.home

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
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.CyclePredictions
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

@Composable
internal fun PhaseRowsSection(predictions: CyclePredictions, today: JalaliDate) {
    val colors = LocalRitmeColors.current
    val rows = listOf(
        stringResource(R.string.phases_window) to offsetDate(today, max(0, predictions.daysUntilFertileWindow)),
        stringResource(R.string.phases_ovulation) to offsetDate(today, max(0, predictions.daysUntilOvulation)),
        stringResource(R.string.phases_pms) to
            "${offsetDate(today, predictions.daysUntilPmsStart)} - ${offsetDate(today, predictions.daysUntilPmsEnd)}",
        stringResource(R.string.phases_next_period) to offsetDate(today, predictions.daysUntilNextPeriod),
    )
    HomeCard(colors) {
        rows.forEachIndexed { i, (label, date) ->
            if (i > 0) Spacer(Modifier.height(14.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                Text(label, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted, fontWeight = FontWeight.Bold)
                Box(Modifier.clip(RoundedCornerShape(12.dp)).background(colors.pinkContainer).padding(horizontal = 14.dp, vertical = 8.dp)) {
                    Text(date, style = MaterialTheme.typography.labelMedium, color = colors.pink, fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

@Composable
internal fun RecommendationsSection(message: DailyMessage?) {
    val colors = LocalRitmeColors.current
    val items = message?.dos?.take(3)?.takeIf { it.isNotEmpty() }
        ?: listOf(stringResource(R.string.reco_default_1), stringResource(R.string.reco_default_2))
    HomeCard(colors) {
        Text(stringResource(R.string.reco_title), style = MaterialTheme.typography.titleMedium, color = colors.pink)
        Spacer(Modifier.height(12.dp))
        items.forEachIndexed { i, text ->
            if (i > 0) Spacer(Modifier.height(8.dp))
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(10.dp)).background(colors.pinkContainer).padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(Modifier.size(34.dp).clip(CircleShape).background(colors.surface), contentAlignment = Alignment.Center) {
                    Text("✓", style = MaterialTheme.typography.labelLarge, color = colors.success)
                }
                Spacer(Modifier.size(12.dp))
                Text(text, style = MaterialTheme.typography.bodyMedium, color = colors.ink, modifier = Modifier.weight(1f), textAlign = TextAlign.Start)
            }
        }
    }
}

@Composable
internal fun SmartTipSection(message: DailyMessage?) {
    val colors = LocalRitmeColors.current
    val body = message?.longMessage?.takeIf { it.isNotBlank() } ?: stringResource(R.string.smarttip_default_body)
    val quote = message?.actionSuggestion?.takeIf { it.isNotBlank() } ?: stringResource(R.string.smarttip_default_quote)
    HomeCard(colors) {
        Text(stringResource(R.string.smarttip_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(10.dp))
        Text(body, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted, textAlign = TextAlign.Start)
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.pinkContainer).padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("✨", style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.size(10.dp))
            Text(quote, style = MaterialTheme.typography.labelLarge, color = colors.ink, modifier = Modifier.weight(1f), textAlign = TextAlign.Start)
        }
    }
}

/**
 * «وظایف امروز» — today's to-dos from the reminders backend (custom-type reminders scheduled for
 * today). Tapping a row checks it off, which toggles the reminder's active flag (the API has no
 * separate completed state). Added from the daily-log day planner, so the two stay in sync.
 */
@Composable
internal fun TodayTasksSection(tasks: List<Reminder>, onToggle: (Reminder) -> Unit) {
    val colors = LocalRitmeColors.current
    val completed = tasks.count { !it.isActive }
    val percent = if (tasks.isEmpty()) 0 else completed * 100 / tasks.size
    HomeCard(colors) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.tasks_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
            if (tasks.isNotEmpty()) {
                Text(
                    "${completed.toPersianDigits()}/${tasks.size.toPersianDigits()}",
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.success,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
        Spacer(Modifier.height(12.dp))
        if (tasks.isEmpty()) {
            Text(
                stringResource(R.string.tasks_empty),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            return@HomeCard
        }
        tasks.forEach { task ->
            val done = !task.isActive
            Row(
                Modifier.fillMaxWidth().clickable { onToggle(task) }.padding(vertical = 9.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    task.title,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (done) colors.inkMuted else colors.ink,
                    textDecoration = if (done) TextDecoration.LineThrough else null,
                    modifier = Modifier.weight(1f),
                    textAlign = TextAlign.Start,
                )
                Spacer(Modifier.size(10.dp))
                Box(
                    Modifier.size(22.dp).clip(CircleShape)
                        .background(if (done) colors.pink else colors.background)
                        .border(1.dp, if (done) colors.pink else colors.outline, CircleShape),
                    contentAlignment = Alignment.Center,
                ) {
                    if (done) Text("✓", style = MaterialTheme.typography.labelSmall, color = colors.onPink)
                }
            }
        }
        Spacer(Modifier.height(10.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.tasks_progress), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
            Text("${percent.toPersianDigits()}٪", style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        }
        Spacer(Modifier.height(6.dp))
        Box(Modifier.fillMaxWidth().height(8.dp).clip(RoundedCornerShape(99.dp)).background(colors.outline)) {
            Box(Modifier.fillMaxWidth(percent / 100f).height(8.dp).clip(RoundedCornerShape(99.dp)).background(colors.pink))
        }
    }
}

@Composable
internal fun MyCyclesSection(predictions: CyclePredictions, today: JalaliDate) {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Text(stringResource(R.string.cycles_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(12.dp))
        Text(
            text = stringResource(R.string.cycles_current_day, predictions.cycleDay.toPersianDigits()),
            style = MaterialTheme.typography.bodyLarge,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            text = stringResource(R.string.cycles_started, offsetDate(today, -(predictions.cycleDay - 1))),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
        )
    }
}

@Composable
internal fun CycleSummarySection(predictions: CyclePredictions) {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Text(stringResource(R.string.cyclesummary_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(stringResource(R.string.cyclesummary_length), style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted)
            Text(stringResource(R.string.cycle_days, predictions.cycleLength.toPersianDigits()), style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
        }
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
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                painter = painterResource(R.drawable.ic_flame),
                contentDescription = null,
                tint = colors.warning,
                modifier = Modifier.size(20.dp),
            )
            Spacer(Modifier.size(8.dp))
            Text(stringResource(R.string.challenge_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        }
        Spacer(Modifier.height(12.dp))
        Column(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(12.dp))
                .border(2.dp, colors.success, RoundedCornerShape(12.dp))
                .padding(12.dp),
        ) {
            Text(stringResource(R.string.challenge_item), style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
            Spacer(Modifier.height(4.dp))
            Text(stringResource(R.string.challenge_desc), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        }
    }
}

/** Today's doctor & medication reminders, rendered as cards. Empty is handled by the caller. */
@Composable
internal fun ReminderCardsSection(reminders: List<Reminder>) {
    val colors = LocalRitmeColors.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        reminders.forEach { reminder ->
            ReminderCard(reminder.type.cardIconRes(), reminder.title, reminder.subtitle, colors)
        }
    }
}

@Composable
private fun ReminderCard(
    @androidx.annotation.DrawableRes icon: Int,
    title: String,
    subtitle: String?,
    colors: RitmeColors,
) {
    HomeCard(colors) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(38.dp).clip(CircleShape).background(colors.pinkContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(painter = painterResource(icon), contentDescription = null, tint = colors.pink, modifier = Modifier.size(18.dp))
            }
            Spacer(Modifier.size(10.dp))
            Column {
                Text(title, style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
                if (!subtitle.isNullOrBlank()) {
                    Text(subtitle, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
                }
            }
        }
    }
}

@androidx.annotation.DrawableRes
private fun ReminderType.cardIconRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.drawable.ic_stetho
    ReminderType.MEDICATION -> R.drawable.ic_pill
    ReminderType.APPOINTMENT -> R.drawable.ic_calendar
    ReminderType.CUSTOM -> R.drawable.ic_bell
}

@Composable
internal fun WeekSummarySection() {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Text(stringResource(R.string.weeksummary_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            SummaryTile(R.drawable.ic_smile, R.string.weeksummary_mood, R.string.weeksummary_mood_value, colors.pink, colors, Modifier.weight(1f))
            SummaryTile(R.drawable.ic_moon, R.string.weeksummary_sleep, R.string.weeksummary_sleep_value, colors.info, colors, Modifier.weight(1f))
            SummaryTile(R.drawable.ic_zap, R.string.weeksummary_energy, R.string.weeksummary_energy_value, colors.warning, colors, Modifier.weight(1f))
        }
        Spacer(Modifier.height(10.dp))
        Text(stringResource(R.string.challenge_desc), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
    }
}

@Composable
internal fun TodayStatusSection() {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Text(stringResource(R.string.todaystatus_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
        Spacer(Modifier.height(12.dp))
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            SummaryTile(R.drawable.ic_heart, R.string.todaystatus_bp, R.string.todaystatus_normal, colors.pink, colors, Modifier.weight(1f))
            SummaryTile(R.drawable.ic_drop, R.string.todaystatus_glucose, R.string.todaystatus_normal, colors.info, colors, Modifier.weight(1f))
            SummaryTile(R.drawable.ic_zap, R.string.todaystatus_hr, R.string.todaystatus_normal, colors.success, colors, Modifier.weight(1f))
        }
    }
}

@Composable
private fun SummaryTile(
    @androidx.annotation.DrawableRes icon: Int,
    @androidx.annotation.StringRes label: Int,
    @androidx.annotation.StringRes value: Int,
    tint: androidx.compose.ui.graphics.Color,
    colors: RitmeColors,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier.clip(RoundedCornerShape(12.dp)).background(colors.pinkContainer).padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(painter = painterResource(icon), contentDescription = null, tint = tint, modifier = Modifier.size(18.dp))
        Spacer(Modifier.height(6.dp))
        Text(stringResource(label), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        Spacer(Modifier.height(2.dp))
        Text(stringResource(value), style = MaterialTheme.typography.labelMedium, color = colors.ink, fontWeight = FontWeight.Bold)
    }
}

@Composable
internal fun ArticlesSection() {
    val colors = LocalRitmeColors.current
    HomeCard(colors) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(stringResource(R.string.articles_title), style = MaterialTheme.typography.titleMedium, color = colors.ink)
            Text(stringResource(R.string.articles_read_more), style = MaterialTheme.typography.labelSmall, color = colors.pink, fontWeight = FontWeight.Bold)
        }
        Spacer(Modifier.height(12.dp))
        Row(
            Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            repeat(3) { index ->
                Column(Modifier.width(190.dp)) {
                    Box(
                        Modifier
                            .fillMaxWidth()
                            .height(96.dp)
                            .clip(RoundedCornerShape(12.dp))
                            .background(
                                Brush.linearGradient(
                                    when (index % 3) {
                                        0 -> listOf(colors.pinkLight, colors.accent)
                                        1 -> listOf(colors.accent, colors.info)
                                        else -> listOf(colors.warning, colors.pinkLight)
                                    },
                                ),
                            ),
                    )
                    Spacer(Modifier.height(6.dp))
                    Text(
                        stringResource(R.string.articles_article1),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                        maxLines = 2,
                    )
                    Text(
                        stringResource(R.string.articles_min, 9.toPersianDigits()),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.inkMuted,
                    )
                }
            }
        }
    }
}

/** Formats "today + n days" as a Jalali "day month" label. */
private fun offsetDate(today: JalaliDate, days: Int): String = today.addDays(days).formatDayMonth()
