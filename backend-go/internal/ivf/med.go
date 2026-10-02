package ivf

import (
	"cmp"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/care"
	carestore "github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/ivf/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Med is an IVF medicine: the care medication reminder it is plus the IVF fields.
type Med struct {
	ID        uint64
	CycleID   uint64
	Role      string
	Route     string
	TriggerAt time.Time // trigger only
	Stock     *Stock    // nil = inventory not tracked
	Care      care.Medication
}

// Stock is the inventory as counted: Units of Unit at CountedAt, each lasting DosesPerUnit doses.
type Stock struct {
	Units        int
	Unit         string
	DosesPerUnit int
	CountedAt    time.Time
}

func medFrom(m store.IvfMed, r store.Reminder) Med {
	out := Med{
		ID: m.ID, CycleID: m.CycleID, Role: m.Role, Route: m.Route, TriggerAt: nullTime(m.TriggerAt),
		Care: care.ParseMedication(carestore.Reminder(r)),
	}
	if m.StockUnits.Valid {
		out.Stock = &Stock{
			Units: int(m.StockUnits.Int16), Unit: m.StockUnit.String, DosesPerUnit: max(int(m.DosesPerUnit), 1),
			CountedAt: nullTime(m.StockCountedAt),
		}
	}
	return out
}

// IsTrigger reports whether the medicine is the trigger shot.
func (m Med) IsTrigger() bool { return m.Role == RoleTrigger }

// ReminderID is the care reminder the medicine is.
func (m Med) ReminderID() uint64 { return m.Care.Row.ID }

// Name is the medicine's name (the care reminder title).
func (m Med) Name() string { return m.Care.Row.Title }

// DosesPerDay is how many doses a day the medicine is scheduled on day (0 when inactive or not on day).
func (m Med) DosesPerDay(day civildate.Date) int {
	if !m.Care.Row.IsActive || !m.Care.Covers(day) {
		return 0
	}
	return len(m.Care.Meta.Times)
}

// Inventory is the stock as of today.
type Inventory struct {
	DosesLeft int
	UnitsLeft int
	DaysLeft  int // valid when Scheduled
	RunsOutOn civildate.Date
	Scheduled bool // the medicine has doses today, so days of supply are known
	Low       bool
}

// Inventory computes the stock left after used doses (care intakes since the count): doses and units left, days of
// supply at today's doses per day, the first day short, and low = it lasts LowSupplyDays or fewer and runs out
// before the medicine's end date (or it is empty).
func (m Med) Inventory(used int, today civildate.Date) Inventory {
	s := m.Stock
	if s == nil {
		return Inventory{}
	}
	left := max(s.Units*s.DosesPerUnit-used, 0)
	inv := Inventory{DosesLeft: left, UnitsLeft: (left + s.DosesPerUnit - 1) / s.DosesPerUnit}
	perDay := m.DosesPerDay(today)
	if perDay == 0 {
		inv.Low = left == 0 && m.Care.Row.IsActive
		return inv
	}
	inv.Scheduled = true
	inv.DaysLeft = left / perDay
	inv.RunsOutOn = today.AddDays(inv.DaysLeft)
	needed := -1 // unknown: no end date
	if m.Care.Row.EndsOn.Valid && !m.Care.Row.EndsOn.Date.Before(today) {
		needed = (today.DiffDays(m.Care.Row.EndsOn.Date) + 1) * perDay
	}
	inv.Low = inv.DaysLeft <= LowSupplyDays && (needed < 0 || left < needed)
	return inv
}

// Dose is one scheduled dose of a day.
type Dose struct {
	Med   Med
	Date  civildate.Date
	Slot  string
	Taken bool
	Site  string
}

// doseKey identifies a dose of a medicine reminder.
type doseKey struct {
	reminderID uint64
	date       civildate.Date
	slot       string
}

// DosesOn lists the doses of day across meds (active, scheduled on day), sorted by time then medicine; taken from
// the care intakes, site from the IVF dose logs.
func DosesOn(day civildate.Date, meds []Med, taken map[doseKey]bool, sites map[doseKey]string) []Dose {
	out := []Dose{}
	for _, m := range meds {
		if m.DosesPerDay(day) == 0 {
			continue
		}
		for _, slot := range m.Care.Meta.Times {
			k := doseKey{m.ReminderID(), day, slot}
			out = append(out, Dose{Med: m, Date: day, Slot: slot, Taken: taken[k], Site: sites[k]})
		}
	}
	slices.SortStableFunc(out, func(a, b Dose) int {
		return cmp.Or(cmp.Compare(a.Slot, b.Slot), cmp.Compare(a.Med.ID, b.Med.ID))
	})
	return out
}

// SiteUse is one logged injection site.
type SiteUse struct {
	Site string
	Date civildate.Date
	Slot string
}

// Rotate picks, over the active sites in rotation order (catalog order), the last site used (uses are newest
// first; only active sites count) and the suggested next one: a site never used yet in rotation order, else the
// least recently used. suggested is "" without sites.
func Rotate(codes []string, uses []SiteUse) (last *SiteUse, suggested string) {
	lastUse := map[string]int{} // site → index in uses (0 = newest)
	for i, u := range uses {
		if !slices.Contains(codes, u.Site) {
			continue
		}
		if _, seen := lastUse[u.Site]; !seen {
			lastUse[u.Site] = i
		}
		if last == nil {
			use := u
			last = &use
		}
	}
	oldest := -1
	for _, c := range codes {
		i, used := lastUse[c]
		if !used {
			return last, c
		}
		if i > oldest {
			oldest, suggested = i, c
		}
	}
	return last, suggested
}
