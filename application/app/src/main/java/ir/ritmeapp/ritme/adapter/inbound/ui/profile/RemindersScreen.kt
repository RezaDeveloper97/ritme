package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
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
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SelectableChip
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
 * The reminders manager (web `/profile/reminders`): the list with per-row active toggles and
 * inline delete confirmation, plus an inline create form (type, title, recurrence, time).
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
                    Box(
                        Modifier.size(38.dp).clip(CircleShape).background(colors.pinkContainer)
                            .clickable { viewModel.onIntent(RemindersIntent.ToggleForm) },
                        contentAlignment = Alignment.Center,
                    ) {
                        Icon(
                            painter = painterResource(if (state.formOpen) R.drawable.ic_x else R.drawable.ic_plus),
                            contentDescription = stringResource(R.string.reminders_add),
                            tint = colors.pink,
                            modifier = Modifier.size(18.dp),
                        )
                    }
                },
            )
        },
    ) { padding ->
        when {
            state.loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.pink)
            }

            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                if (state.formOpen) {
                    item(key = "form") { AddReminderForm(state, viewModel::onIntent, colors) }
                }
                if (state.reminders.isEmpty() && !state.formOpen) {
                    item(key = "empty") { EmptyState(colors) }
                }
                items(state.reminders, key = { it.id }) { reminder ->
                    ReminderRow(reminder, state, viewModel::onIntent, colors)
                }
            }
        }
    }
}

