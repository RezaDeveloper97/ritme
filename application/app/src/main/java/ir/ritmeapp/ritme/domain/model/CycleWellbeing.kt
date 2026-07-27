package ir.ritmeapp.ritme.domain.model

/**
 * Phase-based wellbeing estimates derived purely from the current cycle phase — the small "engine"
 * that lets the Home dashboard's «خلاصه هفته» card show real, cycle-driven scores instead of static
 * placeholders. There is no per-day mood/sleep/energy signal in the API, so these are typical-for-
 * this-phase projections (the same way cycle apps estimate wellbeing from the menstrual phase):
 * energy dips during menstruation, peaks around ovulation, and tapers through the luteal phase.
 *
 * Kept locale- and framework-free so it is unit-testable (§9). Returns [UNKNOWN] (all zeros) when the
 * phase is not yet known, which the UI renders as placeholder dashes.
 */
data class CycleWellbeing(
    val moodPercent: Int,
    val energyPercent: Int,
    val sleepPercent: Int,
) {
    companion object {
        val UNKNOWN = CycleWellbeing(moodPercent = 0, energyPercent = 0, sleepPercent = 0)

        /** Projects the typical mood/energy/sleep scores for [phase]; [UNKNOWN] when it is null. */
        fun from(phase: CyclePhase?): CycleWellbeing = when (phase) {
            CyclePhase.MENSTRUATION -> CycleWellbeing(moodPercent = 62, energyPercent = 45, sleepPercent = 70)
            CyclePhase.FOLLICULAR -> CycleWellbeing(moodPercent = 82, energyPercent = 78, sleepPercent = 80)
            CyclePhase.OVULATION -> CycleWellbeing(moodPercent = 90, energyPercent = 88, sleepPercent = 76)
            CyclePhase.LUTEAL -> CycleWellbeing(moodPercent = 68, energyPercent = 60, sleepPercent = 64)
            null -> UNKNOWN
        }

        /** True once there is a real phase to project from (otherwise the card shows dashes). */
        fun isKnown(phase: CyclePhase?): Boolean = phase != null
    }
}
