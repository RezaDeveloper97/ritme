package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyOnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyProfile
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancySymptomLog
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent
import ir.ritmeapp.ritme.domain.model.PregnancyWeeklyLog

/**
 * Outbound port for pregnancy mode (`/pregnancy/…`). One port for the whole resource family:
 * the pregnancy screens always use these together and they share one backend feature, so
 * splitting further would only multiply fakes (§4 I, judged per role not per endpoint).
 */
interface PregnancyGateway {

    /** Switches the account into pregnancy mode. */
    suspend fun activate(): AppResult<Unit>

    /** Switches back to cycle mode. */
    suspend fun deactivate(): AppResult<Unit>

    /** Submits the pregnancy onboarding answers (dating source + history). */
    suspend fun completeOnboarding(answers: PregnancyOnboardingAnswers): AppResult<Unit>

    /** Gestational age, due date, and risk flags for the tracker hero. */
    suspend fun status(): AppResult<PregnancyStatus>

    /** The pregnancy profile, or null when none exists yet (backend 404). */
    suspend fun profile(): AppResult<PregnancyProfile?>

    /** Localized educational content for [week] (1..40). */
    suspend fun contentForWeek(week: Int): AppResult<PregnancyWeekContent>

    /** The symptom log for [date], or null when nothing was logged. */
    suspend fun symptomLog(date: GregorianDate): AppResult<PregnancySymptomLog?>

    /** Upserts a symptom log; returns how many new alerts it raised. */
    suspend fun saveSymptomLog(log: PregnancySymptomLog): AppResult<Int>

    /** The weekly check-in for [week], or null when not yet filled. */
    suspend fun weeklyLog(week: Int): AppResult<PregnancyWeeklyLog?>

    /** Upserts the weekly check-in. */
    suspend fun saveWeeklyLog(log: PregnancyWeeklyLog): AppResult<Unit>

    /** Upserts a fetal-movement entry. */
    suspend fun saveFetalMovement(log: FetalMovementLog): AppResult<Unit>

    /** Every alert, newest first. */
    suspend fun alerts(): AppResult<List<PregnancyAlert>>

    /** Marks one alert read. */
    suspend fun markAlertRead(id: Long): AppResult<Unit>

    /** Dismisses one alert. */
    suspend fun dismissAlert(id: Long): AppResult<Unit>

    /** Marks every alert read. */
    suspend fun markAllAlertsRead(): AppResult<Unit>
}
