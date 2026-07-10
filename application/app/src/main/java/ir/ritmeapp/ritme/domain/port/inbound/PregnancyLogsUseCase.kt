package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.PregnancySymptomLog
import ir.ritmeapp.ritme.domain.model.PregnancyWeeklyLog

/**
 * Inbound port: the three pregnancy-log forms (daily symptoms, weekly check-in,
 * fetal movement).
 */
interface PregnancyLogsUseCase {

    /** The symptom log saved for [date], or null. */
    suspend fun symptomLog(date: GregorianDate): AppResult<PregnancySymptomLog?>

    /** Upserts a symptom log; returns how many new alerts it raised. */
    suspend fun saveSymptomLog(log: PregnancySymptomLog): AppResult<Int>

    /** The weekly check-in for [week], or null. */
    suspend fun weeklyLog(week: Int): AppResult<PregnancyWeeklyLog?>

    /** Upserts the weekly check-in. */
    suspend fun saveWeeklyLog(log: PregnancyWeeklyLog): AppResult<Unit>

    /** Upserts a fetal-movement entry. */
    suspend fun saveFetalMovement(log: FetalMovementLog): AppResult<Unit>
}
