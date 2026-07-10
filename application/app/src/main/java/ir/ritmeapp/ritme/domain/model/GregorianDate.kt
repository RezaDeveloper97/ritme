package ir.ritmeapp.ritme.domain.model

/**
 * A plain year/month/day in the Gregorian calendar — the shape the backend speaks at the API
 * boundary (`birthday`, `last_period_start` are ISO `yyyy-MM-dd`). Pure data with no notion of
 * "now": the current date is read in an adapter and handed in, so the domain stays clock-free
 * and testable (CLAUDE.md §2).
 */
data class GregorianDate(
    val year: Int,
    val month: Int,
    val day: Int,
) {
    /** ISO `yyyy-MM-dd`, the format every date field in the Ritme API expects. */
    fun toIso(): String = "%04d-%02d-%02d".format(year, month, day)

    /** This date expressed in the Jalali (Shamsi) calendar Iranian users think in. */
    fun toJalali(): JalaliDate = JalaliDate.fromGregorian(this)

    companion object {
        /** Parses an ISO `yyyy-MM-dd` (the API's date shape); null if it isn't that shape. */
        fun parseIso(value: String): GregorianDate? {
            val parts = value.trim().split('-')
            if (parts.size != 3) return null
            val y = parts[0].toIntOrNull() ?: return null
            val m = parts[1].toIntOrNull() ?: return null
            val d = parts[2].toIntOrNull() ?: return null
            return GregorianDate(y, m, d)
        }
    }
}
