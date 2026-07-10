package ir.ritmeapp.ritme.domain.model

/**
 * The backend's acknowledgement that a one-time code was dispatched for a mobile number
 * (`POST /api/v1/auth/send-otp`). [newUser] is true when this mobile had no account yet — the
 * verification step will register it — and [expireSeconds] is how long the code stays valid,
 * which the verification screen uses to gate a resend.
 */
data class OtpChallenge(
    val newUser: Boolean,
    val expireSeconds: Int,
)
