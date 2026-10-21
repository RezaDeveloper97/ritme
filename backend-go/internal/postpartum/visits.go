package postpartum

import (
	"slices"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/postpartum/guide"
)

// Postpartum visits (T-M9-06, D-135; SPEC v6 §3.1, ACOG CO 736): a first contact by contact_by_day, the in-app
// reminder of the postpartum check-up from reminder_day and the comprehensive visit by comprehensive_by_day, all in
// days after the birth (clinical config postpartum.visits, version 1 = 21 / 42 / 84).
//
// A visit is done when a day she logged as «I had my postpartum visit» (postpartum_recovery_logs.postpartum_visit,
// PUT /postpartum/recovery) falls in its window: the contact window is the birth day to contact_by_day; the 6-week
// and comprehensive visits share the window after it, open-ended (a late comprehensive visit is still that visit).
// Care appointments carry no postpartum tag, so they do not count (open item for the advisor sheet / T-M10).

// Visit kinds, in display order.
const (
	VisitContact       = "contact"
	VisitReminder      = "reminder"
	VisitComprehensive = "comprehensive"
)

// VisitKinds are the kinds in display order.
var VisitKinds = []string{VisitContact, VisitReminder, VisitComprehensive}

// Visit statuses.
const (
	VisitDone     = "done"
	VisitUpcoming = "upcoming" // its day is still ahead
	VisitDue      = "due"      // the window is open and its day has come
	VisitOverdue  = "overdue"  // past its last day without a logged visit
)

// VisitDays are the visit days after the birth valid on one day.
type VisitDays struct{ ContactBy, Reminder, ComprehensiveBy int }

// VisitDaysOn reads postpartum.visits valid on d (a zero d = version 1).
func VisitDaysOn(d civildate.Date) VisitDays {
	return VisitDays{
		ContactBy:       clinicalconfig.Int(clinicalconfig.KeyPostpartumVisits, "contact_by_day", d),
		Reminder:        clinicalconfig.Int(clinicalconfig.KeyPostpartumVisits, "reminder_day", d),
		ComprehensiveBy: clinicalconfig.Int(clinicalconfig.KeyPostpartumVisits, "comprehensive_by_day", d),
	}
}

// Visit is one postpartum visit of a birth.
type Visit struct {
	Kind   string
	DueOn  civildate.Date
	Done   bool
	DoneOn *civildate.Date // the first logged visit day in the window
	Status string
}

// VisitsOn are the three visits of the birth on birth as seen on today, with the config valid on today. visited
// are the days she logged a postpartum visit (any order; days before the birth are ignored).
func VisitsOn(birth, today civildate.Date, visited []civildate.Date) []Visit {
	days := VisitDaysOn(today)
	contactEnd := birth.AddDays(days.ContactBy)
	first := func(from civildate.Date, until *civildate.Date) *civildate.Date {
		var best *civildate.Date
		for _, d := range visited {
			if d.Before(from) || (until != nil && d.After(*until)) {
				continue
			}
			if best == nil || d.Before(*best) {
				v := d
				best = &v
			}
		}
		return best
	}
	contactDone := first(birth, &contactEnd)
	laterDone := first(contactEnd.AddDays(1), nil)

	reminderOn := birth.AddDays(days.Reminder)
	compBy := birth.AddDays(days.ComprehensiveBy)
	out := make([]Visit, 0, len(VisitKinds))
	add := func(kind string, due civildate.Date, done *civildate.Date, status string) {
		v := Visit{Kind: kind, DueOn: due, DoneOn: done, Done: done != nil, Status: status}
		if v.Done {
			v.Status = VisitDone
		}
		out = append(out, v)
	}
	contact := VisitUpcoming
	if today.After(contactEnd) {
		contact = VisitOverdue
	}
	add(VisitContact, contactEnd, contactDone, contact)
	reminder := VisitUpcoming
	if !today.Before(reminderOn) {
		reminder = VisitDue
	}
	add(VisitReminder, reminderOn, laterDone, reminder)
	comp := VisitUpcoming
	switch {
	case today.After(compBy):
		comp = VisitOverdue
	case !today.Before(reminderOn):
		comp = VisitDue
	}
	add(VisitComprehensive, compBy, laterDone, comp)
	return out
}

// NoticeKinds are the visits that get the lazy in-app notice on today: the 6-week reminder from reminder_day and the
// comprehensive visit once past its last day, each only while no visit is logged (the notice itself is written once
// per kind and birth by Service.EnsureVisitNotices).
func NoticeKinds(visits []Visit) []string {
	var out []string
	for _, v := range visits {
		switch {
		case v.Done:
		case v.Kind == VisitReminder && v.Status == VisitDue:
			out = append(out, v.Kind)
		case v.Kind == VisitComprehensive && v.Status == VisitOverdue:
			out = append(out, v.Kind)
		}
	}
	return out
}

// IsUrgentAlert reports whether an alert key is an urgent red flag (call 115 now).
func IsUrgentAlert(key string) bool { return slices.Contains(guide.RedFlagAlerts, key) }
