package ir.ritmeapp.ritme.domain.model

/**
 * One user-created reminder (`/reminders`). [scheduledAt] is the ISO datetime for one-shot
 * reminders; recurring ones carry [recurrence] + [recurrenceTime] (HH:mm) instead. The
 * backend owns delivery; the app only creates, edits, and lists them.
 */
data class Reminder(
    val id: Long,
    val type: ReminderType,
    val title: String,
    val subtitle: String?,
    val notes: String?,
    val scheduledAt: String?,
    val recurrence: ReminderRecurrence,
    val recurrenceTime: String?,
    val isActive: Boolean,
)

/** The reminder categories the backend accepts (`GET /reminders/enums`). */
enum class ReminderType(val apiValue: String) {
    DOCTOR("doctor"),
    MEDICATION("medication"),
    APPOINTMENT("appointment"),
    CUSTOM("custom");

    companion object {
        fun fromApi(value: String?): ReminderType = entries.firstOrNull { it.apiValue == value } ?: CUSTOM
    }
}

/** How often a reminder repeats. */
enum class ReminderRecurrence(val apiValue: String) {
    NONE("none"),
    DAILY("daily"),
    WEEKLY("weekly"),
    MONTHLY("monthly");

    companion object {
        fun fromApi(value: String?): ReminderRecurrence = entries.firstOrNull { it.apiValue == value } ?: NONE
    }
}

/**
 * The writable fields of a reminder — what create/update sends. Kept separate from [Reminder]
 * so the id/ownership stay server-assigned and a draft can exist before it has an id.
 */
data class ReminderDraft(
    val type: ReminderType,
    val title: String,
    val subtitle: String? = null,
    val notes: String? = null,
    val scheduledAt: String? = null,
    val recurrence: ReminderRecurrence = ReminderRecurrence.NONE,
    val recurrenceTime: String? = null,
    val isActive: Boolean = true,
)
