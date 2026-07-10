package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.JalaliDate
import java.util.Calendar

/**
 * Reads "today" from the device clock in the adapter layer, keeping the domain [JalaliDate]/
 * [GregorianDate] value objects clock-free (CLAUDE.md §2). Used to seed date pickers and bound
 * their year ranges. Uses the device's default time zone, which is what a local user expects.
 */
fun todayGregorian(): GregorianDate {
    val c = Calendar.getInstance()
    return GregorianDate(c.get(Calendar.YEAR), c.get(Calendar.MONTH) + 1, c.get(Calendar.DAY_OF_MONTH))
}

/** Today in the Jalali calendar the user thinks in. */
fun todayJalali(): JalaliDate = todayGregorian().toJalali()
