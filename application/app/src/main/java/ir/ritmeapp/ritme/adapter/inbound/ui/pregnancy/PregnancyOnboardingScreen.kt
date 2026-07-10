package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * Pregnancy-mode setup (web `/pregnancy/onboarding`): pick the dating source (LMP / ultrasound /
 * manual weeks) with its conditional inputs, plus optional history, pre-existing conditions,
 * and blood type — submitted once to `POST /pregnancy/onboarding`.
 */
@Composable
fun PregnancyOnboardingScreen(
    onBack: () -> Unit,
    onCompleted: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: PregnancyOnboardingViewModel = viewModel(factory = container.pregnancyOnboardingViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:pregnancy_onboarding:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.PregnancyOnboarding.route, null, System.currentTimeMillis()),
        )
        viewModel.effects.collect { effect ->
            when (effect) {
                PregnancyOnboardingEffect.Completed -> onCompleted()
            }
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = { ScreenHeader(title = stringResource(R.string.preg_ob_title), onBack = onBack) },
        bottomBar = { SubmitBar(state, viewModel::onIntent, colors) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(key = "subtitle") {
                Text(
                    stringResource(R.string.preg_ob_subtitle),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
            item(key = "sources") { SourceCards(state, viewModel::onIntent, colors) }
            item(key = "history") { HistoryCard(state, viewModel::onIntent, colors) }
            item(key = "conditions") { ConditionsCard(state, viewModel::onIntent, colors) }
            item(key = "blood") { BloodCard(state, viewModel::onIntent, colors) }
            item(key = "tail") { Spacer(Modifier.height(4.dp)) }
        }
    }
}

@Composable
private fun SourceCards(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(
            stringResource(R.string.preg_ob_source_label),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        SourceCard(
            source = PregnancyAgeSource.LMP,
            title = stringResource(R.string.preg_ob_source_lmp),
            hint = stringResource(R.string.preg_ob_hint_lmp),
            state = state,
            onIntent = onIntent,
            colors = colors,
        ) {
            Text(
                stringResource(R.string.preg_ob_lmp_date),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(6.dp))
            JalaliDatePicker(
                value = state.lmpDate,
                onValueChange = { onIntent(PregnancyOnboardingIntent.LmpChanged(it)) },
                minYear = todayJalali().year - 1,
                maxYear = todayJalali().year,
            )
        }
        SourceCard(
            source = PregnancyAgeSource.ULTRASOUND,
            title = stringResource(R.string.preg_ob_source_ultrasound),
            hint = stringResource(R.string.preg_ob_hint_ultrasound),
            state = state,
            onIntent = onIntent,
            colors = colors,
        ) {
            Text(
                stringResource(R.string.preg_ob_ultrasound_date),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(6.dp))
            JalaliDatePicker(
                value = state.ultrasoundDate,
                onValueChange = { onIntent(PregnancyOnboardingIntent.UltrasoundDateChanged(it)) },
                minYear = todayJalali().year - 1,
                maxYear = todayJalali().year,
            )
            Spacer(Modifier.height(10.dp))
            Text(
                stringResource(R.string.preg_ob_ultrasound_age),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
            )
            WeeksDaysPickers(
                weeks = state.ultrasoundWeeks,
                days = state.ultrasoundDays,
                onWeeks = { onIntent(PregnancyOnboardingIntent.UltrasoundWeeksChanged(it)) },
                onDays = { onIntent(PregnancyOnboardingIntent.UltrasoundDaysChanged(it)) },
            )
        }
        SourceCard(
            source = PregnancyAgeSource.MANUAL,
            title = stringResource(R.string.preg_ob_source_manual),
            hint = stringResource(R.string.preg_ob_hint_manual),
            state = state,
            onIntent = onIntent,
            colors = colors,
        ) {
            Text(
                stringResource(R.string.preg_ob_manual_age),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
            )
            WeeksDaysPickers(
                weeks = state.manualWeeks,
                days = state.manualDays,
                onWeeks = { onIntent(PregnancyOnboardingIntent.ManualWeeksChanged(it)) },
                onDays = { onIntent(PregnancyOnboardingIntent.ManualDaysChanged(it)) },
            )
        }
    }
}

@Composable
private fun SourceCard(
    source: PregnancyAgeSource,
    title: String,
    hint: String,
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
    expandedContent: @Composable () -> Unit,
) {
    val selected = state.source == source
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) colors.pink else colors.outline,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable { onIntent(PregnancyOnboardingIntent.SourceSelected(source)) }
            .padding(14.dp),
    ) {
        Text(
            text = title,
            style = MaterialTheme.typography.bodyLarge,
            color = if (selected) colors.pink else colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(2.dp))
        Text(hint, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        if (selected) {
            Spacer(Modifier.height(12.dp))
            expandedContent()
        }
    }
}

