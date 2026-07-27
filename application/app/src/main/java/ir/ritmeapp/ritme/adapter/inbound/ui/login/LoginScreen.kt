package ir.ritmeapp.ritme.adapter.inbound.ui.login

import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.interaction.MutableInteractionSource
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
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.PersianDigitsTransformation
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The root login/signup screen: mobile entry. Mirrors the web signup form (frontend
 * `screens/auth-signup`): a top-aligned white form with a sparkled title, a bordered phone field
 * carrying a trailing user icon, a terms-&-conditions gate, and a bottom-pinned primary CTA.
 * Owns the [LoginViewModel], records the last-safe-screen + breadcrumb on entry (§7.2), and once
 * the backend has SMSed a code hands the mobile number to [onValidated]. Being the start
 * destination it is intentionally not swipe-back-wrapped (nothing to pop to).
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
        onTermsChanged = viewModel::onTermsChanged,
        onSubmit = viewModel::onSubmit,
        modifier = modifier,
    )
}

@Composable
private fun LoginContent(
    state: LoginUiState,
    snackbarHostState: SnackbarHostState,
    onPhoneChanged: (String) -> Unit,
    onTermsChanged: (Boolean) -> Unit,
    onSubmit: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.surface,
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .imePadding(),
        ) {
            Column(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 22.dp),
            ) {
                Spacer(Modifier.height(56.dp))
                TitleRow(colors)
                Spacer(Modifier.height(12.dp))
                Text(
                    text = stringResource(R.string.auth_signup_subtitle),
                    style = MaterialTheme.typography.bodyMedium.copy(fontSize = 14.sp),
                    color = colors.inkMuted,
                    textAlign = TextAlign.Start,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(32.dp))
                PhoneField(state, onPhoneChanged, onSubmit, colors)
                Spacer(Modifier.height(26.dp))
                TermsRow(state.termsAccepted, onTermsChanged, colors)
                ErrorMessage(state.status, colors)
            }
            RitmePrimaryButton(
                text = stringResource(
                    if (state.isSubmitting) R.string.auth_signup_sending else R.string.auth_signup_submit,
                ),
                onClick = onSubmit,
                enabled = state.canSubmit,
                modifier = Modifier.padding(start = 16.dp, end = 16.dp, top = 14.dp, bottom = 8.dp),
            )
        }
    }
}

/** Title + pink sparkle, grouped at the reading start (right in RTL) like the web `.titr` row. */
@Composable
private fun TitleRow(colors: RitmeColors) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.Start,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = stringResource(R.string.auth_signup_title),
            style = MaterialTheme.typography.titleMedium.copy(fontSize = 16.sp, fontWeight = FontWeight.Bold),
            color = colors.ink,
        )
        Spacer(Modifier.size(8.dp))
        Icon(
            painter = painterResource(R.drawable.ic_sparkle),
            contentDescription = null,
            tint = colors.pink,
            modifier = Modifier.size(22.dp),
        )
    }
}

/**
 * Bordered phone field matching the web `.field`: a static label above a 52dp rounded box with
 * an LTR numeric input (Persian digits) and a trailing user icon. The border turns pink on focus.
 */
