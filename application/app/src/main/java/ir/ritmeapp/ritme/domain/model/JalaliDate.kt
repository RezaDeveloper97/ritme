package ir.ritmeapp.ritme.domain.model

/**
 * A plain year/month/day in the Jalali (Shamsi / Persian) calendar — the calendar Iranian
 * users actually think and pick dates in, even though the Ritme API only ever speaks
 * Gregorian ISO (see [GregorianDate]). Keeping this a pure value object lets the UI show
 * Shamsi dates while the network layer keeps sending `yyyy-MM-dd`, with a single trusted
 * conversion in between.
 *
 * The conversion is the accurate "jalaali-js" algorithm (Behdad Esfahbod / Roozbeh
 * Pournader): it uses an astronomical breaks-array to decide leap years exactly, rather
 * than the common approximate 33-year modulo rule which drifts for some years. The math is
 * kept in a private [JalaliMath] object so this class stays a thin, immutable façade.
 */
data class JalaliDate(
    val year: Int,
    val month: Int,
    val day: Int,
) {
    /** This date expressed in the Gregorian calendar the backend expects. */
    fun toGregorian(): GregorianDate = JalaliMath.jalToJdn(year, month, day).let(JalaliMath::d2g)

    /** ISO `yyyy-MM-dd` (Gregorian) — the shape every date field in the Ritme API speaks. */
    fun toIso(): String = toGregorian().toIso()

    /** Persian name of this date's month (فروردین … اسفند); calendar data, not UI formatting. */
    val monthName: String get() = MONTH_NAMES[month - 1]

    /**
     * True when this Jalali year is a leap year (اسفند has 30 days instead of 29). In the
     * jalaali-js model the `leap` field is "years since the last leap year", so a value of 0
     * means the year itself is the leap year — this is deliberately not `== 1`.
     */
    fun isLeapYear(): Boolean = JalaliMath.jalCal(year, withoutLeap = false).leap == 0

    /**
     * Number of days in this date's month: months 1..6 have 31, months 7..11 have 30, and
     * month 12 (اسفند) has 30 in a leap year, otherwise 29.
     */
    fun daysInMonth(): Int = when {
        month in 1..FIRST_HALF_LAST_MONTH -> LONG_MONTH_DAYS
        month in (FIRST_HALF_LAST_MONTH + 1) until LAST_MONTH -> SHORT_MONTH_DAYS
        else -> if (isLeapYear()) SHORT_MONTH_DAYS else ESFAND_COMMON_DAYS
    }

    /** Julian Day Number of this date — the basis for weekday and day arithmetic. */
    fun toJdn(): Int = JalaliMath.jalToJdn(year, month, day)

    /** This date shifted by [days] (may cross month/year boundaries), via day-number arithmetic. */
    fun addDays(days: Int): JalaliDate = JalaliMath.jdnToJal(toJdn() + days)

    /** Day of week in the Persian convention: 0 = شنبه (Saturday) … 6 = جمعه (Friday). */
    fun dayOfWeek(): Int {
        // Sakamoto on the Gregorian equivalent gives 0 = Sunday … 6 = Saturday; shift to Saturday-first.
        val g = toGregorian()
        return (gregorianDowSunday0(g.year, g.month, g.day) + 1) % DAYS_IN_WEEK
    }

    companion object {
        /** Lowest Jalali year the breaks-array algorithm can convert. */
        const val MIN_YEAR: Int = -61

        /** Highest Jalali year the breaks-array algorithm can convert. */
        const val MAX_YEAR: Int = 3177

        /** The twelve Persian month names, index 0 = فروردین … index 11 = اسفند. */
        val MONTH_NAMES: List<String> = listOf(
            "فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور",
            "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند",
        )

        /** The Jalali date corresponding to the given Gregorian date. */
        fun fromGregorian(g: GregorianDate): JalaliDate =
            JalaliMath.jdnToJal(JalaliMath.g2d(g.year, g.month, g.day))

        /** True when (y, m, d) is a real Jalali date (valid month and day-in-month). */
        fun isValid(year: Int, month: Int, day: Int): Boolean {
            if (year < MIN_YEAR || year > MAX_YEAR) return false
            if (month < 1 || month > LAST_MONTH) return false
            if (day < 1) return false
            return day <= JalaliDate(year, month, 1).daysInMonth()
        }

        /**
         * The weeks of a Jalali month as Saturday-first rows for a calendar grid. Leading `null`s
         * pad to the first-of-month's weekday and trailing `null`s complete the last week, so every
         * row has exactly 7 cells.
         */
        fun monthMatrix(year: Int, month: Int): List<List<JalaliDate?>> {
            val first = JalaliDate(year, month, 1)
            val lead = first.dayOfWeek()
            val days = first.daysInMonth()
            val cells = ArrayList<JalaliDate?>(lead + days)
            repeat(lead) { cells.add(null) }
            for (d in 1..days) cells.add(JalaliDate(year, month, d))
            while (cells.size % DAYS_IN_WEEK != 0) cells.add(null)
            return cells.chunked(DAYS_IN_WEEK)
        }

        /** Gregorian day of week via Sakamoto's method: 0 = Sunday … 6 = Saturday. */
        private fun gregorianDowSunday0(y: Int, m: Int, d: Int): Int {
            val t = intArrayOf(0, 3, 2, 5, 0, 3, 5, 1, 4, 6, 2, 4)
            val yy = if (m < 3) y - 1 else y
            return (yy + yy / 4 - yy / 100 + yy / 400 + t[m - 1] + d) % DAYS_IN_WEEK
        }

        private const val DAYS_IN_WEEK = 7
        private const val FIRST_HALF_LAST_MONTH = 6
        private const val LAST_MONTH = 12
        private const val LONG_MONTH_DAYS = 31
        private const val SHORT_MONTH_DAYS = 30
        private const val ESFAND_COMMON_DAYS = 29
    }
}

