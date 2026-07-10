package ir.ritmeapp.ritme.adapter.inbound.ui.log

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.DailyHealthLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderDraft
import ir.ritmeapp.ritme.domain.model.getOrNull
import ir.ritmeapp.ritme.domain.port.inbound.HealthLogUseCase
import ir.ritmeapp.ritme.domain.port.inbound.ManageRemindersUseCase
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Daily-log screen state machine. Day switching reloads the saved log for that day and
 * replaces the draft; edits mutate only the draft until [LogIntent.Save] upserts it.
 * Future days are unreachable (the next-day intent stops at today).
 */
class LogViewModel(
    private val healthLog: HealthLogUseCase,
    private val reminders: ManageRemindersUseCase,
    private val today: JalaliDate,
    initialDateIso: String?,
) : ViewModel() {

    private val _state = MutableStateFlow(
        LogUiState(date = resolveInitialDay(initialDateIso), isToday = false),
    )
    val state: StateFlow<LogUiState> = _state.asStateFlow()

    private var loadJob: Job? = null

    init {
        refreshTodayFlag()
        loadDay(_state.value.date)
        loadReminders()
    }

    fun onIntent(intent: LogIntent) {
        when (intent) {
            LogIntent.PreviousDay -> moveDay(-1)
            LogIntent.NextDay -> moveDay(+1)
            is LogIntent.OpenCategory -> _state.update { it.copy(openCategory = intent.category) }
            LogIntent.CloseSheet -> _state.update { it.copy(openCategory = null) }
            is LogIntent.SetValue -> setValue(intent)
            LogIntent.Save -> save()
            is LogIntent.NewTaskTitleChanged -> _state.update { it.copy(newTaskTitle = intent.title, taskError = false) }
            is LogIntent.NewTaskTypeChanged -> _state.update { it.copy(newTaskType = intent.type) }
            LogIntent.AddTask -> addTask()
            is LogIntent.ToggleTaskDone -> toggleTaskDone(intent.reminder)
            is LogIntent.DeleteTask -> deleteTask(intent.id)
        }
    }

    private fun resolveInitialDay(initialDateIso: String?): JalaliDate {
        val requested = initialDateIso?.let(GregorianDate::parseIso)?.toJalali() ?: return today
        // Future days can't be logged; clamp to today like the web page does.
        return if (requested.toJdn() > today.toJdn()) today else requested
    }

    private fun moveDay(delta: Int) {
        val target = _state.value.date.addDays(delta)
        if (target.toJdn() > today.toJdn()) return
        _state.update { it.copy(date = target, values = emptyMap(), saveState = LogSaveState.IDLE) }
        refreshTodayFlag()
        loadDay(target)
    }

    private fun refreshTodayFlag() {
        _state.update { it.copy(isToday = it.date.toJdn() == today.toJdn()) }
    }

    private fun loadDay(day: JalaliDate) {
        loadJob?.cancel()
        loadJob = viewModelScope.launch {
            _state.update { it.copy(loading = true) }
            val result = healthLog.logFor(day.toGregorian())
            _state.update { current ->
                if (current.date != day) return@update current
                current.copy(
                    loading = false,
                    values = (result as? AppResult.Success)?.value?.values ?: emptyMap(),
                )
            }
        }
    }

    private fun setValue(intent: LogIntent.SetValue) {
        _state.update { current ->
            val values = current.values.toMutableMap()
            if (intent.value == null) values.remove(intent.field) else values[intent.field] = intent.value
            current.copy(values = values, saveState = LogSaveState.IDLE)
        }
    }

    private fun save() {
        val snapshot = _state.value
        if (!snapshot.canSave) return
        viewModelScope.launch {
            Breadcrumbs.add("log:save")
            _state.update { it.copy(saveState = LogSaveState.SAVING) }
            val result = healthLog.save(DailyHealthLog(snapshot.date.toGregorian(), snapshot.values))
            _state.update {
                it.copy(saveState = if (result is AppResult.Success) LogSaveState.SAVED else LogSaveState.ERROR)
            }
        }
    }

    private fun loadReminders() {
        viewModelScope.launch {
            reminders.list().getOrNull()?.let { list -> _state.update { it.copy(reminders = list) } }
        }
    }

    /** Adds a doctor/medication reminder or to-do scheduled for the day being edited. */
    private fun addTask() {
        val snapshot = _state.value
        if (!snapshot.canAddTask) return
        viewModelScope.launch {
            Breadcrumbs.add("log:task:add")
            _state.update { it.copy(addingTask = true, taskError = false) }
            val draft = ReminderDraft(
                type = snapshot.newTaskType,
                title = snapshot.newTaskTitle.trim(),
                scheduledAt = snapshot.date.toIso(),
            )
            when (val result = reminders.create(draft)) {
                is AppResult.Failure -> _state.update { it.copy(addingTask = false, taskError = true) }
                is AppResult.Success -> _state.update {
                    it.copy(addingTask = false, newTaskTitle = "", reminders = it.reminders + result.value)
                }
            }
        }
    }

    private fun toggleTaskDone(reminder: Reminder) {
        viewModelScope.launch {
            Breadcrumbs.add("log:task:toggle")
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
            reminders.update(reminder.id, draft).getOrNull()?.let { updated ->
                _state.update { current ->
                    current.copy(reminders = current.reminders.map { if (it.id == updated.id) updated else it })
                }
            }
        }
    }

    private fun deleteTask(id: Long) {
        viewModelScope.launch {
            Breadcrumbs.add("log:task:delete")
            if (reminders.delete(id) is AppResult.Success) {
                _state.update { current -> current.copy(reminders = current.reminders.filterNot { it.id == id }) }
            }
        }
    }
}
