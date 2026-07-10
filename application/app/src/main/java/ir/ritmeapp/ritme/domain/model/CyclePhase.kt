package ir.ritmeapp.ritme.domain.model

/**
 * The four menstrual-cycle phases the backend reports (`cycle/today.phase`,
 * `messages/daily.context_info.phase`). Note the backend emits `menstruation` (not `period`) and
 * never emits a `fertile` phase — fertile/period visual state is driven by the boolean flags on
 * [CycleCalculation], not by this enum. The Persian label is resolved in the UI layer.
 */
enum class CyclePhase(val apiValue: String) {
    MENSTRUATION("menstruation"),
    FOLLICULAR("follicular"),
    OVULATION("ovulation"),
    LUTEAL("luteal"),
    ;

    companion object {
        /** Maps a backend phase string to a [CyclePhase], or null if absent/unknown. */
        fun fromApi(value: String?): CyclePhase? = entries.firstOrNull { it.apiValue == value }
    }
}
