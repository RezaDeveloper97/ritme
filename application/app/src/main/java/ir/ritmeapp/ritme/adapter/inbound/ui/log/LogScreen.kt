package ir.ritmeapp.ritme.adapter.inbound.ui.log

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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatDayMonth
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.HealthLogCategory
import ir.ritmeapp.ritme.domain.model.HealthLogControl
import ir.ritmeapp.ritme.domain.model.HealthLogField
import ir.ritmeapp.ritme.domain.model.HealthLogValue
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlin.math.roundToInt

/**
 * The daily health-log screen (web `/log`): a day switcher, eleven category cards that open a
 * bottom sheet of data-driven controls, and a sticky save button. The form renders itself from
 * the [HealthLogField] catalog, so a new backend field appears here without new UI code.
 */
@Composable
fun LogScreen(
    initialDateIso: String?,
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: LogViewModel = viewModel(factory = container.logViewModelFactory(initialDateIso))
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:log:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Log.ROUTE, initialDateIso, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = {
            Column {
                SaveBar(state, onSave = { viewModel.onIntent(LogIntent.Save) }, colors)
                RitmeBottomBar(active = RitmeTab.LOG, mode = TrackingMode.CYCLE, onNavigate = onNavigate)
            }
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item(key = "header") {
                Column(Modifier.fillMaxWidth(), horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(
                        stringResource(R.string.log_title),
                        style = MaterialTheme.typography.titleLarge,
                        color = colors.ink,
                        fontWeight = FontWeight.Bold,
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        stringResource(R.string.log_subtitle),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.inkMuted,
                        textAlign = TextAlign.Center,
                    )
                }
            }
            item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
            item(key = "daytasks") { DayTasksSection(state, viewModel::onIntent, colors) }
            items(HealthLogCategory.entries, key = { it.name }) { category ->
                CategoryCard(
                    category = category,
                    count = state.countIn(category),
                    onClick = { viewModel.onIntent(LogIntent.OpenCategory(category)) },
                    colors = colors,
                )
            }
            item(key = "tail") { Spacer(Modifier.height(4.dp)) }
        }
    }

    val openCategory = state.openCategory
    if (openCategory != null) {
        CategorySheet(
            category = openCategory,
            state = state,
            onIntent = viewModel::onIntent,
            colors = colors,
        )
    }
}

@Composable
private fun DaySwitcher(state: LogUiState, onIntent: (LogIntent) -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            // RTL: the start chevron moves one day back in time.
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.log_prev_day), {
                onIntent(LogIntent.PreviousDay)
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
                    onIntent(LogIntent.NextDay)
                })
            }
        }
    }
}

/** The three item kinds the day planner can add: a plain to-do plus the two health reminders. */
private val TASK_TYPES = listOf(ReminderType.CUSTOM, ReminderType.DOCTOR, ReminderType.MEDICATION)

