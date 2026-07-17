package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The calendar's quick-edit decision rules — the same thresholds as the web calendar,
 * so both clients offer identical actions for the same tapped day.
 */
class PeriodDayRulesTest {

    private val today = GregorianDate(2026, 7, 14)

    private fun day(month: Int, dayOfMonth: Int) = GregorianDate(2026, month, dayOfMonth)

    private val closedPeriod = LoggedPeriod(id = 1, start = day(7, 4), end = day(7, 8))

    @Test
    fun `future day offers nothing`() {
        val action = PeriodDayRules.actionFor(day(7, 20), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.None, action)
    }

    @Test
    fun `day inside a period offers removal - middle day trims the end`() {
        val action = PeriodDayRules.actionFor(day(7, 6), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.TrimEnd(closedPeriod, day(7, 5)), action)
    }

    @Test
    fun `first day of a period moves the start forward`() {
        val action = PeriodDayRules.actionFor(day(7, 4), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.TrimStart(closedPeriod, day(7, 5)), action)
    }

    @Test
    fun `single-day period is deleted outright`() {
        val single = LoggedPeriod(id = 2, start = day(7, 10), end = day(7, 10))
        val action = PeriodDayRules.actionFor(day(7, 10), today, listOf(single))
        assertEquals(PeriodDayAction.DeletePeriod(single), action)
    }

    @Test
    fun `day just after a period extends its end`() {
        val action = PeriodDayRules.actionFor(day(7, 10), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.Extend(closedPeriod, newStart = day(7, 4), newEnd = day(7, 10)), action)
    }

    @Test
    fun `day just before a period extends its start and keeps the recorded end`() {
        val action = PeriodDayRules.actionFor(day(7, 1), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.Extend(closedPeriod, newStart = day(7, 1), newEnd = day(7, 8)), action)
    }

    @Test
    fun `day between 8 and 20 days away offers nothing - no duplicate periods mid-cycle`() {
        val action = PeriodDayRules.actionFor(day(6, 22), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.None, action)
    }

    @Test
    fun `day a full cycle away may start a new period`() {
        val action = PeriodDayRules.actionFor(day(6, 10), today, listOf(closedPeriod))
        assertEquals(PeriodDayAction.StartNew, action)
    }

    @Test
    fun `no logged periods - any past day may start one`() {
        val action = PeriodDayRules.actionFor(day(7, 14), today, emptyList())
        assertEquals(PeriodDayAction.StartNew, action)
    }

    @Test
    fun `open period covers the default bleed length`() {
        val open = LoggedPeriod(id = 3, start = day(7, 10), end = null)
        // Day 12 falls inside the assumed 5-day bleed → removal, not extension.
        val action = PeriodDayRules.actionFor(day(7, 12), today, listOf(open))
        assertTrue(action is PeriodDayAction.TrimEnd)
    }
}
