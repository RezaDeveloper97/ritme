@file:OptIn(ExperimentalLayoutApi::class)

package ir.ritmeapp.ritme.adapter.inbound.ui.pregnancy

import android.app.TimePickerDialog
import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.PaddingValues
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
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
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.PersianDigitsTransformation
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.log.LogSaveState
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.domain.model.TrackingMode
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

// Web alert/critical accents that have no exact Ritme token yet.
private val CriticalSymptomRed = Color(0xFFC13438) // TODO token
private val WarningAccentRed = Color(0xFFE5484D) // TODO token
private val FlameAccent = Color(0xFFE9662E) // TODO token

/**
 * The pregnancy log (web `/pregnancy/log`): three tabs — daily symptoms (toggle + severity per
 * symptom, warning group highlighted), the weekly check-in (weight, swelling, vitals, mood), and
 * fetal movement (status, count, times). Each tab saves independently and upserts server-side.
 */
@Composable
fun PregnancyLogScreen(
    initialTab: PregnancyLogTab,
    onBack: () -> Unit,
    onNavigate: (Destination) -> Unit,
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
        // Persistent floating tab bar, like the web pregnancy-log page.
        bottomBar = { RitmeBottomBar(active = RitmeTab.LOG, mode = TrackingMode.PREGNANCY, onNavigate = onNavigate) },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding).imePadding(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(key = "header") { LogHeader(onBack, colors) }
            item(key = "tabs") {
                TabSwitcher(
                    selectedIndex = state.tab.ordinal,
                    onSelect = { index ->
                        viewModel.onIntent(PregnancyLogIntent.SelectTab(PregnancyLogTab.entries[index]))
                    },
                    colors = colors,
                )
            }
            when (state.tab) {
                PregnancyLogTab.SYMPTOMS -> symptomsTab(state, viewModel, colors)
                PregnancyLogTab.WEEKLY -> weeklyTab(state, viewModel, colors)
                PregnancyLogTab.MOVEMENT -> movementTab(state, viewModel, colors)
            }
            item(key = "tail") { Spacer(Modifier.height(8.dp)) }
        }
    }
}

// --- Header + tab switcher ---------------------------------------------------------

/** Start-aligned header block (web `.titr` + `.sub`): back chevron on its own row, then title/subtitle. */
@Composable
private fun LogHeader(onBack: () -> Unit, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth().padding(top = 4.dp)) {
        HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.action_back), onBack)
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.preg_log_title),
            fontSize = 20.sp,
            fontWeight = FontWeight.ExtraBold,
            color = colors.ink,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.preg_log_subtitle),
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
        )
    }
}

/** Web `.seg` tab bar: muted track, the active tab a solid-pink pill with white text. */
@Composable
private fun TabSwitcher(selectedIndex: Int, onSelect: (Int) -> Unit, colors: RitmeColors) {
    val labels = listOf(
        stringResource(R.string.preg_log_tab_symptoms),
        stringResource(R.string.preg_log_tab_weekly),
        stringResource(R.string.preg_log_tab_movement),
    )
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .background(colors.background)
            .padding(4.dp),
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        labels.forEachIndexed { index, label ->
            val active = index == selectedIndex
            Box(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(12.dp))
                    .background(if (active) colors.pink else Color.Transparent)
                    .clickable { onSelect(index) }
                    .padding(vertical = 9.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = label,
                    fontSize = 12.5.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (active) colors.onPink else colors.inkMuted,
                )
            }
        }
    }
}

// --- Symptoms tab -------------------------------------------------------------

