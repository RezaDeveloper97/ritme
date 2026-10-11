package checklist

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Supply reminder defaults (nbd_Shop_Beauty «۳ روز قبل از پریود بعدی»).
const (
	DefaultLeadDays = 3
	MinLeadDays     = 1
	MaxLeadDays     = 7
	// ReminderHour is the Tehran wall-clock hour of the care reminder (the user moves it in the reminder screens).
	ReminderHour = 9
	// ReminderSource marks the care reminder this opt-in owns (reminders.meta.source).
	ReminderSource = "shop_supply"
)

// Supply is the user's opt-in (a missing row is the default: off, DefaultLeadDays, no product).
type Supply struct {
	Enabled   bool
	LeadDays  int
	ProductID uint64
}

// RemindOn is the day of the supply reminder for a predicted next period start: LeadDays before it, not before
// today. ok=false when the opt-in is off, the prediction is unknown, or the predicted start is not after today
// (the period is due or late: no shopping reminder then; the next prediction brings a new one).
func RemindOn(s Supply, next, today civildate.Date) (civildate.Date, bool) {
	if !s.Enabled || next.IsZero() || !next.After(today) {
		return civildate.Date{}, false
	}
	lead := s.LeadDays
	if lead < MinLeadDays || lead > MaxLeadDays {
		lead = DefaultLeadDays
	}
	day := next.AddDays(-lead)
	if day.Before(today) {
		day = today
	}
	return day, true
}

// ReminderAt is the moment of the care reminder on day (ReminderHour, Tehran).
func ReminderAt(day civildate.Date) time.Time {
	return day.TehranMidnight().Add(ReminderHour * time.Hour)
}
