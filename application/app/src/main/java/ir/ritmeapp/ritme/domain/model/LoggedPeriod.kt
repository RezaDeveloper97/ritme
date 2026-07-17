package ir.ritmeapp.ritme.domain.model

/**
 * One period the user actually logged (as opposed to engine predictions) — the unit the
 * calendar's quick edit actions operate on. [end] is null while the period is still open.
 */
data class LoggedPeriod(
    val id: Long,
    val start: GregorianDate,
    val end: GregorianDate?,
) {
    /** Last day this period visibly covers; an open period runs for [defaultBleedDays]. */
    fun coveredEnd(defaultBleedDays: Int): GregorianDate =
        end ?: start.toJalali().addDays(defaultBleedDays - 1).toGregorian()
}
