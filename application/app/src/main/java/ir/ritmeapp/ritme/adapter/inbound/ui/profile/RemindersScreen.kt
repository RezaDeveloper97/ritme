package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.HeaderIconButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeBottomSheet
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmePrimaryButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.RitmeSoftButton
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.WheelPicker
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderRecurrence
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The reminders manager (web `/profile/reminders`): one grouped card of divider-separated rows,
 * each with an active toggle and inline delete confirmation, plus an inline create form (type,
 * title, recurrence, time). Loading and error render as text cards; the empty card carries an
 * «افزودن یادآور» CTA.
 */
@Composable
fun RemindersScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: RemindersViewModel = viewModel(factory = container.remindersViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current
    var timeSheetOpen by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:reminders:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfileReminders.route, null, System.currentTimeMillis()),
        )
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = {
            ScreenHeader(
                title = stringResource(R.string.reminders_title),
                onBack = onBack,
                trailing = {
                    HeaderIconButton(
                        icon = if (state.formOpen) R.drawable.ic_x else R.drawable.ic_plus,
                        contentDescription = stringResource(R.string.reminders_add),
                        onClick = { viewModel.onIntent(RemindersIntent.ToggleForm) },
                        tint = colors.pink,
                    )
                },
            )
        },
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(start = 18.dp, end = 18.dp, top = 4.dp, bottom = 24.dp),
            ) {
                if (state.formOpen) {
                    item(key = "form") {
                        AddReminderForm(
                            state = state,
                            onIntent = viewModel::onIntent,
                            onEditTime = { timeSheetOpen = true },
                            colors = colors,
                        )
                    }
                }
                when {
                    state.loading -> item(key = "loading") {
                        MessageCard(stringResource(R.string.reminders_loading), colors)
                    }

                    state.isError -> item(key = "error") {
                        MessageCard(stringResource(R.string.reminders_load_error), colors)
                    }

                    state.reminders.isNotEmpty() -> item(key = "list") {
                        Column(
                            Modifier
                                .fillMaxWidth()
                                .padding(top = 12.dp)
                                .clip(RoundedCornerShape(16.dp))
                                .background(colors.surface)
                                .border(1.dp, colors.outline, RoundedCornerShape(16.dp)),
                        ) {
                            state.reminders.forEachIndexed { index, reminder ->
                                if (index > 0) {
                                    Box(
                                        Modifier
                                            .fillMaxWidth()
                                            .padding(start = 60.dp)
                                            .height(1.dp)
                                            .background(colors.outline),
                                    )
                                }
                                ReminderRow(reminder, state, viewModel::onIntent, colors)
                            }
                        }
                    }

                    else -> item(key = "empty") {
                        EmptyState(
                            onAdd = { if (!state.formOpen) viewModel.onIntent(RemindersIntent.ToggleForm) },
                            colors = colors,
                        )
                    }
                }
            }

            // Time picker sheet (web's native `<input type="time">`), placed as the last root child.
            RitmeBottomSheet(visible = timeSheetOpen, onDismiss = { timeSheetOpen = false }) {
                val form = state.form
                Text(
                    stringResource(R.string.reminder_form_time),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                Spacer(Modifier.height(8.dp))
                Row(Modifier.fillMaxWidth()) {
                    WheelPicker(
                        count = HOURS_IN_DAY,
                        selectedIndex = form.hour,
                        onSelected = { onIntentTime(viewModel, form.copy(hour = it)) },
                        label = { "%02d".format(it).toPersianDigits() },
                        modifier = Modifier.weight(1f),
                    )
                    WheelPicker(
                        count = MINUTE_STEPS,
                        selectedIndex = form.minute / MINUTE_STEP,
                        onSelected = { onIntentTime(viewModel, form.copy(minute = it * MINUTE_STEP)) },
                        label = { "%02d".format(it * MINUTE_STEP).toPersianDigits() },
                        modifier = Modifier.weight(1f),
                    )
                }
                Spacer(Modifier.height(12.dp))
                RitmePrimaryButton(
                    text = stringResource(R.string.action_done),
                    onClick = { timeSheetOpen = false },
                )
            }
        }
    }
}

private fun onIntentTime(viewModel: RemindersViewModel, form: ReminderFormState) {
    viewModel.onIntent(RemindersIntent.FormChanged(form))
}

