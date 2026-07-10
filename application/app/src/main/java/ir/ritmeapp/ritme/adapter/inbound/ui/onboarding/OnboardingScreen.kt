package ir.ritmeapp.ritme.adapter.inbound.ui.onboarding

import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

private const val BIRTHDAY_MIN_AGE_SPAN = 90
private const val WEIGHT_MIN = 30
private const val WEIGHT_MAX = 150
private const val HEIGHT_MIN = 120
private const val HEIGHT_MAX = 220
private const val PERIOD_MIN = 1
private const val PERIOD_MAX = 10
private const val CYCLE_MIN = 15
private const val CYCLE_MAX = 60

/**
 * The signup wizard shown to a brand-new account after OTP: one screen driving all steps through a
 * single [OnboardingViewModel] (MVI). "Back" (system or button) rewinds a step until the first,
 * then it is a no-op — onboarding is the post-auth root, so there is no going back to OTP. On the
 * last step Continue submits the profile; success navigates home via a one-shot effect.
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
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(horizontal = 24.dp)
                .imePadding(),
        ) {
            Spacer(Modifier.height(16.dp))
            ProgressHeader(state.stepNumber, state.totalSteps, colors)
            Spacer(Modifier.height(28.dp))

            Box(
                Modifier
                    .fillMaxWidth()
                    .weight(1f),
                contentAlignment = Alignment.TopCenter,
            ) {
                StepContent(state, onIntent, today, colors)
            }

            state.errorMessage?.let { message ->
                Text(
                    text = message,
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.error,
                    modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                    textAlign = TextAlign.Center,
                )
            }
            ContinueButton(state, onIntent, colors)
            Spacer(Modifier.height(20.dp))
        }
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
            NumberStep(
                R.string.onboarding_weight_title, null, WEIGHT_MIN, WEIGHT_MAX,
                answers.weightKg ?: WEIGHT_MIN, { onIntent(OnboardingIntent.WeightChanged(it)) },
                R.string.onboarding_weight_unit, colors,
            )

        OnboardingStep.HEIGHT ->
            NumberStep(
                R.string.onboarding_height_title, null, HEIGHT_MIN, HEIGHT_MAX,
                answers.heightCm ?: HEIGHT_MIN, { onIntent(OnboardingIntent.HeightChanged(it)) },
                R.string.onboarding_height_unit, colors,
            )

        OnboardingStep.INTENTION ->
            IntentionStep(answers.intention, { onIntent(OnboardingIntent.IntentionSelected(it)) }, colors)

        OnboardingStep.PERIOD ->
            NumberStep(
                R.string.onboarding_period_title, null, PERIOD_MIN, PERIOD_MAX,
                answers.periodDuration ?: PERIOD_MIN, { onIntent(OnboardingIntent.PeriodChanged(it)) },
                R.string.onboarding_days_unit, colors,
            )

        OnboardingStep.CYCLE ->
            NumberStep(
                R.string.onboarding_cycle_title, R.string.onboarding_cycle_subtitle, CYCLE_MIN, CYCLE_MAX,
                answers.cycleDuration ?: CYCLE_MIN, { onIntent(OnboardingIntent.CycleChanged(it)) },
                R.string.onboarding_days_unit, colors,
            )

        OnboardingStep.LAST_PERIOD ->
            LastPeriodStep(
                date = answers.lastPeriod ?: today,
                onDate = { onIntent(OnboardingIntent.LastPeriodChanged(it)) },
                minYear = today.year - 1,
                maxYear = today.year,
                colors = colors,
            )

        OnboardingStep.CONDITIONS ->
            ConditionsStep(answers.conditions, { onIntent(OnboardingIntent.ConditionToggled(it)) }, colors)
    }
}

@Composable
private fun ProgressHeader(stepNumber: Int, totalSteps: Int, colors: RitmeColors) {
    val fraction by animateFloatAsState(
        targetValue = stepNumber.toFloat() / totalSteps.toFloat(),
        label = "onboarding-progress",
    )
    Column {
        Text(
            text = stringResource(R.string.onboarding_step_counter, stepNumber.toPersianDigits(), totalSteps.toPersianDigits()),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
        )
        Spacer(Modifier.height(8.dp))
        Box(
            Modifier
                .fillMaxWidth()
                .height(6.dp)
                .clip(RoundedCornerShape(3.dp))
                .background(colors.outline),
        ) {
            Box(
                Modifier
                    .fillMaxHeight()
                    .fillMaxWidth(fraction)
                    .clip(RoundedCornerShape(3.dp))
                    .background(colors.pink),
            )
        }
    }
}

@Composable
private fun ContinueButton(
    state: OnboardingUiState,
    onIntent: (OnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    Button(
        onClick = { onIntent(OnboardingIntent.Next) },
        enabled = state.canProceed && !state.submitting,
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
        if (state.submitting) {
            CircularProgressIndicator(Modifier.size(22.dp), strokeWidth = 2.dp, color = colors.onPink)
        } else {
            Text(
                text = stringResource(if (state.isLastStep) R.string.onboarding_finish else R.string.onboarding_continue),
                style = MaterialTheme.typography.labelLarge,
            )
        }
    }
}
