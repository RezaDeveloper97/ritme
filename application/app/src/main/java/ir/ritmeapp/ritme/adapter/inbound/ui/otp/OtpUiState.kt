package ir.ritmeapp.ritme.adapter.inbound.ui.otp

/**
 * Immutable snapshot for the one-time-code step. [codeLength] is fixed for the challenge and
 * gates submission; the async outcome lives in [status] as a sealed type (CLAUDE.md §4).
 */
data class OtpUiState(
    val codeInput: String = "",
    val codeLength: Int = CODE_LENGTH,
    val status: OtpStatus = OtpStatus.Idle,
) {
    val isBusy: Boolean get() = status is OtpStatus.Submitting || status is OtpStatus.Resending
    val canSubmit: Boolean get() = codeInput.length == codeLength && !isBusy

    companion object {
        /** Ritme OTP codes are always 4 digits (backend `code` validation: `size:4`). */
        const val CODE_LENGTH = 4
    }
}

sealed interface OtpStatus {
    data object Idle : OtpStatus
    data object Submitting : OtpStatus
    data object Resending : OtpStatus
    data class Error(val key: AuthErrorKey, val serverMessage: String? = null) : OtpStatus

    /** Code accepted — tokens stored; navigate home. */
    data object Authenticated : OtpStatus
}
