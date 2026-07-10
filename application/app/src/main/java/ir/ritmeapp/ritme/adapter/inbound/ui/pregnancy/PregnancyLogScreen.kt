package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.Segmented
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.log.LogSaveState
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.PregnancyLogTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.PregnancySymptom
import ir.ritmeapp.ritme.domain.model.PregnancySymptomGroup
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.model.SymptomSeverity
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlin.math.roundToInt

/**
 * The pregnancy log (web `/pregnancy/log`): three tabs — daily symptoms (toggle + severity per
 * symptom, warning group highlighted), the weekly check-in (weight, swelling, vitals, mood), and
 * fetal movement (status, count, times). Each tab saves independently and upserts server-side.
 */
@Composable
fun PregnancyLogScreen(
    initialTab: PregnancyLogTab,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: PregnancyLogViewModel = viewModel(factory = container.pregnancyLogViewModelFactory(initialTab))
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:pregnancy_log:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.PregnancyLog.ROUTE, initialTab.name, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = { ScreenHeader(title = stringResource(R.string.preg_log_title), onBack = onBack) },
        bottomBar = { SaveBar(state, viewModel::onIntent, colors) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(key = "subtitle") {
                Text(
                    stringResource(R.string.preg_log_subtitle),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
            item(key = "tabs") {
                Segmented(
                    options = listOf(
                        stringResource(R.string.preg_log_tab_symptoms),
                        stringResource(R.string.preg_log_tab_weekly),
                        stringResource(R.string.preg_log_tab_movement),
                    ),
                    selectedIndex = state.tab.ordinal,
                    onSelected = { index ->
                        viewModel.onIntent(PregnancyLogIntent.SelectTab(PregnancyLogTab.entries[index]))
                    },
                )
            }
            when (state.tab) {
                PregnancyLogTab.SYMPTOMS -> symptomsTab(state, viewModel, colors)
                PregnancyLogTab.WEEKLY -> weeklyTab(state, viewModel, colors)
                PregnancyLogTab.MOVEMENT -> movementTab(state, viewModel, colors)
            }
            item(key = "tail") { Spacer(Modifier.height(4.dp)) }
        }
    }
}

// --- Symptoms tab -------------------------------------------------------------

private fun androidx.compose.foundation.lazy.LazyListScope.symptomsTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
    item(key = "group_common") {
        SymptomGroupCard(
            title = R.string.preg_group_common,
            group = PregnancySymptomGroup.COMMON,
            state = state,
            onIntent = viewModel::onIntent,
            colors = colors,
        )
    }
    item(key = "group_aches") {
        SymptomGroupCard(
            title = R.string.preg_group_aches,
            group = PregnancySymptomGroup.PAIN,
            state = state,
            onIntent = viewModel::onIntent,
            colors = colors,
        )
    }
    item(key = "group_warning") {
        SymptomGroupCard(
            title = R.string.preg_group_warning,
            group = PregnancySymptomGroup.WARNING,
            state = state,
            onIntent = viewModel::onIntent,
            colors = colors,
            warning = true,
        )
    }
    item(key = "sym_notes") {
        NotesCard(
            value = state.symptomNotes,
            onChange = { viewModel.onIntent(PregnancyLogIntent.SymptomNotesChanged(it)) },
            colors = colors,
        )
    }
    if (state.alertsRaised > 0) {
        item(key = "alerts_raised") {
            Text(
                text = stringResource(R.string.preg_alerts_raised, state.alertsRaised.toPersianDigits()),
                style = MaterialTheme.typography.labelMedium,
                color = colors.warning,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        }
    }
}

@Composable
private fun DaySwitcher(state: PregnancyLogUiState, onIntent: (PregnancyLogIntent) -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.log_prev_day), {
                onIntent(PregnancyLogIntent.PreviousDay)
            })
            Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = state.date.formatDayMonth(),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                if (state.isToday) {
                    Text(
                        stringResource(R.string.log_today),
                        style = MaterialTheme.typography.labelSmall,
                        color = colors.pink,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
            if (state.isToday) {
                Spacer(Modifier.size(38.dp))
            } else {
                HeaderIconButton(R.drawable.ic_chevron_left, stringResource(R.string.log_next_day), {
                    onIntent(PregnancyLogIntent.NextDay)
                })
            }
        }
    }
}

