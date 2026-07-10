package ir.ritmeapp.ritme.adapter.inbound.ui.login

/**
 * Immutable snapshot the login screen renders from. [phoneInput] holds raw ASCII digits; the
 * Persian rendering is a display-only transformation. The async outcome lives in [status] as a
 * sealed type so the UI never juggles loose boolean/nullable flags (CLAUDE.md §4).
 */
data class LoginUiState(
    val phoneInput: String = "",
    val isPhoneValid: Boolean = false,
    val status: LoginStatus = LoginStatus.Idle,
) {
    val isSubmitting: Boolean get() = status is LoginStatus.Submitting
    val canSubmit: Boolean get() = !isSubmitting && isPhoneValid
}

/** The mutually-exclusive states the login action can be in. */
sealed interface LoginStatus {
    data object Idle : LoginStatus
    data object Submitting : LoginStatus
    data class Error(val key: LoginErrorKey, val serverMessage: String? = null) : LoginStatus

    /** A code was sent — advance to the OTP step for this mobile number. [newUser] gates onboarding. */
    data class Validated(val mobileNational: String, val newUser: Boolean) : LoginStatus
}

/** Error categories the UI maps to localized copy (keeps user-facing strings out of the VM). */
enum class LoginErrorKey { InvalidPhone, Network, Server, Unexpected }
