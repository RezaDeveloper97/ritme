package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
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
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlin.math.roundToInt

private const val BIRTHDAY_MIN_AGE_SPAN = 90
private const val DEFAULT_WEIGHT_KG = 60
private const val DEFAULT_HEIGHT_CM = 165
private const val DEFAULT_PERIOD_DAYS = 5
private const val DEFAULT_CYCLE_DAYS = 28
private const val RING_DURATION_MS = 2300
private const val EDGE_SWIPE_ZONE_DP = 24
private const val EDGE_SWIPE_COMMIT_DP = 56
private const val RING_SIZE_DP = 200
private const val RING_STROKE_DP = 13

/**
 * The signup wizard shown to a brand-new account after OTP: one screen driving every step through a
 * single [OnboardingViewModel] (MVI). Each step shows a start-aligned title with a NavBack chevron
 * and an "N / M" counter (no progress bar — matching the web). The intention step auto-advances on
 * tap; the last step hands off to an animated progress-ring [OnboardingStep.SETTING_UP] screen that
 * saves once and navigates home once both the ring and the save settle.
 */
@Composable
fun OnboardingScreen(
    onDone: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: OnboardingViewModel = viewModel(factory = container.onboardingViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:onboarding:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Onboarding.route, null, System.currentTimeMillis()),
        )
    }

    LaunchedEffect(Unit) {
        viewModel.effects.collect { effect ->
            when (effect) {
                OnboardingEffect.NavigateHome -> onDone()
            }
        }
    }

    BackHandler(enabled = state.canGoBack) { viewModel.onIntent(OnboardingIntent.Back) }

    OnboardingContent(state = state, onIntent = viewModel::onIntent, modifier = modifier)
}

@Composable
private fun OnboardingContent(
    state: OnboardingUiState,
    onIntent: (OnboardingIntent) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    val today = remember { todayJalali() }

    Scaffold(modifier = modifier.fillMaxSize(), containerColor = colors.background) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            if (state.isSettingUp) {
                SettingUpStep(onRingDone = { onIntent(OnboardingIntent.SettingUpRingDone) }, colors = colors)
            } else {
                Wizard(state, onIntent, today, colors)
                if (state.canGoBack) EdgeSwipeBack { onIntent(OnboardingIntent.Back) }
            }
        }
    }
}

@Composable
private fun Wizard(
    state: OnboardingUiState,
    onIntent: (OnboardingIntent) -> Unit,
    today: JalaliDate,
    colors: RitmeColors,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = 22.dp)
            .imePadding(),
    ) {
        Spacer(Modifier.height(8.dp))
        OnboardingHeader(state.stepNumber, state.totalSteps, { onIntent(OnboardingIntent.Back) }, colors)
        Spacer(Modifier.height(6.dp))

        AnimatedContent(
            targetState = state.step,
            transitionSpec = {
                val forward = targetState.ordinal >= initialState.ordinal
                val dir = if (forward) 1 else -1
                (slideInHorizontally { w -> dir * w } + fadeIn()) togetherWith
                    (slideOutHorizontally { w -> -dir * w } + fadeOut())
            },
            modifier = Modifier.fillMaxWidth().weight(1f),
            label = "onboarding-step",
        ) { _ ->
            StepContent(state, onIntent, today, colors)
        }

        val errorText = when (state.validation) {
            OnboardingValidation.SELECT_SOURCE -> stringResource(R.string.ob_basis_select_source)
            OnboardingValidation.FILL_REQUIRED -> stringResource(R.string.ob_basis_fill_required)
            null -> state.errorMessage
        }
        if (errorText != null) {
            Text(
                text = errorText,
                style = MaterialTheme.typography.labelMedium,
                color = colors.error,
                modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                textAlign = TextAlign.Center,
            )
        }

        // The intention step routes forward on tap (web has no Continue there).
        if (state.step != OnboardingStep.INTENTION) {
            RitmePrimaryButton(
                text = stringResource(if (state.isLastStep) R.string.ob_finish else R.string.onboarding_continue),
                onClick = { onIntent(OnboardingIntent.Next) },
                enabled = state.canProceed,
            )
        }
        Spacer(Modifier.height(16.dp))
    }
}

@Composable
private fun StepContent(
    state: OnboardingUiState,
    onIntent: (OnboardingIntent) -> Unit,
    today: JalaliDate,
    colors: RitmeColors,
) {
    val answers = state.answers
    when (state.step) {
        OnboardingStep.NAME ->
            NameStep(answers.name.orEmpty(), { onIntent(OnboardingIntent.NameChanged(it)) }, colors)

        OnboardingStep.BIRTHDAY ->
            BirthdayStep(
                date = answers.birthday ?: today,
                onDate = { onIntent(OnboardingIntent.BirthdayChanged(it)) },
                minYear = today.year - BIRTHDAY_MIN_AGE_SPAN,
                maxYear = today.year,
                colors = colors,
            )

        OnboardingStep.WEIGHT ->
            WeightStep(
                kg = answers.weightKg ?: DEFAULT_WEIGHT_KG,
                unit = state.weightUnit,
                onKg = { onIntent(OnboardingIntent.WeightChanged(it)) },
                onUnit = { onIntent(OnboardingIntent.WeightUnitChanged(it)) },
                colors = colors,
            )

        OnboardingStep.HEIGHT ->
            HeightStep(
                cm = answers.heightCm ?: DEFAULT_HEIGHT_CM,
                unit = state.heightUnit,
                onCm = { onIntent(OnboardingIntent.HeightChanged(it)) },
                onUnit = { onIntent(OnboardingIntent.HeightUnitChanged(it)) },
                colors = colors,
            )

        OnboardingStep.INTENTION ->
            IntentionStep(
                selected = answers.intention,
                onSelect = {
                    onIntent(OnboardingIntent.IntentionSelected(it))
                    onIntent(OnboardingIntent.Next)
                },
                colors = colors,
            )

        OnboardingStep.PREGNANCY_BASIS ->
            PregnancyBasisStep(state.pregnancyBasis, today, onIntent, colors)

        OnboardingStep.PERIOD ->
            PeriodStep(answers.periodDuration ?: DEFAULT_PERIOD_DAYS, { onIntent(OnboardingIntent.PeriodChanged(it)) }, colors)

        OnboardingStep.CYCLE ->
            CycleStep(answers.cycleDuration ?: DEFAULT_CYCLE_DAYS, { onIntent(OnboardingIntent.CycleChanged(it)) }, colors)

        OnboardingStep.LAST_PERIOD ->
            LastPeriodStep(answers.lastPeriod, { onIntent(OnboardingIntent.LastPeriodChanged(it)) }, colors)

        OnboardingStep.CONDITIONS ->
            ConditionsStep(answers.conditions, { onIntent(OnboardingIntent.ConditionToggled(it)) }, colors)

        OnboardingStep.SETTING_UP -> Unit
    }
}

