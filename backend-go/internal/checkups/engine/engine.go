// Package engine is the periodic-checkups status engine (docs/checkups/README.md, T-M4-01): a pure,
// clock-injected function from (catalog types, the user's records and settings, birthday, cycle
// prediction, today) to one item per applicable checkup — status, section, next due — plus the
// summary behind the home ring. No DB, no HTTP, no labels (T-M4-02 renders those).
//
// Rules, per applicable type:
//
//   - last = the latest record's done_on (ties → the higher id); that record's next_due_on, when
//     set, is a user override and wins over everything below.
//   - next due = last + interval_months (calendar months, clamped to the month's last day).
//     A range interval (mammography 12–24) is due from the minimum and overdue only after the
//     maximum. A monthly cycle-timed type (self-exam, cycle days 7–10) is due at the window of the
//     first predicted cycle that starts after `last` and overdue after that window; without cycle
//     data it falls back to last + 1 month.
//   - status: not_yet (age below age_min; next due = the day the user reaches it) · due (never
//     recorded — except a monthly cycle-timed type placed on its upcoming window, which follows the
//     lead days like a recorded one — or today ≥ next due − remind_lead_days) · overdue (today after the due-by date) ·
//     soon (within 60 days) · up_to_date · disabled (the user switched it off; kept in the list,
//     left out of the summary).
//   - section: this_month for a cycle-timed type that is due (a never-recorded monthly one also
//     soon) with its next due inside the current Jalali month, or a monthly cycle-timed type (the
//     self-exam) that is overdue; else overdue for any other overdue item (a cycle-timed Pap smear
//     included); else the type's category.
//   - summary: total = every enabled item; up_to_date counts up_to_date + soon + not_yet (the
//     artboard ring «۴ از ۶» counts the not-yet mammography as fine); due; overdue.
//
// Types are skipped (no item) when inactive, hidden in pregnancy while the user is pregnant, or
// when the user is older than age_max.
package engine