/** A centered muted-text card used for the loading and error states (web `.card` padding 24). */
@Composable
private fun MessageCard(text: String, colors: RitmeColors) {
    SurfaceCard(modifier = Modifier.padding(top = 12.dp)) {
        Text(
            text = text,
            style = MaterialTheme.typography.bodyMedium.copy(fontSize = 14.sp),
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
        )
    }
}

@Composable
private fun EmptyState(onAdd: () -> Unit, colors: RitmeColors) {
    SurfaceCard(modifier = Modifier.padding(top = 12.dp)) {
        Column(
            Modifier.fillMaxWidth().padding(vertical = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier.size(56.dp).clip(CircleShape).background(PinkSoft),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_bell),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(26.dp),
                )
            }
            Spacer(Modifier.height(14.dp))
            Text(
                stringResource(R.string.reminders_empty_title),
                style = MaterialTheme.typography.titleMedium.copy(fontSize = 15.sp),
                color = colors.ink,
                fontWeight = FontWeight.ExtraBold,
            )
            Spacer(Modifier.height(6.dp))
            Text(
                stringResource(R.string.reminders_empty_subtitle),
                style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp, lineHeight = 22.sp),
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
            )
            Spacer(Modifier.height(18.dp))
            RitmePrimaryButton(text = stringResource(R.string.reminders_add), onClick = onAdd)
        }
    }
}

@Composable
private fun AddReminderForm(
    state: RemindersUiState,
    onIntent: (RemindersIntent) -> Unit,
    onEditTime: () -> Unit,
    colors: RitmeColors,
) {
    val form = state.form
    SurfaceCard(modifier = Modifier.padding(top = 12.dp)) {
        Text(
            stringResource(R.string.reminder_form_title),
            style = MaterialTheme.typography.titleMedium.copy(fontSize = 14.sp),
            color = colors.ink,
            fontWeight = FontWeight.ExtraBold,
        )
        Spacer(Modifier.height(12.dp))

        FieldLabel(stringResource(R.string.reminder_form_type), colors)
        DropdownField(
            selected = form.type,
            options = ReminderType.entries.map { it to stringResource(it.labelRes()) },
            onSelect = { onIntent(RemindersIntent.FormChanged(form.copy(type = it))) },
            colors = colors,
        )

        Spacer(Modifier.height(12.dp))
        FieldLabel(stringResource(R.string.reminder_form_title_label), colors)
        OutlinedTextField(
            value = form.title,
            onValueChange = {
                if (it.length <= TITLE_MAX) onIntent(RemindersIntent.FormChanged(form.copy(title = it)))
            },
            placeholder = { Text(stringResource(R.string.reminder_form_title_placeholder), color = colors.inkMuted) },
            singleLine = true,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(12.dp))
        FieldLabel(stringResource(R.string.reminder_form_subtitle_label), colors)
        OutlinedTextField(
            value = form.subtitle,
            onValueChange = {
                if (it.length <= SUBTITLE_MAX) onIntent(RemindersIntent.FormChanged(form.copy(subtitle = it)))
            },
            placeholder = { Text(stringResource(R.string.reminder_form_subtitle_placeholder), color = colors.inkMuted) },
            singleLine = true,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(12.dp))
        FieldLabel(stringResource(R.string.reminder_form_recurrence), colors)
        DropdownField(
            selected = form.recurrence,
            options = ReminderRecurrence.entries.map { it to stringResource(it.labelRes()) },
            onSelect = { onIntent(RemindersIntent.FormChanged(form.copy(recurrence = it))) },
            colors = colors,
        )

        if (form.recurrence != ReminderRecurrence.NONE) {
            Spacer(Modifier.height(12.dp))
            FieldLabel(stringResource(R.string.reminder_form_time), colors)
            FieldBox(onClick = onEditTime, colors = colors) {
                Text(
                    text = "%02d:%02d".format(form.hour, form.minute).toPersianDigits(),
                    style = MaterialTheme.typography.bodyLarge.copy(fontSize = 15.sp),
                    color = colors.ink,
                    modifier = Modifier.weight(1f),
                )
                Icon(
                    painter = painterResource(R.drawable.ic_alarm),
                    contentDescription = null,
                    tint = colors.inkMuted,
                    modifier = Modifier.size(18.dp),
                )
            }
        }

        if (state.formError) {
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.reminder_form_error),
                style = MaterialTheme.typography.labelSmall.copy(fontSize = 13.sp),
                color = colors.error,
            )
        }

        Spacer(Modifier.height(16.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalAlignment = Alignment.CenterVertically) {
            RitmePrimaryButton(
                text = if (state.submitting) {
                    stringResource(R.string.edit_saving)
                } else {
                    stringResource(R.string.reminder_form_submit)
                },
                onClick = { onIntent(RemindersIntent.Submit) },
                enabled = form.canSubmit && !state.submitting,
                modifier = Modifier.weight(1f),
            )
            RitmeSoftButton(
                text = stringResource(R.string.action_cancel),
                onClick = { onIntent(RemindersIntent.ToggleForm) },
                enabled = !state.submitting,
            )
        }
    }
}

