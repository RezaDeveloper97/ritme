package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import ir.ritmeapp.ritme.domain.model.AppError

/**
 * Error categories the OTP screen maps to localized copy (keeps user-facing strings out of the
 * VM). Mirrors the web `authErrorKey` buckets so raw server strings never leak to the user (§11).
 */
enum class AuthErrorKey { InvalidCode, TooMany, Network, Generic }

private const val STATUS_TOO_MANY_REQUESTS = 429
private const val STATUS_BAD_REQUEST = 400
private const val STATUS_UNPROCESSABLE = 422

/**
 * Maps a transport/domain [AppError] onto the closed set of OTP-screen error categories, matching
 * the web: 400/422 → wrong-or-expired code, 429 → too-many-attempts, no HTTP status → network,
 * anything else → a calm generic message. The backend's raw message is never shown.
 */
fun AppError.toOtpError(): OtpStatus.Error = when (this) {
    is AppError.Network, is AppError.Timeout -> OtpStatus.Error(AuthErrorKey.Network)
    is AppError.Http -> when (statusCode) {
        STATUS_TOO_MANY_REQUESTS -> OtpStatus.Error(AuthErrorKey.TooMany)
        STATUS_BAD_REQUEST, STATUS_UNPROCESSABLE -> OtpStatus.Error(AuthErrorKey.InvalidCode)
        else -> OtpStatus.Error(AuthErrorKey.Generic)
    }
    is AppError.Validation -> OtpStatus.Error(AuthErrorKey.InvalidCode)
    is AppError.Parsing, is AppError.Storage, is AppError.Unexpected ->
        OtpStatus.Error(AuthErrorKey.Generic)
}
