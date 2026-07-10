package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.model.PhoneNumber

/**
 * Inbound port: complete a login by verifying the one-time code sent to [mobile]. On success
 * the issued token is persisted before returning, so callers need only react to the outcome.
 */
interface VerifyOtpUseCase {
    suspend operator fun invoke(mobile: PhoneNumber, code: String): AppResult<AuthTokens>
}
