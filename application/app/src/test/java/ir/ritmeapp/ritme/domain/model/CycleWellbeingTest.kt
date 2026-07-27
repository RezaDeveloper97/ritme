package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CycleWellbeingTest {

    @Test
    fun `energy peaks at ovulation and dips during menstruation`() {
        val ovulation = CycleWellbeing.from(CyclePhase.OVULATION)
        val menstruation = CycleWellbeing.from(CyclePhase.MENSTRUATION)
        assertTrue(ovulation.energyPercent > menstruation.energyPercent)
    }

    @Test
    fun `unknown phase returns all zeros and is not known`() {
        assertEquals(CycleWellbeing.UNKNOWN, CycleWellbeing.from(null))
        assertFalse(CycleWellbeing.isKnown(null))
        assertTrue(CycleWellbeing.isKnown(CyclePhase.LUTEAL))
    }

    @Test
    fun `every known phase projects in-range percentages`() {
        CyclePhase.entries.forEach { phase ->
            val w = CycleWellbeing.from(phase)
            listOf(w.moodPercent, w.energyPercent, w.sleepPercent).forEach {
                assertTrue("$phase in range", it in 1..100)
            }
        }
    }
}
