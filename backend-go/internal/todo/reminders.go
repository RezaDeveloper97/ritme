package todo

import (
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// ReminderCategory is the notification category of task reminders: a reminder the user set on a dated, timed task
// («یادآور» in Todo_Add) is a personal appointment, so it follows the «قرارها» switch and the quiet hours.
const ReminderCategory = notifications.Appointments

// Occurrence is one task reminder due on a day.
type Occurrence struct {
	Task     Task
	At       time.Time // Tehran wall-clock
	Decision notifications.Decision
	Message  notifications.Message
}

// minuteOf parses HH:MM.
func minuteOf(hhmm string) (int, bool) {
	if len(hhmm) != 5 || hhmm[2] != ':' {
		return 0, false
	}
	h, err1 := strconv.Atoi(hhmm[:2])
	m, err2 := strconv.Atoi(hhmm[3:])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// PlanReminders lists the reminders of day: open tasks with remind on, due that day at a time. Each goes through
// notifications.Decide (switched-off category, quiet hours) and its copy is rendered later through
// notifications.Render, so «متن خنثی» hides the title on the lock screen. There is no push sender in the API yet: a
// sender asks this for the pushes of a day, like vitals.PlanReminders and reminders.Plan.
func PlanReminders(p notifications.Preferences, tasks []Task, day civildate.Date, locale string) []Occurrence {
	var out []Occurrence
	for _, t := range tasks {
		minute, ok := minuteOf(t.DueTime)
		if !t.Remind || t.Done() || t.DueDate != day || !ok {
			continue
		}
		at := day.TehranMidnight().Add(time.Duration(minute) * time.Minute)
		out = append(out, Occurrence{
			Task: t, At: at,
			Decision: notifications.Decide(p, ReminderCategory, at, time.Time{}),
			Message: notifications.Message{
				Category: ReminderCategory,
				Title:    T("reminder.title", locale),
				Body:     Tp("reminder.body", map[string]string{"title": t.Title}, locale),
				URL:      "/todo",
			},
		})
	}
	return out
}
