package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import ir.ritmeapp.ritme.domain.model.AppError

/** Error categories the OTP screen maps to localized copy (keeps user-facing strings out of the VM). */
enum class AuthErrorKey { WrongCode, Network, Server, Unexpected }

/**
 * Maps a transport/domain [AppError] onto the closed set of OTP-screen error categories,
 * preserving the backend's own message for HTTP rejections (e.g. "Invalid OTP code",
 * "OTP has expired") so the user sees the real reason.
 */
fun AppError.toOtpError(): OtpStatus.Error = when (this) {
    is AppError.Network, is AppError.Timeout -> OtpStatus.Error(AuthErrorKey.Network)
    is AppError.Http -> OtpStatus.Error(AuthErrorKey.WrongCode, message)
    is AppError.Validation -> OtpStatus.Error(AuthErrorKey.WrongCode)
    is AppError.Parsing, is AppError.Unexpected -> OtpStatus.Error(AuthErrorKey.Unexpected)
}
