// Package conditionnudges is the heavy pain / heavy bleeding nudges of the condition programs (CB-COND-06, IA_Map
// cross-link «ثبت درد زیاد → پیشنهاد دفترچه درد و پزشک»):
//
//   - heavy_pain: pain ≥ PainMinScore on MinDays+ days of the current cycle while not enrolled in the endometriosis
//     program → suggest the pain diary (/programs/pain);
//   - heavy_bleeding: heavy or very heavy flow on MinDays+ days of the current cycle while not enrolled in the
//     heavy-bleeding program → suggest the PBAC chart (/programs/bleeding).
//
// The facts come from bloom's day log (health_log_entries through healthlog.Service — the pain diary of CB-COND-01
// writes its score there too), the enrolments from condition_enrolments and the cycle start from cycle_histories:
// no parallel storage, nothing is persisted — a nudge is computed per request and disappears once she enrols (or
// the cycle turns over). Only the cycle-tracking modes (cycle, ttc, teen) are nudged; pregnancy, postpartum and
// menopause have their own bleeding / pain alerts.
//
// The texts are admin-editable message_contents rows of group `condition_nudge` (item = rule key, payload
// {title, body, action, doctor_action}, `{days}` placeholder); a missing text falls back to the embedded copy in
// lang/<code>/condition_nudges.json. All thresholds and copy are [needs clinical review].
package conditionnudges

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Group is the message_contents group of the nudge texts.
const Group = "condition_nudge"

// Rule keys (= message_contents item keys).
const (
	RuleHeavyPain     = "heavy_pain"
	RuleHeavyBleeding = "heavy_bleeding"
)

// Rules are every rule, in display order.
var Rules = []string{RuleHeavyPain, RuleHeavyBleeding}

// Program codes (conditions.ProgramEndo / ProgramHeavyBleeding — repeated here so the messages engine does not
// depend on the conditions package).
const (
	ProgramEndo          = "endo"
	ProgramHeavyBleeding = "heavy_bleeding"
)

// Thresholds [needs clinical review].
const (
	// PainMinScore is the lowest 0–10 pain score that counts as strong pain.
	PainMinScore = 7
	// MinDays is how many qualifying days of the cycle raise a nudge.
	MinDays = 2
	// MaxCycleDays: a latest period start older than this is not "the current cycle" (no period logged since);
	// the window is then the last FallbackCycleDays (or the profile's cycle length).
	MaxCycleDays = 45
	// FallbackCycleDays is the window without a usable period start or profile cycle length.
	FallbackCycleDays = 28
)

// painSevere is the taxonomy pain level a 7–10 score maps to (conditions.LevelFor); a log-sheet entry with a level
// but no score counts by its level.
const painSevere = "severe"

// heavyFlow are the bleeding.flow options that count as heavy.
var heavyFlow = []string{"heavy", "very_heavy"}

// nudgedModes are the life modes the nudges run in.
var nudgedModes = []string{string(enums.LifeModeCycle), string(enums.LifeModeTTC), string(enums.LifeModeTeen)}

// rule is one nudge rule.
type rule struct {
	key, program, link string
	day                func([]taxonomy.Entry) bool
}

var rules = []rule{
	{key: RuleHeavyPain, program: ProgramEndo, link: "/programs/pain", day: strongPainDay},
	{key: RuleHeavyBleeding, program: ProgramHeavyBleeding, link: "/programs/bleeding", day: heavyFlowDay},
}

// Facts are one user's inputs to the rules.
type Facts struct {
	Mode     string          // effective life mode (enums.ResolveLifeMode)
	Enrolled map[string]bool // program → enrolled
	From, To civildate.Date  // the current cycle window (inclusive)
	Days     []healthlog.DayEntries
}

// Hit is one raised nudge.
type Hit struct {
	Rule, Program, Link string
	Dates               []civildate.Date // qualifying days, oldest first
}

// Detect runs every rule on f (pure).
func Detect(f Facts) []Hit {
	if !slices.Contains(nudgedModes, f.Mode) {
		return nil
	}
	var out []Hit
	for _, r := range rules {
		if f.Enrolled[r.program] {
			continue
		}
		var dates []civildate.Date
		for _, d := range f.Days {
			if d.Date.Before(f.From) || d.Date.After(f.To) {
				continue
			}
			if r.day(d.Entries) {
				dates = append(dates, d.Date)
			}
		}
		if len(dates) >= MinDays {
			out = append(out, Hit{Rule: r.key, Program: r.program, Link: r.link, Dates: dates})
		}
	}
	return out
}

// strongPainDay: any pain location scored ≥ PainMinScore (a location without a score counts when its level is
// severe). "No pain" and "no" items never count.
func strongPainDay(day []taxonomy.Entry) bool {
	for _, e := range day {
		if e.Category != "pain" || e.Param != "location" || (e.Code.Valid && e.Code.String == taxonomy.No) {
			continue
		}
		if e.Num.Valid {
			if f, err := strconv.ParseFloat(e.Num.String, 64); err == nil {
				if f >= PainMinScore {
					return true
				}
				continue
			}
		}
		if e.Code.Valid && e.Code.String == painSevere {
			return true
		}
	}
	return false
}

// heavyFlowDay: bleeding.flow is heavy or very heavy (the old log's "high" / "very_high" intensity is projected
// onto the same options).
func heavyFlowDay(day []taxonomy.Entry) bool {
	for _, e := range day {
		if e.Category == "bleeding" && e.Param == "flow" && e.Code.Valid && slices.Contains(heavyFlow, e.Code.String) {
			return true
		}
	}
	return false
}

// CycleWindow is the current cycle [from, today]: from the latest period start on or before today when it is less
// than MaxCycleDays old, else the last cycleLen days (FallbackCycleDays when cycleLen is unset or implausible).
func CycleWindow(today civildate.Date, latestStart *civildate.Date, cycleLen int) (civildate.Date, civildate.Date) {
	if latestStart != nil && !latestStart.After(today) && latestStart.DiffDays(today) < MaxCycleDays {
		return *latestStart, today
	}
	if cycleLen < 21 || cycleLen > MaxCycleDays {
		cycleLen = FallbackCycleDays
	}
	return today.AddDays(-(cycleLen - 1)), today
}
