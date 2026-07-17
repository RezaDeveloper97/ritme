package ir.ritmeapp.ritme.domain.model

/**
 * The one period edit the calendar offers for a tapped day, decided by [PeriodDayRules].
 * Mirrors the web calendar's day sheet: start a new period, stretch a nearby one to cover
 * the day, or unmark a day that is inside one.
 */
sealed interface PeriodDayAction {

    /** The day is far from every logged period → a genuinely new period may start here. */
    data object StartNew : PeriodDayAction

    /** The day sits just outside [period] → widen it to [newStart]..[newEnd]. */
    data class Extend(
        val period: LoggedPeriod,
        val newStart: GregorianDate,
        val newEnd: GregorianDate?,
    ) : PeriodDayAction

    /** Unmarking this day of [period] — the exact edit depends on where the day falls. */
    sealed interface RemoveDay : PeriodDayAction {
        val period: LoggedPeriod
    }

    /** The day is the period's only day → the whole record goes. */
    data class DeletePeriod(override val period: LoggedPeriod) : RemoveDay

    /** The day is the first day → the period now starts at [newStart]. */
    data class TrimStart(override val period: LoggedPeriod, val newStart: GregorianDate) : RemoveDay

    /** Any later day → the period is cut to end at [newEnd] (days are contiguous). */
    data class TrimEnd(override val period: LoggedPeriod, val newEnd: GregorianDate) : RemoveDay

    /** Nothing sensible to offer (future day, or too close to a period to start a new one). */
    data object None : PeriodDayAction
}

/**
 * Pure decision rules for what the calendar offers on a tapped day — the same thresholds
 * as the web calendar so both clients behave identically.
 */
object PeriodDayRules {

    /** A day this close to a logged period reads as the SAME bleed → extend, never start. */
    const val EXTEND_GAP_DAYS = 7

    /**
     * Quick "start period" needs at least a rough cycle of distance from every logged
     * period: two real periods can't be closer, so anything nearer is a correction.
     */
    const val NEW_PERIOD_MIN_GAP_DAYS = 21

    /** Bleeding length assumed for a still-open period (matches the backend default). */
    const val DEFAULT_BLEED_DAYS = 5

    /** The single edit action offered for [day], given every logged period. */
    fun actionFor(
        day: GregorianDate,
        today: GregorianDate,
        periods: List<LoggedPeriod>,
        defaultBleedDays: Int = DEFAULT_BLEED_DAYS,
    ): PeriodDayAction {
        val dayJdn = day.jdn()
        if (dayJdn > today.jdn()) return PeriodDayAction.None

        coveringPeriod(dayJdn, periods, defaultBleedDays)?.let { period ->
            return removalFor(day, dayJdn, period, defaultBleedDays)
        }

        var nearest: Pair<LoggedPeriod, Int>? = null
        for (period in periods) {
            val gap = gapToPeriod(dayJdn, period, defaultBleedDays)
            if (nearest == null || gap < nearest.second) nearest = period to gap
        }
        val (period, gap) = nearest ?: return PeriodDayAction.StartNew
        return when {
            gap <= EXTEND_GAP_DAYS -> extensionFor(day, dayJdn, period, defaultBleedDays)
            gap < NEW_PERIOD_MIN_GAP_DAYS -> PeriodDayAction.None
            else -> PeriodDayAction.StartNew
        }
    }

    private fun coveringPeriod(dayJdn: Int, periods: List<LoggedPeriod>, defaultBleedDays: Int): LoggedPeriod? =
        periods.firstOrNull { period ->
            dayJdn >= period.start.jdn() && dayJdn <= period.coveredEnd(defaultBleedDays).jdn()
        }

    private fun removalFor(
        day: GregorianDate,
        dayJdn: Int,
        period: LoggedPeriod,
        defaultBleedDays: Int,
    ): PeriodDayAction.RemoveDay {
        val startJdn = period.start.jdn()
        val endJdn = period.coveredEnd(defaultBleedDays).jdn()
        return when {
            dayJdn == startJdn && dayJdn == endJdn -> PeriodDayAction.DeletePeriod(period)
            dayJdn == startJdn -> PeriodDayAction.TrimStart(period, day.plusDays(1))
            else -> PeriodDayAction.TrimEnd(period, day.plusDays(-1))
        }
    }

    private fun extensionFor(
        day: GregorianDate,
        dayJdn: Int,
        period: LoggedPeriod,
        defaultBleedDays: Int,
    ): PeriodDayAction.Extend =
        if (dayJdn < period.start.jdn()) {
            // Extending backwards keeps the recorded end (possibly still open).
            PeriodDayAction.Extend(period, newStart = day, newEnd = period.end)
        } else {
            PeriodDayAction.Extend(period, newStart = period.start, newEnd = day)
        }

    /** Whole days between the day and the period's covered range (never negative). */
    private fun gapToPeriod(dayJdn: Int, period: LoggedPeriod, defaultBleedDays: Int): Int {
        val startJdn = period.start.jdn()
        val endJdn = period.coveredEnd(defaultBleedDays).jdn()
        return when {
            dayJdn < startJdn -> startJdn - dayJdn
            dayJdn > endJdn -> dayJdn - endJdn
            else -> 0
        }
    }

    private fun GregorianDate.jdn(): Int = toJalali().toJdn()

    private fun GregorianDate.plusDays(days: Int): GregorianDate = toJalali().addDays(days).toGregorian()
}
