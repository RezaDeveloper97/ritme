package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.runtime.Immutable
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft
import ir.ritmeapp.ritme.domain.model.ReminderRecurrence
import ir.ritmeapp.ritme.domain.model.ReminderType
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.ManageRemindersUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** The new-reminder form draft. */
@Immutable
data class ReminderFormState(
    val type: ReminderType = ReminderType.DOCTOR,
    val title: String = "",
    val subtitle: String = "",
    val recurrence: ReminderRecurrence = ReminderRecurrence.NONE,
    val hour: Int = DEFAULT_HOUR,
    val minute: Int = 0,
) {
    val canSubmit: Boolean get() = title.isNotBlank()

    companion object {
        // Web `AddReminderForm` defaults the time input to 08:00.
        const val DEFAULT_HOUR = 8
    }
}

/** The reminders screen's single immutable state (§4 MVI). */
@Immutable
data class RemindersUiState(
    val loading: Boolean = true,
    val isError: Boolean = false,
    val reminders: List<Reminder> = emptyList(),
    val formOpen: Boolean = false,
    val form: ReminderFormState = ReminderFormState(),
    val submitting: Boolean = false,
    val formError: Boolean = false,
    val confirmingDeleteId: Long? = null,
    val deleting: Boolean = false,
    /** The reminder whose active-toggle request is in flight (its row disables meanwhile). */
    val togglingId: Long? = null,
)

sealed interface RemindersIntent {
    data object Retry : RemindersIntent
    data object ToggleForm : RemindersIntent
    data class FormChanged(val form: ReminderFormState) : RemindersIntent
    data object Submit : RemindersIntent
    data class ToggleActive(val reminder: Reminder) : RemindersIntent
    data class AskDelete(val id: Long) : RemindersIntent
    data object CancelDelete : RemindersIntent
    data class ConfirmDelete(val id: Long) : RemindersIntent
}

/** Reminders CRUD state machine over `/reminders`. */
class RemindersViewModel(
    private val reminders: ManageRemindersUseCase,
) : ViewModel() {

    private val _state = MutableStateFlow(RemindersUiState())
    val state: StateFlow<RemindersUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun onIntent(intent: RemindersIntent) {
        when (intent) {
            RemindersIntent.Retry -> load()
            RemindersIntent.ToggleForm ->
                _state.update { it.copy(formOpen = !it.formOpen, formError = false) }

            is RemindersIntent.FormChanged -> _state.update { it.copy(form = intent.form, formError = false) }
            RemindersIntent.Submit -> submit()
            is RemindersIntent.ToggleActive -> toggleActive(intent.reminder)
            is RemindersIntent.AskDelete -> _state.update { it.copy(confirmingDeleteId = intent.id) }
            RemindersIntent.CancelDelete -> _state.update { it.copy(confirmingDeleteId = null) }
            is RemindersIntent.ConfirmDelete -> delete(intent.id)
        }
    }

    private fun load() {
        viewModelScope.launch {
            _state.update { it.copy(loading = true, isError = false) }
            when (val result = reminders.list()) {
                is AppResult.Failure -> _state.update { it.copy(loading = false, isError = true) }
                is AppResult.Success -> _state.update { it.copy(loading = false, reminders = result.value) }
            }
        }
    }

    private fun submit() {
        val snapshot = _state.value
        if (!snapshot.form.canSubmit || snapshot.submitting) return
        viewModelScope.launch {
            Breadcrumbs.add("reminders:create")
            _state.update { it.copy(submitting = true, formError = false) }
            val form = snapshot.form
            val draft = ReminderDraft(
                type = form.type,
                title = form.title.trim(),
                subtitle = form.subtitle.trim().takeIf { it.isNotEmpty() },
                recurrence = form.recurrence,
                recurrenceTime = if (form.recurrence != ReminderRecurrence.NONE) {
                    "%02d:%02d".format(form.hour, form.minute)
                } else {
                    null
                },
            )
            when (reminders.create(draft)) {
                is AppResult.Failure -> _state.update { it.copy(submitting = false, formError = true) }
                is AppResult.Success -> {
                    _state.update {
                        it.copy(submitting = false, formOpen = false, form = ReminderFormState())
                    }
                    load()
                }
            }
        }
    }

    private fun toggleActive(reminder: Reminder) {
        if (_state.value.togglingId != null) return
        viewModelScope.launch {
            Breadcrumbs.add("reminders:toggle")
            _state.update { it.copy(togglingId = reminder.id) }
            val draft = ReminderDraft(
                type = reminder.type,
                title = reminder.title,
                subtitle = reminder.subtitle,
                notes = reminder.notes,
                scheduledAt = reminder.scheduledAt,
                recurrence = reminder.recurrence,
                recurrenceTime = reminder.recurrenceTime,
                isActive = !reminder.isActive,
            )
            val updated = reminders.update(reminder.id, draft).getOrNull()
            _state.update { current ->
                current.copy(
                    togglingId = null,
                    reminders = if (updated != null) {
                        current.reminders.map { if (it.id == updated.id) updated else it }
                    } else {
                        current.reminders
                    },
                )
            }
        }
    }

    private fun delete(id: Long) {
        if (_state.value.deleting) return
        viewModelScope.launch {
            Breadcrumbs.add("reminders:delete")
            _state.update { it.copy(deleting = true) }
            if (reminders.delete(id) is AppResult.Success) {
                _state.update { current ->
                    current.copy(
                        deleting = false,
                        confirmingDeleteId = null,
                        reminders = current.reminders.filterNot { it.id == id },
                    )
                }
            } else {
                _state.update { it.copy(deleting = false, confirmingDeleteId = null) }
            }
        }
    }
}
