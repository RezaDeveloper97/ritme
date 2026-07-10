package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Verifies the jalaali-js conversion against hand-checked Gregorian/Jalali vectors, in both
 * directions, plus round-tripping and calendar-shape rules. If the algorithm ever drifts
 * (e.g. someone swaps in the approximate 33-year leap rule) these anchor dates catch it.
 */
class JalaliDateTest {

    private data class Vector(
        val g: GregorianDate,
        val j: JalaliDate,
    )

    private val vectors = listOf(
        Vector(GregorianDate(1995, 5, 15), JalaliDate(1374, 2, 25)),
        Vector(GregorianDate(2024, 3, 20), JalaliDate(1403, 1, 1)), // Nowruz 1403
        Vector(GregorianDate(2026, 7, 10), JalaliDate(1405, 4, 19)),
        Vector(GregorianDate(1979, 2, 11), JalaliDate(1357, 11, 22)),
        Vector(GregorianDate(2000, 1, 1), JalaliDate(1378, 10, 11)),
    )

    @Test
    fun gregorianToJalaliMatchesVectors() {
        for (v in vectors) {
            assertEquals("G→J for ${v.g.toIso()}", v.j, JalaliDate.fromGregorian(v.g))
            assertEquals("G→J via GregorianDate for ${v.g.toIso()}", v.j, v.g.toJalali())
        }
    }

    @Test
    fun jalaliToGregorianMatchesVectors() {
        for (v in vectors) {
            assertEquals("J→G for ${v.j}", v.g, v.j.toGregorian())
            assertEquals("J→G ISO for ${v.j}", v.g.toIso(), v.j.toIso())
        }
    }

    @Test
    fun roundTripJalaliGregorianJalali() {
        val samples = listOf(
            JalaliDate(1374, 2, 25),
            JalaliDate(1403, 1, 1),
            JalaliDate(1405, 4, 19),
            JalaliDate(1357, 11, 22),
            JalaliDate(1378, 10, 11),
            JalaliDate(1403, 12, 30), // leap-year اسفند 30
            JalaliDate(1399, 6, 31),
        )
        for (j in samples) {
            assertEquals("round-trip for $j", j, j.toGregorian().toJalali())
        }
    }

    @Test
    fun daysInMonthFollowsCalendarShape() {
        for (month in 1..6) {
            assertEquals("month $month has 31 days", 31, JalaliDate(1403, month, 1).daysInMonth())
        }
        for (month in 7..11) {
            assertEquals("month $month has 30 days", 30, JalaliDate(1403, month, 1).daysInMonth())
        }
    }

    @Test
    fun esfandDependsOnLeapYear() {
        // 1403 is a leap year → اسفند has 30 days; 1404 is common → 29 days.
        assertTrue(JalaliDate(1403, 12, 1).isLeapYear())
        assertEquals(30, JalaliDate(1403, 12, 1).daysInMonth())
        assertFalse(JalaliDate(1404, 12, 1).isLeapYear())
        assertEquals(29, JalaliDate(1404, 12, 1).daysInMonth())
    }

    @Test
    fun monthNamesArePersian() {
        assertEquals("فروردین", JalaliDate(1403, 1, 1).monthName)
        assertEquals("اسفند", JalaliDate(1403, 12, 1).monthName)
    }

    @Test
    fun isValidRejectsImpossibleDates() {
        assertTrue(JalaliDate.isValid(1403, 12, 30))
        assertFalse(JalaliDate.isValid(1404, 12, 30)) // اسفند 30 in a common year
        assertFalse(JalaliDate.isValid(1403, 7, 31)) // month 7 only has 30 days
        assertFalse(JalaliDate.isValid(1403, 13, 1)) // no 13th month
        assertFalse(JalaliDate.isValid(1403, 0, 1))
        assertTrue(JalaliDate.isValid(1403, 1, 31))
    }
}
