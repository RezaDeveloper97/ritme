package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.domain.model.JalaliDate

/**
 * A hand-built Jalali (Shamsi) date picker: three [WheelPicker]s — day, month, year — laid out in
 * a Row that the ambient RTL direction renders right-to-left (day on the right, year on the left),
 * the way Persian date pickers read. No third-party date-picker library (CLAUDE.md §3).
 *
 * Presentational and controlled: the parent owns the [value]; changing the year or month re-clamps
 * the day to that month's length (so اسفند 30 can't survive into a common year), and every change
 * is reported through [onValueChange].
 */
@Composable
fun JalaliDatePicker(
    value: JalaliDate,
    onValueChange: (JalaliDate) -> Unit,
    minYear: Int,
    maxYear: Int,
    modifier: Modifier = Modifier,
) {
    val yearCount = (maxYear - minYear + 1).coerceAtLeast(1)
    val daysInMonth = JalaliDate(value.year, value.month, 1).daysInMonth()

    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        WheelPicker(
            count = daysInMonth,
            selectedIndex = (value.day - 1).coerceIn(0, daysInMonth - 1),
            onSelected = { onValueChange(value.copy(day = it + 1)) },
            label = { (it + 1).toPersianDigits() },
            modifier = Modifier.weight(1f),
        )
        WheelPicker(
            count = JalaliDate.MONTH_NAMES.size,
            selectedIndex = (value.month - 1).coerceIn(0, JalaliDate.MONTH_NAMES.lastIndex),
            onSelected = { onMonthChange(value, it + 1, onValueChange) },
            label = { JalaliDate.MONTH_NAMES[it] },
            modifier = Modifier.weight(1.3f),
        )
        WheelPicker(
            count = yearCount,
            selectedIndex = (value.year - minYear).coerceIn(0, yearCount - 1),
            onSelected = { onYearChange(value, minYear + it, onValueChange) },
            label = { (minYear + it).toPersianDigits() },
            modifier = Modifier.weight(1.2f),
        )
    }
}

/** Applies a new month, shrinking the day if the new month is shorter (e.g. 31 → 30/29). */
private fun onMonthChange(current: JalaliDate, newMonth: Int, onValueChange: (JalaliDate) -> Unit) {
    val maxDay = JalaliDate(current.year, newMonth, 1).daysInMonth()
    onValueChange(JalaliDate(current.year, newMonth, current.day.coerceAtMost(maxDay)))
}

/** Applies a new year, shrinking the day if اسفند loses its 30th in a common year. */
private fun onYearChange(current: JalaliDate, newYear: Int, onValueChange: (JalaliDate) -> Unit) {
    val maxDay = JalaliDate(newYear, current.month, 1).daysInMonth()
    onValueChange(JalaliDate(newYear, current.month, current.day.coerceAtMost(maxDay)))
}