import (
	"cmp"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Category is checkup_types.category.
type Category string

// Categories.
const (
	CategoryMonthly    Category = "monthly"
	CategorySixMonthly Category = "six_monthly"
	CategoryAnnual     Category = "annual"
	CategoryMultiYear  Category = "multi_year"
	CategoryAgeBased   Category = "age_based"
	CategoryCustom     Category = "custom"
)

// Status is an item's status.
type Status string

// Statuses.
const (
	StatusOverdue  Status = "overdue"
	StatusDue      Status = "due"
	StatusSoon     Status = "soon"
	StatusUpToDate Status = "up_to_date"
	StatusNotYet   Status = "not_yet"
	StatusDisabled Status = "disabled"
)

// Section is the list section an item is shown in: this_month, overdue, or a Category value.
type Section string

// Sections that are not categories.
const (
	SectionThisMonth Section = "this_month"
	SectionOverdue   Section = "overdue"
)

// SoonDays is how far ahead a next due date turns an up-to-date item into `soon`.
const SoonDays = 60

// Type is the part of a checkup_types row the engine reads. Zero ints mean NULL.
type Type struct {
	ID       uint64
	Key      string // "" for a custom checkup
	UserID   uint64 // 0 for the shared catalog
	Category Category
	// IntervalMonths is the (minimum) interval; IntervalMonthsMax > 0 makes it a range.
	IntervalMonths    int
	IntervalMonthsMax int
	AgeMin, AgeMax    int
	// CycleDayFrom/CycleDayTo: the best cycle days (1 = first day of the period).
	CycleDayFrom, CycleDayTo int
	RemindLeadDays           int
	HideInPregnancy          bool
	IsActive                 bool
	SortOrder                int
}

// CycleTimed reports whether the type has a cycle-day window.
func (t Type) CycleTimed() bool { return t.CycleDayFrom > 0 && t.CycleDayTo >= t.CycleDayFrom }

// Record is the part of a checkup_records row the engine reads.
type Record struct {
	ID     uint64
	TypeID uint64
	DoneOn civildate.Date
	// NextDueOn is the user's override; the zero Date is NULL.
	NextDueOn civildate.Date
}

// Setting is a user_checkup_settings row. A type without a row is enabled with reminders on.
type Setting struct {
	TypeID  uint64
	Enabled bool
	Remind  bool
}

// Input is everything one evaluation reads.
type Input struct {
	Types    []Type
	Records  []Record
	Settings []Setting
	// Birthday is the profile birthday; the zero Date means unknown (age rules are not applied).
	Birthday civildate.Date
	// Pregnant hides the types marked hide_in_pregnancy.
	Pregnant bool
	// Cycle is the cycle prediction (see CycleFromHistory); the zero value means no cycle data.
	Cycle Cycle
}

// Item is the evaluated state of one checkup.
type Item struct {
	TypeID   uint64
	Key      string
	Category Category
	IsCustom bool
	Status   Status
	Section  Section
	// LastDoneOn is the latest record's done_on (zero = never recorded).
	LastDoneOn civildate.Date
	// NextDueOn is when the checkup is next due (zero when never recorded and not cycle-timed);
	// for not_yet it is the day the user reaches age_min.
	NextDueOn civildate.Date
	// DueBy is the last day before the item turns overdue (the range maximum, the end of the
	// cycle window, else NextDueOn).
	DueBy civildate.Date
	// NextDueOverridden: NextDueOn is the user's own next_due_on.
	NextDueOverridden bool
	// CycleTimed: NextDueOn was placed on a predicted cycle window.
	CycleTimed bool
	Enabled    bool
	Remind     bool
}

// Summary is the ring and counts line (disabled items excluded).
type Summary struct {
	Total    int
	UpToDate int
	Due      int
	Overdue  int
}

// Result is one evaluation.
type Result struct {
	// Age in completed years; nil when the birthday is unknown.
	Age     *int
	Items   []Item
	Summary Summary
}

// Evaluate runs the engine for the day `now` reports in Tehran.
func Evaluate(in Input, now civildate.Nower) Result {
	today := civildate.Today(now)
	res := Result{Items: []Item{}}
	var age int
	if !in.Birthday.IsZero() {
		age = in.Birthday.AgeOn(today)
		res.Age = &age
	}

	latest := latestRecords(in.Records)
	settings := make(map[uint64]Setting, len(in.Settings))
	for _, s := range in.Settings {
		settings[s.TypeID] = s
	}

	types := slices.Clone(in.Types)
	slices.SortStableFunc(types, func(a, b Type) int {
		return cmp.Or(cmp.Compare(a.SortOrder, b.SortOrder), cmp.Compare(a.ID, b.ID))
	})
	monthEnd := JalaliMonthEnd(today)

	for _, t := range types {
		if !t.IsActive || (in.Pregnant && t.HideInPregnancy) {
			continue
		}
		if res.Age != nil && t.AgeMax > 0 && age > t.AgeMax {
			continue
		}
		item := evaluateType(t, latest[t.ID], in, res.Age, today)
		if s, ok := settings[t.ID]; ok {
			item.Enabled, item.Remind = s.Enabled, s.Remind
		}
		if !item.Enabled {
			item.Status = StatusDisabled
		}
		item.Section = section(t, item, monthEnd)
		res.Items = append(res.Items, item)
		res.Summary.add(item.Status)
	}
	return res
}

func (s *Summary) add(st Status) {
	switch st {
	case StatusDisabled:
		return
	case StatusDue:
		s.Due++
	case StatusOverdue:
		s.Overdue++
	default: // up_to_date, soon, not_yet
		s.UpToDate++
	}
	s.Total++
}

// latestRecords picks, per type, the record with the latest done_on (ties → higher id).
func latestRecords(records []Record) map[uint64]Record {
	out := make(map[uint64]Record, len(records))
	for _, r := range records {
		cur, ok := out[r.TypeID]
		if !ok || r.DoneOn.After(cur.DoneOn) || (r.DoneOn == cur.DoneOn && r.ID > cur.ID) {
			out[r.TypeID] = r
		}
	}
	return out
}

func evaluateType(t Type, last Record, in Input, age *int, today civildate.Date) Item {
	item := Item{
		TypeID:     t.ID,
		Key:        t.Key,
		Category:   t.Category,
		IsCustom:   t.UserID != 0,
		LastDoneOn: last.DoneOn,
		Enabled:    true,
		Remind:     true,
	}

	if age != nil && t.AgeMin > 0 && *age < t.AgeMin {
		item.Status = StatusNotYet
		item.NextDueOn = addYears(in.Birthday, t.AgeMin)
		item.DueBy = item.NextDueOn
		return item
	}

	switch {
	case !last.NextDueOn.IsZero():
		item.NextDueOn, item.DueBy, item.NextDueOverridden = last.NextDueOn, last.NextDueOn, true
	case last.DoneOn.IsZero():
		// Never recorded: due now; a cycle-timed type still gets its upcoming window.
		if w, ok := upcomingWindow(t, in.Cycle, today); ok {
			item.NextDueOn, item.DueBy, item.CycleTimed = w.from, w.to, true
			if t.Category == CategoryMonthly {
				// A monthly habit (the self-exam) is not "due" weeks before its window: that read
				// «موعدش رسیده» next to «۱۸ روز دیگر» (audit 5d). It follows the lead days instead.
				item.Status = status(t, item, today)
				return item
			}
		}
		item.Status = StatusDue
		return item
	default:
		d := NextDueAfter(t, last.DoneOn, in.Cycle)
		item.NextDueOn, item.DueBy, item.CycleTimed = d.From, d.By, d.CycleTimed
	}
	item.Status = status(t, item, today)
	return item
}

func status(t Type, item Item, today civildate.Date) Status {
	switch {
	case today.After(item.DueBy):
		return StatusOverdue
	case !today.Before(item.NextDueOn.AddDays(-t.RemindLeadDays)):
		return StatusDue
	case !today.Before(item.NextDueOn.AddDays(-SoonDays)):
		return StatusSoon
	default:
		return StatusUpToDate
	}
}

func section(t Type, item Item, monthEnd civildate.Date) Section {
	switch {
	case item.Status == StatusDisabled:
		return Section(t.Category)
	case t.CycleTimed() && item.Status == StatusOverdue && t.Category == CategoryMonthly:
		// A missed monthly window (the self-exam) is this month's miss, not a long-overdue visit.
		return SectionThisMonth
	case t.CycleTimed() && (item.Status == StatusDue || neverDoneMonthly(t, item)) &&
		(item.NextDueOn.IsZero() || !item.NextDueOn.After(monthEnd)):
		return SectionThisMonth
	case item.Status == StatusOverdue:
		return SectionOverdue
	default:
		return Section(t.Category)
	}
}

// neverDoneMonthly: a never-recorded monthly cycle-timed type (the self-exam) whose window is
// still ahead — `soon` since audit 5d, but it keeps its place in «این ماه» like the `due` it was.
func neverDoneMonthly(t Type, item Item) bool {
	return t.Category == CategoryMonthly && item.LastDoneOn.IsZero() && item.Status == StatusSoon
}

// Due is the computed due window of a checkup done on a given day.
type Due struct {
	// From is the next due date; By is the last day before it turns overdue.
	From, By   civildate.Date
	CycleTimed bool
}

// NextDueAfter computes when a checkup done on doneOn is next due (the MarkDone preview). It
// ignores user overrides.
func NextDueAfter(t Type, doneOn civildate.Date, c Cycle) Due {
	if w, ok := windowAfter(t, c, doneOn); ok {
		return Due{From: w.from, By: w.to, CycleTimed: true}
	}
	from := AddMonths(doneOn, max(1, t.IntervalMonths))
	by := from
	if t.IntervalMonthsMax > t.IntervalMonths {
		by = AddMonths(doneOn, t.IntervalMonthsMax)
	}
	return Due{From: from, By: by}
}

// AddMonths adds n calendar months, clamping the day to the target month's length
// (Carbon addMonthsNoOverflow: 31 Jan + 1 month = 28/29 Feb).
func AddMonths(d civildate.Date, n int) civildate.Date {
	m := int(d.Month) - 1 + n
	y := d.Year + m/12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	month := time.Month(m + 1)
	return civildate.New(y, month, min(d.Day, daysIn(y, month)))
}

func addYears(d civildate.Date, n int) civildate.Date { return AddMonths(d, 12*n) }

func daysIn(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