/**
 * The jalaali-js integer date math (Behdad Esfahbod / Roozbeh Pournader), ported verbatim.
 *
 * Kotlin's `Int` division truncates toward zero and `%` takes the dividend's sign, which is
 * exactly what the reference `div(a,b) = ~~(a/b)` and `mod(a,b) = a - ~~(a/b)*b` do — so the
 * native operators are used directly instead of re-deriving those helpers.
 */
private object JalaliMath {

    /** Result of [jalCal]: leap flag, the paired Gregorian year, and March day of Farvardin 1. */
    data class JalCal(val leap: Int, val gy: Int, val march: Int)

    private val BREAKS = intArrayOf(
        -61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181,
        1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
    )

    /** Gregorian (proleptic) date to Julian Day Number. */
    fun g2d(gy: Int, gm: Int, gd: Int): Int {
        var d = (gy + (gm - 8) / 6 + 100100) * 1461 / 4 +
            (153 * ((gm + 9) % 12) + 2) / 5 +
            gd - 34840408
        d = d - (gy + 100100 + (gm - 8) / 6) / 100 * 3 / 4 + 752
        return d
    }

    /** Julian Day Number to Gregorian (proleptic) date. */
    fun d2g(jdn: Int): GregorianDate {
        var j = 4 * jdn + 139361631
        j += (4 * jdn + 183187720) / 146097 * 3 / 4 * 4 - 3908
        val i = j % 1461 / 4 * 5 + 308
        val gd = i % 153 / 5 + 1
        val gm = i / 153 % 12 + 1
        val gy = j / 1461 - 100100 + (8 - gm) / 6
        return GregorianDate(gy, gm, gd)
    }

    /**
     * Determines leap-year status and the Gregorian March day on which the given Jalali year's
     * Farvardin 1 falls, using the astronomical breaks array. When [withoutLeap] is true the
     * (more expensive) leap computation is skipped and leap is reported as 0.
     */
    fun jalCal(jy: Int, withoutLeap: Boolean): JalCal {
        val bl = BREAKS.size
        val gy = jy + 621
        var leapJ = -14
        var jp = BREAKS[0]

        require(jy >= jp && jy < BREAKS[bl - 1]) { "Invalid Jalali year $jy" }

        var jump = 0
        var i = 1
        while (i < bl) {
            val jm = BREAKS[i]
            jump = jm - jp
            if (jy < jm) break
            leapJ += jump / 33 * 8 + jump % 33 / 4
            jp = jm
            i += 1
        }
        var n = jy - jp

        leapJ += n / 33 * 8 + (n % 33 + 3) / 4
        if (jump % 33 == 4 && jump - n == 4) leapJ += 1

        val leapG = gy / 4 - (gy / 100 + 1) * 3 / 4 - 150
        val march = 20 + leapJ - leapG

        var leap = 0
        if (!withoutLeap) {
            if (jump - n < 6) n = n - jump + (jump + 4) / 33 * 33
            leap = ((n + 1) % 33 - 1) % 4
            if (leap == -1) leap = 4
        }

        return JalCal(leap = leap, gy = gy, march = march)
    }

    /** Jalali date to Julian Day Number. */
    fun jalToJdn(jy: Int, jm: Int, jd: Int): Int {
        val r = jalCal(jy, withoutLeap = true)
        return g2d(r.gy, 3, r.march) + (jm - 1) * 31 - jm / 7 * (jm - 7) + jd - 1
    }

    /** Julian Day Number to Jalali date. */
    fun jdnToJal(jdn: Int): JalaliDate {
        val gy = d2g(jdn).year
        var jy = gy - 621
        val r = jalCal(jy, withoutLeap = false)
        val jdn1f = g2d(gy, 3, r.march)
        var jm: Int
        var jd: Int

        var k = jdn - jdn1f
        if (k >= 0) {
            if (k <= 185) {
                jm = 1 + k / 31
                jd = k % 31 + 1
                return JalaliDate(jy, jm, jd)
            } else {
                k -= 186
            }
        } else {
            jy -= 1
            k += 179
            if (r.leap == 1) k += 1
        }
        jm = 7 + k / 30
        jd = k % 30 + 1
        return JalaliDate(jy, jm, jd)
    }
}
