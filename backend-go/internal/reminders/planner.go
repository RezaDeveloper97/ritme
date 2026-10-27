// Package reminders plans the cycle reminders of B-N1-09 — before period, PMS start, fertile window, daily log
// and pill — from a user's notification preferences (internal/notifications) and the cycle engine's prediction.
//
// There is no push sender yet: a sender (cron, web-push worker) asks Plan for the occurrences of a day and sends
// only those whose Decision says Send, rendered through notifications.Render. Every occurrence has already gone
// through notifications.Decide, so a switched-off category, quiet hours and the category's time are honoured in
// one place:
//
//	prefs, _ := notifications.Load(ctx, q, userID)
//	for _, o := range reminders.Plan(prefs, reminders.CycleFrom(engineState, today), today) {
//	    if o.Decision.Send { … } else if !o.Decision.DeferUntil.IsZero() { … retry then … }
//	}
package reminders

import (
	"time"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PMSDays mirrors the engine's default `pms_possible` run (Params.PMSDays): the 3 days before the period.
const PMSDays = 3

// Cycle is the part of the prediction the planner needs; zero dates = unknown (no period logged yet).
type Cycle struct {
	NextPeriodStart    civildate.Date
	PMSStart           civildate.Date
	FertileWindowStart civildate.Date
	// CycleLength is the effective length used for the prediction (0 = unknown).
	CycleLength int
}

// CycleFrom reads the upcoming dates off the engine's display chain (State.FutureCycles, T-M9-36, D-136), so a sender
// uses the same predictions as the app: the first future period that starts after today, the PMS window
// next − PMSDays .. next − 1 (State.Params), and the first fertile window that starts today or later.
// Hormonal contraception or pregnancy empty the fertile window / the whole list (E-13), so those reminders do not
// fire. No anchor (or a postpartum / over-late user) = the zero Cycle.
func CycleFrom(es *engine.State, today civildate.Date) Cycle {
	day := engine.DayOf(today)
	var c Cycle
	for _, fc := range es.FutureCycles(day) {
		if c.NextPeriodStart.IsZero() && fc.PeriodStart > day {
			c.NextPeriodStart = fc.PeriodStart.Civil()
			c.PMSStart = fc.PeriodStart.Civil().AddDays(-es.Params().PMSDays)
		}
		if c.FertileWindowStart.IsZero() && fc.FertileStart.IsSet() && fc.FertileStart >= day {
			c.FertileWindowStart = fc.FertileStart.Civil()
		}
	}
	if c.NextPeriodStart.IsZero() {
		return Cycle{}
	}
	c.CycleLength, _ = es.CycleLength()
	return c
}

// PMSCycleDay is the cycle day the PMS reminder fires on («روز ۲۷ سیکل» for a 29-day cycle); 0 = unknown. It is the
// first day of the PMS window next − PMSDays .. next − 1, where next is cycle day cycleLength + 1.
func PMSCycleDay(cycleLength int) int { return PMSCycleDayOf(cycleLength, PMSDays) }

// PMSCycleDayOf is PMSCycleDay for a PMS window of pmsDays days (the engine's PMSDays parameter).
func PMSCycleDayOf(cycleLength, pmsDays int) int {
	if cycleLength <= pmsDays {
		return 0
	}
	return cycleLength - pmsDays + 1
}

// Occurrence is one reminder due on the planned day.
type Occurrence struct {
	Category notifications.Category
	At       time.Time // Tehran wall-clock
	Decision notifications.Decision
}

// Plan lists the reminders due on day, in screen order, each gated by notifications.Decide at its own time.
// Cycle-bound reminders need a known cycle; daily log and pill fire every day. Switched-off categories are
// still listed (Decision.Reason = category_off) so a sender can log why nothing went out.
func Plan(p notifications.Preferences, c Cycle, day civildate.Date) []Occurrence {
	var out []Occurrence
	for _, cat := range notifications.Timed {
		if !dueOn(p, c, cat, day) {
			continue
		}
		minute, _ := p.TimeOf(cat)
		at := day.TehranMidnight().Add(time.Duration(minute) * time.Minute)
		out = append(out, Occurrence{Category: cat, At: at, Decision: notifications.Decide(p, cat, at, time.Time{})})
	}
	return out
}

func dueOn(p notifications.Preferences, c Cycle, cat notifications.Category, day civildate.Date) bool {
	switch cat {
	case notifications.BeforePeriod:
		return !c.NextPeriodStart.IsZero() && c.NextPeriodStart.AddDays(-p.DaysBeforePeriod()) == day
	case notifications.PMS:
		return !c.PMSStart.IsZero() && c.PMSStart == day
	case notifications.FertileWindow:
		return !c.FertileWindowStart.IsZero() && c.FertileWindowStart == day
	case notifications.DailyLog, notifications.Pill:
		return true
	}
	return false
}
