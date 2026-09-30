// Package reminders plans the cycle reminders of B-N1-09 — before period, PMS start, fertile window, daily log
// and pill — from a user's notification preferences (internal/notifications) and the cycle engine's prediction.
//
// There is no push sender yet: a sender (cron, web-push worker) asks Plan for the occurrences of a day and sends
// only those whose Decision says Send, rendered through notifications.Render. Every occurrence has already gone
// through notifications.Decide, so a switched-off category, quiet hours and the category's time are honoured in
// one place:
//
//	prefs, _ := notifications.Load(ctx, q, userID)
//	for _, o := range reminders.Plan(prefs, reminders.CycleFrom(anchor, cycleLen, periodLen, today), today) {
//	    if o.Decision.Send { … } else if !o.Decision.DeferUntil.IsZero() { … retry then … }
//	}
package reminders

import (
	"time"

	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// PMSDays mirrors the resolver's `pms_possible` run and the home PMS window: the 3 days before the period.
const PMSDays = 3

// Cycle is the part of the prediction the planner needs; zero dates = unknown (no period logged yet).
type Cycle struct {
	NextPeriodStart    civildate.Date
	PMSStart           civildate.Date
	FertileWindowStart civildate.Date
	// CycleLength is the effective length used for the prediction (0 = unknown).
	CycleLength int
}

// CycleFrom projects the anchor (last confirmed period start) forward like the engine (view.Predict) to the
// first cycle whose period starts after today, so every date is upcoming. A zero anchor = unknown cycle.
func CycleFrom(anchor civildate.Date, cycleLength, periodLength int, today civildate.Date) Cycle {
	if anchor.IsZero() || cycleLength <= 0 {
		return Cycle{}
	}
	p := view.Predict(anchor, cycleLength, max(1, periodLength), today, enums.EffectiveSource(""))
	next := p.NextPeriodStart
	fertile := p.FertileWindowStart
	if fertile.Before(today) { // this cycle's window is over: plan the next one
		fertile = fertile.AddDays(cycleLength)
	}
	return Cycle{
		NextPeriodStart:    next,
		PMSStart:           next.AddDays(-PMSDays),
		FertileWindowStart: fertile,
		CycleLength:        cycleLength,
	}
}

// PMSCycleDay is the cycle day the PMS reminder fires on («روز ۲۷ سیکل» for a 29-day cycle); 0 = unknown.
func PMSCycleDay(cycleLength int) int {
	if cycleLength <= PMSDays {
		return 0
	}
	return cycleLength - PMSDays + 1
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