/**
 * Day-scoped planner: the doctor/medication reminders and to-dos set for the day being edited,
 * with inline add, done-toggle, and delete. "Done" maps to an inactive reminder (§ no completed
 * flag in the API). Backed by the same `/reminders` data the Home dashboard reads.
 */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun DayTasksSection(state: LogUiState, onIntent: (LogIntent) -> Unit, colors: RitmeColors) {
    val tasks = state.dayTasks
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(
                stringResource(R.string.daytasks_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            if (tasks.isNotEmpty()) {
                Text(
                    stringResource(R.string.daytasks_progress, state.doneTaskCount.toPersianDigits(), tasks.size.toPersianDigits()),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.success,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.daytasks_subtitle), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        Spacer(Modifier.height(12.dp))

        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            TASK_TYPES.forEach { type ->
                SelectableChip(
                    label = stringResource(type.taskLabelRes()),
                    selected = state.newTaskType == type,
                    onClick = { onIntent(LogIntent.NewTaskTypeChanged(type)) },
                )
            }
        }
        Spacer(Modifier.height(10.dp))
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            OutlinedTextField(
                value = state.newTaskTitle,
                onValueChange = { onIntent(LogIntent.NewTaskTitleChanged(it)) },
                placeholder = { Text(stringResource(R.string.daytasks_placeholder), color = colors.inkMuted) },
                singleLine = true,
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = colors.pink,
                    unfocusedBorderColor = colors.outline,
                ),
                modifier = Modifier.weight(1f),
            )
            Spacer(Modifier.width(8.dp))
            Button(
                onClick = { onIntent(LogIntent.AddTask) },
                enabled = state.canAddTask,
                colors = ButtonDefaults.buttonColors(
                    containerColor = colors.pink,
                    contentColor = colors.onPink,
                    disabledContainerColor = colors.outline,
                    disabledContentColor = colors.inkMuted,
                ),
                shape = RoundedCornerShape(12.dp),
                modifier = Modifier.height(52.dp),
            ) {
                Text(
                    if (state.addingTask) stringResource(R.string.daytasks_adding) else stringResource(R.string.daytasks_add),
                    fontWeight = FontWeight.Bold,
                )
            }
        }
        if (state.taskError) {
            Spacer(Modifier.height(8.dp))
            Text(stringResource(R.string.daytasks_error), style = MaterialTheme.typography.labelSmall, color = colors.error)
        }
        Spacer(Modifier.height(12.dp))
        if (tasks.isEmpty()) {
            Text(
                stringResource(R.string.daytasks_empty),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        } else {
            tasks.forEachIndexed { i, task ->
                if (i > 0) Spacer(Modifier.height(2.dp))
                TaskRow(task, onIntent, colors)
            }
        }
    }
}

@Composable
private fun TaskRow(task: Reminder, onIntent: (LogIntent) -> Unit, colors: RitmeColors) {
    val done = !task.isActive
    Row(Modifier.fillMaxWidth().padding(vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
        Box(
            Modifier.size(34.dp).clip(CircleShape).background(colors.pinkContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painter = painterResource(task.type.taskIconRes()), contentDescription = null, tint = colors.pink, modifier = Modifier.size(16.dp))
        }
        Spacer(Modifier.width(10.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = task.title,
                style = MaterialTheme.typography.bodyMedium,
                color = if (done) colors.inkMuted else colors.ink,
                fontWeight = FontWeight.Bold,
                textDecoration = if (done) TextDecoration.LineThrough else null,
            )
            val subtitle = task.subtitle
            if (!subtitle.isNullOrBlank()) {
                Text(subtitle, style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
            }
        }
        Box(
            Modifier.size(24.dp).clip(CircleShape)
                .background(if (done) colors.pink else colors.background)
                .border(1.dp, if (done) colors.pink else colors.outline, CircleShape)
                .clickable { onIntent(LogIntent.ToggleTaskDone(task)) },
            contentAlignment = Alignment.Center,
        ) {
            if (done) Text("✓", style = MaterialTheme.typography.labelSmall, color = colors.onPink)
        }
        Spacer(Modifier.width(6.dp))
        Icon(
            painter = painterResource(R.drawable.ic_trash),
            contentDescription = stringResource(R.string.action_delete),
            tint = colors.error,
            modifier = Modifier
                .size(30.dp)
                .clip(CircleShape)
                .clickable { onIntent(LogIntent.DeleteTask(task.id)) }
                .padding(6.dp),
        )
    }
}

@androidx.annotation.StringRes
private fun ReminderType.taskLabelRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.string.reminder_type_doctor
    ReminderType.MEDICATION -> R.string.reminder_type_medication
    ReminderType.APPOINTMENT -> R.string.reminder_type_appointment
    ReminderType.CUSTOM -> R.string.daytasks_type_custom
}

@androidx.annotation.DrawableRes
private fun ReminderType.taskIconRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.drawable.ic_stetho
    ReminderType.MEDICATION -> R.drawable.ic_pill
    ReminderType.APPOINTMENT -> R.drawable.ic_calendar
    ReminderType.CUSTOM -> R.drawable.ic_bell
}

