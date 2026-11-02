// Package conditionnudges is the heavy pain / heavy bleeding nudges of the condition programs (CB-COND-06, IA_Map
// cross-link «ثبت درد زیاد → پیشنهاد دفترچه درد و پزشک»). Since T-M10-09 (D-166) each nudge is a message rule
// (rule_key cond.<name>, seeded by goose 00115, slot alerts_inbox, content / suggestion):
//
//   - cond.heavy_pain: cond.heavy_pain_days ≥ 2 (pain ≥ PainMinScore on a day of the current cycle) while
//     cond.programs does not hold endo → suggest the pain diary (/programs/pain);
//   - cond.heavy_bleeding: cond.heavy_bleeding_days ≥ 2 (heavy or very heavy flow) while not enrolled in
//     heavy_bleeding → suggest the PBAC chart (/programs/bleeding).
//
// The facts come from bloom's day log (health_log_entries through healthlog.Service — the pain diary of CB-COND-01
// writes its score there too), the enrolments from condition_enrolments and the cycle window from cycle_histories (v1)
// or the v6 engine; the counts become the cond.* facts (messages/facts/conditions.go) and the nudges run through the
// alerts inbox (one row per rule, merged on a new day, resolved once she enrols or the cycle turns over). Only the
// cycle-tracking modes (cycle, ttc, teen) are nudged; pregnancy, postpartum and menopause have their own alerts.
// GET /messages/nudges is an adapter over the inbox with the D-41 shape.
//
// The adapter's texts are unchanged: admin-editable message_contents rows of group `condition_nudge` (item = rule key,
// payload {title, body, action, doctor_action}, `{days}` placeholder); a missing text falls back to the embedded copy
// in lang/<code>/condition_nudges.json. The seeded rules carry the same copy (copy/<code>.json) for the inbox and the
// feed. All thresholds and copy are [needs clinical review].
package conditionnudges

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/healthlog/taxonomy"
	mfacts "github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
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

