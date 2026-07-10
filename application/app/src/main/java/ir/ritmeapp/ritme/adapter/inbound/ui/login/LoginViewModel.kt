package ir.ritmeapp.ritme.adapter.inbound.ui.login

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the login screen: mobile entry, then a `send-otp` request. Depends only on the
 * inbound port — no networking/parsing here (§4). On success it surfaces
 * [LoginStatus.Validated], which the screen turns into navigation to the OTP step.
 */
class LoginViewModel(
    private val sendOtp: SendOtpUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(LoginUiState())
    val state: StateFlow<LoginUiState> = _state.asStateFlow()

    fun onPhoneChanged(raw: String) {
        val sanitized = raw.filter { it.isDigit() || it == '+' }.take(MAX_PHONE_LENGTH)
        _state.update {
            it.copy(
                phoneInput = sanitized,
                isPhoneValid = PhoneNumber.isValid(sanitized),
                status = if (it.status is LoginStatus.Error) LoginStatus.Idle else it.status,
            )
        }
    }

    fun onSubmit() {
        when (val parsed = PhoneNumber.parse(_state.value.phoneInput)) {
            is AppResult.Failure ->
                _state.update { it.copy(status = LoginStatus.Error(LoginErrorKey.InvalidPhone)) }

            is AppResult.Success -> {
                Breadcrumbs.add("ui:login:send_otp")
                _state.update { it.copy(status = LoginStatus.Submitting) }
                viewModelScope.launch {
                    when (val result = sendOtp(parsed.value)) {
                        is AppResult.Success -> _state.update {
                            it.copy(status = LoginStatus.Validated(parsed.value.national, result.value.newUser))
                        }

                        is AppResult.Failure -> _state.update {
                            it.copy(status = result.error.toStatus())
                        }
                    }
                }
            }
        }
    }

    /** Resets the one-shot [LoginStatus.Validated] so returning here won't re-navigate. */
    fun onValidatedHandled() {
        _state.update { if (it.status is LoginStatus.Validated) it.copy(status = LoginStatus.Idle) else it }
    }

    private companion object {
        const val MAX_PHONE_LENGTH = 15
    }
}

/** Maps a transport/domain failure onto the closed set of UI error categories. */
internal fun AppError.toStatus(): LoginStatus.Error = when (this) {
    is AppError.Network, is AppError.Timeout -> LoginStatus.Error(LoginErrorKey.Network)
    is AppError.Http -> LoginStatus.Error(LoginErrorKey.Server, message)
    is AppError.Validation -> LoginStatus.Error(LoginErrorKey.InvalidPhone)
    is AppError.Parsing, is AppError.Storage, is AppError.Unexpected ->
        LoginStatus.Error(LoginErrorKey.Unexpected)
}
