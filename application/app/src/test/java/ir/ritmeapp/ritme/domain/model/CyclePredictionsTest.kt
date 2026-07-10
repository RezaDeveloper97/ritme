package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class CyclePredictionsTest {

    private fun calc(
        cycleDay: Int? = 10,
        cycleLength: Int? = 28,
        ovulation: Int? = 14,
        fertility: Double? = 20.4,
    ) = CycleCalculation(
        cycleDay = cycleDay,
        cycleLength = cycleLength,
        phase = CyclePhase.FOLLICULAR,
        estimatedOvulationDay = ovulation,
        fertilityPercent = fertility,
        isFertileWindow = false,
        isPeriodTomorrow = false,
        isRecalculating = false,
    )

    @Test
    fun `returns null when there is no cycle data`() {
        assertNull(CyclePredictions.from(calc(cycleDay = null)))
    }

    @Test
    fun `derives day offsets from the calculation`() {
        val p = CyclePredictions.from(calc(cycleDay = 10, cycleLength = 28, ovulation = 14))!!
        assertEquals(18, p.daysUntilNextPeriod) // 28 - 10
        assertEquals(4, p.daysUntilOvulation) // 14 - 10
        assertEquals(0, p.daysUntilFertileWindow) // 4 - 4
        assertEquals(14, p.daysUntilPmsStart) // 18 - 4
        assertEquals(17, p.daysUntilPmsEnd) // 18 - 1
    }

    @Test
    fun `rounds fertility percent and never goes negative on next period`() {
        val p = CyclePredictions.from(calc(cycleDay = 30, cycleLength = 28, fertility = 20.6))!!
        assertEquals(0, p.daysUntilNextPeriod) // max(0, 28 - 30)
        assertEquals(21, p.fertilityPercent) // round(20.6)
    }

    @Test
    fun `progress percent reflects cycle position`() {
        val p = CyclePredictions.from(calc(cycleDay = 14, cycleLength = 28))!!
        assertEquals(50, p.progressPercent)
    }
}