@Composable
private fun PhoneField(
    state: LoginUiState,
    onPhoneChanged: (String) -> Unit,
    onSubmit: () -> Unit,
    colors: RitmeColors,
) {
    val interactionSource = remember { MutableInteractionSource() }
    val focused by interactionSource.collectIsFocusedAsState()
    val shape = RoundedCornerShape(14.dp)
    Column(Modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.auth_phone_label),
            style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp),
            color = colors.inkMuted,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Start,
        )
        Spacer(Modifier.height(8.dp))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp)
                .clip(shape)
                .background(colors.surface)
                .border(1.5.dp, if (focused) colors.pinkLight else colors.outline, shape)
                .padding(horizontal = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(Modifier.weight(1f)) {
                CompositionLocalProviderLtr {
                    BasicTextField(
                        value = state.phoneInput,
                        onValueChange = onPhoneChanged,
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        interactionSource = interactionSource,
                        textStyle = TextStyle(
                            color = colors.ink,
                            fontSize = 15.sp,
                            textAlign = TextAlign.Start,
                        ),
                        cursorBrush = SolidColor(colors.pink),
                        visualTransformation = PersianDigitsTransformation,
                        keyboardOptions = KeyboardOptions(
                            keyboardType = KeyboardType.Phone,
                            imeAction = ImeAction.Done,
                        ),
                        keyboardActions = KeyboardActions(onDone = { if (state.canSubmit) onSubmit() }),
                        decorationBox = { inner ->
                            if (state.phoneInput.isEmpty()) {
                                Text(
                                    text = stringResource(R.string.auth_phone_placeholder),
                                    style = TextStyle(color = colors.placeholder, fontSize = 15.sp),
                                )
                            }
                            inner()
                        },
                    )
                }
            }
            Icon(
                painter = painterResource(R.drawable.ic_user),
                contentDescription = null,
                tint = colors.placeholder,
                modifier = Modifier.size(18.dp),
            )
        }
    }
}

/** Tappable terms sentence + pink checkbox (web `.cbx.pink`) that gates the CTA. */
@Composable
private fun TermsRow(
    accepted: Boolean,
    onTermsChanged: (Boolean) -> Unit,
    colors: RitmeColors,
) {
    val termsText = buildAnnotatedString {
        append(stringResource(R.string.auth_terms_prefix))
        withStyle(SpanStyle(color = colors.pink, fontWeight = FontWeight.Bold)) {
            append(stringResource(R.string.auth_terms_link))
        }
        append(stringResource(R.string.auth_terms_suffix))
    }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onTermsChanged(!accepted) },
        verticalAlignment = Alignment.Top,
    ) {
        Text(
            text = termsText,
            style = MaterialTheme.typography.bodySmall.copy(fontSize = 13.sp),
            color = colors.inkMuted,
            textAlign = TextAlign.Start,
            modifier = Modifier.weight(1f),
        )
        Spacer(Modifier.size(10.dp))
        PinkCheckbox(accepted, colors)
    }
}

/** 20dp pink checkbox: light-pink border when off, brand fill + white check when on. */
@Composable
private fun PinkCheckbox(checked: Boolean, colors: RitmeColors) {
    val shape = RoundedCornerShape(4.dp)
    // Web `.cbx.pink` off-border is #FF92B7 — a one-off not present in the token set.
    val offBorder = Color(0xFFFF92B7) // TODO token
    Box(
        modifier = Modifier
            .size(20.dp)
            .clip(shape)
            .background(if (checked) colors.pink else colors.surface)
            .border(1.5.dp, if (checked) colors.pink else offBorder, shape),
        contentAlignment = Alignment.Center,
    ) {
        if (checked) {
            Icon(
                painter = painterResource(R.drawable.ic_check),
                contentDescription = null,
                tint = colors.onPink,
                modifier = Modifier.size(13.dp),
            )
        }
    }
}

@Composable
private fun ErrorMessage(status: LoginStatus, colors: RitmeColors) {
    if (status is LoginStatus.Error) {
        Spacer(Modifier.height(16.dp))
        Text(
            text = stringResource(status.key.messageRes()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.error,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Start,
        )
    }
}

/** Forces LTR for the numeric input so Persian phone digits read left-to-right (web `dir="ltr"`). */
@Composable
private fun CompositionLocalProviderLtr(content: @Composable () -> Unit) {
    androidx.compose.runtime.CompositionLocalProvider(
        LocalLayoutDirection provides LayoutDirection.Ltr,
        content = content,
    )
}

@StringRes
private fun LoginErrorKey.messageRes(): Int = when (this) {
    LoginErrorKey.InvalidPhone -> R.string.login_error_invalid_phone
    LoginErrorKey.TooMany -> R.string.auth_error_too_many
    LoginErrorKey.Network -> R.string.auth_error_network
    LoginErrorKey.Server -> R.string.auth_error_generic
    LoginErrorKey.Unexpected -> R.string.auth_error_generic
}