private fun LazyListScope.symptomsTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
    item(key = "group_common") {
        SymptomGroupCard(R.string.preg_group_common, PregnancySymptomGroup.COMMON, R.drawable.ic_sparkle, colors.pink, state, viewModel::onIntent, colors)
    }
    item(key = "group_aches") {
        SymptomGroupCard(R.string.preg_group_aches, PregnancySymptomGroup.PAIN, R.drawable.ic_sparkle, colors.pink, state, viewModel::onIntent, colors)
    }
    item(key = "group_warning") {
        SymptomGroupCard(
            R.string.preg_group_warning,
            PregnancySymptomGroup.WARNING,
            R.drawable.ic_shield,
            WarningAccentRed,
            state,
            viewModel::onIntent,
            colors,
            hintRes = R.string.preg_critical_hint,
        )
    }
    item(key = "sym_notes") {
        NotesCard(state.symptomNotes, { viewModel.onIntent(PregnancyLogIntent.SymptomNotesChanged(it)) }, colors)
    }
    if (state.alertsRaised > 0) {
        item(key = "alerts_raised") {
            AlertsRaisedCard(state.alertsRaised, R.drawable.ic_flame, FlameAccent, colors)
        }
    }
    item(key = "sym_save") {
        SaveSection(
            state.symptomSave,
            PregnancyLogIntent.SaveSymptoms,
            enabled = state.symptoms.isNotEmpty() || state.symptomNotes.isNotBlank(),
            onIntent = viewModel::onIntent,
            colors = colors,
        )
    }
}

@Composable
private fun DaySwitcher(state: PregnancyLogUiState, onIntent: (PregnancyLogIntent) -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(16.dp))
            .padding(horizontal = 12.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        HeaderIconButton(
            R.drawable.ic_chevron_right,
            stringResource(R.string.log_prev_day),
            { onIntent(PregnancyLogIntent.PreviousDay) },
        )
        Row(
            modifier = Modifier.weight(1f),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_calendar),
                contentDescription = null,
                tint = colors.ink,
                modifier = Modifier.size(15.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                text = state.date.formatDayMonth(),
                fontSize = 14.5.sp,
                fontWeight = FontWeight.ExtraBold,
                color = colors.ink,
            )
            if (state.isToday) {
                Spacer(Modifier.width(8.dp))
                Box(
                    modifier = Modifier
                        .clip(RoundedCornerShape(20.dp))
                        .background(colors.pinkContainer)
                        .padding(horizontal = 9.dp, vertical = 2.dp),
                ) {
                    Text(
                        stringResource(R.string.log_today),
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        color = colors.pink,
                    )
                }
            }
        }
        HeaderIconButton(
            R.drawable.ic_chevron_left,
            stringResource(R.string.log_next_day),
            { if (!state.isToday) onIntent(PregnancyLogIntent.NextDay) },
            modifier = Modifier.alpha(if (state.isToday) 0.3f else 1f),
        )
    }
}

@Composable
private fun SymptomGroupCard(
    @androidx.annotation.StringRes title: Int,
    group: PregnancySymptomGroup,
    @DrawableRes icon: Int,
    tint: Color,
    state: PregnancyLogUiState,
    onIntent: (PregnancyLogIntent) -> Unit,
    colors: RitmeColors,
    @androidx.annotation.StringRes hintRes: Int? = null,
) {
    val critical = group == PregnancySymptomGroup.WARNING
    PgCard(icon, stringResource(title), colors, tint = tint, hint = hintRes?.let { stringResource(it) }) {
        PregnancySymptom.byGroup[group].orEmpty().forEach { symptom ->
            SymptomRow(symptom, state.symptoms[symptom], critical, onIntent, colors)
        }
    }
}

@Composable
private fun SymptomRow(
    symptom: PregnancySymptom,
    severity: SymptomSeverity?,
    critical: Boolean,
    onIntent: (PregnancyLogIntent) -> Unit,
    colors: RitmeColors,
) {
    Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = stringResource(symptom.labelRes()),
                fontSize = 13.5.sp,
                fontWeight = FontWeight.Bold,
                color = if (critical) CriticalSymptomRed else colors.ink,
                modifier = Modifier.weight(1f),
            )
            BrandSwitch(severity != null, colors) { on ->
                onIntent(PregnancyLogIntent.SymptomToggled(symptom, on))
            }
        }
        if (severity != null) {
            Spacer(Modifier.height(8.dp))
            SeverityChips(severity) { onIntent(PregnancyLogIntent.SymptomSeverityChanged(symptom, it)) }
        }
    }
    HairlineDivider(colors)
}

// --- Weekly tab -------------------------------------------------------------------

