package ir.ritmeapp.ritme.adapter.inbound.ui.log

import androidx.compose.runtime.Immutable
import ir.ritmeapp.ritme.domain.model.HealthLogCategory
import ir.ritmeapp.ritme.domain.model.HealthLogField
import ir.ritmeapp.ritme.domain.model.HealthLogValue
import ir.ritmeapp.ritme.domain.model.JalaliDate
import ir.ritmeapp.ritme.domain.model.Reminder
import ir.ritmeapp.ritme.domain.model.ReminderType

/** Where auto-save currently stands (drives the header status pill: saving / saved / error). */
enum class LogSaveState { IDLE, SAVING, SAVED, ERROR }

/**
 * The daily-log screen's single immutable state (§4 MVI): the day being edited, the draft
 * values (prefilled from the server when the day already has a log), and which category
 * sheet is open.
 */
@Immutable
data class LogUiState(
    val date: JalaliDate,
    val isToday: Boolean,
    val loading: Boolean = true,
    val values: Map<HealthLogField, HealthLogValue> = emptyMap(),
    val saveState: LogSaveState = LogSaveState.IDLE,
    val openCategory: HealthLogCategory? = null,
    val reminders: List<Reminder> = emptyList(),
    val newTaskTitle: String = "",
    val newTaskType: ReminderType = ReminderType.CUSTOM,
    val addingTask: Boolean = false,
    val taskError: Boolean = false,
) {
    /** The «{n} مورد» badge per category card. */
    fun countIn(category: HealthLogCategory): Int = values.keys.count { it.category == category }

    /** Reminders/to-dos scheduled for the day being edited (one-off `scheduledAt` on [date]). */
    val dayTasks: List<Reminder>
        get() = reminders.filter { it.scheduledAt?.take(ISO_DATE_LENGTH) == date.toIso() }

    /** Done maps to an inactive reminder — the backend has no separate completed flag. */
    val doneTaskCount: Int get() = dayTasks.count { !it.isActive }

    val canAddTask: Boolean get() = newTaskTitle.isNotBlank() && !addingTask

    private companion object {
        const val ISO_DATE_LENGTH = 10
    }
}

/** Everything the user can do on the log screen (single `onIntent` entry point, §4). */
sealed interface LogIntent {
    data object PreviousDay : LogIntent
    data object NextDay : LogIntent
    data class OpenCategory(val category: HealthLogCategory) : LogIntent
    data object CloseSheet : LogIntent

    /** Sets or clears (null) one field's value in the draft; each edit auto-saves (debounced). */
    data class SetValue(val field: HealthLogField, val value: HealthLogValue?) : LogIntent

    // ── Day planner: doctor/medication reminders + to-dos for the edited day ──
    data class NewTaskTitleChanged(val title: String) : LogIntent
    data class NewTaskTypeChanged(val type: ReminderType) : LogIntent
    data object AddTask : LogIntent
    data class ToggleTaskDone(val reminder: Reminder) : LogIntent
    data class DeleteTask(val id: Long) : LogIntent
}