@Composable
private fun EmptyState(colors: RitmeColors) {
    SurfaceCard {
        Column(Modifier.fillMaxWidth().padding(vertical = 12.dp), horizontalAlignment = Alignment.CenterHorizontally) {
            Icon(
                painter = painterResource(R.drawable.ic_alarm),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(32.dp),
            )
            Spacer(Modifier.height(10.dp))
            Text(
                stringResource(R.string.reminders_empty_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(4.dp))
            Text(
                stringResource(R.string.reminders_empty_subtitle),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
            )
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun AddReminderForm(state: RemindersUiState, onIntent: (RemindersIntent) -> Unit, colors: RitmeColors) {
    val form = state.form
    SurfaceCard {
        Text(
            stringResource(R.string.reminder_form_title),
            style = MaterialTheme.typography.titleMedium,
            color = colors.ink,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(12.dp))

        Text(stringResource(R.string.reminder_form_type), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(6.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            ReminderType.entries.forEach { type ->
                SelectableChip(
                    label = stringResource(type.labelRes()),
                    selected = form.type == type,
                    onClick = { onIntent(RemindersIntent.FormChanged(form.copy(type = type))) },
                )
            }
        }

        Spacer(Modifier.height(12.dp))
        Text(stringResource(R.string.reminder_form_title_label), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(6.dp))
        OutlinedTextField(
            value = form.title,
            onValueChange = { onIntent(RemindersIntent.FormChanged(form.copy(title = it))) },
            placeholder = { Text(stringResource(R.string.reminder_form_title_placeholder), color = colors.inkMuted) },
            singleLine = true,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(12.dp))
        Text(stringResource(R.string.reminder_form_subtitle_label), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(6.dp))
        OutlinedTextField(
            value = form.subtitle,
            onValueChange = { onIntent(RemindersIntent.FormChanged(form.copy(subtitle = it))) },
            placeholder = { Text(stringResource(R.string.reminder_form_subtitle_placeholder), color = colors.inkMuted) },
            singleLine = true,
            colors = OutlinedTextFieldDefaults.colors(
                focusedBorderColor = colors.pink,
                unfocusedBorderColor = colors.outline,
            ),
            modifier = Modifier.fillMaxWidth(),
        )

        Spacer(Modifier.height(12.dp))
        Text(stringResource(R.string.reminder_form_recurrence), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
        Spacer(Modifier.height(6.dp))
        FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            ReminderRecurrence.entries.forEach { recurrence ->
                SelectableChip(
                    label = stringResource(recurrence.labelRes()),
                    selected = form.recurrence == recurrence,
                    onClick = { onIntent(RemindersIntent.FormChanged(form.copy(recurrence = recurrence))) },
                )
            }
        }

        if (form.recurrence != ReminderRecurrence.NONE) {
            Spacer(Modifier.height(12.dp))
            Text(stringResource(R.string.reminder_form_time), style = MaterialTheme.typography.labelMedium, color = colors.inkMuted)
            Row(Modifier.fillMaxWidth()) {
                WheelPicker(
                    count = HOURS_IN_DAY,
                    selectedIndex = form.hour,
                    onSelected = { onIntent(RemindersIntent.FormChanged(form.copy(hour = it))) },
                    label = { "%02d".format(it).toPersianDigits() },
                    visibleCount = 3,
                    modifier = Modifier.weight(1f),
                )
                WheelPicker(
                    count = MINUTE_STEPS,
                    selectedIndex = form.minute / MINUTE_STEP,
                    onSelected = { onIntent(RemindersIntent.FormChanged(form.copy(minute = it * MINUTE_STEP))) },
                    label = { "%02d".format(it * MINUTE_STEP).toPersianDigits() },
                    visibleCount = 3,
                    modifier = Modifier.weight(1f),
                )
            }
        }

        if (state.formError) {
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.reminder_form_error),
                style = MaterialTheme.typography.labelSmall,
                color = colors.error,
            )
        }

        Spacer(Modifier.height(12.dp))
        Button(
            onClick = { onIntent(RemindersIntent.Submit) },
            enabled = form.canSubmit && !state.submitting,
            colors = ButtonDefaults.buttonColors(
                containerColor = colors.pink,
                contentColor = colors.onPink,
                disabledContainerColor = colors.outline,
                disabledContentColor = colors.inkMuted,
            ),
            shape = RoundedCornerShape(14.dp),
            modifier = Modifier.fillMaxWidth().height(46.dp),
        ) {
            Text(
                text = if (state.submitting) {
                    stringResource(R.string.edit_saving)
                } else {
                    stringResource(R.string.reminder_form_submit)
                },
                fontWeight = FontWeight.Bold,
            )
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
    SurfaceCard {
        Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(38.dp).clip(CircleShape).background(colors.pinkContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(reminder.type.iconRes()),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(18.dp),
                )
            }
            Spacer(Modifier.width(10.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = reminder.title,
                    style = MaterialTheme.typography.bodyMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = scheduleSummary(reminder),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                )
            }
            Switch(
                checked = reminder.isActive,
                onCheckedChange = { onIntent(RemindersIntent.ToggleActive(reminder)) },
                colors = SwitchDefaults.colors(
                    checkedTrackColor = colors.pink,
                    checkedThumbColor = colors.onPink,
                    uncheckedTrackColor = colors.outline,
                    uncheckedThumbColor = colors.surface,
                ),
            )
            Spacer(Modifier.width(6.dp))
            Icon(
                painter = painterResource(R.drawable.ic_trash),
                contentDescription = stringResource(R.string.action_delete),
                tint = colors.error,
                modifier = Modifier
                    .size(30.dp)
                    .clip(CircleShape)
                    .clickable { onIntent(RemindersIntent.AskDelete(reminder.id)) }
                    .padding(6.dp),
            )
        }
        if (state.confirmingDeleteId == reminder.id) {
            Spacer(Modifier.height(10.dp))
            Row(
                Modifier.fillMaxWidth().clip(RoundedCornerShape(12.dp)).background(colors.periodContainer).padding(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    stringResource(R.string.reminder_delete_question),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.ink,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = stringResource(R.string.action_delete),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.error,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier
                        .clip(RoundedCornerShape(10.dp))
                        .clickable { onIntent(RemindersIntent.ConfirmDelete(reminder.id)) }
                        .padding(horizontal = 10.dp, vertical = 6.dp),
                )
                Text(
                    text = stringResource(R.string.action_cancel),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                    modifier = Modifier
                        .clip(RoundedCornerShape(10.dp))
                        .clickable { onIntent(RemindersIntent.CancelDelete) }
                        .padding(horizontal = 10.dp, vertical = 6.dp),
                )
            }
        }
    }
}

/** «هفتگی ساعت ۰۹:۰۰» for recurring reminders, or the one-shot date. */
@Composable
private fun scheduleSummary(reminder: Reminder): String {
    if (reminder.recurrence != ReminderRecurrence.NONE) {
        val recurrence = stringResource(reminder.recurrence.labelRes())
        val time = reminder.recurrenceTime?.toPersianDigits()
        return if (time != null) stringResource(R.string.reminder_recurrence_at, recurrence, time) else recurrence
    }
    val date = reminder.scheduledAt?.substringBefore('T')?.substringBefore(' ')
        ?.let(GregorianDate::parseIso)?.toJalali()
    return date?.let { "${it.day.toPersianDigits()} ${it.monthName} ${it.year.toPersianDigits()}" }
        ?: stringResource(R.string.recurrence_none)
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

private const val HOURS_IN_DAY = 24
private const val MINUTE_STEP = 5
private const val MINUTE_STEPS = 60 / MINUTE_STEP
