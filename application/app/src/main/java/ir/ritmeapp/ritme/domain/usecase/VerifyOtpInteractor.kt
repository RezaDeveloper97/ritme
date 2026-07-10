package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.VerifyOtpUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore

/**
 * Default [VerifyOtpUseCase]. Owns the "verify, then persist the token" policy so no UI or
 * adapter has to remember it: a successful login is not complete until the token is stored.
 */
class VerifyOtpInteractor(
    private val authGateway: AuthGateway,
    private val tokenStore: TokenStore,
) : VerifyOtpUseCase {

    override suspend fun invoke(mobile: PhoneNumber, code: String): AppResult<AuthTokens> {
        val result = authGateway.verifyOtp(mobile, code)
        if (result is AppResult.Success) tokenStore.save(result.value)
        return result
    }
}