private fun LazyListScope.weeklyTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    val week = state.week
    if (week == null) {
        item(key = "weekly_loading") { LoadingCard(colors) }
        return
    }
    val draft = state.weekly
    val update: (WeeklyDraft) -> Unit = { viewModel.onIntent(PregnancyLogIntent.WeeklyChanged(it)) }

    item(key = "week_header") {
        Text(
            text = stringResource(R.string.preg_weekly_week, week.toPersianDigits()),
            fontSize = 14.sp,
            fontWeight = FontWeight.ExtraBold,
            color = colors.ink,
            modifier = Modifier.padding(horizontal = 2.dp),
        )
    }
    item(key = "weight") {
        PgCard(R.drawable.ic_chart, stringResource(R.string.preg_weekly_weight), colors) {
            WeightField(draft.weightKg, colors) { update(draft.copy(weightKg = it)) }
        }
    }
    item(key = "swelling") {
        PgCard(R.drawable.ic_drop, stringResource(R.string.preg_weekly_swelling), colors) {
            FieldRow(stringResource(R.string.preg_weekly_swelling), colors) {
                BrandSwitch(draft.hasSwelling, colors) { on ->
                    update(draft.copy(hasSwelling = on, swellingLocations = if (on) draft.swellingLocations else emptyList()))
                }
            }
            if (draft.hasSwelling) {
                Text(
                    stringResource(R.string.preg_weekly_swelling_where),
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = colors.inkMuted,
                )
                Spacer(Modifier.height(8.dp))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
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
            FieldRow(stringResource(R.string.preg_weekly_breath), colors) {
                BrandSwitch(draft.hasShortnessOfBreath, colors) { update(draft.copy(hasShortnessOfBreath = it)) }
            }
        }
    }
    item(key = "vitals") {
        PgCard(R.drawable.ic_stetho, stringResource(R.string.preg_weekly_bp_device), colors) {
            FieldRow(stringResource(R.string.preg_weekly_bp_device), colors) {
                BrandSwitch(draft.hasBpDevice, colors) { update(draft.copy(hasBpDevice = it)) }
            }
            if (draft.hasBpDevice) {
                Spacer(Modifier.height(6.dp))
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    NumberFieldRow(stringResource(R.string.preg_weekly_systolic), draft.systolic, colors, Modifier.weight(1f)) {
                        update(draft.copy(systolic = it))
                    }
                    NumberFieldRow(stringResource(R.string.preg_weekly_diastolic), draft.diastolic, colors, Modifier.weight(1f)) {
                        update(draft.copy(diastolic = it))
                    }
                }
            }
            Spacer(Modifier.height(12.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                NumberFieldRow(stringResource(R.string.preg_weekly_fasting), draft.fastingSugar?.roundToInt(), colors, Modifier.weight(1f)) {
                    update(draft.copy(fastingSugar = it?.toDouble()))
                }
                NumberFieldRow(stringResource(R.string.preg_weekly_postmeal), draft.postMealSugar?.roundToInt(), colors, Modifier.weight(1f)) {
                    update(draft.copy(postMealSugar = it?.toDouble()))
                }
            }
        }
    }
    item(key = "mood") {
        PgCard(R.drawable.ic_smile, stringResource(R.string.preg_weekly_mood), colors) {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                FetalMovementLog.MOOD_OPTIONS.forEach { option ->
                    SelectableChip(
                        label = moodLabel(option),
                        selected = draft.mood == option,
                        onClick = { update(draft.copy(mood = if (draft.mood == option) null else option)) },
                    )
                }
            }
            Spacer(Modifier.height(12.dp))
            MentalRow(stringResource(R.string.preg_mental_anxiety), draft.anxiety, colors) { update(draft.copy(anxiety = it)) }
            MentalRow(stringResource(R.string.preg_mental_mood_swings), draft.moodSwings, colors) { update(draft.copy(moodSwings = it)) }
            MentalRow(stringResource(R.string.preg_mental_depression), draft.depression, colors) { update(draft.copy(depression = it)) }
        }
    }
    item(key = "weekly_notes") {
        NotesCard(state.weekly.notes, { update(draft.copy(notes = it)) }, colors)
    }
    item(key = "weekly_save") {
        SaveSection(
            state.weeklySave,
            PregnancyLogIntent.SaveWeekly,
            enabled = state.weekly.hasAnything,
            onIntent = viewModel::onIntent,
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
    HairlineDivider(colors)
    Column(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
        FieldRow(label, colors) {
            BrandSwitch(severity != null, colors) { on -> onChange(if (on) SymptomSeverity.MILD else null) }
        }
        if (severity != null) {
            Spacer(Modifier.height(8.dp))
            SeverityChips(severity) { onChange(it) }
        }
    }
}

// --- Movement tab -----------------------------------------------------------------

private fun LazyListScope.movementTab(
    state: PregnancyLogUiState,
    viewModel: PregnancyLogViewModel,
    colors: RitmeColors,
) {
    val draft = state.movement
    val update: (MovementDraft) -> Unit = { viewModel.onIntent(PregnancyLogIntent.MovementChanged(it)) }

    item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
    if (!state.fetalTrackingActive) {
        item(key = "notyet") { NotYetCard(colors) }
    }
    item(key = "status") {
        PgCard(R.drawable.ic_heart, stringResource(R.string.preg_movement_status), colors) {
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
        PgCard(
            R.drawable.ic_chart,
            stringResource(R.string.preg_movement_count),
            colors,
            hint = stringResource(R.string.preg_movement_tracking_from),
        ) {
            OutlinedTextField(
                value = draft.count?.toString().orEmpty(),
                onValueChange = { text ->
                    update(draft.copy(count = text.filter(Char::isDigit).take(3).toIntOrNull()?.coerceIn(0, MAX_MOVEMENTS)))
                },
                singleLine = true,
                placeholder = { Text(stringResource(R.string.preg_count_placeholder), color = colors.inkMuted) },
                visualTransformation = PersianDigitsTransformation,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                colors = fieldColors(colors),
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(12.dp))
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                TimeField(stringResource(R.string.preg_movement_first), draft.firstTime, colors, Modifier.weight(1f)) {
                    update(draft.copy(firstTime = it))
                }
                TimeField(stringResource(R.string.preg_movement_last), draft.lastTime, colors, Modifier.weight(1f)) {
                    update(draft.copy(lastTime = it))
                }
            }
        }
    }
    item(key = "movement_notes") {
        NotesCard(draft.notes, { update(draft.copy(notes = it)) }, colors)
    }
    item(key = "movement_save") {
        SaveSection(
            state.movementSave,
            PregnancyLogIntent.SaveMovement,
            enabled = draft.status != null,
            onIntent = viewModel::onIntent,
            colors = colors,
        )
    }
}

@Composable
private fun NotYetCard(colors: RitmeColors) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.Top) {
            Icon(
                painter = painterResource(R.drawable.ic_info),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.padding(top = 2.dp).size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                stringResource(R.string.preg_movement_not_yet),
                fontSize = 12.5.sp,
                color = colors.inkMuted,
                lineHeight = 22.sp,
            )
        }
    }
}

