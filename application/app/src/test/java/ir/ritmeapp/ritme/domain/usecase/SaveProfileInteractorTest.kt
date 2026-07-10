package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyIntention
import ir.ritmeapp.ritme.domain.model.UserProfile
import ir.ritmeapp.ritme.domain.port.outbound.ProfileGateway
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class SaveProfileInteractorTest {

    /** Hand-written fake port (CLAUDE.md §9 — no mocking framework). */
    private class FakeProfileGateway(
        private val result: AppResult<Unit> = AppResult.Success(Unit),
    ) : ProfileGateway {
        var saved: OnboardingAnswers? = null
        override suspend fun saveProfile(answers: OnboardingAnswers): AppResult<Unit> {
            saved = answers
            return result
        }

        override suspend fun fetchProfile(): AppResult<UserProfile> =
            AppResult.Failure(AppError.Unexpected("not used in this test"))
    }

    private val answers = OnboardingAnswers(
        name = "Sara",
        intention = PregnancyIntention.AVOIDING,
        periodDuration = 5,
        cycleDuration = 28,
    )

    @Test
    fun `passes the collected answers through to the gateway`() = runBlocking {
        val gateway = FakeProfileGateway()
        SaveProfileInteractor(gateway)(answers)
        assertEquals(answers, gateway.saved)
    }

    @Test
    fun `returns success from the gateway unchanged`() = runBlocking {
        val result = SaveProfileInteractor(FakeProfileGateway())(answers)
        assertTrue(result is AppResult.Success)
    }

    @Test
    fun `propagates a gateway failure unchanged`() = runBlocking {
        val failure = AppResult.Failure(AppError.Http(422, "Validation failed", "{}"))
        val result = SaveProfileInteractor(FakeProfileGateway(result = failure))(answers)
        assertSame(failure, result)
    }
}
