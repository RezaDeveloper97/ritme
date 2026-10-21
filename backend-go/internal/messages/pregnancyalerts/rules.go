// Package pregnancyalerts is the pregnancy v2 alert-rule registry of the message engine
// (T-M7-04, docs/pregnancy-v2/README.md § Message engine integration).
//
// Detectors are Go; everything else is data: each rule reads enabled / level / window_days /
// params (from the default-language row) and its texts (request locale → default language) from
// message_contents group `pregnancy_alert`, item_key = rule key (admin-web
// pregnancy-alert-rules). A rule without a live row does not fire, except the emergency rules
// (EmergencyRules, T-M9-05): their path is fixed and a missing or disabled row falls back to the
// embedded copy (emergency/<locale>.json). Clinical thresholds of the pregnancy rules (BP, GDM,
// kicks, 5-1-1) are the versioned clinical config (internal/clinicalconfig), not row params. Hits are persisted to
// pregnancy_alerts (alert_type `v2:<rule>`, v1 alert_level mapped from the 4 levels, the v2
// metadata in trigger_symptoms) and deduplicated per rule + dedupe key per window. Texts are
// rendered when an alert is read, so an admin edit shows without a deploy.
package pregnancyalerts

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/clinicalconfig"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy/labor"
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
	"week_entered", "bp_high", "sugar_high", "fetal_movement", "contractions_511",
}

// WeightMissingFromWeekday is weight_missing_week's default `from_weekday`: the 6th day of the
// pregnancy week (0-based 5), used while the rule row has no `params.from_weekday` (or null).
const WeightMissingFromWeekday = 5

// EmergencyRules always fire (SPEC v6 §4.2 "fixed emergency paths", T-M9-05): admins edit their copy, never switch
// them off. Fluid leakage is a critical_symptom; the bp_high row's switch only governs its high tier.
var EmergencyRules = []string{"fetal_movement", "contractions_511", "critical_symptom", "bp_high"}

// MandatoryCriticalSymptoms are always part of critical_symptom, whatever the row's `symptoms` list says.
var MandatoryCriticalSymptoms = []string{"bleeding", "fluid_leakage", "severe_sudden_pain"}

// Variants of a rule's copy (a Hit's Variant; texts from the row's `variants.<name>`, else the embedded copy).
const (
	VariantKicks  = "kicks"  // fetal_movement raised by a kick-count session
	VariantSevere = "severe" // bp_high at or above pregnancy.bp.severe
)

// Config is the behaviour part of a rule row.
type Config struct {
	Enabled    bool
	Level      string
	WindowDays int
	Params     map[string]any
	// Forced: the row was missing or disabled and an emergency rule runs on its embedded copy.
	Forced bool
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
	// The contractions of the running contraction session (B-N5-03) and its id.
	Contractions       []labor.Contraction
	ContractionSession uint64
	// Now is the evaluation instant (a kick session running longer than its window counts as ended at
	// start + window); TotalDays the gestational age in days on Today.
	Now       time.Time
	TotalDays int
	// Kicks are the kick-count sessions of the last KickLookbackDays (ended, plus the running one).
	Kicks []Kick
	// Readings are vitals readings handed over by internal/vitals (EvaluateReadings).
	Readings []Reading
}

// Kick is one kick-count session (pregnancy_kick_sessions).
type Kick struct {
	ID       uint64
	Start    time.Time
	End      time.Time // zero while running
	Count    int
	TargetAt time.Time // the target-th kick (tenth_kick_at), zero when not reached
}

// Reading is a vitals reading of a pregnant user. Systolic / Diastolic are 0 when not a BP reading; MgDl is 0 when not
// a glucose reading. Slot is the GDM slot of a glucose reading (GlucoseFasting / GlucoseOneHour / GlucoseTwoHour).
type Reading struct {
	Date                civildate.Date
	Systolic, Diastolic int
	MgDl                float64
	Slot                string
}

