package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.JalaliDatePicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.PregnancyAgeSource
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** Unselected dating-method dot (web `#D8DEE5`); no matching brand token yet. */
private val UnselectedDot = Color(0xFFD8DEE5) // TODO token

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
        bottomBar = { SubmitBar(state, viewModel::onIntent, colors) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(key = "header") { HeaderBlock(onBack, colors) }
            item(key = "sources") { SourceMethodCard(state, viewModel::onIntent, colors) }
            state.source?.let { source ->
                item(key = "conditional") { ConditionalCard(source, state, viewModel::onIntent, colors) }
            }
            item(key = "history") { HistoryCard(state, viewModel::onIntent, colors) }
            item(key = "conditions") { ConditionsCard(state, viewModel::onIntent, colors) }
            item(key = "blood") { BloodCard(state, viewModel::onIntent, colors) }
            item(key = "tail") { Spacer(Modifier.height(4.dp)) }
        }
    }
}

/** Start-aligned header: back chevron, a large title, then a muted subtitle (web `.titr`/`.sub`). */
@Composable
private fun HeaderBlock(onBack: () -> Unit, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth()) {
        HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.action_back), onBack)
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.preg_ob_title),
            fontSize = 20.sp,
            fontWeight = FontWeight.ExtraBold,
            color = colors.ink,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.preg_ob_subtitle),
            style = MaterialTheme.typography.bodySmall,
            color = colors.inkMuted,
        )
    }
}

/**
 * The web `PgCard`: a white surface with an optional tinted icon badge + title header and an
 * optional muted hint sub-line, then the section body.
 */
@Composable
private fun PgCard(
    title: String,
    iconRes: Int? = null,
    hint: String? = null,
    content: @Composable () -> Unit,
) {
    val colors = LocalRitmeColors.current
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            if (iconRes != null) {
                Box(
                    Modifier
                        .size(30.dp)
                        .clip(CircleShape)
                        .background(colors.pink.copy(alpha = 0.1f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painter = painterResource(iconRes),
                        contentDescription = null,
                        tint = colors.pink,
                        modifier = Modifier.size(16.dp),
                    )
                }
                Spacer(Modifier.width(9.dp))
            }
            Text(
                title,
                fontSize = 15.sp,
                fontWeight = FontWeight.ExtraBold,
                color = colors.ink,
            )
        }
        Spacer(Modifier.height(if (hint != null) 4.dp else 12.dp))
        if (hint != null) {
            Text(hint, style = MaterialTheme.typography.bodySmall, color = colors.inkMuted)
            Spacer(Modifier.height(12.dp))
        }
        content()
    }
}

@Composable
private fun SourceMethodCard(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    PgCard(title = stringResource(R.string.preg_ob_source_label), iconRes = R.drawable.ic_calendar) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            MethodOption(
                title = stringResource(R.string.preg_ob_source_lmp),
                hint = stringResource(R.string.preg_ob_hint_lmp),
                selected = state.source == PregnancyAgeSource.LMP,
                onClick = { onIntent(PregnancyOnboardingIntent.SourceSelected(PregnancyAgeSource.LMP)) },
                colors = colors,
            )
            MethodOption(
                title = stringResource(R.string.preg_ob_source_ultrasound),
                hint = stringResource(R.string.preg_ob_hint_ultrasound),
                selected = state.source == PregnancyAgeSource.ULTRASOUND,
                onClick = { onIntent(PregnancyOnboardingIntent.SourceSelected(PregnancyAgeSource.ULTRASOUND)) },
                colors = colors,
            )
            MethodOption(
                title = stringResource(R.string.preg_ob_source_manual),
                hint = stringResource(R.string.preg_ob_hint_manual),
                selected = state.source == PregnancyAgeSource.MANUAL,
                onClick = { onIntent(PregnancyOnboardingIntent.SourceSelected(PregnancyAgeSource.MANUAL)) },
                colors = colors,
            )
        }
    }
}

