package ir.ritmeapp.ritme.adapter.inbound.ui.foundation

import ir.ritmeapp.ritme.domain.model.JalaliDate

/** "۲۵ اردیبهشت" — day + Persian month name, with Persian digits. UI-layer display formatting. */
fun JalaliDate.formatDayMonth(): String = "${day.toPersianDigits()} $monthName"

/** "۲۵ اردیبهشت ۱۴۰۳" — full Jalali date with Persian digits. */
fun JalaliDate.formatFull(): String = "${day.toPersianDigits()} $monthName ${year.toPersianDigits()}"
