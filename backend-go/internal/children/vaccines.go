package children

import (
	"github.com/ritme/backend-go/internal/children/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Vaccine schedule windows.
const (
	// RemindDaysBefore is when a visit reminder fires («یادآور واکسن … ۳ روز قبل از هر مراجعه»).
	RemindDaysBefore = 3
	// SoonDays: an upcoming visit within this many days is «به‌زودی».
	SoonDays = 30
	// OverdueAfterDays: a visit past due by more than this many days is overdue (until then just due).
	OverdueAfterDays = 30
)

// Visit / dose statuses.
const (
	StatusDone     = "done"
	StatusDue      = "due"
	StatusOverdue  = "overdue"
	StatusSoon     = "soon"
	StatusUpcoming = "upcoming"
)

// Dose is one scheduled dose and, when given, its record.
type Dose struct {
	Entry
	Given *store.ChildVaccineDose
}

// Visit is the doses of one schedule age (بدو تولد, ۲ ماهگی, …) with the due date from the birth date.
type Visit struct {
	Code      string // birth|m2|…
	AgeMonths int
	DueDate   civildate.Date
	RemindOn  civildate.Date
	Doses     []Dose
	Status    string
	Given     int
}

// Schedule is a child's vaccine schedule on a day.
type Schedule struct {
	Visits          []Visit
	GivenDoses      int
	TotalDoses      int
	CompletedVisits int
	Next            *Visit // first visit not completed; nil when everything is given
	UpToDate        bool   // no due or overdue visit
}

// BuildSchedule groups the catalog doses into visits (catalog order; a visit sits where its first dose is), dates
// them from birth and marks what the child got.
func BuildSchedule(es []Entry, given []store.ChildVaccineDose, birth, today civildate.Date) Schedule {
	byCode := make(map[string]*store.ChildVaccineDose, len(given))
	for i := range given {
		byCode[given[i].DoseCode] = &given[i]
	}
	var visits []Visit
	index := map[string]int{}
	for _, e := range es {
		i, ok := index[e.Visit]
		if !ok {
			due := AddMonths(birth, e.AgeMonths)
			visits = append(visits, Visit{Code: e.Visit, AgeMonths: e.AgeMonths, DueDate: due, RemindOn: due.AddDays(-RemindDaysBefore)})
			i = len(visits) - 1
			index[e.Visit] = i
		}
		visits[i].Doses = append(visits[i].Doses, Dose{Entry: e, Given: byCode[e.Code]})
	}
	s := Schedule{UpToDate: true}
	for i := range visits {
		v := &visits[i]
		for _, d := range v.Doses {
			if d.Given != nil {
				v.Given++
			}
		}
		v.Status = visitStatus(v, today)
		s.GivenDoses += v.Given
		s.TotalDoses += len(v.Doses)
		if v.Status == StatusDone {
			s.CompletedVisits++
		}
		if v.Status == StatusDue || v.Status == StatusOverdue {
			s.UpToDate = false
		}
	}
	s.Visits = visits
	for i := range s.Visits {
		if s.Visits[i].Status != StatusDone {
			s.Next = &s.Visits[i]
			break
		}
	}
	return s
}

func visitStatus(v *Visit, today civildate.Date) string {
	switch {
	case v.Given == len(v.Doses):
		return StatusDone
	case v.DueDate.After(today):
		if today.DiffDays(v.DueDate) <= SoonDays {
			return StatusSoon
		}
		return StatusUpcoming
	case v.DueDate.DiffDays(today) > OverdueAfterDays:
		return StatusOverdue
	}
	return StatusDue
}

// DoseStatus is done for a given dose, else its visit's status.
func (v Visit) DoseStatus(d Dose) string {
	if d.Given != nil {
		return StatusDone
	}
	return v.Status
}

// VisitLabel is «بدو تولد», «۲ ماهگی», «۶ سالگی».
func VisitLabel(ageMonths int, locale string) string {
	switch {
	case ageMonths == 0:
		return T("visits.birth", locale)
	case ageMonths >= 24 && ageMonths%12 == 0:
		return Tp("visits.years", map[string]string{"n": num(ageMonths/12, locale)}, locale)
	}
	return Tp("visits.months", map[string]string{"n": num(ageMonths, locale)}, locale)
}

// Reminder is a visit whose reminder is due: from RemindOn until OverdueAfterDays after the due date, not completed.
type Reminder struct {
	Visit Visit
	Days  int // days until the due date (negative when past)
}

// DueReminders are the schedule's visits to remind about on today.
func DueReminders(s Schedule, today civildate.Date) []Reminder {
	var out []Reminder
	for _, v := range s.Visits {
		if v.Status == StatusDone || v.RemindOn.After(today) || v.DueDate.DiffDays(today) > OverdueAfterDays {
			continue
		}
		out = append(out, Reminder{Visit: v, Days: today.DiffDays(v.DueDate)})
	}
	return out
}
