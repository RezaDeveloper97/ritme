package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class JalaliCalendarTest {

    @Test
    fun `dayOfWeek is Saturday-first — Farvardin 1 1403 is Wednesday`() {
        // 1403-01-01 == 2024-03-20, a Wednesday → Persian index 4 (شنبه=0).
        assertEquals(4, JalaliDate(1403, 1, 1).dayOfWeek())
    }

    @Test
    fun `addDays moves within and across month boundaries`() {
        assertEquals(JalaliDate(1403, 1, 2), JalaliDate(1403, 1, 1).addDays(1))
        // Farvardin has 31 days, so +31 lands on Ordibehesht 1.
        assertEquals(JalaliDate(1403, 2, 1), JalaliDate(1403, 1, 1).addDays(31))
    }

    @Test
    fun `addDays is consistent with the Gregorian mapping`() {
        assertEquals("2024-03-21", JalaliDate(1403, 1, 1).addDays(1).toIso())
    }

    @Test
    fun `addDays round-trips`() {
        val d = JalaliDate(1402, 11, 5)
        assertEquals(d, d.addDays(40).addDays(-40))
    }

    @Test
    fun `monthMatrix pads to Saturday-first full weeks`() {
        val weeks = JalaliDate.monthMatrix(1403, 1)
        // Farvardin 1403: Wednesday start → 4 leading nulls, 31 days → 5 rows of 7.
        assertEquals(5, weeks.size)
        weeks.forEach { assertEquals(7, it.size) }
        assertNull(weeks[0][0])
        assertNull(weeks[0][3])
        assertEquals(JalaliDate(1403, 1, 1), weeks[0][4])
        assertEquals(31, weeks.flatten().count { it != null })
    }
}