// --- Shared bits -------------------------------------------------------------------

/** Web `PgCard`: a surface card with a leading tinted round icon badge before the title. */
@Composable
private fun PgCard(
    @DrawableRes icon: Int,
    title: String,
    colors: RitmeColors,
    tint: Color = colors.pink,
    hint: String? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    SurfaceCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(30.dp)
                    .clip(CircleShape)
                    .background(tint.copy(alpha = 0.1f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(icon),
                    contentDescription = null,
                    tint = tint,
                    modifier = Modifier.size(16.dp),
                )
            }
            Spacer(Modifier.width(9.dp))
            Text(title, fontSize = 15.sp, fontWeight = FontWeight.ExtraBold, color = colors.ink)
        }
        if (hint != null) {
            Spacer(Modifier.height(4.dp))
            Text(hint, fontSize = 12.5.sp, color = colors.inkMuted, lineHeight = 20.sp)
        }
        Spacer(Modifier.height(12.dp))
        content()
    }
}

/** Web `.card` loading placeholder shown on the weekly tab until the gestational week resolves. */
@Composable
private fun LoadingCard(colors: RitmeColors) {
    SurfaceCard {
        Text(
            stringResource(R.string.loading),
            style = MaterialTheme.typography.bodyMedium,
            color = colors.inkMuted,
            modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp),
            textAlign = TextAlign.Center,
        )
    }
}

