package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.model.OtpChallenge
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
import org.junit.Assert.assertTrue
import org.junit.Test

class OtpAuthInteractorTest {

    /** Hand-written fake port (CLAUDE.md §9 — no mocking framework). */
    private class FakeAuthGateway(
        private val sendResult: AppResult<OtpChallenge> = AppResult.Success(OtpChallenge(false, 120)),
        private val verifyResult: AppResult<AuthTokens> = AppResult.Success(AuthTokens("token")),
    ) : AuthGateway {
        var sentTo: PhoneNumber? = null
        var verifiedCode: String? = null

        override suspend fun sendOtp(mobile: PhoneNumber): AppResult<OtpChallenge> {
            sentTo = mobile
            return sendResult
        }

        override suspend fun verifyOtp(mobile: PhoneNumber, code: String): AppResult<AuthTokens> {
            verifiedCode = code
            return verifyResult
        }
    }

    private class FakeTokenStore : TokenStore {
        var saved: AuthTokens? = null
        override suspend fun save(tokens: AuthTokens) { saved = tokens }
        override suspend fun load(): AuthTokens? = saved
        override suspend fun clear() { saved = null }
    }

    private val mobile = (PhoneNumber.parse("09123456789") as AppResult.Success).value

    @Test
    fun `sendOtp passes the mobile through to the gateway`() = runBlocking {
        val gateway = FakeAuthGateway()
        SendOtpInteractor(gateway)(mobile)
        assertEquals(mobile, gateway.sentTo)
    }

    @Test
    fun `sendOtp propagates a gateway failure unchanged`() = runBlocking {
        val failure = AppResult.Failure(AppError.Network("offline"))
        val result = SendOtpInteractor(FakeAuthGateway(sendResult = failure))(mobile)
        assertSame(failure, result)
    }

    @Test
    fun `verifyOtp persists the issued token on success`() = runBlocking {
        val store = FakeTokenStore()
        val gateway = FakeAuthGateway(verifyResult = AppResult.Success(AuthTokens("abc")))

        val result = VerifyOtpInteractor(gateway, store)(mobile, "1234")

        assertTrue(result is AppResult.Success)
        assertEquals("1234", gateway.verifiedCode)
        assertEquals(AuthTokens("abc"), store.saved)
    }

    @Test
    fun `verifyOtp does not persist anything on failure`() = runBlocking {
        val store = FakeTokenStore()
        val failure = AppResult.Failure(AppError.Http(422, "Invalid OTP code", "{}"))
        val result = VerifyOtpInteractor(FakeAuthGateway(verifyResult = failure), store)(mobile, "0000")

        assertSame(failure, result)
        assertNull(store.saved)
    }
}
