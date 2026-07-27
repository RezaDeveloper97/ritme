package ir.ritmeapp.ritme.adapter.inbound.ui.otp

import androidx.activity.compose.LocalOnBackPressedDispatcherOwner
import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeSoftButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The one-time-code step. Mirrors the web OTP page (frontend `screens/auth-otp`): a white screen
 * with a NavBack header, an edit-phone pill, four auto-advancing digit cells, a resend countdown,
 * and a bottom-pinned verify CTA. Records the last-safe-screen on entry (§7.2) and, on a verified
 * code, calls [onAuthenticated]. Swipe-back is provided by the nav host wrapper; the header/edit
 * controls trigger the same system Back so no navigation wiring lives here.
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
    val backDispatcher = LocalOnBackPressedDispatcherOwner.current?.onBackPressedDispatcher
    val onBack: () -> Unit = { backDispatcher?.onBackPressed() }

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
        onBack = onBack,
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
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    val focusRequester = remember { FocusRequester() }

    // Auto-focus the code on entry, and re-focus after a failed verify (which cleared it).
    LaunchedEffect(Unit) { focusRequester.requestFocus() }
    LaunchedEffect(state.status) {
        if (state.status is OtpStatus.Error) focusRequester.requestFocus()
    }

    Scaffold(modifier = modifier.fillMaxSize(), containerColor = colors.surface) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .imePadding(),
        ) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp)
                    .padding(horizontal = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                HeaderIconButton(
                    icon = R.drawable.ic_chevron_right,
                    contentDescription = stringResource(R.string.action_back),
                    onClick = onBack,
                )
            }

            Column(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 22.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Column(Modifier.fillMaxWidth()) {
                    Text(
                        text = stringResource(R.string.auth_otp_title),
                        style = MaterialTheme.typography.titleLarge.copy(
                            fontSize = 20.sp,
                            fontWeight = FontWeight.ExtraBold,
                        ),
                        color = colors.ink,
                        textAlign = TextAlign.Start,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Spacer(Modifier.height(12.dp))
                    Text(
                        text = stringResource(R.string.auth_otp_subtitle, mobile.toPersianDigits()),
                        style = MaterialTheme.typography.bodyMedium.copy(fontSize = 13.sp),
                        color = colors.inkMuted,
                        textAlign = TextAlign.Start,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                Spacer(Modifier.height(16.dp))
                RitmeSoftButton(
                    text = stringResource(R.string.auth_otp_edit_phone),
                    onClick = onBack,
                    leadingIcon = {
                        Icon(
                            painter = painterResource(R.drawable.ic_pencil),
                            contentDescription = null,
                            tint = colors.steel,
                            modifier = Modifier.size(15.dp),
                        )
                    },
                )
                Spacer(Modifier.height(30.dp))

                OtpCells(
                    code = state.codeInput,
                    length = state.codeLength,
                    focusRequester = focusRequester,
                    onCodeChanged = onCodeChanged,
                    onSubmit = onSubmit,
                    colors = colors,
                )

                ErrorMessage(state.status, colors)

                Spacer(Modifier.height(26.dp))
                ResendRow(state, onResend, colors)
            }

            RitmePrimaryButton(
                text = stringResource(
                    if (state.status is OtpStatus.Submitting) R.string.auth_otp_verifying else R.string.auth_otp_submit,
                ),
                onClick = onSubmit,
                enabled = state.canSubmit,
                modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 14.dp, bottom = 8.dp),
            )
        }
    }
}

/**
 * Four auto-advancing digit cells (web `.otp-box`). The digits are drawn as static cells; a
 * transparent, edge-to-edge [BasicTextField] overlaid on top captures input so backspace and
 * auto-advance fall out of normal single-field editing while each cell still shows one digit.
 */
@Composable
private fun OtpCells(
    code: String,
    length: Int,
    focusRequester: FocusRequester,
    onCodeChanged: (String) -> Unit,
    onSubmit: () -> Unit,
    colors: RitmeColors,
) {
    // dir="ltr" in the web — digits fill left-to-right regardless of the RTL page.
    CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
        Box {
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                repeat(length) { index ->
                    OtpCell(
                        char = code.getOrNull(index),
                        focused = index == code.length,
                        colors = colors,
                    )
                }
            }
            BasicTextField(
                value = code,
                onValueChange = onCodeChanged,
                modifier = Modifier
                    .matchParentSize()
                    .focusRequester(focusRequester),
                singleLine = true,
                textStyle = TextStyle(color = Color.Transparent),
                cursorBrush = SolidColor(Color.Transparent),
                keyboardOptions = KeyboardOptions(
                    keyboardType = KeyboardType.NumberPassword,
                    imeAction = ImeAction.Done,
                ),
                keyboardActions = KeyboardActions(onDone = { onSubmit() }),
                decorationBox = { inner -> inner() },
            )
        }
    }
}

@Composable
private fun OtpCell(char: Char?, focused: Boolean, colors: RitmeColors) {
    val shape = RoundedCornerShape(16.dp)
    val borderColor = when {
        char != null -> colors.pink
        focused -> colors.pinkLight
        else -> colors.outline
    }
    Box(
        modifier = Modifier
            .size(width = 60.dp, height = 56.dp)
            .clip(shape)
            .background(if (char != null) colors.pinkContainer else colors.surface)
            .border(1.5.dp, borderColor, shape),
        contentAlignment = Alignment.Center,
    ) {
        if (char != null) {
            Text(
                text = char.toString().toPersianDigits(),
                style = MaterialTheme.typography.headlineSmall.copy(
                    fontSize = 24.sp,
                    fontWeight = FontWeight.ExtraBold,
                ),
                color = colors.ink,
            )
        }
    }
}

/** Counts down to the resend link (web timer): "…تا ۰۰:ss" while locked, tappable label at 0. */
@Composable
private fun ResendRow(state: OtpUiState, onResend: () -> Unit, colors: RitmeColors) {
    when {
        state.status is OtpStatus.Resending ->
            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp, color = colors.pink)

        state.canResend ->
            Text(
                text = stringResource(R.string.auth_otp_resend),
                style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                color = colors.pink,
                modifier = Modifier.clickable(onClick = onResend),
            )

        else -> {
            val time = PERSIAN_MINUTES_PREFIX + state.secondsRemaining.toString().padStart(2, '0').toPersianDigits()
            Text(
                text = stringResource(R.string.auth_otp_resend_timer, time),
                style = MaterialTheme.typography.bodySmall.copy(fontSize = 13.sp),
                color = colors.inkMuted,
            )
        }
    }
}

@Composable
private fun ErrorMessage(status: OtpStatus, colors: RitmeColors) {
    if (status is OtpStatus.Error) {
        Spacer(Modifier.height(18.dp))
        Text(
            text = stringResource(status.key.messageRes()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.error,
            textAlign = TextAlign.Center,
        )
    }
}

/** The resend timer never exceeds 59s, so the minutes segment is always "۰۰". */
private const val PERSIAN_MINUTES_PREFIX = "۰۰:"

@StringRes
private fun AuthErrorKey.messageRes(): Int = when (this) {
    AuthErrorKey.InvalidCode -> R.string.auth_error_invalid_code
    AuthErrorKey.TooMany -> R.string.auth_error_too_many
    AuthErrorKey.Network -> R.string.auth_error_network
    AuthErrorKey.Generic -> R.string.auth_error_generic
}