/** Web `FieldRow`: a label with a trailing control on a space-between row. */
@Composable
private fun FieldRow(label: String, colors: RitmeColors, trailing: @Composable () -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(vertical = 9.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            label,
            fontSize = 13.5.sp,
            fontWeight = FontWeight.Bold,
            color = colors.ink,
            modifier = Modifier.weight(1f),
        )
        trailing()
    }
}

/** Severity (mild/moderate/severe) as wrapping pill chips — web `Segmented`. */
@Composable
private fun SeverityChips(selected: SymptomSeverity, onSelect: (SymptomSeverity) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        SymptomSeverity.entries.forEach { severity ->
            SelectableChip(
                label = severityLabel(severity),
                selected = selected == severity,
                onClick = { if (selected != severity) onSelect(severity) },
            )
        }
    }
}

/** Always-visible weight entry (web NumberField): a decimal field rendered with Persian digits. */
@Composable
private fun WeightField(value: Double?, colors: RitmeColors, onChange: (Double?) -> Unit) {
    var text by remember { mutableStateOf(value.toWeightText()) }
    LaunchedEffect(value) {
        // Re-sync only when the value changes externally (prefill), not from our own typing.
        if (value != text.toDoubleOrNull()) text = value.toWeightText()
    }
    OutlinedTextField(
        value = text,
        onValueChange = { raw ->
            val sanitized = sanitizeDecimal(raw)
            text = sanitized
            onChange(sanitized.toDoubleOrNull())
        },
        singleLine = true,
        placeholder = { Text(stringResource(R.string.preg_weight_placeholder), color = colors.inkMuted) },
        visualTransformation = PersianDigitsTransformation,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
        colors = fieldColors(colors),
        modifier = Modifier.fillMaxWidth(),
    )
}