@Composable
private fun CategoryCard(
    category: HealthLogCategory,
    count: Int,
    onClick: () -> Unit,
    colors: RitmeColors,
) {
    SurfaceCard(Modifier.clickable(onClick = onClick)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(40.dp).clip(CircleShape).background(colors.pinkContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(category.iconRes()),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(20.dp),
                )
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    stringResource(category.labelRes()),
                    style = MaterialTheme.typography.bodyLarge,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = if (count > 0) {
                        stringResource(R.string.log_selected_count, count.toPersianDigits())
                    } else {
                        stringResource(category.hintRes())
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = if (count > 0) colors.pink else colors.inkMuted,
                )
            }
            Icon(
                // RTL: forward affordance points toward the text end.
                painter = painterResource(R.drawable.ic_chevron_left),
                contentDescription = null,
                tint = colors.inkMuted,
                modifier = Modifier.size(18.dp),
            )
        }
    }
}

@Composable
private fun SaveBar(state: LogUiState, onSave: () -> Unit, colors: RitmeColors) {
    Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
        if (state.saveState == LogSaveState.ERROR) {
            Text(
                stringResource(R.string.log_save_error),
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(6.dp))
        }
        Button(
            onClick = onSave,
            enabled = state.canSave,
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
                text = when (state.saveState) {
                    LogSaveState.SAVING -> stringResource(R.string.log_saving)
                    LogSaveState.SAVED -> stringResource(R.string.log_saved)
                    else -> stringResource(R.string.log_save)
                },
                fontWeight = FontWeight.Bold,
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CategorySheet(
    category: HealthLogCategory,
    state: LogUiState,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(
        onDismissRequest = { onIntent(LogIntent.CloseSheet) },
        sheetState = sheetState,
        containerColor = colors.surface,
    ) {
        Column(Modifier.fillMaxWidth().padding(horizontal = 20.dp)) {
            Text(
                stringResource(category.labelRes()),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Text(
                stringResource(R.string.log_sheet_hint),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
            Spacer(Modifier.height(12.dp))
            LazyColumn(
                modifier = Modifier.fillMaxWidth().weight(1f, fill = false),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                items(HealthLogField.byCategory[category].orEmpty(), key = { it.name }) { field ->
                    FieldRow(field, state.values[field], onIntent, colors)
                }
                item(key = "done") {
                    Button(
                        onClick = { onIntent(LogIntent.CloseSheet) },
                        colors = ButtonDefaults.buttonColors(containerColor = colors.pink, contentColor = colors.onPink),
                        shape = RoundedCornerShape(14.dp),
                        modifier = Modifier.fillMaxWidth().height(46.dp),
                    ) {
                        Text(stringResource(R.string.action_done), fontWeight = FontWeight.Bold)
                    }
                }
                item(key = "space") { Spacer(Modifier.height(12.dp)) }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun FieldRow(
    field: HealthLogField,
    value: HealthLogValue?,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    when (val control = field.control) {
        is HealthLogControl.Choice -> ChoiceField(field, control.options, value, single = true, onIntent, colors)

        is HealthLogControl.MultiChoice ->
            ChoiceField(field, control.options, value, single = false, onIntent, colors)

        HealthLogControl.Degree ->
            ChoiceField(field, DEGREE_OPTIONS, value, single = true, onIntent, colors)

        HealthLogControl.Toggle -> ToggleField(field, value, onIntent, colors)

        is HealthLogControl.Measure -> MeasureField(field, control, value, onIntent, colors)

        HealthLogControl.Note -> NoteField(field, value, onIntent, colors)
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChoiceField(
    field: HealthLogField,
    options: List<String>,
    value: HealthLogValue?,
    single: Boolean,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val selectedSingle = (value as? HealthLogValue.Choice)?.option
    val selectedMulti = (value as? HealthLogValue.MultiChoice)?.options ?: emptyList()
    Column {
        FieldLabel(field, colors)
        Spacer(Modifier.height(8.dp))
        FlowRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            options.forEach { option ->
                val isSelected = if (single) option == selectedSingle else option in selectedMulti
                OptionChip(field, option, isSelected, colors) {
                    val next: HealthLogValue? = if (single) {
                        if (isSelected) null else HealthLogValue.Choice(option)
                    } else {
                        val updated = if (isSelected) selectedMulti - option else selectedMulti + option
                        if (updated.isEmpty()) null else HealthLogValue.MultiChoice(updated)
                    }
                    onIntent(LogIntent.SetValue(field, next))
                }
            }
        }
    }
}

@Composable
private fun OptionChip(
    field: HealthLogField,
    option: String,
    selected: Boolean,
    colors: RitmeColors,
    onClick: () -> Unit,
) {
    val labelRes = optionLabelRes(field, option)
    SelectableChip(
        label = labelRes?.let { stringResource(it) } ?: option,
        selected = selected,
        onClick = onClick,
    )
}

@Composable
private fun ToggleField(
    field: HealthLogField,
    value: HealthLogValue?,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val enabled = (value as? HealthLogValue.Toggle)?.enabled == true
    Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
        FieldLabel(field, colors, Modifier.weight(1f))
        RitmeSwitch(enabled, colors) { on ->
            onIntent(LogIntent.SetValue(field, if (on) HealthLogValue.Toggle(true) else null))
        }
    }
}

@Composable
private fun MeasureField(
    field: HealthLogField,
    control: HealthLogControl.Measure,
    value: HealthLogValue?,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val current = (value as? HealthLogValue.Number)?.value
    val steps = ((control.max - control.min) / control.step).roundToInt() + 1
    val defaultIndex = steps / 2
    Column {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            FieldLabel(field, colors, Modifier.weight(1f))
            RitmeSwitch(current != null, colors) { on ->
                val defaultValue = control.min + defaultIndex * control.step
                onIntent(LogIntent.SetValue(field, if (on) HealthLogValue.Number(defaultValue) else null))
            }
        }
        if (current != null) {
            Spacer(Modifier.height(6.dp))
            WheelPicker(
                count = steps,
                selectedIndex = (((current - control.min) / control.step).roundToInt()).coerceIn(0, steps - 1),
                onSelected = { index ->
                    onIntent(LogIntent.SetValue(field, HealthLogValue.Number(control.min + index * control.step)))
                },
                label = { index -> formatMeasure(control.min + index * control.step) },
                visibleCount = 3,
            )
            Text(
                text = stringResource(
                    if (field == HealthLogField.WEIGHT) R.string.log_unit_kg else R.string.log_unit_celsius,
                ),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
            )
        }
    }
}

@Composable
private fun NoteField(
    field: HealthLogField,
    value: HealthLogValue?,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val text = (value as? HealthLogValue.Text)?.text.orEmpty()
    Column {
        FieldLabel(field, colors)
        Spacer(Modifier.height(8.dp))
        OutlinedTextField(
            value = text,
            onValueChange = { updated ->
                onIntent(
                    LogIntent.SetValue(
                        field,
                        updated.takeIf { it.isNotBlank() }?.let { HealthLogValue.Text(it) },
                    ),
                )
            },
            placeholder = { Text(stringResource(R.string.log_notes_placeholder), color = colors.inkMuted) },
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
private fun FieldLabel(field: HealthLogField, colors: RitmeColors, modifier: Modifier = Modifier) {
    Text(
        text = stringResource(field.labelRes()),
        style = MaterialTheme.typography.bodyMedium,
        color = colors.ink,
        fontWeight = FontWeight.Bold,
        modifier = modifier,
    )
}

/** The brand-colored yes/no switch shared by toggle and measure rows. */
@Composable
private fun RitmeSwitch(checked: Boolean, colors: RitmeColors, onCheckedChange: (Boolean) -> Unit) {
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

/** «۶۲٫۵» — trims a whole number's decimal and localizes digits. */
private fun formatMeasure(value: Double): String {
    val rounded = (value * 10).roundToInt() / 10.0
    val text = if (rounded % 1.0 == 0.0) rounded.toInt().toString() else rounded.toString()
    return text.toPersianDigits()
}

private val DEGREE_OPTIONS = listOf("low", "medium", "high")
