package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.LoggedPeriod
import ir.ritmeapp.ritme.domain.model.PeriodDayAction

/**
 * Inbound port: the calendar's quick period edits — list what's logged and execute the
 * one [PeriodDayAction] the rules offer for a tapped day.
 */
interface ManagePeriodsUseCase {

    /** Every logged period, for deciding which day offers which action. */
    suspend fun history(): AppResult<List<LoggedPeriod>>

    /** Executes [action] for [day] (start / extend / trim / delete). [PeriodDayAction.None] is a no-op. */
    suspend fun apply(action: PeriodDayAction, day: GregorianDate): AppResult<Unit>
}
