package ir.ritmeapp.ritme.adapter.inbound.ui.login

import androidx.annotation.StringRes
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.PersianDigitsTransformation
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The root login screen: mobile entry. Owns the [LoginViewModel], records the last-safe-screen
 * + breadcrumb on entry (§7.2), and once the backend has SMSed a code hands the mobile number
 * to [onValidated]. Being the start destination, it is intentionally not swipe-back-wrapped.
 */
@Composable
fun LoginScreen(
    onValidated: (mobile: String, newUser: Boolean) -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: LoginViewModel = viewModel(factory = container.loginViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:login:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Login.route, null, System.currentTimeMillis()),
        )
    }

    LaunchedEffect(state.status) {
        val status = state.status
        if (status is LoginStatus.Validated) {
            onValidated(status.mobileNational, status.newUser)
            viewModel.onValidatedHandled()
        }
    }

    LoginContent(
        state = state,
        snackbarHostState = snackbarHostState,
        onPhoneChanged = viewModel::onPhoneChanged,
        onSubmit = viewModel::onSubmit,
        modifier = modifier,
    )
}

@Composable
private fun LoginContent(
    state: LoginUiState,
    snackbarHostState: SnackbarHostState,
    onPhoneChanged: (String) -> Unit,
    onSubmit: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 24.dp)
                .imePadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(72.dp))
            BrandMark()
            Spacer(Modifier.height(28.dp))
            Text(
                text = stringResource(R.string.login_title),
                style = MaterialTheme.typography.headlineLarge,
                color = colors.ink,
            )
            Spacer(Modifier.height(12.dp))
            Text(
                text = stringResource(R.string.login_subtitle),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(32.dp))
            PhoneField(state, onPhoneChanged, onSubmit, colors)
            ErrorMessage(state.status, colors)
            Spacer(Modifier.height(24.dp))
            SubmitButton(state, onSubmit, colors)
        }
    }
}

@Composable
private fun BrandMark() {
    Image(
        painter = painterResource(R.drawable.ritme_logo),
        contentDescription = stringResource(R.string.cd_app_logo),
        modifier = Modifier
            .size(104.dp)
            .clip(RoundedCornerShape(26.dp)),
    )
}

@Composable
private fun PhoneField(
    state: LoginUiState,
    onPhoneChanged: (String) -> Unit,
    onSubmit: () -> Unit,
    colors: RitmeColors,
) {
    OutlinedTextField(
        value = state.phoneInput,
        onValueChange = onPhoneChanged,
        modifier = Modifier.fillMaxWidth(),
        label = { Text(stringResource(R.string.login_phone_label)) },
        placeholder = { Text(stringResource(R.string.login_phone_placeholder)) },
        singleLine = true,
        isError = state.status is LoginStatus.Error,
        visualTransformation = PersianDigitsTransformation,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone, imeAction = ImeAction.Done),
        keyboardActions = KeyboardActions(onDone = { if (state.canSubmit) onSubmit() }),
        shape = MaterialTheme.shapes.medium,
        colors = fieldColors(colors),
    )
}

@Composable
private fun ErrorMessage(status: LoginStatus, colors: RitmeColors) {
    if (status is LoginStatus.Error) {
        Spacer(Modifier.height(8.dp))
        Text(
            text = status.serverMessage ?: stringResource(status.key.messageRes()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.error,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun SubmitButton(
    state: LoginUiState,
    onSubmit: () -> Unit,
    colors: RitmeColors,
) {
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
        if (state.isSubmitting) {
            CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp, color = colors.onPink)
        } else {
            Text(
                text = stringResource(R.string.login_submit),
                style = MaterialTheme.typography.labelLarge,
            )
        }
    }
}

@Composable
private fun fieldColors(colors: RitmeColors) = OutlinedTextFieldDefaults.colors(
    focusedBorderColor = colors.pink,
    unfocusedBorderColor = colors.outline,
    focusedLabelColor = colors.pink,
    cursorColor = colors.pink,
    errorBorderColor = colors.error,
)

@StringRes
private fun LoginErrorKey.messageRes(): Int = when (this) {
    LoginErrorKey.InvalidPhone -> R.string.login_error_invalid_phone
    LoginErrorKey.Network -> R.string.login_error_network
    LoginErrorKey.Server -> R.string.login_error_server
    LoginErrorKey.Unexpected -> R.string.login_error_unexpected
}
