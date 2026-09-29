// Package pregnancyalerts is the pregnancy v2 alert-rule registry of the message engine
// (T-M7-04, docs/pregnancy-v2/README.md § Message engine integration).
//
// Detectors are Go; everything else is data: each rule reads enabled / level / window_days /
// params (from the default-language row) and its texts (request locale → default language) from
// message_contents group `pregnancy_alert`, item_key = rule key (admin-web
// pregnancy-alert-rules). A rule without a live row does not fire. Hits are persisted to
// pregnancy_alerts (alert_type `v2:<rule>`, v1 alert_level mapped from the 4 levels, the v2
// metadata in trigger_symptoms) and deduplicated per rule + dedupe key per window. Texts are
// rendered when an alert is read, so an admin edit shows without a deploy.
package pregnancyalerts

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Levels are the four v2 levels, info → urgent (the legend order).
var Levels = []string{"info", "suggestion", "follow_up", "urgent"}

// V1Level maps a v2 level onto the pregnancy_alerts.alert_level enum (info|warning|emergency).
func V1Level(level string) string {
	switch level {
	case "urgent":
		return "emergency"
	case "follow_up":
		return "warning"
	default:
		return "info"
	}
}

// RuleKeys are the registered detectors, in display order (= registry.AlertRules).
var RuleKeys = []string{
	"vomiting_streak", "severe_symptom_count", "critical_symptom", "weight_missing_week",
	"week_entered", "bp_high", "sugar_high", "fetal_movement",
}

// WeightMissingFromWeekday is weight_missing_week's default `from_weekday`: the 6th day of the
// pregnancy week (0-based 5). Not in the admin params schema yet, so the code default applies.
const WeightMissingFromWeekday = 5

// Config is the behaviour part of a rule row.
type Config struct {
	Enabled    bool
	Level      string
	WindowDays int
	Params     map[string]any
}

