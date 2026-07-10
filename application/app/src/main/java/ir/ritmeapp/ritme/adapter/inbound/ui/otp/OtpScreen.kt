package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.PersianDigitsTransformation
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The one-time-code step. Records the last-safe-screen on entry (§7.2) and, on a verified
 * code, calls [onAuthenticated]. Swipe-back is provided by the nav host wrapper.
 */
@Composable
fun OtpScreen(
    mobile: String,
    onAuthenticated: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: OtpViewModel =
        viewModel(factory = container.otpViewModelFactory(mobile))
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:otp:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Otp.ROUTE, mobile, System.currentTimeMillis()),
        )
    }

    LaunchedEffect(state.status) {
        if (state.status is OtpStatus.Authenticated) {
            onAuthenticated()
            viewModel.onNavigationHandled()
        }
    }

    OtpContent(
        state = state,
        mobile = mobile,
        onCodeChanged = viewModel::onCodeChanged,
        onSubmit = viewModel::onSubmit,
        onResend = viewModel::onResend,
        modifier = modifier,
    )
}

@Composable
private fun OtpContent(
    state: OtpUiState,
    mobile: String,
    onCodeChanged: (String) -> Unit,
    onSubmit: () -> Unit,
    onResend: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Scaffold(modifier = modifier.fillMaxSize(), containerColor = colors.background) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 24.dp)
                .imePadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(64.dp))
            Text(
                text = stringResource(R.string.otp_title),
                style = MaterialTheme.typography.headlineMedium,
                color = colors.ink,
            )
            Spacer(Modifier.height(10.dp))
            Text(
                text = stringResource(R.string.otp_subtitle, mobile.toPersianDigits()),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(28.dp))
            OutlinedTextField(
                value = state.codeInput,
                onValueChange = onCodeChanged,
                modifier = Modifier.fillMaxWidth(),
                label = { Text(stringResource(R.string.otp_label)) },
                singleLine = true,
                isError = state.status is OtpStatus.Error,
                visualTransformation = PersianDigitsTransformation,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { onSubmit() }),
                shape = MaterialTheme.shapes.medium,
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = colors.pink,
                    unfocusedBorderColor = colors.outline,
                    focusedLabelColor = colors.pink,
                    cursorColor = colors.pink,
                    errorBorderColor = colors.error,
                ),
            )
            ErrorMessage(state.status, colors)
            Spacer(Modifier.height(24.dp))
            Button(
                onClick = onSubmit,
                enabled = state.canSubmit,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.buttonColors(
                    containerColor = colors.pink,
                    contentColor = colors.onPink,
                    disabledContainerColor = colors.pink.copy(alpha = 0.4f),
                    disabledContentColor = colors.onPink.copy(alpha = 0.8f),
                ),
            ) {
                if (state.status is OtpStatus.Submitting) {
                    CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp, color = colors.onPink)
                } else {
                    Text(stringResource(R.string.otp_submit), style = MaterialTheme.typography.labelLarge)
                }
            }
            Spacer(Modifier.height(8.dp))
            TextButton(onClick = onResend, enabled = !state.isBusy) {
                if (state.status is OtpStatus.Resending) {
                    CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = colors.pink)
                } else {
                    Text(
                        text = stringResource(R.string.otp_resend),
                        color = colors.pink,
                        style = MaterialTheme.typography.labelLarge,
                    )
                }
            }
        }
    }
}

@Composable
private fun ErrorMessage(status: OtpStatus, colors: RitmeColors) {
    if (status is OtpStatus.Error) {
        Spacer(Modifier.height(8.dp))
        Text(
            text = status.serverMessage ?: stringResource(status.key.messageRes()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.error,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@StringRes
private fun AuthErrorKey.messageRes(): Int = when (this) {
    AuthErrorKey.WrongCode -> R.string.otp_error_wrong_code
    AuthErrorKey.Network -> R.string.login_error_network
    AuthErrorKey.Server -> R.string.login_error_server
    AuthErrorKey.Unexpected -> R.string.login_error_unexpected
}