/** A labelled integer field (web NumberField); an empty label renders the field alone. */
@Composable
private fun NumberFieldRow(
    label: String,
    value: Int?,
    colors: RitmeColors,
    modifier: Modifier = Modifier,
    onChange: (Int?) -> Unit,
) {
    Column(modifier.fillMaxWidth()) {
        if (label.isNotEmpty()) {
            Text(label, fontSize = 13.sp, fontWeight = FontWeight.Bold, color = colors.ink)
            Spacer(Modifier.height(6.dp))
        }
        OutlinedTextField(
            value = value?.toString().orEmpty(),
            onValueChange = { text -> onChange(text.filter(Char::isDigit).take(3).toIntOrNull()) },
            singleLine = true,
            visualTransformation = PersianDigitsTransformation,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            colors = fieldColors(colors),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

/** A tappable time field (web `<input type="time">`) that opens a native time picker. */
@Composable
private fun TimeField(
    label: String,
    value: String?,
    colors: RitmeColors,
    modifier: Modifier = Modifier,
    onChange: (String?) -> Unit,
) {
    val context = LocalContext.current
    Column(modifier) {
        Text(label, fontSize = 13.sp, fontWeight = FontWeight.Bold, color = colors.ink)
        Spacer(Modifier.height(6.dp))
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp)
                .clip(RoundedCornerShape(14.dp))
                .background(colors.surface)
                .border(1.5.dp, colors.outline, RoundedCornerShape(14.dp))
                .clickable {
                    val hour = value?.substringBefore(':')?.toIntOrNull() ?: DEFAULT_TIME_HOUR
                    val minute = value?.substringAfter(':')?.toIntOrNull() ?: 0
                    TimePickerDialog(
                        context,
                        { _, pickedHour, pickedMinute -> onChange("%02d:%02d".format(pickedHour, pickedMinute)) },
                        hour,
                        minute,
                        true,
                    ).show()
                }
                .padding(horizontal = 16.dp),
            contentAlignment = Alignment.CenterStart,
        ) {
            Text(
                text = value?.toPersianDigits() ?: stringResource(R.string.preg_time_placeholder),
                fontSize = 15.sp,
                color = if (value != null) colors.ink else colors.inkMuted,
            )
        }
    }
}

@Composable
private fun NotesCard(value: String, onChange: (String) -> Unit, colors: RitmeColors) {
    PgCard(R.drawable.ic_pencil, stringResource(R.string.preg_notes), colors) {
        OutlinedTextField(
            value = value,
            onValueChange = onChange,
            placeholder = { Text(stringResource(R.string.preg_notes_placeholder), color = colors.inkMuted) },
            minLines = 3,
            colors = fieldColors(colors),
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

/** Web alerts-raised `.card`: a start accent bar + tinted icon + count. */
@Composable
private fun AlertsRaisedCard(count: Int, @DrawableRes icon: Int, accent: Color, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(IntrinsicSize.Min)
            .clip(RoundedCornerShape(16.dp))
            .background(colors.surface)
            .border(1.dp, colors.outline, RoundedCornerShape(16.dp)),
    ) {
        Box(Modifier.width(3.dp).fillMaxHeight().background(accent))
        Row(
            modifier = Modifier.padding(horizontal = 13.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = accent,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(8.dp))
            Text(
                text = stringResource(R.string.preg_alerts_raised, count.toPersianDigits()),
                fontSize = 12.5.sp,
                fontWeight = FontWeight.Bold,
                color = colors.ink,
            )
        }
    }
}

/** Inline save button (web `.btn-primary`): pink error line above, check icon on the saved state. */
@Composable
private fun SaveSection(
    saveState: LogSaveState,
    action: PregnancyLogIntent,
    enabled: Boolean,
    onIntent: (PregnancyLogIntent) -> Unit,
    colors: RitmeColors,
) {
    Column(Modifier.fillMaxWidth()) {
        if (saveState == LogSaveState.ERROR) {
            Text(
                stringResource(R.string.log_save_error),
                fontSize = 12.5.sp,
                fontWeight = FontWeight.Bold,
                color = colors.pink,
                modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp),
                textAlign = TextAlign.Center,
            )
        }
        RitmePrimaryButton(
            text = when (saveState) {
                LogSaveState.SAVING -> stringResource(R.string.log_saving)
                LogSaveState.SAVED -> stringResource(R.string.edit_saved)
                else -> stringResource(R.string.action_save)
            },
            onClick = { onIntent(action) },
            enabled = enabled && saveState != LogSaveState.SAVING,
            leadingIcon = if (saveState == LogSaveState.SAVED) {
                {
                    Icon(
                        painter = painterResource(R.drawable.ic_check),
                        contentDescription = null,
                        tint = colors.onPink,
                        modifier = Modifier.size(18.dp),
                    )
                }
            } else {
                null
            },
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
private fun HairlineDivider(colors: RitmeColors) {
    Box(Modifier.fillMaxWidth().height(1.dp).background(colors.outline))
}

@Composable
private fun fieldColors(colors: RitmeColors) = OutlinedTextFieldDefaults.colors(
    focusedBorderColor = colors.pink,
    unfocusedBorderColor = colors.outline,
    cursorColor = colors.pink,
)

@Composable
private fun severityLabel(severity: SymptomSeverity): String = when (severity) {
    SymptomSeverity.MILD -> stringResource(R.string.severity_mild)
    SymptomSeverity.MODERATE -> stringResource(R.string.severity_moderate)
    SymptomSeverity.SEVERE -> stringResource(R.string.severity_severe)
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
private fun moodLabel(option: String): String = when (option) {
    "good" -> stringResource(R.string.preg_mood_good)
    "moderate" -> stringResource(R.string.preg_mood_moderate)
    else -> stringResource(R.string.preg_mood_poor)
}

@Composable
private fun swellingLabel(location: String): String = when (location) {
    "feet" -> stringResource(R.string.preg_swelling_feet)
    "hands" -> stringResource(R.string.preg_swelling_hands)
    else -> stringResource(R.string.preg_swelling_face)
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

private fun Double?.toWeightText(): String = this?.let {
    if (it % 1.0 == 0.0) it.toInt().toString() else it.toString()
} ?: ""

/** Keep only digits and a single decimal point, capped to a sensible length. */
private fun sanitizeDecimal(raw: String): String {
    val filtered = raw.filter { it.isDigit() || it == '.' }
    val firstDot = filtered.indexOf('.')
    val single = if (firstDot < 0) {
        filtered
    } else {
        filtered.substring(0, firstDot + 1) + filtered.substring(firstDot + 1).replace(".", "")
    }
    return single.take(6)
}

private const val MAX_MOVEMENTS = 100
private const val DEFAULT_TIME_HOUR = 9