/** The weeks (1..42) + extra days (0..6) pair used by the ultrasound/manual sources. */
@Composable
private fun WeeksDaysPickers(weeks: Int, days: Int, onWeeks: (Int) -> Unit, onDays: (Int) -> Unit) {
    val weekUnit = stringResource(R.string.preg_ob_weeks)
    val dayUnit = stringResource(R.string.preg_ob_days)
    Row(Modifier.fillMaxWidth()) {
        WheelPicker(
            count = PregnancyOnboardingUiState.MAX_WEEKS,
            selectedIndex = (weeks - 1).coerceIn(0, PregnancyOnboardingUiState.MAX_WEEKS - 1),
            onSelected = { onWeeks(it + 1) },
            label = { "${(it + 1).toPersianDigits()} $weekUnit" },
            visibleCount = 3,
            modifier = Modifier.weight(1f),
        )
        WheelPicker(
            count = PregnancyOnboardingUiState.MAX_EXTRA_DAYS + 1,
            selectedIndex = days.coerceIn(0, PregnancyOnboardingUiState.MAX_EXTRA_DAYS),
            onSelected = onDays,
            label = { "${it.toPersianDigits()} $dayUnit" },
            visibleCount = 3,
            modifier = Modifier.weight(1f),
        )
    }
}

@Composable
private fun HistoryCard(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard {
        Text(
            stringResource(R.string.preg_ob_history_title),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        ToggleRow(stringResource(R.string.preg_ob_miscarriage), state.hasMiscarriageHistory, colors) {
            onIntent(PregnancyOnboardingIntent.MiscarriageToggled(it))
        }
        ToggleRow(stringResource(R.string.preg_ob_high_risk), state.hasHighRiskHistory, colors) {
            onIntent(PregnancyOnboardingIntent.HighRiskToggled(it))
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ConditionsCard(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.preg_ob_conditions_title),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            Text(
                stringResource(R.string.preg_ob_optional),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
        }
        Spacer(Modifier.height(10.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            PregnancyOnboardingUiState.CONDITION_OPTIONS.forEach { condition ->
                SelectableChip(
                    label = conditionLabel(condition),
                    selected = condition in state.conditions,
                    onClick = { onIntent(PregnancyOnboardingIntent.ConditionToggled(condition)) },
                )
            }
        }
    }
}

@Composable
private fun conditionLabel(condition: String): String = when (condition) {
    "chronic_hypertension" -> stringResource(R.string.preg_ob_cond_hypertension)
    "diabetes" -> stringResource(R.string.preg_ob_cond_diabetes)
    "hypothyroidism" -> stringResource(R.string.preg_ob_cond_hypothyroidism)
    "hyperthyroidism" -> stringResource(R.string.preg_ob_cond_hyperthyroidism)
    else -> stringResource(R.string.preg_ob_cond_none)
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun BloodCard(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard {
        Text(
            stringResource(R.string.preg_ob_blood_type),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            PregnancyOnboardingUiState.BLOOD_TYPES.forEach { type ->
                SelectableChip(
                    label = type,
                    selected = state.bloodType == type,
                    onClick = { onIntent(PregnancyOnboardingIntent.BloodTypeSelected(type)) },
                )
            }
        }
        Spacer(Modifier.height(12.dp))
        Text(
            stringResource(R.string.preg_ob_rh),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            PregnancyOnboardingUiState.RH_FACTORS.forEach { factor ->
                SelectableChip(
                    label = if (factor == "positive") {
                        stringResource(R.string.preg_ob_rh_positive)
                    } else {
                        stringResource(R.string.preg_ob_rh_negative)
                    },
                    selected = state.rhFactor == factor,
                    onClick = { onIntent(PregnancyOnboardingIntent.RhSelected(factor)) },
                )
            }
        }
    }
}

@Composable
private fun ToggleRow(label: String, checked: Boolean, colors: RitmeColors, onChange: (Boolean) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            modifier = Modifier.weight(1f),
        )
        Switch(
            checked = checked,
            onCheckedChange = onChange,
            colors = SwitchDefaults.colors(
                checkedTrackColor = colors.pink,
                checkedThumbColor = colors.onPink,
                uncheckedTrackColor = colors.outline,
                uncheckedThumbColor = colors.surface,
            ),
        )
    }
}

@Composable
private fun SubmitBar(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    Column(Modifier.fillMaxWidth().padding(16.dp)) {
        val error = when (state.error) {
            PregnancyOnboardingError.SELECT_SOURCE -> stringResource(R.string.preg_ob_select_source)
            PregnancyOnboardingError.FILL_REQUIRED -> stringResource(R.string.preg_ob_fill_required)
            PregnancyOnboardingError.SUBMIT_FAILED -> stringResource(R.string.error_generic)
            PregnancyOnboardingError.NONE -> null
        }
        if (error != null) {
            Text(
                text = error,
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
                modifier = Modifier.fillMaxWidth().padding(bottom = 6.dp),
                textAlign = TextAlign.Center,
            )
        }
        Button(
            onClick = { onIntent(PregnancyOnboardingIntent.Submit) },
            enabled = !state.submitting,
            colors = ButtonDefaults.buttonColors(
                containerColor = colors.pink,
                contentColor = colors.onPink,
                disabledContainerColor = colors.outline,
                disabledContentColor = colors.inkMuted,
            ),
            shape = RoundedCornerShape(14.dp),
            modifier = Modifier.fillMaxWidth().height(48.dp),
        ) {
            Text(
                text = if (state.submitting) {
                    stringResource(R.string.preg_ob_submitting)
                } else {
                    stringResource(R.string.preg_ob_submit)
                },
                fontWeight = FontWeight.Bold,
            )
        }
    }
}
