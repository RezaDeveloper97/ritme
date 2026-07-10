package ir.ritmeapp.ritme.domain.port.inbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.OtpChallenge
import ir.ritmeapp.ritme.domain.model.PhoneNumber

/**
 * Inbound port: start (or resend) a login by asking the backend to SMS a one-time code to the
 * user's mobile number. ViewModels depend on this interface, never on the concrete gateway.
 */
interface SendOtpUseCase {
    suspend operator fun invoke(mobile: PhoneNumber): AppResult<OtpChallenge>
}
