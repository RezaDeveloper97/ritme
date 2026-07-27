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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalConfiguration
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
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeBottomSheet
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
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
 * bottom sheet of data-driven controls, and a day planner at the bottom. Every edit auto-saves
 * (no Save button); the header shows a small save-status pill. The form renders itself from the
 * [HealthLogField] catalog, so a new backend field appears here without new UI code.
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

    // Keep the last opened category so the sheet can finish its exit animation after it closes.
    var sheetCategory by remember { mutableStateOf<HealthLogCategory?>(null) }
    LaunchedEffect(state.openCategory) {
        state.openCategory?.let { sheetCategory = it }
    }

    Box(modifier = modifier.fillMaxSize()) {
        Scaffold(
            modifier = Modifier.fillMaxSize(),
            containerColor = colors.background,
            bottomBar = {
                RitmeBottomBar(active = RitmeTab.LOG, mode = TrackingMode.CYCLE, onNavigate = onNavigate)
            },
        ) { padding ->
            LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                item(key = "header") {
                    Row(
                        Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.Top,
                        horizontalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        Column(Modifier.weight(1f)) {
                            Text(
                                stringResource(R.string.log_title),
                                style = MaterialTheme.typography.titleLarge,
                                color = colors.ink,
                                fontWeight = FontWeight.Bold,
                            )
                            Spacer(Modifier.height(6.dp))
                            Text(
                                stringResource(R.string.log_subtitle),
                                style = MaterialTheme.typography.labelMedium,
                                color = colors.inkMuted,
                            )
                        }
                        SaveStatusPill(state.saveState, colors)
                    }
                }
                item(key = "day") { DaySwitcher(state, viewModel::onIntent, colors) }
                items(HealthLogCategory.entries, key = { it.name }) { category ->
                    CategoryCard(
                        category = category,
                        count = state.countIn(category),
                        onClick = { viewModel.onIntent(LogIntent.OpenCategory(category)) },
                        colors = colors,
                    )
                }
                item(key = "daytasks") { DayTasksSection(state, viewModel::onIntent, colors) }
                item(key = "tail") { Spacer(Modifier.height(4.dp)) }
            }
        }

        // Web `.sheet`: overlays the whole screen (nav included) as the last child of the root Box.
        RitmeBottomSheet(
            visible = state.openCategory != null,
            onDismiss = { viewModel.onIntent(LogIntent.CloseSheet) },
        ) {
            sheetCategory?.let { category ->
                CategorySheetContent(category, state, viewModel::onIntent, colors)
            }
        }
    }
}

