package ir.ritmeapp.ritme.adapter.inbound.ui.otp

/**
 * Immutable snapshot for the one-time-code step. [codeLength] is fixed for the challenge and
 * gates submission; [secondsRemaining] drives the resend countdown (web: 59 → 0, then the resend
 * link becomes tappable). The async outcome lives in [status] as a sealed type (CLAUDE.md §4).
 */
data class OtpUiState(
    val codeInput: String = "",
    val codeLength: Int = CODE_LENGTH,
    val secondsRemaining: Int = RESEND_SECONDS,
    val status: OtpStatus = OtpStatus.Idle,
) {
    val isBusy: Boolean get() = status is OtpStatus.Submitting || status is OtpStatus.Resending
    val canSubmit: Boolean get() = codeInput.length == codeLength && !isBusy

    /** Resend is only offered once the countdown has elapsed (mirrors the web timer gate). */
    val canResend: Boolean get() = secondsRemaining <= 0 && !isBusy

    companion object {
        /** Ritme OTP codes are always 4 digits (backend `code` validation: `size:4`). */
        const val CODE_LENGTH = 4

        /** Seconds the resend link stays locked after a code is sent (web starts at 59). */
        const val RESEND_SECONDS = 59
    }
}

sealed interface OtpStatus {
    data object Idle : OtpStatus
    data object Submitting : OtpStatus
    data object Resending : OtpStatus
    data class Error(val key: AuthErrorKey) : OtpStatus

    /** Code accepted — tokens stored; navigate home. */
    data object Authenticated : OtpStatus
}