@Composable
private fun FieldLabel(text: String, colors: RitmeColors) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp),
        color = colors.inkMuted,
        modifier = Modifier.padding(bottom = 8.dp),
    )
}

/** The web `.field`: a 52dp, 14dp-rounded, 1.5dp-outlined surface box holding a start-aligned row. */
@Composable
private fun FieldBox(
    onClick: () -> Unit,
    colors: RitmeColors,
    content: @Composable androidx.compose.foundation.layout.RowScope.() -> Unit,
) {
    Row(
        Modifier
            .fillMaxWidth()
            .height(52.dp)
            .clip(RoundedCornerShape(14.dp))
            .background(colors.surface)
            .border(1.5.dp, colors.outline, RoundedCornerShape(14.dp))
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        content = content,
    )
}

/** A `.field`-styled select: shows the current label, opens a menu of options (web `<select>`). */
@Composable
private fun <T> DropdownField(
    selected: T,
    options: List<Pair<T, String>>,
    onSelect: (T) -> Unit,
    colors: RitmeColors,
) {
    var expanded by remember { mutableStateOf(false) }
    val currentLabel = options.firstOrNull { it.first == selected }?.second.orEmpty()
    Box(Modifier.fillMaxWidth()) {
        FieldBox(onClick = { expanded = true }, colors = colors) {
            Text(
                text = currentLabel,
                style = MaterialTheme.typography.bodyLarge.copy(fontSize = 15.sp),
                color = colors.ink,
                modifier = Modifier.weight(1f),
            )
            Icon(
                painter = painterResource(R.drawable.ic_chevron_down),
                contentDescription = null,
                tint = colors.inkMuted,
                modifier = Modifier.size(18.dp),
            )
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { (value, label) ->
                DropdownMenuItem(
                    text = { Text(label, color = colors.ink) },
                    onClick = {
                        onSelect(value)
                        expanded = false
                    },
                )
            }
        }
    }
}

@Composable
private fun ReminderRow(
    reminder: Reminder,
    state: RemindersUiState,
    onIntent: (RemindersIntent) -> Unit,
    colors: RitmeColors,
) {
    val confirming = state.confirmingDeleteId == reminder.id
    val deletePending = confirming && state.deleting
    val togglePending = state.togglingId == reminder.id
    val second = secondLine(reminder)

    Column {
        Row(
            Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                Modifier
                    .size(34.dp)
                    .alpha(if (reminder.isActive) 1f else INACTIVE_ALPHA)
                    .clip(RoundedCornerShape(11.dp))
                    .background(colors.background),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(reminder.type.iconRes()),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(19.dp),
                )
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = reminder.title,
                    style = MaterialTheme.typography.bodyMedium.copy(fontSize = 14.sp),
                    color = if (reminder.isActive) colors.ink else colors.inkMuted,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (second.isNotBlank()) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        text = second,
                        style = MaterialTheme.typography.labelSmall.copy(fontSize = 12.sp),
                        color = colors.inkMuted,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Spacer(Modifier.width(10.dp))
            Switch(
                checked = reminder.isActive,
                onCheckedChange = { onIntent(RemindersIntent.ToggleActive(reminder)) },
                enabled = !togglePending && !deletePending,
                colors = SwitchDefaults.colors(
                    checkedTrackColor = colors.pink,
                    checkedThumbColor = colors.onPink,
                    uncheckedTrackColor = colors.outline,
                    uncheckedThumbColor = colors.surface,
                ),
            )
            Spacer(Modifier.width(6.dp))
            Icon(
                painter = painterResource(if (confirming) R.drawable.ic_x else R.drawable.ic_trash),
                contentDescription = stringResource(R.string.action_delete),
                tint = colors.error,
                modifier = Modifier
                    .size(30.dp)
                    .alpha(if (deletePending) INACTIVE_ALPHA else 1f)
                    .clip(CircleShape)
                    .clickable(enabled = !deletePending) {
                        onIntent(
                            if (confirming) RemindersIntent.CancelDelete else RemindersIntent.AskDelete(reminder.id),
                        )
                    }
                    .padding(6.dp),
            )
        }
        if (confirming) {
            Row(
                Modifier.fillMaxWidth().padding(start = 46.dp, end = 14.dp, bottom = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text(
                    text = if (deletePending) {
                        stringResource(R.string.reminder_delete_deleting)
                    } else {
                        stringResource(R.string.reminder_delete_question)
                    },
                    style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp),
                    color = colors.inkMuted,
                    modifier = Modifier.weight(1f),
                )
                ConfirmPill(
                    text = stringResource(R.string.action_delete),
                    textColor = colors.error,
                    background = colors.error.copy(alpha = 0.10f),
                    enabled = !deletePending,
                    onClick = { onIntent(RemindersIntent.ConfirmDelete(reminder.id)) },
                )
                ConfirmPill(
                    text = stringResource(R.string.action_cancel),
                    textColor = colors.ink,
                    background = colors.background,
                    enabled = !deletePending,
                    onClick = { onIntent(RemindersIntent.CancelDelete) },
                )
            }
        }
    }
}

