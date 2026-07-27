package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

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
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.JalaliDate

private val DAY_NAMES = listOf("ش", "ی", "د", "س", "چ", "پ", "ج")

/**
 * A hand-built Jalali month-grid day picker (CLAUDE.md §3 — no library), mirroring the web
 * `JalaliCalendar`: a month/year header with prev/next chevrons, a ش…ج weekday row, and a
 * 7-column grid of day buttons where the selected day is a solid brand-pink circle
 * (web `.cday.on`). Wrapped in a surface card like the web. Reports the full picked
 * year/month/day so navigating months never leaves a stale same-numbered highlight.
 */
@Composable
fun JalaliMonthCalendar(
    selected: JalaliDate?,
    onSelect: (JalaliDate) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    val today = remember { todayJalali() }
    var year by remember { mutableIntStateOf(selected?.year ?: today.year) }
    var month by remember { mutableIntStateOf(selected?.month ?: today.month) }

    val firstOfMonth = JalaliDate(year, month, 1)
    val offset = firstOfMonth.dayOfWeek() // 0 = شنبه
    val daysInMonth = firstOfMonth.daysInMonth()

    fun goPrev() {
        if (month == 1) {
            year -= 1; month = 12
        } else {
            month -= 1
        }
    }

    fun goNext() {
        if (month == 12) {
            year += 1; month = 1
        } else {
            month += 1
        }
    }

    Column(
        modifier = modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(16.dp))
            .padding(horizontal = 14.dp, vertical = 16.dp),
    ) {
        // Month header. In RTL the first child sits on the right: prev-month (chevron_right).
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            HeaderIconButton(R.drawable.ic_chevron_right, "ماه قبل", ::goPrev)
            Text(
                text = "${firstOfMonth.monthName} ${year.toPersianDigits()}",
                style = MaterialTheme.typography.titleSmall,
                fontWeight = FontWeight.Bold,
                color = colors.ink,
            )
            HeaderIconButton(R.drawable.ic_chevron_left, "ماه بعد", ::goNext)
        }
        Spacer(Modifier.height(14.dp))

        // Weekday header.
        Row(Modifier.fillMaxWidth()) {
            DAY_NAMES.forEach { name ->
                Text(
                    text = name,
                    modifier = Modifier.weight(1f),
                    textAlign = TextAlign.Center,
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold,
                    color = colors.inkMuted,
                )
            }
        }
        Spacer(Modifier.height(8.dp))

        // Day grid: leading blanks for the first-day offset, then the days, padded to full weeks.
        val cells = buildList {
            repeat(offset) { add(null) }
            for (d in 1..daysInMonth) add(d)
            while (size % 7 != 0) add(null)
        }
        cells.chunked(7).forEach { week ->
            Row(Modifier.fillMaxWidth()) {
                week.forEach { day ->
                    Box(
                        modifier = Modifier.weight(1f).height(40.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (day != null) {
                            val isSelected = selected?.year == year && selected.month == month && selected.day == day
                            Box(
                                modifier = Modifier
                                    .size(36.dp)
                                    .clip(CircleShape)
                                    .background(if (isSelected) colors.pink else colors.surface)
                                    .clickable { onSelect(JalaliDate(year, month, day)) },
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(
                                    text = day.toPersianDigits(),
                                    style = MaterialTheme.typography.bodyMedium.copy(fontSize = 14.sp),
                                    fontWeight = if (isSelected) FontWeight.Bold else FontWeight.SemiBold,
                                    color = if (isSelected) colors.onPink else colors.ink,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
