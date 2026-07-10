package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.UserProfile

/**
 * Outbound port for the user's profile (`/profile`). The domain states the need in its own
 * vocabulary ([OnboardingAnswers], [UserProfile]); the adapter owns the JSON shape, the bearer
 * token, and the date conversion. Never throws across the boundary — every outcome is an
 * [AppResult].
 */
interface ProfileGateway {

    /** Reads the full profile + BMI the profile screens render from. */
    suspend fun fetchProfile(): AppResult<UserProfile>

    /** Upserts the fields set in [answers]; unset fields are left untouched server-side. */
    suspend fun saveProfile(answers: OnboardingAnswers): AppResult<Unit>
}
