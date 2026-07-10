package ir.ritmeapp.ritme.domain.usecase

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OtpChallenge
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.domain.port.outbound.AuthGateway

/**
 * Default [SendOtpUseCase]. A thin delegation today, but the deliberate seam where
 * resend throttling / attempt counting will live without touching UI or network code.
 */
class SendOtpInteractor(
    private val authGateway: AuthGateway,
) : SendOtpUseCase {

    override suspend fun invoke(mobile: PhoneNumber): AppResult<OtpChallenge> =
        authGateway.sendOtp(mobile)
}