/** A single dating-method row: a leading radio dot (checked when selected) + title + hint. */
@Composable
private fun MethodOption(
    title: String,
    hint: String,
    selected: Boolean,
    onClick: () -> Unit,
    colors: RitmeColors,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(
                width = if (selected) 2.dp else 1.dp,
                color = if (selected) colors.pink else colors.outline,
                shape = RoundedCornerShape(16.dp),
            )
            .clickable(onClick = onClick)
            .padding(horizontal = 12.dp, vertical = 11.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Box(
            Modifier
                .padding(top = 2.dp)
                .size(20.dp)
                .clip(CircleShape)
                .background(if (selected) colors.pink else UnselectedDot),
            contentAlignment = Alignment.Center,
        ) {
            if (selected) {
                Icon(
                    painter = painterResource(R.drawable.ic_check),
                    contentDescription = null,
                    tint = colors.onPink,
                    modifier = Modifier.size(13.dp),
                )
            }
        }
        Spacer(Modifier.width(10.dp))
        Column {
            Text(
                title,
                fontSize = 14.sp,
                fontWeight = FontWeight.ExtraBold,
                color = colors.ink,
            )
            Spacer(Modifier.height(2.dp))
            Text(hint, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        }
    }
}

/** The conditional inputs card shown below the chooser once a method is picked. */
@Composable
private fun ConditionalCard(
    source: PregnancyAgeSource,
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    when (source) {
        PregnancyAgeSource.LMP -> PgCard(title = stringResource(R.string.preg_ob_lmp_date)) {
            JalaliDatePicker(
                value = state.lmpDate,
                onValueChange = { onIntent(PregnancyOnboardingIntent.LmpChanged(it)) },
                minYear = todayJalali().year - 1,
                maxYear = todayJalali().year,
            )
        }

        PregnancyAgeSource.ULTRASOUND -> PgCard(title = stringResource(R.string.preg_ob_ultrasound_date)) {
            JalaliDatePicker(
                value = state.ultrasoundDate,
                onValueChange = { onIntent(PregnancyOnboardingIntent.UltrasoundDateChanged(it)) },
                minYear = todayJalali().year - 1,
                maxYear = todayJalali().year,
            )
            Spacer(Modifier.height(12.dp))
            Text(
                stringResource(R.string.preg_ob_ultrasound_age),
                fontSize = 13.sp,
                fontWeight = FontWeight.Bold,
                color = colors.ink,
            )
            Spacer(Modifier.height(8.dp))
            WeeksField(state.ultrasoundWeeks, colors) {
                onIntent(PregnancyOnboardingIntent.UltrasoundWeeksChanged(it))
            }
            Spacer(Modifier.height(10.dp))
            DaysSegmented(state.ultrasoundDays, colors) {
                onIntent(PregnancyOnboardingIntent.UltrasoundDaysChanged(it))
            }
        }

        PregnancyAgeSource.MANUAL -> PgCard(title = stringResource(R.string.preg_ob_manual_age)) {
            WeeksField(state.manualWeeks, colors) {
                onIntent(PregnancyOnboardingIntent.ManualWeeksChanged(it))
            }
            Spacer(Modifier.height(10.dp))
            DaysSegmented(state.manualDays, colors) {
                onIntent(PregnancyOnboardingIntent.ManualDaysChanged(it))
            }
        }
    }
}

/** Numeric weeks entry (web `NumberField`, min 1 max 42) with a bold label above the field. */
@Composable
private fun WeeksField(weeks: Int?, colors: RitmeColors, onWeeks: (Int?) -> Unit) {
    Column(Modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.preg_ob_weeks),
            fontSize = 13.sp,
            fontWeight = FontWeight.Bold,
            color = colors.ink,
        )
        Spacer(Modifier.height(6.dp))
        OutlinedTextField(
            value = weeks?.toString().orEmpty(),
            onValueChange = { text ->
                onWeeks(
                    text.filter(Char::isDigit).take(2).toIntOrNull()
                        ?.coerceAtMost(PregnancyOnboardingUiState.MAX_WEEKS),
                )
            },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            shape = RoundedCornerShape(14.dp),
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
                focusedContainerColor = colors.surface,
                unfocusedContainerColor = colors.surface,
                focusedTextColor = colors.ink,
                unfocusedTextColor = colors.ink,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

/** Extra-days picker (web `Segmented`): a bold «روز» label over a chip row of 0..6. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun DaysSegmented(days: Int?, colors: RitmeColors, onDays: (Int?) -> Unit) {
    Column(Modifier.fillMaxWidth()) {
        Text(
            stringResource(R.string.preg_ob_days),
            fontSize = 13.sp,
            fontWeight = FontWeight.Bold,
            color = colors.ink,
        )
        Spacer(Modifier.height(8.dp))
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            for (day in 0..PregnancyOnboardingUiState.MAX_EXTRA_DAYS) {
                SelectableChip(
                    label = day.toPersianDigits(),
                    selected = days == day,
                    onClick = { onDays(if (days == day) null else day) },
                )
            }
        }
    }
}

@Composable
private fun HistoryCard(
    state: PregnancyOnboardingUiState,
    onIntent: (PregnancyOnboardingIntent) -> Unit,
    colors: RitmeColors,
) {
    PgCard(title = stringResource(R.string.preg_ob_history_title), iconRes = R.drawable.ic_shield) {
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
    PgCard(
        title = stringResource(R.string.preg_ob_conditions_title),
        iconRes = R.drawable.ic_stetho,
        hint = stringResource(R.string.preg_ob_optional),
    ) {
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
    PgCard(
        title = stringResource(R.string.preg_ob_blood_type),
        iconRes = R.drawable.ic_drop,
        hint = stringResource(R.string.preg_ob_optional),
    ) {
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
            fontSize = 13.sp,
            fontWeight = FontWeight.Bold,
            color = colors.ink,
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
            fontWeight = FontWeight.Bold,
            modifier = Modifier.weight(1f),
        )
        Switch(
            checked = checked,
            onCheckedChange = onChange,
            colors = SwitchDefaults.colors(
                checkedTrackColor = colors.pink,
                checkedThumbColor = colors.onPink,
                uncheckedTrackColor = UnselectedDot,
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
                color = colors.pink,
                fontWeight = FontWeight.Bold,
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
            shape = RoundedCornerShape(16.dp),
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
