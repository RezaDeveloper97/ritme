// Package v2 is the pregnancy v2 read API (docs/pregnancy-v2/README.md): dating preview, the
// Today screen, the week page and the per-week user state under /api/v1/pregnancy/v2. The
// dating math is pregnancy/calc; admin-authored content comes from pregnancy_week_details and
// message_contents (pregnancy_week_tip, pregnancy_setup).
package v2

import (
	"math"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

const (
	// TermWeeks is the length of the progress bar (40 weeks = 280 days).
	TermWeeks = 40
	// MaxWeek is the last week the v2 screens show (overdue pregnancies keep counting to 42).
	MaxWeek = 42
	// birthRangeDays is the half-width of the "usual birth range" around the due date before the
	// dating uncertainty is added (most babies are born within ±2 weeks of it).
	birthRangeDays = 14
)

// trimesterStartWeeks are the completed weeks at which trimesters 2 and 3 begin (calc.Trimester).
var trimesterStartWeeks = [3]int{0, 13, 28}

// Dating is the resolved gestational age of a profile on a given day.
type Dating struct {
	Source      string
	Weeks       int // completed weeks
	Days        int // 0..6
	TotalDays   int
	Confidence  string
	Uncertainty int
	Due         civildate.Date
	Today       civildate.Date
	BasisDate   civildate.Date // LMP, ultrasound or manual entry date
}

// Resolve dates p on today; ok is false when the profile lacks dating data.
func Resolve(p *store.PregnancyProfile, today civildate.Date) (Dating, bool) {
	if p == nil {
		return Dating{}, false
	}
	c := calc.New(p, "en", today)
	ga := c.GestationalAge()
	due, ok := c.EDDDate()
	if !ga.Valid || !ok {
		return Dating{}, false
	}
	d := Dating{
		Source:      p.AgeSource.String,
		Weeks:       ga.Weeks,
		Days:        ga.Days,
		TotalDays:   ga.TotalDays,
		Confidence:  string(ga.Confidence),
		Uncertainty: ga.Uncertainty,
		Due:         due,
		Today:       today,
	}
	switch d.Source {
	case "lmp":
		d.BasisDate = p.LmpDate.Date
	case "ultrasound":
		d.BasisDate = p.UltrasoundDate.Date
	default:
		d.BasisDate = p.ManualEntryDate.Date
	}
	return d, true
}

// Start is the (estimated) LMP: day 0 of week 1.
func (d Dating) Start() civildate.Date { return d.Due.AddDays(-280) }

// CurrentWeek is the 1-based week shown to the user, clamped to 1..MaxWeek.
func (d Dating) CurrentWeek() int { return min(MaxWeek, max(1, d.Weeks+1)) }

// Trimester of the current day.
func (d Dating) Trimester() int { return calc.Trimester(d.Weeks) }

// WeekRange is the first and last day of 1-based week n.
func (d Dating) WeekRange(n int) (civildate.Date, civildate.Date) {
	from := d.Start().AddDays((n - 1) * 7)
	return from, from.AddDays(6)
}

// BirthRange is the usual birth range around the due date.
func (d Dating) BirthRange() (civildate.Date, civildate.Date) {
	w := birthRangeDays + d.Uncertainty
	return d.Due.AddDays(-w), d.Due.AddDays(w)
}

// DaysLeft is today → due date (negative when overdue).
func (d Dating) DaysLeft() int { return d.Today.DiffDays(d.Due) }

// Percent is the progress through 40 weeks, 0..100.
func (d Dating) Percent() int {
	return min(100, max(0, int(math.Round(float64(d.TotalDays)*100/float64(TermWeeks*7)))))
}

// Relation of 1-based week n to the current week.
func (d Dating) Relation(n int) string {
	switch cur := d.CurrentWeek(); {
	case n < cur:
		return "past"
	case n == cur:
		return "current"
	}
	return "future"
}