func (c Config) intParam(key string, def int) int {
	switch v := c.Params[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func (c Config) listParam(key string) []string {
	raw, _ := c.Params[key].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Day is one logged day: symptom → severity (mild|moderate|severe; "" when unset).
type Day map[string]string

// Facts are the user's data the detectors read, for the window ending Today.
type Facts struct {
	Today     civildate.Date
	Week      int // current 1-based week
	WeekDay   int // 0..6 days into the current week
	Source    string
	Days      map[civildate.Date]Day
	Weekly    []store.PregnancyWeeklyLog // logged within the longest window, plus the current week's
	Fetal     []store.PregnancyFetalMovement
	HasWeight bool // a weight is logged for the current week
}

// Hit is one detector firing: a dedupe key and the placeholder values of its texts. On is the
// day the fact happened (the week start, the streak's last day); zero means the evaluation day.
type Hit struct {
	Rule   string
	Dedupe string
	Vars   [][2]string
	On     civildate.Date
}

func (f Facts) windowStart(w int) civildate.Date { return f.Today.AddDays(-(max(1, w) - 1)) }

func (f Facts) inWindow(d civildate.Date, w int) bool {
	return !d.Before(f.windowStart(w)) && !d.After(f.Today)
}

// Detect runs one rule; nil when it does not fire (or the rule is unknown / disabled).
func Detect(rule string, c Config, f Facts) []Hit {
	if !c.Enabled {
		return nil
	}
	switch rule {
	case "vomiting_streak":
		return vomitingStreak(c, f)
	case "severe_symptom_count":
		return severeCount(c, f)
	case "critical_symptom":
		return critical(c, f)
	case "weight_missing_week":
		// Not before day from_weekday (0-based, default WeightMissingFromWeekday) of the week, so
		// the user has had most of the week to log it (review #11, T-M2-34).
		if f.Week >= c.intParam("from_week", 1) && f.WeekDay >= c.intParam("from_weekday", WeightMissingFromWeekday) && !f.HasWeight {
			w := strconv.Itoa(f.Week)
			return []Hit{{Rule: rule, Dedupe: "w" + w, Vars: [][2]string{{"week", w}}}}
		}
	case "week_entered":
		if f.WeekDay < max(1, c.WindowDays) {
			w := strconv.Itoa(f.Week)
			return []Hit{{
				Rule: rule, Dedupe: "w" + w, Vars: [][2]string{{"week", w}, {"basis", f.Source}},
				On: f.Today.AddDays(-f.WeekDay),
			}}
		}
	case "bp_high":
		return bpHigh(c, f)
	case "sugar_high":
		return sugarHigh(c, f)
	case "fetal_movement":
		return fetal(c, f)
	}
	return nil
}

// vomitingStreak: consecutive days with vomiting ending today (or yesterday, when today is not
// logged yet), capped at the window. Fires at min_streak_days, or when severe_min_count > 0 and
// that many streak days were severe.
func vomitingStreak(c Config, f Facts) []Hit {
	end := f.Today
	if _, ok := f.Days[end]["vomiting"]; !ok {
		end = end.AddDays(-1)
	}
	streak, severe := 0, 0
	start := end
	for d := end; f.inWindow(d, c.WindowDays); d = d.AddDays(-1) {
		sev, ok := f.Days[d]["vomiting"]
		if !ok {
			break
		}
		streak++
		start = d
		if sev == "severe" {
			severe++
		}
	}
	minStreak, minSevere := c.intParam("min_streak_days", 3), c.intParam("severe_min_count", 0)
	if streak == 0 || (streak < minStreak && (minSevere <= 0 || severe < minSevere)) {
		return nil
	}
	return []Hit{{Rule: "vomiting_streak", Dedupe: start.String(), Vars: [][2]string{
		{"days", strconv.Itoa(streak)}, {"severe_count", strconv.Itoa(severe)},
	}, On: end}}
}

// severeCount: severe (day, symptom) pairs of the listed symptoms within the window.
func severeCount(c Config, f Facts) []Hit {
	symptoms := c.listParam("symptoms")
	n := 0
	for d, day := range f.Days {
		if !f.inWindow(d, c.WindowDays) {
			continue
		}
		for s, sev := range day {
			if sev == "severe" && slices.Contains(symptoms, s) {
				n++
			}
		}
	}
	if n == 0 || n < c.intParam("min_count", 3) {
		return nil
	}
	return []Hit{{Rule: "severe_symptom_count", Dedupe: "", Vars: [][2]string{{"count", strconv.Itoa(n)}}}}
}

// critical: every listed critical symptom logged within the window; spotting only up to
// spotting_until_week (or when severe), like the v1 rule.
func critical(c Config, f Facts) []Hit {
	var out []Hit
	until := c.intParam("spotting_until_week", 12)
	for d := f.windowStart(c.WindowDays); !d.After(f.Today); d = d.AddDays(1) {
		day := f.Days[d]
		for _, s := range c.listParam("symptoms") {
			sev, ok := day[s]
			if !ok {
				continue
			}
			if s == "spotting" && f.Week > until && sev != "severe" {
				continue
			}
			out = append(out, Hit{Rule: "critical_symptom", Dedupe: s + "@" + d.String(), Vars: [][2]string{{"symptom", s}}})
		}
	}
	return out
}

func bpHigh(c Config, f Facts) []Hit {
	var out []Hit
	sMin, dMin := c.intParam("systolic_min", 140), c.intParam("diastolic_min", 90)
	for _, w := range f.Weekly {
		if !f.inWindow(w.LogDate, c.WindowDays) || (!w.SystolicPressure.Valid && !w.DiastolicPressure.Valid) {
			continue
		}
		hi := (w.SystolicPressure.Valid && int(w.SystolicPressure.Int32) >= sMin) ||
			(w.DiastolicPressure.Valid && int(w.DiastolicPressure.Int32) >= dMin)
		if !hi {
			continue
		}
		sys, dia := nullInt(w.SystolicPressure.Int32, w.SystolicPressure.Valid), nullInt(w.DiastolicPressure.Int32, w.DiastolicPressure.Valid)
		out = append(out, Hit{Rule: "bp_high", Dedupe: fmt.Sprintf("%s:%s/%s", w.LogDate, sys, dia),
			Vars: [][2]string{{"systolic", sys}, {"diastolic", dia}}})
	}
	return out
}

func sugarHigh(c Config, f Facts) []Hit {
	var out []Hit
	fMax, pMax := float64(c.intParam("fasting_max", 95)), float64(c.intParam("post_meal_max", 140))
	for _, w := range f.Weekly {
		if !f.inWindow(w.LogDate, c.WindowDays) {
			continue
		}
		fv, fok := decimal(w.FastingBloodSugar.String, w.FastingBloodSugar.Valid)
		pv, pok := decimal(w.PostMealBloodSugar.String, w.PostMealBloodSugar.Valid)
		if !(fok && fv > fMax) && !(pok && pv > pMax) {
			continue
		}
		fs, ps := orNone(w.FastingBloodSugar.String, fok), orNone(w.PostMealBloodSugar.String, pok)
		out = append(out, Hit{Rule: "sugar_high", Dedupe: fmt.Sprintf("%s:%s/%s", w.LogDate, fs, ps),
			Vars: [][2]string{{"fasting", fs}, {"post_meal", ps}}})
	}
	return out
}

func fetal(c Config, f Facts) []Hit {
	var out []Hit
	from, statuses := c.intParam("from_week", 24), c.listParam("statuses")
	for _, m := range f.Fetal {
		if !f.inWindow(m.LogDate, c.WindowDays) || int(m.PregnancyWeek) < from || !slices.Contains(statuses, m.MovementStatus) {
			continue
		}
		out = append(out, Hit{Rule: "fetal_movement", Dedupe: m.LogDate.String() + ":" + m.MovementStatus,
			Vars: [][2]string{{"status", m.MovementStatus}}})
	}
	return out
}

// none marks a missing reading in a placeholder value (rendered with the locale's "none").
const none = "\x00none"

func nullInt(v int32, ok bool) string {
	if !ok {
		return none
	}
	return strconv.Itoa(int(v))
}

func decimal(s string, ok bool) (float64, bool) {
	if !ok {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

func orNone(s string, ok bool) string {
	if !ok {
		return none
	}
	return s
}