@Composable
private fun ConfirmPill(
    text: String,
    textColor: androidx.compose.ui.graphics.Color,
    background: androidx.compose.ui.graphics.Color,
    enabled: Boolean,
    onClick: () -> Unit,
) {
    Text(
        text = text,
        style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp),
        color = textColor,
        fontWeight = FontWeight.Bold,
        modifier = Modifier
            .alpha(if (enabled) 1f else INACTIVE_ALPHA)
            .clip(RoundedCornerShape(10.dp))
            .background(background)
            .clickable(enabled = enabled, onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 6.dp),
    )
}

/** Second line: the optional subtitle plus a human schedule summary, joined by « · » (web parity). */
@Composable
private fun secondLine(reminder: Reminder): String {
    val subtitle = reminder.subtitle?.takeIf { it.isNotBlank() }
    val schedule = scheduleText(reminder)
    return listOfNotNull(subtitle, schedule).joinToString(" · ")
}

/** «هفتگی ساعت ۰۹:۰۰» for recurring reminders, or the one-off Jalali date (null when neither). */
@Composable
private fun scheduleText(reminder: Reminder): String? {
    if (reminder.recurrence != ReminderRecurrence.NONE) {
        val recurrence = stringResource(reminder.recurrence.labelRes())
        val time = reminder.recurrenceTime?.toPersianDigits()
        return if (time != null) stringResource(R.string.reminder_recurrence_at, recurrence, time) else recurrence
    }
    val date = reminder.scheduledAt?.substringBefore('T')?.substringBefore(' ')
        ?.let(GregorianDate::parseIso)?.toJalali()
    return date?.let { "${it.day.toPersianDigits()} ${it.monthName} ${it.year.toPersianDigits()}" }
}

@androidx.annotation.StringRes
private fun ReminderType.labelRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.string.reminder_type_doctor
    ReminderType.MEDICATION -> R.string.reminder_type_medication
    ReminderType.APPOINTMENT -> R.string.reminder_type_appointment
    ReminderType.CUSTOM -> R.string.reminder_type_custom
}

@DrawableRes
private fun ReminderType.iconRes(): Int = when (this) {
    ReminderType.DOCTOR -> R.drawable.ic_stetho
    ReminderType.MEDICATION -> R.drawable.ic_pill
    ReminderType.APPOINTMENT -> R.drawable.ic_calendar
    ReminderType.CUSTOM -> R.drawable.ic_bell
}

@androidx.annotation.StringRes
private fun ReminderRecurrence.labelRes(): Int = when (this) {
    ReminderRecurrence.NONE -> R.string.recurrence_none
    ReminderRecurrence.DAILY -> R.string.recurrence_daily
    ReminderRecurrence.WEEKLY -> R.string.recurrence_weekly
    ReminderRecurrence.MONTHLY -> R.string.recurrence_monthly
}

/** Web `--pink-soft` (#FFCFCF): the empty-state badge tint. */
private val PinkSoft = androidx.compose.ui.graphics.Color(0xFFFFCFCF) // TODO token (--pink-soft)

private const val HOURS_IN_DAY = 24
private const val MINUTE_STEP = 5
private const val MINUTE_STEPS = 60 / MINUTE_STEP
private const val TITLE_MAX = 120
private const val SUBTITLE_MAX = 160
private const val INACTIVE_ALPHA = 0.55f
