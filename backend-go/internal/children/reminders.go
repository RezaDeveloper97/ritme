package children

import (
	"sort"
	"time"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Vaccine reminders («یادآور واکسن و پایش رشد · ۳ روز قبل از هر مراجعه»).
//
// There is no push sender in the API yet (see internal/reminders): GET /children/reminders is the in-app due list,
// and a sender (cron / web-push worker) asks PlanReminders for the pushes of a day. A visit's reminder fires once, on
// RemindOn (due date − RemindDaysBefore), under the notification category checkups, so a switched-off category,
// quiet hours and the category's time are honoured by notifications.Decide, and the copy goes through
// notifications.Render («متن خنثی» hides the child's name and vaccine on the lock screen).

// ReminderCategory is the notification category vaccine reminders use.
const ReminderCategory = notifications.Checkups

// Occurrence is one vaccine reminder due on the planned day.
type Occurrence struct {
	ChildID  uint64
	Visit    Visit
	At       time.Time // Tehran wall-clock
	Decision notifications.Decision
	Message  notifications.Message
}

// PlanReminders lists the reminders due on day for one child's schedule, each gated by notifications.Decide at the
// category's time. Switched-off categories are still listed (Decision.Reason = category_off).
func PlanReminders(p notifications.Preferences, childID uint64, childName string, s Schedule, day civildate.Date, locale string) []Occurrence {
	var out []Occurrence
	minute, _ := p.TimeOf(ReminderCategory)
	at := day.TehranMidnight().Add(time.Duration(minute) * time.Minute)
	for _, v := range s.Visits {
		if v.Status == StatusDone || v.RemindOn != day {
			continue
		}
		params := map[string]string{
			"name":  childName,
			"visit": VisitLabel(v.AgeMonths, locale),
			"days":  num(RemindDaysBefore, locale),
			"date":  Digits(v.DueDate.String(), locale),
		}
		out = append(out, Occurrence{
			ChildID: childID, Visit: v, At: at,
			Decision: notifications.Decide(p, ReminderCategory, at, time.Time{}),
			Message: notifications.Message{
				Category: ReminderCategory,
				Title:    Tp("reminder.title", params, locale),
				Body:     Tp("reminder.body", params, locale),
				URL:      "/children",
			},
		})
	}
	return out
}

// sortReminders orders the due list soonest first (most overdue at the top).
func sortReminders(items []*jsonx.OrderedMap) {
	sort.SliceStable(items, func(i, j int) bool {
		a, _ := items[i].Get("days_left")
		b, _ := items[j].Get("days_left")
		ai, _ := a.(int)
		bi, _ := b.(int)
		return ai < bi
	})
}
