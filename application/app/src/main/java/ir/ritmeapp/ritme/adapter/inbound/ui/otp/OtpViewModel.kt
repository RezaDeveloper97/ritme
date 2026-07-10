package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.domain.port.inbound.VerifyOtpUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the one-time-code step for [mobile]. Verifies the code ([onSubmit]) or resends a
 * fresh one ([onResend]). Depends only on inbound ports (§4); the verify use case persists the
 * token, so this VM just maps outcomes into [OtpStatus].
 */
class OtpViewModel(
    private val mobile: PhoneNumber,
    private val verifyOtp: VerifyOtpUseCase,
    private val sendOtp: SendOtpUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(OtpUiState())
    val state: StateFlow<OtpUiState> = _state.asStateFlow()

    fun onCodeChanged(raw: String) {
        val digits = raw.filter { it.isDigit() }.take(_state.value.codeLength)
        _state.update {
            it.copy(
                codeInput = digits,
                status = if (it.status is OtpStatus.Error) OtpStatus.Idle else it.status,
            )
        }
    }

    fun onSubmit() {
        if (!_state.value.canSubmit) return
        Breadcrumbs.add("ui:otp:submit")
        _state.update { it.copy(status = OtpStatus.Submitting) }
        viewModelScope.launch {
            _state.update {
                when (val result = verifyOtp(mobile, it.codeInput)) {
                    is AppResult.Success -> it.copy(status = OtpStatus.Authenticated)
                    is AppResult.Failure -> it.copy(status = result.error.toOtpError())
                }
            }
        }
    }

    fun onResend() {
        if (_state.value.isBusy) return
        Breadcrumbs.add("ui:otp:resend")
        _state.update { it.copy(status = OtpStatus.Resending) }
        viewModelScope.launch {
            _state.update {
                when (val result = sendOtp(mobile)) {
                    is AppResult.Success -> it.copy(codeInput = "", status = OtpStatus.Idle)
                    is AppResult.Failure -> it.copy(status = result.error.toOtpError())
                }
            }
        }
    }

    fun onNavigationHandled() {
        _state.update { if (it.status is OtpStatus.Authenticated) it.copy(status = OtpStatus.Idle) else it }
    }
}