// Day rule and window [needs clinical review]. How many qualifying days raise a nudge and the program that silences
// it are conditions of the seeded cond.* rules (T-M10-09).
const (
	// PainMinScore is the lowest 0–10 pain score that counts as strong pain.
	PainMinScore = 7
	// MaxCycleDays (CYCLE_ENGINE v1 only): a latest period start older than this is not "the current cycle" (no period
	// logged since); the window is then the last FallbackCycleDays (or the profile's cycle length). The v6 window is
	// EngineCycleWindow.
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

// KeyPrefix is the rule_key prefix of the condition nudge rules.
const KeyPrefix = "cond."

// Def is one seeded condition nudge rule (goose 00115 is generated from these and the copy; its int test pins them).
type Def struct {
	Key, Msg, Name string // rule_key, the GET answer's key, the admin title
	Program, Link  string
	Kind, Level    string
	Prio           int
	Groups         []rules.Group
	// Fact is the day-count fact of the rule (its qualifying days).
	Fact string
	day  func([]taxonomy.Entry) bool
}

func cnd(fact, op string, v any) rules.Condition {
	return rules.Condition{Fact: fact, Op: op, Value: v}
}

// Defs are the nudge rules in display order: alerts inbox, content / suggestion (a pointer to a program, not a
// warning), «two qualifying days of the current cycle while not enrolled in the program».
var Defs = []Def{
	{Key: KeyPrefix + RuleHeavyPain, Msg: RuleHeavyPain, Name: "Condition nudge: strong pain on several days",
		Program: ProgramEndo, Link: "/programs/pain", Kind: rules.KindContent, Level: rules.LevelSuggestion, Prio: 450,
		Fact: "cond.heavy_pain_days", day: strongPainDay,
		Groups: []rules.Group{{Op: "and", Items: []rules.Condition{cnd("cond.heavy_pain_days", "gte", 2),
			cnd("cond.programs", "nin", []any{ProgramEndo})}}}},
	{Key: KeyPrefix + RuleHeavyBleeding, Msg: RuleHeavyBleeding, Name: "Condition nudge: heavy flow on several days",
		Program: ProgramHeavyBleeding, Link: "/programs/bleeding", Kind: rules.KindContent, Level: rules.LevelSuggestion,
		Prio: 440, Fact: "cond.heavy_bleeding_days", day: heavyFlowDay,
		Groups: []rules.Group{{Op: "and", Items: []rules.Condition{cnd("cond.heavy_bleeding_days", "gte", 2),
			cnd("cond.programs", "nin", []any{ProgramHeavyBleeding})}}}},
}

// Facts are one user's inputs to the rules.
type Facts struct {
	Mode     string          // effective life mode (enums.ResolveLifeMode)
	Enrolled map[string]bool // program → enrolled
	From, To civildate.Date  // the current cycle window (inclusive); zero = no current cycle (v6): nothing is nudged
	Days     []healthlog.DayEntries
}

// Hit is one raised nudge.
type Hit struct {
	Rule, Program, Link string
	Dates               []civildate.Date // qualifying days, oldest first
}

// Holds reports whether the rule of d holds on f for this user (served or code-held; the adapter: its inbox row is
// open).
type Holds func(d *Def, f rules.Facts) bool

// CodeHolds evaluates the code-held rules.
func CodeHolds(d *Def, f rules.Facts) bool {
	r := d.Rule(0)
	return rules.MatchRule(&r, f)
}

// Detect runs the code-held rules on f (pure).
func Detect(f Facts) []Hit { return DetectWith(f, CodeHolds) }

// DetectWith builds the nudges of f whose rules hold, in display order.
func DetectWith(f Facts, holds Holds) []Hit {
	dates := qualifying(f)
	vals := rules.Facts(mfacts.ConditionValues(Input(f)))
	var out []Hit
	for i := range Defs {
		d := &Defs[i]
		if len(dates[d.Key]) > 0 && holds(d, vals) {
			out = append(out, Hit{Rule: d.Msg, Program: d.Program, Link: d.Link, Dates: dates[d.Key]})
		}
	}
	return out
}

// qualifying are the qualifying days of each rule inside the window (none outside a nudged mode or a current cycle).
func qualifying(f Facts) map[string][]civildate.Date {
	out := map[string][]civildate.Date{}
	if !slices.Contains(nudgedModes, f.Mode) || f.From.IsZero() {
		return out
	}
	for _, r := range Defs {
		for _, d := range f.Days {
			if d.Date.Before(f.From) || d.Date.After(f.To) {
				continue
			}
			if r.day(d.Entries) {
				out[r.Key] = append(out[r.Key], d.Date)
			}
		}
	}
	return out
}

// Input is the cond.* fact input of f: the qualifying day counts (0 outside a nudged mode or a current cycle) and the
// enrolled programs.
func Input(f Facts) *mfacts.ConditionInput {
	q := qualifying(f)
	in := &mfacts.ConditionInput{PainDays: len(q[KeyPrefix+RuleHeavyPain]), BleedingDays: len(q[KeyPrefix+RuleHeavyBleeding]),
		Programs: []string{}}
	for p, on := range f.Enrolled {
		if on {
			in.Programs = append(in.Programs, p)
		}
	}
	slices.Sort(in.Programs)
	return in
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

// EngineCycleWindow is the v6 current cycle [current_period_start, today] of the engine status of today (T-M9-37,
// D-136): the engine's anchor (real periods, skips, the declared start) and its lateness rule replace MaxCycleDays and
// the profile length. ok=false — no current cycle, nothing is nudged — without an anchor, when the period is late
// beyond max_late_days (unresolved cycle), or when nothing is predicted (pregnancy, postpartum not returned).
func EngineCycleWindow(st *engine.State, today civildate.Date) (from, to civildate.Date, ok bool) {
	t := engine.DayOf(today)
	s := st.Resolve(t, t)
	start, next := s.Anchors.CurrentPeriodStart, s.Anchors.PredictedNextPeriodStart
	if start == engine.NoDay || next == engine.NoDay || start > t ||
		s.Warnings.Has(engine.WarnCycleUnresolvedAfterExpectedPeriod) {
		return civildate.Date{}, civildate.Date{}, false
	}
	return start.Civil(), today, true
}
