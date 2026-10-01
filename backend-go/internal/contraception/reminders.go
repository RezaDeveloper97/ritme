package contraception

import (
	"encoding/json"
	"time"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Reminder kinds a method creates (contraception_reminders.kind), in display order.
const (
	KindPillRefill         = "pill_refill"
	KindIUDStringCheck     = "iud_string_check"
	KindIUDFollowup        = "iud_followup"
	KindIUDReplacement     = "iud_replacement"
	KindInjectionNext      = "injection_next"
	KindImplantReplacement = "implant_replacement"
)

// ReminderKinds is every kind in display order.
var ReminderKinds = []string{
	KindPillRefill, KindIUDStringCheck, KindIUDFollowup, KindIUDReplacement, KindInjectionNext, KindImplantReplacement,
}

// Reminder types in `reminders` (enums.ReminderType values): visits are care appointments, the rest custom.
const (
	TypeAppointment = care.TypeAppointment
	TypeCustom      = "custom"
)

// ReminderHour is the Tehran wall-clock hour the method's reminders are set to (appointments at 09:00, reminded
// a day before; the monthly string check at 09:00). The user moves them in the care screens.
const ReminderHour = 9

// Spec is one reminder a method wants.
type Spec struct {
	Kind string
	Type string // TypeAppointment | TypeCustom
	Due  civildate.Date
	// Monthly is a repeating custom reminder (recurrence monthly from Due); otherwise one-off at Due.
	Monthly bool
	// Topic is the care appointment topic (appointments only).
	Topic string
}

// addMonths moves d by n calendar months (Go normalisation: 31 Jan + 1 month = 3 Mar).
func addMonths(d civildate.Date, months int) civildate.Date {
	return civildate.FromTime(time.Date(d.Year, d.Month+time.Month(months), d.Day, 0, 0, 0, 0, time.UTC))
}

// Plan is the reminders a saved method wants as of today, in display order.
func Plan(m Method, today civildate.Date) []Spec {
	var out []Spec
	switch {
	case IsPill(m.Method):
		if r := m.Refill(today); r.Known {
			due := r.RefillOn
			if due.Before(today) {
				due = today
			}
			out = append(out, Spec{Kind: KindPillRefill, Type: TypeCustom, Due: due})
		}
	case IsIUD(m.Method):
		if m.InsertedOn.IsZero() {
			return nil
		}
		out = append(out, Spec{Kind: KindIUDStringCheck, Type: TypeCustom, Due: addMonths(m.InsertedOn, 1), Monthly: true})
		if !m.FollowupDone {
			out = append(out, Spec{Kind: KindIUDFollowup, Type: TypeAppointment, Due: m.FollowupOn(), Topic: "checkup"})
		}
		if due := m.IUDReplaceOn(); !due.IsZero() {
			out = append(out, Spec{Kind: KindIUDReplacement, Type: TypeAppointment, Due: due, Topic: "checkup"})
		}
	case m.Method == MethodInjection:
		if due := m.NextInjectionOn(); !due.IsZero() {
			out = append(out, Spec{Kind: KindInjectionNext, Type: TypeAppointment, Due: due, Topic: "other"})
		}
	case m.Method == MethodImplant:
		if !m.ReplaceOn.IsZero() {
			out = append(out, Spec{Kind: KindImplantReplacement, Type: TypeAppointment, Due: m.ReplaceOn, Topic: "checkup"})
		}
	}
	return out
}

// At is the moment the spec fires: Due at ReminderHour, Tehran.
func (s Spec) At() time.Time {
	return s.Due.TehranMidnight().Add(ReminderHour * time.Hour)
}

// Columns are the `reminders` columns a spec writes.
type Columns struct {
	ScheduledAt    *time.Time
	Recurrence     string
	RecurrenceTime *string
	StartsOn       *civildate.Date
	Meta           json.RawMessage // appointments only, on insert
}

// Columns maps the spec onto the reminders row: an appointment is one-off at At() with a v1 appointment meta
// (in person, remind 1 day before, no prep); a monthly custom reminder recurs from Due at ReminderHour; a one-off
// custom reminder is scheduled at At().
func (s Spec) Columns() Columns {
	at := s.At()
	if s.Monthly {
		t := time.Date(0, 1, 1, ReminderHour, 0, 0, 0, time.UTC).Format(time.TimeOnly)
		due := s.Due
		return Columns{Recurrence: "monthly", RecurrenceTime: &t, StartsOn: &due}
	}
	c := Columns{ScheduledAt: &at, Recurrence: "none"}
	if s.Type == TypeAppointment {
		meta, _ := json.Marshal(care.AppointmentMeta{ // a struct of plain fields always marshals
			V: care.MetaVersion, Kind: "in_person", Topic: s.Topic, RemindBefore: care.DefaultRemindBefore,
			Prep: []care.PrepItem{}, Status: care.StatusScheduled,
		})
		c.Meta = meta
	}
	return c
}