@Composable
private fun SymptomGroupCard(
    title: Int,
    group: PregnancySymptomGroup,
    state: PregnancyLogUiState,
    onIntent: (PregnancyLogIntent) -> Unit,
    colors: RitmeColors,
    warning: Boolean = false,
) {
    SurfaceCard {
        Text(
            text = stringResource(title),
            style = MaterialTheme.typography.bodyLarge,
            color = if (warning) colors.error else colors.ink,
            fontWeight = FontWeight.Bold,
        )
        if (warning) {
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.preg_critical_hint),
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
            )
        }
        Spacer(Modifier.height(8.dp))
        PregnancySymptom.byGroup[group].orEmpty().forEach { symptom ->
            SymptomRow(symptom, state.symptoms[symptom], onIntent, colors)
        }
    }
}

@Composable
private fun SymptomRow(
    symptom: PregnancySymptom,
    severity: SymptomSeverity?,
    onIntent: (PregnancyLogIntent) -> Unit,
    colors: RitmeColors,
) {
    Column(Modifier.fillMaxWidth().padding(vertical = 6.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = stringResource(symptom.labelRes()),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                modifier = Modifier.weight(1f),
            )
            BrandSwitch(severity != null, colors) { on ->
                onIntent(PregnancyLogIntent.SymptomToggled(symptom, on))
            }
        }
        if (severity != null) {
            Spacer(Modifier.height(6.dp))
            SeveritySegments(severity) { onIntent(PregnancyLogIntent.SymptomSeverityChanged(symptom, it)) }
        }
    }
}

@Composable
private fun SeveritySegments(selected: SymptomSeverity, onSelected: (SymptomSeverity) -> Unit) {
    Segmented(
        options = listOf(
            stringResource(R.string.severity_mild),
            stringResource(R.string.severity_moderate),
            stringResource(R.string.severity_severe),
        ),
        selectedIndex = selected.ordinal,
        onSelected = { onSelected(SymptomSeverity.entries[it]) },
    )
}

private fun PregnancySymptom.labelRes(): Int = when (this) {
    PregnancySymptom.NAUSEA -> R.string.preg_sym_nausea
    PregnancySymptom.VOMITING -> R.string.preg_sym_vomiting
    PregnancySymptom.FATIGUE -> R.string.preg_sym_fatigue
    PregnancySymptom.HEADACHE -> R.string.preg_sym_headache
    PregnancySymptom.DIZZINESS -> R.string.preg_sym_dizziness
    PregnancySymptom.BREAST_PAIN -> R.string.preg_sym_breast_pain
    PregnancySymptom.LOWER_ABDOMINAL_PAIN -> R.string.preg_sym_lower_abdominal_pain
    PregnancySymptom.CRAMPING -> R.string.preg_sym_cramping
    PregnancySymptom.BACK_PAIN -> R.string.preg_sym_back_pain
    PregnancySymptom.PELVIC_PRESSURE -> R.string.preg_sym_pelvic_pressure
    PregnancySymptom.SPOTTING -> R.string.preg_sym_spotting
    PregnancySymptom.BLEEDING -> R.string.preg_sym_bleeding
    PregnancySymptom.FLUID_LEAKAGE -> R.string.preg_sym_fluid_leakage
    PregnancySymptom.SEVERE_SUDDEN_PAIN -> R.string.preg_sym_severe_sudden_pain
}

// --- Weekly tab -------------------------------------------------------------------

