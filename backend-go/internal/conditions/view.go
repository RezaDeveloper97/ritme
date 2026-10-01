package conditions

import (
	"github.com/ritme/backend-go/internal/catalog"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

func ptr[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// OverviewJSON is GET /conditions: every program with the user's enrolment.
func OverviewJSON(list []Enrolment) *jsonx.OrderedMap {
	programs := make([]*jsonx.OrderedMap, len(list))
	for i, e := range list {
		var on any
		if e.Enrolled {
			on = e.EnrolledOn
		}
		programs[i] = jsonx.Obj("code", e.Program, "enrolled", e.Enrolled, "enrolled_on", on)
	}
	return jsonx.Obj("programs", programs)
}

// PainJSON is one pain-diary day.
func PainJSON(d PainDay) *jsonx.OrderedMap {
	return jsonx.Obj(
		"date", d.Date,
		"score", ptr(d.Score),
		"locations", jsonx.List(d.Locations),
		"relief", jsonx.List(d.Relief),
		"types", jsonx.List(d.Types),
		"associated", jsonx.List(d.Associated),
		"missed_activity", ptr(d.MissedActivity),
		"analgesic", ptr(d.Analgesic),
		"analgesic_time", ptr(d.AnalgesicTime),
		"analgesic_effect", ptr(d.AnalgesicEffect),
	)
}

func scoresJSON(list []ItemScore) *jsonx.OrderedMap {
	m := jsonx.NewObject()
	for _, s := range list {
		m.Set(s.Code, s.Score)
	}
	return m
}

func meanOrNil(d PMDDDay) any {
	if len(d.Scores) == 0 {
		return nil
	}
	return d.Mean()
}

// PMDDJSON is one questionnaire day: scores {item: 1–6} and their mean (null when nothing was rated).
func PMDDJSON(d PMDDDay) *jsonx.OrderedMap {
	return jsonx.Obj("date", d.Date, "scores", scoresJSON(d.Scores), "mean", meanOrNil(d))
}

func windowJSON(w Window) *jsonx.OrderedMap { return jsonx.Obj("from", w.From, "to", w.To) }

// ChartJSON is GET /conditions/pmdd/chart.
func ChartJSON(ch Chart, loc catalog.Localizer) *jsonx.OrderedMap {
	cycles := make([]*jsonx.OrderedMap, len(ch.Cycles))
	for i, c := range ch.Cycles {
		days := make([]*jsonx.OrderedMap, len(c.Days))
		for j, d := range c.Days {
			days[j] = jsonx.Obj(
				"date", d.Day.Date,
				"cycle_day", d.CycleDay,
				"in_period", d.InPeriod,
				"late_luteal", d.LateLuteal,
				"mean", d.Day.Mean(),
				"scores", scoresJSON(d.Day.Scores),
			)
		}
		cycles[i] = jsonx.Obj(
			"start", c.Start,
			"end", c.End,
			"closed", c.Closed,
			"complete", c.Complete,
			"period", windowJSON(c.Period),
			"late_luteal", windowJSON(c.LateLuteal),
			"follicular", windowJSON(c.Follicular),
			"luteal_days", c.LutealDays,
			"follicular_days", c.FollicularDays,
			"luteal_mean", ptr(c.LutealMean),
			"follicular_mean", ptr(c.FollicularMean),
			"days", days,
		)
	}
	return jsonx.Obj(
		"required_cycles", RequiredCycles,
		"complete_cycles", ch.CompleteCycles,
		"ready", ch.Ready,
		"pattern", ch.Pattern,
		"cycles", cycles,
		"alerts", alertsJSON(ch.Alerts, loc),
	)
}

func alertsJSON(items []catalog.Item, loc catalog.Localizer) []*jsonx.OrderedMap {
	out := make([]*jsonx.OrderedMap, len(items))
	for i, it := range items {
		out[i] = loc.Public(it)
	}
	return out
}

// PBACJSON is one pad-chart day with its period and the ≥ 100 alert.
func PBACJSON(v PBACView, loc catalog.Localizer) *jsonx.OrderedMap {
	var period, alert any
	if p := v.Period; p != nil {
		period = jsonx.Obj("start", p.Start, "end", p.End, "day", p.DayNumber, "score", p.Score, "days_logged", p.DaysLogged)
	}
	if v.Alert != nil {
		alert = loc.Public(*v.Alert)
	}
	d := v.Day
	return jsonx.Obj(
		"date", d.Date,
		"light", d.Light,
		"medium", d.Medium,
		"heavy", d.Heavy,
		"clots", ptr(d.Clots),
		"flooding", d.Flooding,
		"score", d.Score(),
		"period", period,
		"alert_threshold", AlertScore,
		"alert", alert,
	)
}
