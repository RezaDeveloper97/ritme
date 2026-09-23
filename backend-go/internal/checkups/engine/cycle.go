package engine

import (
	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Cycle is the cycle prediction the engine places cycle-timed checkups on: a known period start
// and the effective cycle length. Predicted starts are Start + k·Length for every integer k.
// The zero value means "no cycle data" (monthly fallback).
type Cycle struct {
	Start  civildate.Date
	Length int
}

// Known reports whether the prediction is usable.
func (c Cycle) Known() bool { return !c.Start.IsZero() && c.Length > 0 }

// CycleFromHistory derives the prediction from the cycle engine (read-only reuse of
// internal/cycle, the same calls as the cycle view): the resolved current period start for today
// and the effective cycle length. No resolvable anchor (no history, no profile last period) →
// the zero Cycle.
func CycleFromHistory(histories []model.History, profile *model.Profile, now civildate.Nower) Cycle {
	today := civildate.Today(now)
	m := metrics.Calculate(histories, profile)
	st := resolver.Resolve(histories, profile, today, today, m)
	if st.CurrentPeriodStart.IsZero() {
		return Cycle{}
	}
	return Cycle{Start: st.CurrentPeriodStart, Length: max(1, st.EffectiveCycleLength)}
}

// startOnOrBefore is the predicted cycle start on or before d.
func (c Cycle) startOnOrBefore(d civildate.Date) civildate.Date {
	diff := c.Start.DiffDays(d)
	k := diff / c.Length
	if diff%c.Length < 0 {
		k-- // floor division
	}
	return c.Start.AddDays(k * c.Length)
}

type window struct{ from, to civildate.Date }

func (t Type) windowOf(cycleStart civildate.Date) window {
	return window{from: cycleStart.AddDays(t.CycleDayFrom - 1), to: cycleStart.AddDays(t.CycleDayTo - 1)}
}

// placesOnCycle: only monthly cycle-timed types ride the predicted cycles; longer intervals
// (Pap 36 months, cycle days 10–20) keep the calendar date and use the window as advice only.
func placesOnCycle(t Type, c Cycle) bool {
	return t.CycleTimed() && t.IntervalMonths <= 1 && c.Known()
}

// windowAfter is the window of the first predicted cycle that starts after doneOn.
func windowAfter(t Type, c Cycle, doneOn civildate.Date) (window, bool) {
	if !placesOnCycle(t, c) {
		return window{}, false
	}
	return t.windowOf(c.startOnOrBefore(doneOn).AddDays(c.Length)), true
}

// upcomingWindow is the current cycle's window while it has not ended, else the next cycle's.
func upcomingWindow(t Type, c Cycle, today civildate.Date) (window, bool) {
	if !placesOnCycle(t, c) {
		return window{}, false
	}
	w := t.windowOf(c.startOnOrBefore(today))
	if today.After(w.to) {
		w = t.windowOf(c.startOnOrBefore(today).AddDays(c.Length))
	}
	return w, true
}