/** Web `.hdr`: a mirrored back chevron on the start edge, a dimmed "N / M" counter on the end edge. */
@Composable
private fun OnboardingHeader(stepNumber: Int, totalSteps: Int, onBack: () -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier.fillMaxWidth().height(44.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.action_back), onBack, tint = colors.ink)
        Spacer(Modifier.weight(1f))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = stepNumber.toPersianDigits(),
                style = MaterialTheme.typography.labelLarge,
                color = colors.inkMuted,
            )
            Text(
                text = " / ${totalSteps.toPersianDigits()}",
                style = MaterialTheme.typography.labelLarge,
                color = colors.inkMuted.copy(alpha = 0.5f),
            )
        }
    }
}

/**
 * A thin physical-left-edge zone that pops a step on a rightward drag past the commit threshold —
 * the Telegram-style swipe-back (CLAUDE.md §5b), narrowed to the edge so it never fights the
 * centered rulers/wheels. The system Back button does the same pop.
 */
@Composable
private fun BoxScope.EdgeSwipeBack(onBack: () -> Unit) {
    val commitPx = with(LocalDensity.current) { EDGE_SWIPE_COMMIT_DP.dp.toPx() }
    CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
        Box(
            Modifier
                .align(Alignment.CenterStart)
                .fillMaxHeight()
                .width(EDGE_SWIPE_ZONE_DP.dp)
                .pointerInput(Unit) {
                    var total = 0f
                    detectHorizontalDragGestures(
                        onDragStart = { total = 0f },
                        onDragEnd = { if (total > commitPx) onBack() },
                        onHorizontalDrag = { change, amount -> total += amount; change.consume() },
                    )
                },
        )
    }
}

/**
 * The post-submit reassurance screen (web `SettingUpPage`): a centered title/subtitle, a custom-drawn
 * pink progress ring counting 0→100٪ (Compose [Animatable] + [Canvas], §5), a medical disclaimer, and
 * a wait line. Reports [onRingDone] when the ring finishes; the ViewModel navigates once the save has
 * also settled.
 */
@Composable
private fun SettingUpStep(onRingDone: () -> Unit, colors: RitmeColors) {
    val progress = remember { Animatable(0f) }
    LaunchedEffect(Unit) {
        progress.animateTo(1f, animationSpec = tween(RING_DURATION_MS, easing = LinearEasing))
        onRingDone()
    }
    val pct = (progress.value * 100).roundToInt()

    Column(
        modifier = Modifier.fillMaxSize().padding(horizontal = 22.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.height(80.dp))
        Text(
            text = stringResource(R.string.ob_setting_up_title),
            style = MaterialTheme.typography.headlineSmall.copy(fontSize = 19.sp, lineHeight = 30.sp),
            color = colors.ink,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(12.dp))
        Text(
            text = stringResource(R.string.ob_common_subtitle),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(40.dp))
        Box(Modifier.size(RING_SIZE_DP.dp), contentAlignment = Alignment.Center) {
            Canvas(Modifier.size(RING_SIZE_DP.dp)) {
                val stroke = RING_STROKE_DP.dp.toPx()
                val diameter = size.minDimension - stroke
                val topLeft = Offset(stroke / 2f, stroke / 2f)
                val arcSize = Size(diameter, diameter)
                drawArc(colors.outline, 0f, 360f, false, topLeft, arcSize, style = Stroke(stroke))
                drawArc(
                    color = colors.pink,
                    startAngle = -90f,
                    sweepAngle = progress.value * 360f,
                    useCenter = false,
                    topLeft = topLeft,
                    size = arcSize,
                    style = Stroke(stroke, cap = StrokeCap.Round),
                )
            }
            Text(
                text = "${pct.toPersianDigits()}٪",
                style = MaterialTheme.typography.displaySmall.copy(fontSize = 40.sp, fontWeight = FontWeight.Bold),
                color = colors.steel,
            )
        }
        Spacer(Modifier.height(40.dp))
        Text(
            text = stringResource(R.string.ob_setting_up_disclaimer),
            style = MaterialTheme.typography.bodySmall,
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.weight(1f))
        Text(
            text = stringResource(R.string.ob_setting_up_wait),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(20.dp))
    }
}
