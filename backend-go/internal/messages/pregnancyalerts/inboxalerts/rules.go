// Package inboxalerts is the message-engine side of the pregnancy alerts (T-M10-08, D-165, DECISIONS.md M-10, H-02;
// SPEC §5.4.1, §4.2): the seeded preg.* rules of the alerts_inbox slot, the code-held definitions of the locked safety
// rules (the emergency guarantee: a missing or inactive row still raises the alert with the embedded copy), the
// per-call evaluation over the user's pregnancy data, and the pregnancyalerts.Backend the pregnancy paths call
// (rendering inbox rows in the v2 alert shape for /pregnancy/v2/alerts* and the write responses).
package inboxalerts

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// KeyPrefix is the rule_key prefix of the pregnancy alert rules.
const KeyPrefix = "preg."

// Def is one seeded pregnancy alert rule (the migration 00113 rows are generated from these; the int test pins them).
type Def struct {
	Key    string // rule_key
	Legacy string // the v2 rule key the pregnancy clients read (rule_key in the v2 alert shape)
	Name   string // the admin title
	Kind   string
	Level  string // stored level (rules.Level*)
	Prio   int
	Locked bool // safety: only the medical advisor edits; the code-held copy stands in when the row is gone or off
	Groups []rules.Group
	// Actions are the v2 action keys (labels from pregnancyalerts lang `actions.<key>`; call_maternity is the "call"
	// key with the maternity unit label).
	Actions []string
	// Time marks a calendar alert (returned by the daily evaluation only).
	Time bool
	// Readings marks a rule a vitals reading can raise (EvaluateReadings).
	Readings bool
}

func all(items ...rules.Condition) []rules.Group { return []rules.Group{{Op: "and", Items: items}} }

func anyOf(items ...rules.Condition) []rules.Group { return []rules.Group{{Op: "or", Items: items}} }

func c(fact, op string, v any) rules.Condition { return rules.Condition{Fact: fact, Op: op, Value: v} }

var (
	followUp = []string{"ack"}
	call     = []string{"call", "ack"}
)

// Defs are the pregnancy alert rules (SPEC §5.4.1 table, plus the two v2/v1 follow-ups the table does not list:
// severe symptoms of the week and severe depression feelings, so no escalation of today's engines is lost).
var Defs = []Def{
	{Key: "preg.vomiting", Legacy: "vomiting_streak", Name: "Pregnancy: repeated vomiting", Kind: rules.KindClinical,
		Level: rules.LevelWorthFollowUp, Prio: 600, Actions: followUp,
		Groups: anyOf(c("preg.vomiting_days_7", "gte", 4), c("preg.vomiting_severe_7", "gte", 2), c("preg.vomiting_streak", "gte", 3))},
	{Key: "preg.cant_keep_fluids", Legacy: "cant_keep_fluids", Name: "Pregnancy: can't keep fluids down", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: call,
		Groups: all(c("preg.cant_keep_fluids", "eq", true))},
	{Key: "preg.severe_symptoms", Legacy: "severe_symptom_count", Name: "Pregnancy: severe symptoms this week", Kind: rules.KindClinical,
		Level: rules.LevelWorthFollowUp, Prio: 550, Actions: followUp,
		Groups: all(c("preg.severe_symptoms_7", "gte", 3))},
	{Key: "preg.critical_symptom", Legacy: "critical_symptom", Name: "Pregnancy: critical symptom", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: call,
		Groups: all(c("preg.critical_symptom", "in", []any{"bleeding", "fluid_leakage", "severe_sudden_pain", "spotting"}))},
	{Key: "preg.weight_missing", Legacy: "weight_missing_week", Name: "Pregnancy: weight not logged this week", Kind: rules.KindProduct,
		Level: rules.LevelSuggestion, Prio: 50, Actions: []string{"log_weight", "ack"}, Time: true,
		Groups: all(c("preg.days_since_weight", "gte", 7))},
	{Key: "preg.week_entered", Legacy: "week_entered", Name: "Pregnancy: new week", Kind: rules.KindContent,
		Level: rules.LevelInfo, Prio: 150, Actions: []string{"open_week", "ack"}, Time: true,
		Groups: all(c("pregnancy_week", "gte", 1))},
	{Key: "preg.next_visit", Legacy: "next_visit", Name: "Pregnancy: next visit soon", Kind: rules.KindContent,
		Level: rules.LevelInfo, Prio: 160, Actions: []string{"ack"}, Time: true,
		Groups: all(c("preg.next_visit_in_days", "lte", 3))},
	{Key: "preg.fetal_movement", Legacy: "fetal_movement", Name: "Pregnancy: reduced or no fetal movement", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: call,
		Groups: all(c("preg.fetal_movement", "in", []any{"reduced", "none"}))},
	{Key: "preg.fetal_movement_kicks", Legacy: "fetal_movement", Name: "Pregnancy: kick count short of the target", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: call,
		Groups: all(c("preg.fetal_movement", "eq", "low_session"))},
	{Key: "preg.bp_high", Legacy: "bp_high", Name: "Pregnancy: BP ≥ 140/90 (follow-up level)", Kind: rules.KindClinical,
		Level: rules.LevelWorthFollowUp, Prio: 700, Actions: call, Readings: true,
		Groups: all(c("preg.bp_last_category", "eq", "high"), c("preg.bp_high_level", "eq", "follow_up"))},
	{Key: "preg.bp_high_urgent", Legacy: "bp_high", Name: "Pregnancy: BP ≥ 140/90 (urgent level)", Kind: rules.KindClinical,
		Level: rules.LevelFollowSoon, Prio: 790, Actions: call, Readings: true,
		Groups: all(c("preg.bp_last_category", "eq", "high"), c("preg.bp_high_level", "eq", "urgent"))},
	{Key: "preg.bp_severe", Legacy: "bp_high", Name: "Pregnancy: BP ≥ 160/110", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: call, Readings: true,
		Groups: all(c("preg.bp_last_category", "eq", "severe"))},
	{Key: "preg.gdm", Legacy: "sugar_high", Name: "Pregnancy: blood sugar above the GDM targets", Kind: rules.KindClinical,
		Level: rules.LevelWorthFollowUp, Prio: 650, Actions: followUp, Readings: true,
		Groups: all(c("preg.gdm_last", "eq", "above_target"))},
	{Key: "preg.contractions_511", Legacy: "contractions_511", Name: "Pregnancy: contractions 5-1-1", Kind: rules.KindSafety,
		Level: rules.LevelFollowSoon, Prio: 1000, Locked: true, Actions: []string{"call_maternity", "ack"},
		Groups: all(c("preg.contractions_511", "eq", true))},
	{Key: "preg.depression", Legacy: "depression_severe", Name: "Pregnancy: severe depression feelings", Kind: rules.KindClinical,
		Level: rules.LevelWorthFollowUp, Prio: 560, Actions: followUp,
		Groups: all(c("preg.depression_severe", "eq", true))},
}

