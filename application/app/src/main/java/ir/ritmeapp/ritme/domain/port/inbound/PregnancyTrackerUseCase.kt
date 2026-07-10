package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyOnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyProfile
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent

/**
 * Inbound port: everything the pregnancy tracker + its onboarding do — mode switching,
 * status, weekly educational content, and safety alerts.
 */
interface PregnancyTrackerUseCase {

    /** Switches the account into pregnancy mode. */
    suspend fun activate(): AppResult<Unit>

    /** Returns the account to cycle mode. */
    suspend fun deactivate(): AppResult<Unit>

    /** Submits the pregnancy onboarding answers. */
    suspend fun completeOnboarding(answers: PregnancyOnboardingAnswers): AppResult<Unit>

    /** The tracker headline (gestational age, due date, risk flags). */
    suspend fun status(): AppResult<PregnancyStatus>

    /** The pregnancy profile, or null when onboarding hasn't produced one yet. */
    suspend fun profile(): AppResult<PregnancyProfile?>

    /** Localized educational content for [week] (1..40). */
    suspend fun contentForWeek(week: Int): AppResult<PregnancyWeekContent>

    /** Every safety alert, newest first. */
    suspend fun alerts(): AppResult<List<PregnancyAlert>>

    /** Marks one alert read. */
    suspend fun markAlertRead(id: Long): AppResult<Unit>

    /** Dismisses one alert. */
    suspend fun dismissAlert(id: Long): AppResult<Unit>

    /** Marks every alert read. */
    suspend fun markAllAlertsRead(): AppResult<Unit>
}
