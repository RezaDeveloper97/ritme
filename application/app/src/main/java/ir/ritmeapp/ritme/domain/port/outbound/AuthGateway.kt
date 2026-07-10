package ir.ritmeapp.ritme.domain.port.outbound

import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.AuthTokens
import ir.ritmeapp.ritme.domain.model.OtpChallenge
import ir.ritmeapp.ritme.domain.model.PhoneNumber

/**
 * Outbound port for the Ritme OTP login flow. The real implementation lives in the network
 * adapter; the domain knows only this small, role-specific contract (CLAUDE.md §4). No method
 * throws across the boundary — every outcome is an [AppResult].
 *
 * Flow (two steps, no captcha/password/refresh): [sendOtp] SMSes a 4-digit code for the
 * mobile number, then [verifyOtp] exchanges that code for an access token. A first-time
 * mobile is registered automatically during verification.
 */
interface AuthGateway {

    /** Asks the backend to SMS a one-time code to [mobile]. */
    suspend fun sendOtp(mobile: PhoneNumber): AppResult<OtpChallenge>

    /** Exchanges the one-time [code] sent to [mobile] for an access token. */
    suspend fun verifyOtp(mobile: PhoneNumber, code: String): AppResult<AuthTokens>
}