// DefOf is the definition of a rule key.
func DefOf(key string) (Def, bool) {
	i := slices.IndexFunc(Defs, func(d Def) bool { return d.Key == key })
	if i < 0 {
		return Def{}, false
	}
	return Defs[i], true
}

// Copy is one rule's texts in one language.
type Copy struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	Seen       string `json:"seen,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Contact    string `json:"contact,omitempty"`
}

// The embedded copy: copy/<code>.json, rule key → texts. It is the seed of the rules' content (the default language as
// the base, every other file as content.translations.<code>) and the code-held copy of the locked rules.
//
//go:embed copy/*.json
var copyFS embed.FS

var embedded = sync.OnceValue(func() map[string]map[string]Copy {
	out := map[string]map[string]Copy{}
	entries, err := copyFS.ReadDir("copy")
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		b, err := copyFS.ReadFile("copy/" + e.Name())
		if err != nil {
			panic(err)
		}
		var m map[string]Copy
		if err := json.Unmarshal(b, &m); err != nil {
			panic(fmt.Errorf("inboxalerts: copy %s: %w", e.Name(), err))
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = m
	}
	return out
})

// CopyOf is the embedded copy of key in locale (the fallback language when the locale has none).
func CopyOf(key, locale string) (Copy, bool) {
	all := embedded()
	if c, ok := all[locale][key]; ok {
		return c, true
	}
	c, ok := all[lang.FallbackLocale][key]
	return c, ok
}

// Languages are the codes of the embedded copy files, sorted.
func Languages() []string {
	out := make([]string, 0, len(embedded()))
	for code := range embedded() {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}

func (c Copy) content() rules.Content {
	return rules.Content{A: rules.Variant{Title: c.Title, Body: c.Body}, Seen: c.Seen, Confidence: c.Confidence,
		Contact: c.Contact}
}

// storedContent is a message_rules content column: the base language's texts plus content.translations.<code>.
type storedContent struct {
	rules.Content
	Translations map[string]rules.Content `json:"translations,omitempty"`
}

// ContentJSON is the content column of d with base the default language's code.
func (d Def) ContentJSON(base string) (string, error) {
	b, ok := CopyOf(d.Key, base)
	if !ok {
		return "", fmt.Errorf("inboxalerts: no copy for %s", d.Key)
	}
	sc := storedContent{Content: b.content(), Translations: map[string]rules.Content{}}
	for _, code := range Languages() {
		if code == base {
			continue
		}
		if t, ok := embedded()[code][d.Key]; ok {
			sc.Translations[code] = t.content()
		}
	}
	out, err := json.Marshal(sc)
	return string(out), err
}

// ConditionsJSON is the conditions column of d.
func (d Def) ConditionsJSON() string {
	b, _ := json.Marshal(d.Groups)
	return string(b)
}

// Rule is the code-held engine rule of d with id (the emergency guarantee). Its content is the fallback language's
// copy; Render reads the request language's copy directly.
func (d Def) Rule(id uint64) rules.Rule {
	cp, _ := CopyOf(d.Key, lang.FallbackLocale)
	return rules.Rule{ID: id, Level: d.Level, Kind: d.Kind, Priority: d.Prio, Placement: rules.SlotAlertsInbox,
		Content: cp.content(), Groups: d.Groups, HoursFrom: -1, HoursTo: -1, AudiencePercent: 100,
		Status: rules.StatusApproved, IsActive: true, RuleKey: d.Key, Version: 1, ApprovedVersion: 1, Locked: d.Locked}
}

// CodeRuleIDBase is the first id of a code-held rule whose row was deleted (no message_rules id is that large).
const CodeRuleIDBase uint64 = 9_000_000_000_000

// CodeRuleID is the stable id of the code-held rule of a deleted row.
func CodeRuleID(key string) uint64 {
	return CodeRuleIDBase + uint64(max(0, slices.IndexFunc(Defs, func(d Def) bool { return d.Key == key }))) //nolint:gosec // small index
}