@OptIn(ExperimentalLayoutApi::class)
private fun androidx.compose.foundation.lazy.LazyListScope.weeklyTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    val draft = state.weekly
    val update: (WeeklyDraft) -> Unit = { viewModel.onIntent(PregnancyLogIntent.WeeklyChanged(it)) }

    item(key = "week_header") {
        Text(
            text = stringResource(R.string.preg_weekly_week, state.week.toPersianDigits()),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
        )
    }
    item(key = "weight") {
        SurfaceCard {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.preg_weekly_weight),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                BrandSwitch(draft.weightKg != null, colors) { on ->
                    update(draft.copy(weightKg = if (on) DEFAULT_WEIGHT else null))
                }
            }
            val weight = draft.weightKg
            if (weight != null) {
                WheelPicker(
                    count = WEIGHT_STEPS,
                    selectedIndex = (((weight - MIN_WEIGHT) / WEIGHT_STEP).roundToInt()).coerceIn(0, WEIGHT_STEPS - 1),
                    onSelected = { update(draft.copy(weightKg = MIN_WEIGHT + it * WEIGHT_STEP)) },
                    label = { formatWeight(MIN_WEIGHT + it * WEIGHT_STEP) },
                    visibleCount = 3,
                )
            }
        }
    }
    item(key = "swelling") {
        SurfaceCard {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.preg_weekly_swelling),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                BrandSwitch(draft.hasSwelling, colors) { on ->
                    update(draft.copy(hasSwelling = on, swellingLocations = if (on) draft.swellingLocations else emptyList()))
                }
            }
            if (draft.hasSwelling) {
                Spacer(Modifier.height(8.dp))
                Text(
                    stringResource(R.string.preg_weekly_swelling_where),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                )
                Spacer(Modifier.height(6.dp))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    FetalMovementLog.SWELLING_LOCATIONS.forEach { location ->
                        SelectableChip(
                            label = swellingLabel(location),
                            selected = location in draft.swellingLocations,
                            onClick = {
                                val updated = if (location in draft.swellingLocations) {
                                    draft.swellingLocations - location
                                } else {
                                    draft.swellingLocations + location
                                }
                                update(draft.copy(swellingLocations = updated))
                            },
                        )
                    }
                }
            }
            Spacer(Modifier.height(8.dp))
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.preg_weekly_breath),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    modifier = Modifier.weight(1f),
                )
                BrandSwitch(draft.hasShortnessOfBreath, colors) { update(draft.copy(hasShortnessOfBreath = it)) }
            }
        }
    }
    item(key = "vitals") {
        SurfaceCard {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.preg_weekly_bp_device),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.weight(1f),
                )
                BrandSwitch(draft.hasBpDevice, colors) { update(draft.copy(hasBpDevice = it)) }
            }
            if (draft.hasBpDevice) {
                Spacer(Modifier.height(8.dp))
                NumberFieldRow(stringResource(R.string.preg_weekly_systolic), draft.systolic, colors) {
                    update(draft.copy(systolic = it))
                }
                NumberFieldRow(stringResource(R.string.preg_weekly_diastolic), draft.diastolic, colors) {
                    update(draft.copy(diastolic = it))
                }
                NumberFieldRow(stringResource(R.string.preg_weekly_fasting), draft.fastingSugar?.roundToInt(), colors) {
                    update(draft.copy(fastingSugar = it?.toDouble()))
                }
                NumberFieldRow(stringResource(R.string.preg_weekly_postmeal), draft.postMealSugar?.roundToInt(), colors) {
                    update(draft.copy(postMealSugar = it?.toDouble()))
                }
            }
        }
    }
    item(key = "mood") {
        SurfaceCard {
            Text(
                stringResource(R.string.preg_weekly_mood),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(8.dp))
            Segmented(
                options = listOf(
                    stringResource(R.string.preg_mood_good),
                    stringResource(R.string.preg_mood_moderate),
                    stringResource(R.string.preg_mood_poor),
                ),
                selectedIndex = FetalMovementLog.MOOD_OPTIONS.indexOf(draft.mood).coerceAtLeast(-1),
                onSelected = { update(draft.copy(mood = FetalMovementLog.MOOD_OPTIONS[it])) },
            )
            Spacer(Modifier.height(10.dp))
            MentalRow(
                stringResource(R.string.preg_mental_anxiety),
                draft.anxiety,
                colors,
            ) { update(draft.copy(anxiety = it)) }
            MentalRow(
                stringResource(R.string.preg_mental_mood_swings),
                draft.moodSwings,
                colors,
            ) { update(draft.copy(moodSwings = it)) }
            MentalRow(
                stringResource(R.string.preg_mental_depression),
                draft.depression,
                colors,
            ) { update(draft.copy(depression = it)) }
        }
    }
    item(key = "weekly_notes") {
        NotesCard(
            value = state.weekly.notes,
            onChange = { update(draft.copy(notes = it)) },
            colors = colors,
        )
    }
}

