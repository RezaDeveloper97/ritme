package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.PhoneNumber
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toAsciiDigits
import ir.ritmeapp.ritme.domain.port.inbound.SendOtpUseCase
import ir.ritmeapp.ritme.domain.port.inbound.VerifyOtpUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Drives the one-time-code step for [mobile]. Verifies the code ([onSubmit]) or resends a
 * fresh one ([onResend]), and runs the resend countdown. Depends only on inbound ports (§4); the
 * verify use case persists the token, so this VM just maps outcomes into [OtpStatus].
 */
class OtpViewModel(
    private val mobile: PhoneNumber,
    private val verifyOtp: VerifyOtpUseCase,
    private val sendOtp: SendOtpUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(OtpUiState())
    val state: StateFlow<OtpUiState> = _state.asStateFlow()

    private var countdownJob: Job? = null

    init {
        startCountdown()
    }

    fun onCodeChanged(raw: String) {
        // Normalize any Persian/Arabic digits to ASCII first: the backend verifies against an
        // ASCII code (e.g. the test OTP "1111"), so a code typed on a Persian keyboard must not
        // reach it as ۱۱۱۱. Keep only ASCII digits afterward.
        val digits = raw.toAsciiDigits().filter { it in '0'..'9' }.take(_state.value.codeLength)
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
                    // Clear the code so the user retypes from scratch (mirrors the web).
                    is AppResult.Failure -> it.copy(codeInput = "", status = result.error.toOtpError())
                }
            }
        }
    }

    fun onResend() {
        if (_state.value.isBusy || !_state.value.canResend) return
        Breadcrumbs.add("ui:otp:resend")
        _state.update { it.copy(status = OtpStatus.Resending) }
        viewModelScope.launch {
            when (val result = sendOtp(mobile)) {
                is AppResult.Success -> {
                    _state.update { it.copy(codeInput = "", status = OtpStatus.Idle) }
                    startCountdown()
                }

                is AppResult.Failure -> _state.update { it.copy(status = result.error.toOtpError()) }
            }
        }
    }

    fun onNavigationHandled() {
        _state.update { if (it.status is OtpStatus.Authenticated) it.copy(status = OtpStatus.Idle) else it }
    }

    /** Restarts the resend lock at [OtpUiState.RESEND_SECONDS] and ticks it down once per second. */
    private fun startCountdown() {
        countdownJob?.cancel()
        _state.update { it.copy(secondsRemaining = OtpUiState.RESEND_SECONDS) }
        countdownJob = viewModelScope.launch {
            while (_state.value.secondsRemaining > 0) {
                delay(SECOND_MILLIS)
                _state.update { it.copy(secondsRemaining = (it.secondsRemaining - 1).coerceAtLeast(0)) }
            }
        }
    }

    private companion object {
        const val SECOND_MILLIS = 1000L
    }
}