// GDM slots of a Reading.
const (
	GlucoseFasting = "fasting"
	GlucoseOneHour = "one_hour"
	GlucoseTwoHour = "two_hour"
)

// KickLookbackDays: a session that ended or expired yesterday is still evaluated today (the lazy expiry may only be
// seen on the next morning's read).
const KickLookbackDays = 2

// gaWeeks are the completed weeks of gestation on day (from the dating of Today).
func (f Facts) gaWeeks(day civildate.Date) int {
	return (f.TotalDays - day.DiffDays(f.Today)) / 7
}

// Hit is one detector firing: a dedupe key and the placeholder values of its texts. On is the
// day the fact happened (the week start, the streak's last day); zero means the evaluation day.
type Hit struct {
	Rule   string
	Dedupe string
	Vars   [][2]string
	On     civildate.Date
	// Level overrides the row's level ("" = the row's): the emergency tiers are urgent whatever the row says.
	Level string
	// Variant selects a variant of the rule's copy ("" = the base copy).
	Variant string
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
	case "contractions_511":
		return contractions511(c, f)
	}
	return nil
}

// contractions511: the running contraction session met the 5-1-1 rule (once per session), with the thresholds of
// the clinical config (labor.ParamsOn, pregnancy.contractions_511), not the row's params.
func contractions511(_ Config, f Facts) []Hit {
	if f.ContractionSession == 0 {
		return nil
	}
	r := labor.FiveOneOne(f.Contractions, labor.ParamsOn(f.Today))
	if !r.Met {
		return nil
	}
	return []Hit{{Rule: "contractions_511", Dedupe: "s" + strconv.FormatUint(f.ContractionSession, 10), Vars: [][2]string{
		{"count", strconv.Itoa(r.Count)}, {"minutes", strconv.Itoa(int(r.Span / time.Minute))},
		{"interval", labor.Clock(r.AvgInterval)}, {"duration", labor.Clock(r.AvgDuration)},
	}}}
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
	symptoms := c.listParam("symptoms")
	for _, s := range MandatoryCriticalSymptoms {
		if !slices.Contains(symptoms, s) {
			symptoms = append(symptoms, s)
		}
	}
	for d := f.windowStart(c.WindowDays); !d.After(f.Today); d = d.AddDays(1) {
		day := f.Days[d]
		for _, s := range symptoms {
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

// BPTier is the pregnancy class of a reading on day (clinical config pregnancy.bp.high / pregnancy.bp.severe):
// "" below the high threshold, "high", or "severe". Either number at or above a threshold decides.
func BPTier(systolic, diastolic int, day civildate.Date) string {
	n := func(key, field string) int { return int(clinicalconfig.Num(key, field, day)) }
	at := func(key string) bool {
		return (systolic > 0 && systolic >= n(key, "systolic")) || (diastolic > 0 && diastolic >= n(key, "diastolic"))
	}
	switch {
	case at(clinicalconfig.KeyPregnancyBPSevere):
		return "severe"
	case at(clinicalconfig.KeyPregnancyBPHigh):
		return "high"
	}
	return ""
}

// HighBPLevel is the v2 level of the high tier on day: urgent while pregnancy.bp.high.urgent = 1 (today's level),
// follow_up when the advisor sets 0 (H-02). The severe tier is always urgent.
func HighBPLevel(day civildate.Date) string {
	if clinicalconfig.Num(clinicalconfig.KeyPregnancyBPHigh, "urgent", day) >= 1 {
		return "urgent"
	}
	return "follow_up"
}

// bpHit is the hit of one reading (nil below the high threshold; only the severe tier when the row is forced).
func bpHit(c Config, day civildate.Date, sys, dia int, sysOK, diaOK bool) *Hit {
	s, d := 0, 0
	if sysOK {
		s = sys
	}
	if diaOK {
		d = dia
	}
	tier := BPTier(s, d, day)
	if tier == "" || (c.Forced && tier != "severe") {
		return nil
	}
	ss, ds := nullInt(int32(sys), sysOK), nullInt(int32(dia), diaOK) //nolint:gosec // validated mmHg
	h := &Hit{Rule: "bp_high", Dedupe: fmt.Sprintf("%s:%s/%s", day, ss, ds), Vars: [][2]string{{"systolic", ss}, {"diastolic", ds}}}
	if tier == "severe" {
		h.Level, h.Variant = "urgent", VariantSevere
	} else {
		h.Level = HighBPLevel(day)
	}
	return h
}

// bpHigh: two tiers (T-M9-05) over the weekly logs and the handed-over vitals readings of the window.
func bpHigh(c Config, f Facts) []Hit {
	var out []Hit
	for _, w := range f.Weekly {
		if !f.inWindow(w.LogDate, c.WindowDays) || (!w.SystolicPressure.Valid && !w.DiastolicPressure.Valid) {
			continue
		}
		if h := bpHit(c, w.LogDate, int(w.SystolicPressure.Int32), int(w.DiastolicPressure.Int32),
			w.SystolicPressure.Valid, w.DiastolicPressure.Valid); h != nil {
			out = append(out, *h)
		}
	}
	for _, r := range f.Readings {
		if !f.inWindow(r.Date, c.WindowDays) || (r.Systolic <= 0 && r.Diastolic <= 0) {
			continue
		}
		if h := bpHit(c, r.Date, r.Systolic, r.Diastolic, r.Systolic > 0, r.Diastolic > 0); h != nil {
			out = append(out, *h)
		}
	}
	return out
}

// GDMTarget is the GDM target of a slot on day (clinical config pregnancy.gdm.targets): a value at or above it is out
// of target. 0 for an unknown slot.
func GDMTarget(slot string, day civildate.Date) float64 {
	switch slot {
	case GlucoseFasting, GlucoseOneHour, GlucoseTwoHour:
		return clinicalconfig.Num(clinicalconfig.KeyPregnancyGDMTargets, slot, day)
	}
	return 0
}

// AboveGDM reports whether mgdl is at or above the slot's GDM target on day.
func AboveGDM(mgdl float64, slot string, day civildate.Date) bool {
	t := GDMTarget(slot, day)
	return t > 0 && mgdl >= t
}

// sugarHigh: fasting ≥ 95, 1 h ≥ 140, 2 h ≥ 120 mg/dL (clinical config). The weekly log's post-meal value is the 1 h
// slot; the 2 h slot comes from vitals readings.
func sugarHigh(c Config, f Facts) []Hit {
	var out []Hit
	for _, w := range f.Weekly {
		if !f.inWindow(w.LogDate, c.WindowDays) {
			continue
		}
		fv, fok := decimal(w.FastingBloodSugar.String, w.FastingBloodSugar.Valid)
		pv, pok := decimal(w.PostMealBloodSugar.String, w.PostMealBloodSugar.Valid)
		if !(fok && AboveGDM(fv, GlucoseFasting, w.LogDate)) && !(pok && AboveGDM(pv, GlucoseOneHour, w.LogDate)) {
			continue
		}
		fs, ps := orNone(w.FastingBloodSugar.String, fok), orNone(w.PostMealBloodSugar.String, pok)
		out = append(out, Hit{Rule: "sugar_high", Dedupe: fmt.Sprintf("%s:%s/%s", w.LogDate, fs, ps),
			Vars: [][2]string{{"fasting", fs}, {"post_meal", ps}}})
	}
	for _, r := range f.Readings {
		if r.MgDl <= 0 || !f.inWindow(r.Date, c.WindowDays) || !AboveGDM(r.MgDl, r.Slot, r.Date) {
			continue
		}
		v := strconv.FormatFloat(r.MgDl, 'f', -1, 64)
		fs, ps := none, v
		if r.Slot == GlucoseFasting {
			fs, ps = v, none
		}
		out = append(out, Hit{Rule: "sugar_high", Dedupe: fmt.Sprintf("%s:%s:%s", r.Date, r.Slot, v),
			Vars: [][2]string{{"fasting", fs}, {"post_meal", ps}}})
	}
	return out
}

// KickParams are the kick-count thresholds on day (clinical config pregnancy.kicks).
type KickParams struct {
	Target   int
	Window   time.Duration
	FromWeek int // completed weeks
}

// KickParamsOn reads pregnancy.kicks valid on day.
func KickParamsOn(day civildate.Date) KickParams {
	n := func(f string) float64 { return clinicalconfig.Num(clinicalconfig.KeyPregnancyKicks, f, day) }
	return KickParams{Target: int(n("target")), Window: time.Duration(n("window_min") * float64(time.Minute)),
		FromWeek: int(n("from_week"))}
}

// KickEnd is when a session counts as ended: its end, or start + window once a running session has run past it
// (zero while it is still counting).
func KickEnd(k Kick, p KickParams, now time.Time) time.Time {
	if !k.End.IsZero() {
		return k.End
	}
	if exp := k.Start.Add(p.Window); !now.Before(exp) {
		return exp
	}
	return time.Time{}
}

// KickReduced: an ended (or expired) session without the target within the window (H-05: < 10 in 2 h → urgent).
func KickReduced(k Kick, p KickParams, now time.Time) bool {
	if KickEnd(k, p, now).IsZero() {
		return false
	}
	reached := k.Count >= p.Target && !k.TargetAt.IsZero() && k.TargetAt.Sub(k.Start) <= p.Window
	return !reached
}

// fetal: the manual reduced / none log from pregnancy.fetal_movement.manual_from_week (H-04, row level), and every
// kick session that ended or expired short of the target at or after pregnancy.kicks.from_week (always urgent).
func fetal(c Config, f Facts) []Hit {
	var out []Hit
	from := int(clinicalconfig.Num(clinicalconfig.KeyPregnancyManualFetalWeek, "week", f.Today))
	statuses := c.listParam("statuses")
	if len(statuses) == 0 {
		statuses = []string{"reduced", "none"}
	}
	for _, m := range f.Fetal {
		if !f.inWindow(m.LogDate, c.WindowDays) || int(m.PregnancyWeek) < from || !slices.Contains(statuses, m.MovementStatus) {
			continue
		}
		out = append(out, Hit{Rule: "fetal_movement", Dedupe: m.LogDate.String() + ":" + m.MovementStatus,
			Vars: [][2]string{{"status", m.MovementStatus}}})
	}
	for _, k := range f.Kicks {
		day := civildate.InTehran(k.Start)
		p := KickParamsOn(day)
		end := KickEnd(k, p, f.Now)
		if end.IsZero() || !f.inWindow(civildate.InTehran(end), KickLookbackDays) || f.gaWeeks(day) < p.FromWeek ||
			!KickReduced(k, p, f.Now) {
			continue
		}
		minutes := int(p.Window / time.Minute)
		out = append(out, Hit{Rule: "fetal_movement", Dedupe: "kick:" + strconv.FormatUint(k.ID, 10),
			Vars: [][2]string{{"count", strconv.Itoa(k.Count)}, {"target", strconv.Itoa(p.Target)},
				{"minutes", strconv.Itoa(minutes)}},
			On: civildate.InTehran(end), Level: "urgent", Variant: VariantKicks})
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

// BPTierThreshold is one number of a BP tier on day ("high" | "severe"; field "systolic" | "diastolic").
func BPTierThreshold(tier, field string, day civildate.Date) float64 {
	key := clinicalconfig.KeyPregnancyBPHigh
	if tier == "severe" {
		key = clinicalconfig.KeyPregnancyBPSevere
	}
	return clinicalconfig.Num(key, field, day)
}