@Composable
private fun MentalRow(
    label: String,
    severity: SymptomSeverity?,
    colors: RitmeColors,
    onChange: (SymptomSeverity?) -> Unit,
) {
    Column(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                label,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                modifier = Modifier.weight(1f),
            )
            BrandSwitch(severity != null, colors) { on -> onChange(if (on) SymptomSeverity.MILD else null) }
        }
        if (severity != null) {
            Spacer(Modifier.height(6.dp))
            SeveritySegments(severity, onChange)
        }
    }
}

@Composable
private fun NumberFieldRow(label: String, value: Int?, colors: RitmeColors, onChange: (Int?) -> Unit) {
    Column(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
        Text(label, style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(4.dp))
        OutlinedTextField(
            value = value?.toString().orEmpty(),
            onValueChange = { text -> onChange(text.filter(Char::isDigit).take(3).toIntOrNull()) },
            singleLine = true,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

// --- Movement tab -----------------------------------------------------------------

private fun androidx.compose.foundation.lazy.LazyListScope.movementTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    val draft = state.movement
    val update: (MovementDraft) -> Unit = { viewModel.onIntent(PregnancyLogIntent.MovementChanged(it)) }

    item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
    if (!state.fetalTrackingActive) {
        item(key = "notyet") {
            SurfaceCard {
                Text(
                    stringResource(R.string.preg_movement_not_yet),
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
    item(key = "status") {
        SurfaceCard {
            Text(
                stringResource(R.string.preg_movement_status),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(8.dp))
            @OptIn(ExperimentalLayoutApi::class)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                FetalMovementLog.STATUS_OPTIONS.forEach { option ->
                    SelectableChip(
                        label = movementStatusLabel(option),
                        selected = draft.status == option,
                        onClick = { update(draft.copy(status = if (draft.status == option) null else option)) },
                    )
                }
            }
        }
    }
    item(key = "count") {
        SurfaceCard {
            Text(
                stringResource(R.string.preg_movement_count),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.preg_movement_tracking_from),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(6.dp))
            OutlinedTextField(
                value = draft.count?.toString().orEmpty(),
                onValueChange = { text ->
                    update(draft.copy(count = text.filter(Char::isDigit).take(3).toIntOrNull()?.coerceIn(0, MAX_MOVEMENTS)))
                },
                singleLine = true,
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = colors.pink,
                    unfocusedBorderColor = colors.outline,
                ),
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }
    item(key = "times") {
        SurfaceCard {
            TimeRow(stringResource(R.string.preg_movement_first), draft.firstTime, colors) {
                update(draft.copy(firstTime = it))
            }
            Spacer(Modifier.height(8.dp))
            TimeRow(stringResource(R.string.preg_movement_last), draft.lastTime, colors) {
                update(draft.copy(lastTime = it))
            }
        }
    }
    item(key = "movement_notes") {
        NotesCard(value = draft.notes, onChange = { update(draft.copy(notes = it)) }, colors = colors)
    }
}

@Composable
private fun movementStatusLabel(option: String): String = when (option) {
    "not_felt_yet" -> stringResource(R.string.preg_move_not_felt_yet)
    "felt" -> stringResource(R.string.preg_move_felt)
    "normal" -> stringResource(R.string.preg_move_normal)
    "reduced" -> stringResource(R.string.preg_move_reduced)
    "increased" -> stringResource(R.string.preg_move_increased)
    else -> stringResource(R.string.preg_move_none)
}

@Composable
private fun swellingLabel(location: String): String = when (location) {
    "feet" -> stringResource(R.string.preg_swelling_feet)
    "hands" -> stringResource(R.string.preg_swelling_hands)
    else -> stringResource(R.string.preg_swelling_face)
}

/** «HH:mm» picker behind a toggle — off means "not recorded". */
@Composable
private fun TimeRow(label: String, value: String?, colors: RitmeColors, onChange: (String?) -> Unit) {
    val hour = value?.substringBefore(':')?.toIntOrNull() ?: DEFAULT_TIME_HOUR
    val minute = value?.substringAfter(':')?.toIntOrNull() ?: 0
    Column(Modifier.fillMaxWidth()) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                label,
                style = MaterialTheme.typography.bodyMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.weight(1f),
            )
            BrandSwitch(value != null, colors) { on ->
                onChange(if (on) "%02d:%02d".format(DEFAULT_TIME_HOUR, 0) else null)
            }
        }
        if (value != null) {
            Row(Modifier.fillMaxWidth()) {
                WheelPicker(
                    count = HOURS_IN_DAY,
                    selectedIndex = hour.coerceIn(0, HOURS_IN_DAY - 1),
                    onSelected = { onChange("%02d:%02d".format(it, minute)) },
                    label = { "%02d".format(it).toPersianDigits() },
                    visibleCount = 3,
                    modifier = Modifier.weight(1f),
                )
                WheelPicker(
                    count = MINUTE_STEPS,
                    selectedIndex = (minute / MINUTE_STEP).coerceIn(0, MINUTE_STEPS - 1),
                    onSelected = { onChange("%02d:%02d".format(hour, it * MINUTE_STEP)) },
                    label = { "%02d".format(it * MINUTE_STEP).toPersianDigits() },
                    visibleCount = 3,
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

// --- Shared bits -------------------------------------------------------------------

@Composable
private fun NotesCard(value: String, onChange: (String) -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Text(
            stringResource(R.string.preg_notes),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            placeholder = { Text(stringResource(R.string.preg_notes_placeholder), color = colors.inkMuted) },
            minLines = 3,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun BrandSwitch(checked: Boolean, colors: RitmeColors, onCheckedChange: (Boolean) -> Unit) {
    Switch(
        checked = checked,
        onCheckedChange = onCheckedChange,
        colors = SwitchDefaults.colors(
            checkedTrackColor = colors.pink,
            checkedThumbColor = colors.onPink,
            uncheckedTrackColor = colors.outline,
            uncheckedThumbColor = colors.surface,
        ),
    )
}

@Composable
private fun SaveBar(state: PregnancyLogUiState, onIntent: (PregnancyLogIntent) -> Unit, colors: RitmeColors) {
    val (saveState, action, enabled) = when (state.tab) {
        PregnancyLogTab.SYMPTOMS -> Triple(
            state.symptomSave,
            PregnancyLogIntent.SaveSymptoms as PregnancyLogIntent,
            state.symptoms.isNotEmpty() || state.symptomNotes.isNotBlank(),
        )

        PregnancyLogTab.WEEKLY -> Triple(
            state.weeklySave,
            PregnancyLogIntent.SaveWeekly as PregnancyLogIntent,
            state.weekly.hasAnything,
        )

        PregnancyLogTab.MOVEMENT -> Triple(
            state.movementSave,
            PregnancyLogIntent.SaveMovement as PregnancyLogIntent,
            state.movement.status != null,
        )
    }
    Column(Modifier.fillMaxWidth().padding(16.dp)) {
        if (saveState == LogSaveState.ERROR) {
            Text(
                stringResource(R.string.log_save_error),
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
                modifier = Modifier.fillMaxWidth().padding(bottom = 6.dp),
                textAlign = TextAlign.Center,
            )
        }
        Button(
            onClick = { onIntent(action) },
            enabled = enabled && saveState != LogSaveState.SAVING,
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
                text = when (saveState) {
                    LogSaveState.SAVING -> stringResource(R.string.log_saving)
                    LogSaveState.SAVED -> stringResource(R.string.edit_saved)
                    else -> stringResource(R.string.action_save)
                },
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

/** «۶۲٫۵» weight label. */
private fun formatWeight(value: Double): String {
    val rounded = (value * 10).roundToInt() / 10.0
    val text = if (rounded % 1.0 == 0.0) rounded.toInt().toString() else rounded.toString()
    return text.toPersianDigits()
}

private const val DEFAULT_WEIGHT = 65.0
private const val MIN_WEIGHT = 30.0
private const val WEIGHT_STEP = 0.5
private const val WEIGHT_STEPS = ((200.0 - MIN_WEIGHT) / WEIGHT_STEP).toInt() + 1
private const val MAX_MOVEMENTS = 100
private const val HOURS_IN_DAY = 24
private const val MINUTE_STEP = 5
private const val MINUTE_STEPS = 60 / MINUTE_STEP
private const val DEFAULT_TIME_HOUR = 9