/** The header's inline save-status chip (web: 'در حال ذخیره…' / check + 'ذخیره شد' / brand error). */
@Composable
private fun SaveStatusPill(saveState: LogSaveState, colors: RitmeColors) {
    if (saveState == LogSaveState.IDLE) return
    val isError = saveState == LogSaveState.ERROR
    Row(
        Modifier
            .padding(top = 2.dp)
            .clip(RoundedCornerShape(20.dp))
            .background(if (isError) colors.pinkContainer else colors.surface)
            .padding(horizontal = 10.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        if (saveState == LogSaveState.SAVED) {
            Icon(
                painter = painterResource(R.drawable.ic_check),
                contentDescription = null,
                tint = colors.inkMuted,
                modifier = Modifier.size(13.dp),
            )
        }
        Text(
            text = when (saveState) {
                LogSaveState.SAVING -> stringResource(R.string.log_saving)
                LogSaveState.ERROR -> stringResource(R.string.log_save_error)
                else -> stringResource(R.string.log_auto_saved)
            },
            style = MaterialTheme.typography.labelSmall,
            color = if (isError) colors.pink else colors.inkMuted,
            fontWeight = FontWeight.Bold,
        )
    }
}

@Composable
private fun DaySwitcher(state: LogUiState, onIntent: (LogIntent) -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Row(
            Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            // RTL: the start chevron moves one day back in time.
            HeaderIconButton(R.drawable.ic_chevron_right, stringResource(R.string.log_prev_day), {
                onIntent(LogIntent.PreviousDay)
            })
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Icon(
                    painter = painterResource(R.drawable.ic_calendar),
                    contentDescription = null,
                    tint = colors.ink,
                    modifier = Modifier.size(16.dp),
                )
                Text(
                    text = state.date.formatDayMonth(),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                if (state.isToday) {
                    Box(
                        Modifier
                            .clip(RoundedCornerShape(20.dp))
                            .background(colors.pinkContainer)
                            .padding(horizontal = 10.dp, vertical = 3.dp),
                    ) {
                        Text(
                            stringResource(R.string.log_today),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.pink,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
            }
            // Always rendered; disabled + greyed when the day is today (no future days).
            HeaderIconButton(
                icon = R.drawable.ic_chevron_left,
                contentDescription = stringResource(R.string.log_next_day),
                onClick = { if (!state.isToday) onIntent(LogIntent.NextDay) },
                modifier = if (state.isToday) Modifier.alpha(0.3f) else Modifier,
            )
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
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        stringResource(R.string.daytasks_progress, state.doneTaskCount.toPersianDigits(), tasks.size.toPersianDigits()),
                        style = MaterialTheme.typography.labelMedium,
                        color = colors.steel,
                        fontWeight = FontWeight.Bold,
                    )
                    Icon(
                        painter = painterResource(R.drawable.ic_check_circle),
                        contentDescription = null,
                        tint = colors.success,
                        modifier = Modifier.size(16.dp),
                    )
                }
            }
        }
        Spacer(Modifier.height(4.dp))
        Text(stringResource(R.string.daytasks_subtitle), style = MaterialTheme.typography.labelSmall, color = colors.inkMuted)
        Spacer(Modifier.height(12.dp))

        FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            TASK_TYPES.forEach { type ->
                TaskTypeChip(
                    type = type,
                    selected = state.newTaskType == type,
                    colors = colors,
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

/** Web day-planner type button: icon + label, brand-tinted when selected, 10dp rounded. */
@Composable
private fun TaskTypeChip(type: ReminderType, selected: Boolean, colors: RitmeColors, onClick: () -> Unit) {
    Row(
        Modifier
            .clip(RoundedCornerShape(10.dp))
            .background(if (selected) colors.pinkContainer else colors.surface)
            .border(1.dp, if (selected) colors.pink else colors.outline, RoundedCornerShape(10.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Icon(
            painter = painterResource(type.taskIconRes()),
            contentDescription = null,
            tint = if (selected) colors.pink else colors.steel,
            modifier = Modifier.size(14.dp),
        )
        Text(
            stringResource(type.taskLabelRes()),
            style = MaterialTheme.typography.labelMedium,
            color = if (selected) colors.pink else colors.steel,
            fontWeight = FontWeight.Bold,
        )
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
    ReminderType.DOCTOR -> R.string.daytasks_type_doctor
    ReminderType.MEDICATION -> R.string.reminder_type_medication
    ReminderType.APPOINTMENT -> R.string.reminder_type_appointment
    ReminderType.CUSTOM -> R.string.daytasks_type_custom
}

@androidx.annotation.DrawableRes
private fun ReminderType.taskIconRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.drawable.ic_stetho
    ReminderType.MEDICATION -> R.drawable.ic_pill
    ReminderType.APPOINTMENT -> R.drawable.ic_calendar
    ReminderType.CUSTOM -> R.drawable.ic_pencil
}

@Composable
private fun CategoryCard(
    category: HealthLogCategory,
    count: Int,
    onClick: () -> Unit,
    colors: RitmeColors,
) {
    val palette = category.palette()
    SurfaceCard(Modifier.clickable(onClick = onClick)) {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(40.dp).clip(CircleShape).background(palette.soft),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(category.iconRes()),
                    contentDescription = null,
                    tint = palette.icon,
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
                    color = if (count > 0) palette.icon else colors.inkMuted,
                    fontWeight = if (count > 0) FontWeight.Bold else FontWeight.Normal,
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
private fun CategorySheetContent(
    category: HealthLogCategory,
    state: LogUiState,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    // Web sheet caps its scroll body at ~58vh; the list grows to that then scrolls.
    val maxListHeight = (LocalConfiguration.current.screenHeightDp * 0.58f).dp
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.Top,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        Column(Modifier.weight(1f)) {
            Text(
                stringResource(category.labelRes()),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.log_sheet_hint),
                style = MaterialTheme.typography.labelSmall,
                color = colors.inkMuted,
            )
        }
        HeaderIconButton(
            icon = R.drawable.ic_x,
            contentDescription = stringResource(R.string.action_done),
            onClick = { onIntent(LogIntent.CloseSheet) },
        )
    }
    Spacer(Modifier.height(14.dp))
    LazyColumn(
        modifier = Modifier.fillMaxWidth().heightIn(max = maxListHeight),
        verticalArrangement = Arrangement.spacedBy(18.dp),
    ) {
        items(category.orderedFields(), key = { it.name }) { field ->
            FieldRow(field, state.values[field], onIntent, colors)
        }
    }
    Spacer(Modifier.height(16.dp))
    RitmePrimaryButton(text = stringResource(R.string.action_done), onClick = { onIntent(LogIntent.CloseSheet) })
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
        is HealthLogControl.Choice -> ChoiceField(field, control.options, value, single = true, showLabel = true, onIntent, colors)

        // The sheet title already names the category, so multi-select shows chips only (web).
        is HealthLogControl.MultiChoice ->
            ChoiceField(field, control.options, value, single = false, showLabel = false, onIntent, colors)

        HealthLogControl.Degree -> DegreeField(field, value, onIntent, colors)

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
    showLabel: Boolean,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val selectedSingle = (value as? HealthLogValue.Choice)?.option
    val selectedMulti = (value as? HealthLogValue.MultiChoice)?.options ?: emptyList()
    Column {
        if (showLabel) {
            FieldLabel(field, colors)
            Spacer(Modifier.height(8.dp))
        }
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

/** Web `.seg`: field label at the start, a compact inline low/medium/high segment at the end. */
@Composable
private fun DegreeField(
    field: HealthLogField,
    value: HealthLogValue?,
    onIntent: (LogIntent) -> Unit,
    colors: RitmeColors,
) {
    val selected = (value as? HealthLogValue.Choice)?.option
    Row(
        Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        FieldLabel(field, colors, Modifier.weight(1f))
        Row(
            Modifier
                .clip(RoundedCornerShape(14.dp))
                .background(colors.background)
                .padding(3.dp),
            horizontalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            DEGREE_OPTIONS.forEach { option ->
                val isSelected = option == selected
                Box(
                    Modifier
                        .height(30.dp)
                        .then(if (isSelected) Modifier.shadow(2.dp, RoundedCornerShape(11.dp)) else Modifier)
                        .clip(RoundedCornerShape(11.dp))
                        .background(if (isSelected) colors.surface else Color.Transparent)
                        .clickable {
                            onIntent(LogIntent.SetValue(field, if (isSelected) null else HealthLogValue.Choice(option)))
                        }
                        .padding(horizontal = 12.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = optionLabelRes(field, option)?.let { stringResource(it) } ?: option,
                        style = MaterialTheme.typography.labelMedium,
                        color = if (isSelected) colors.pink else colors.inkMuted,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }
        }
    }
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
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                WheelPicker(
                    count = steps,
                    selectedIndex = (((current - control.min) / control.step).roundToInt()).coerceIn(0, steps - 1),
                    onSelected = { index ->
                        onIntent(LogIntent.SetValue(field, HealthLogValue.Number(control.min + index * control.step)))
                    },
                    label = { index -> formatMeasure(control.min + index * control.step) },
                    modifier = Modifier.width(120.dp),
                    visibleCount = 3,
                )
                Spacer(Modifier.width(12.dp))
                Text(
                    text = stringResource(
                        if (field == HealthLogField.WEIGHT) R.string.log_unit_kg else R.string.log_unit_celsius,
                    ),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                    fontWeight = FontWeight.Bold,
                )
            }
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
    // The sheet title already names this "یادداشت", so the textarea stands alone (web).
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

/** The web iOS-style pill switch: 46×28 track (brand on / grey off) with a 22dp white knob. */
@Composable
private fun RitmeSwitch(checked: Boolean, colors: RitmeColors, onCheckedChange: (Boolean) -> Unit) {
    // TODO token: web switch-off track #D8DEE5 has no design token yet.
    val trackColor = if (checked) colors.pink else Color(0xFFD8DEE5)
    Box(
        Modifier
            .width(46.dp)
            .height(28.dp)
            .clip(RoundedCornerShape(99.dp))
            .background(trackColor)
            .clickable { onCheckedChange(!checked) }
            .padding(3.dp),
        contentAlignment = if (checked) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Box(
            Modifier
                .size(22.dp)
                .shadow(2.dp, CircleShape)
                .clip(CircleShape)
                .background(colors.surface),
        )
    }
}

/** «۶۲٫۵» — trims a whole number's decimal and localizes digits. */
private fun formatMeasure(value: Double): String {
    val rounded = (value * 10).roundToInt() / 10.0
    val text = if (rounded % 1.0 == 0.0) rounded.toInt().toString() else rounded.toString()
    return text.toPersianDigits()
}

private val DEGREE_OPTIONS = listOf("low", "medium", "high")
