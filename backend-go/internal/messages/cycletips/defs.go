// Package cycletips holds the cycle home's «نکتهٔ هوشمند» as message-engine rules (D-216, user request 2026-11-28:
// one decision layer for every message). It replaces the cycle part of the legacy manager bundle
// (GET /messages/daily, internal/messages/manager): the phase message, the symptom override and the correlation
// insight become smart_tip rules (rule_key tip.*) an admin edits, approves and measures like any other rule.
//
// The seed (goose 00133) is generated from Defs and pinned by the migration int test: the copy is the legacy
// defaults (internal/messages/content/defaults.json — base language as the content, every other language under
// content.translations), the conditions read the engine's facts and today's log (facts.SymptomsToday) instead of
// the legacy detectors, whose columns no longer exist (D-05).
//
// Not migrated, on purpose: the nutrition / sleep / exercise modules and the pattern layer. The card only reached
// them when the phase message was empty, which never happens for a user with a cycle, and the pattern detectors
// were dead code (D-05); their message groups are retired with the fallback.
package cycletips

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// KeyPrefix starts every rule_key of this package.
const KeyPrefix = "tip."

// Slot, Level and Kind of every rule: the smart-tip slot only takes clinical messages (rules.SlotKinds).
const (
	Slot  = "smart_tip"
	Level = rules.LevelInfo
	Kind  = rules.KindClinical
)

// Priorities: a combination of today's symptoms is more specific than one symptom, which is more specific than the
// phase message. Inside a tier the legacy order is kept (manager: determineOverride, correlationRules).
const (
	prioCorrelation = 700
	prioSymptom     = 600
	prioPhase       = 200
	prioDefault     = 150
)

// Def is one seeded rule.
type Def struct {
	Key    string
	Name   string // the internal title
	Prio   int
	Groups []rules.Group
	Topics []string

	group, item string            // the legacy defaults entry the copy comes from
	titles      map[string]string // locale → title, for the entries that have none of their own
}

func cond(fact, op string, value any) rules.Condition {
	return rules.Condition{Fact: fact, Op: op, Value: value}
}
func all(items ...rules.Condition) []rules.Group { return []rules.Group{{Op: "and", Items: items}} }
func has(code string) rules.Condition            { return cond(facts.SymptomsToday, "contains", code) }
func oneOf(codes ...string) rules.Condition      { return cond(facts.SymptomsToday, "in", codes) }

var cycleModes = cond("mode", "in", []string{"cycle", "ttc"})

// phases: the legacy phase item → the engine's main_phase values it covers (the late luteal «period expected» days
// read the luteal copy; the legacy «ovulation» copy is the fertile phase's).
var phases = []struct {
	item   string
	main   []string
	topics []string
}{
	{"menstruation", []string{"menstrual"}, nil},
	{"follicular", []string{"follicular"}, nil},
	{"ovulation", []string{"fertile"}, []string{rules.TopicFertileWindow}},
	{"luteal", []string{"luteal", "period_expected"}, nil},
}

// symptoms: the layer-2 overrides in the legacy priority order.
var symptoms = []struct {
	item  string
	when  rules.Condition
	fa    string
	en    string
	label string
}{
	{"heavy_flow", has("heavy_flow"), "خونریزی زیاد", "Heavy flow", "heavy flow"},
	{"pain", oneOf("cramps", "headache", "backache"), "درد", "Pain", "pain"},
	{"mood_sad", oneOf("mood_sad", "mood_anxious"), "غم یا اضطراب", "Feeling low or anxious", "low or anxious mood"},
	{"mood_angry", has("mood_angry"), "عصبانیت", "Feeling angry", "anger"},
	{"low_energy", oneOf("low_energy", "fatigue"), "کم\u200cانرژی و خستگی", "Low energy", "low energy"},
	{"bloating", has("bloating"), "نفخ", "Bloating", "bloating"},
	{"acne", has("acne"), "جوش پوست", "Acne", "acne"},
}

// correlations: the layer-3 insights in the legacy order (premium ones keep is_premium).
var correlations = []struct {
	group, item string
	when        []rules.Condition
	topics      []string
	fa, en      string
}{
	{"correlation_cycle", "sleep_mood", []rules.Condition{has("poor_sleep"), oneOf("mood_sad", "mood_angry")}, nil,
		"خواب و خلق", "Sleep and mood"},
	{"correlation_cycle", "stress_physical", []rules.Condition{has("mood_anxious"), oneOf("headache", "cramps")}, nil,
		"استرس و علائم جسمی", "Stress and the body"},
	{"correlation_cycle", "pms_symptoms", []rules.Condition{cond("is_pms_window", "eq", true), oneOf("bloating", "mood_sad")}, nil,
		"روزهای پیش از قاعدگی", "Premenstrual days"},
	{"correlation_cycle", "fertile_window_ttc", []rules.Condition{cond("mode", "eq", "ttc"), cond("is_fertile_window", "eq", true)},
		[]string{rules.TopicFertileWindow}, "پنجرهٔ باروری", "Fertile window"},
	{"correlation_cycle", "luteal_energy_drop", []rules.Condition{cond("is_premium", "eq", true),
		cond("main_phase", "eq", "luteal"), has("low_energy")}, nil, "افت انرژی در فاز لوتئال", "Luteal energy dip"},
	{"correlation_cycle", "hormone_skin", []rules.Condition{cond("is_premium", "eq", true),
		cond("main_phase", "in", []string{"luteal", "menstrual"}), has("acne")}, nil, "پوست و هورمون\u200cها", "Skin and hormones"},
	{"correlation_common", "dehydration_indicator", []rules.Condition{has("headache"), has("fatigue")}, nil,
		"شاید کم\u200cآبی", "Possibly dehydration"},
	{"correlation_common", "sleep_deprivation", []rules.Condition{has("poor_sleep"), has("low_energy")}, nil,
		"کم\u200cخوابی", "Short on sleep"},
}

// Defs are the seeded rules, highest tier first.
var Defs = buildDefs()

func buildDefs() []Def {
	var out []Def
	for i, c := range correlations {
		items := c.when
		if !slices.ContainsFunc(items, func(x rules.Condition) bool { return x.Fact == "mode" }) {
			items = append([]rules.Condition{cycleModes}, items...)
		}
		out = append(out, Def{
			Key: KeyPrefix + "corr." + c.item, Name: "Smart tip: " + strings.ReplaceAll(c.item, "_", " "),
			Prio: prioCorrelation + 10*(len(correlations)-i), Groups: all(items...), Topics: c.topics,
			group: c.group, item: c.item, titles: map[string]string{"fa": c.fa, "en": c.en},
		})
	}
	for i, s := range symptoms {
		out = append(out, Def{
			Key: KeyPrefix + "symptom." + s.item, Name: "Smart tip: " + s.label + " logged today",
			Prio: prioSymptom + 10*(len(symptoms)-i), Groups: all(cycleModes, s.when),
			group: "cycle_override", item: s.item, titles: map[string]string{"fa": s.fa, "en": s.en},
		})
	}
	for _, m := range []struct{ mode, group string }{{"cycle", "cycle_base_non_ttc"}, {"ttc", "cycle_base_ttc"}} {
		for _, p := range phases {
			out = append(out, Def{
				Key: KeyPrefix + m.mode + "." + p.main[0], Name: "Smart tip: " + m.mode + " · " + p.main[0] + " phase",
				Prio: prioPhase, Groups: all(cond("mode", "eq", m.mode), cond("main_phase", "in", p.main)),
				Topics: p.topics, group: m.group, item: p.item,
			})
		}
		out = append(out, Def{
			Key: KeyPrefix + m.mode + ".default", Name: "Smart tip: " + m.mode + " · phase not known",
			Prio: prioDefault, Groups: all(cond("mode", "eq", m.mode)), group: m.group, item: "default",
		})
	}
	return out
}

// suggestion introduces the legacy «action» line inside the body, per language (the fallback for any other one).
var suggestion = map[string]string{"fa": "پیشنهاد: ", "en": "Try this: "}

var defaults = sync.OnceValue(func() map[string]map[string]map[string]map[string]any {
	var d map[string]map[string]map[string]map[string]any
	if err := json.Unmarshal(content.DefaultsJSON(), &d); err != nil {
		panic(fmt.Errorf("cycletips: defaults: %w", err))
	}
	return d
})

func str(m map[string]any, key string) string { s, _ := m[key].(string); return strings.TrimSpace(s) }

// variant is d's title and body in one language: the phase copy has its own short line as the title; a symptom or
// correlation gets the name in titles. The body is the message followed by the suggested action.
func (d Def) variant(locale string) (rules.Variant, bool) {
	p, ok := defaults()[d.group][d.item][locale]
	if !ok {
		return rules.Variant{}, false
	}
	title, body := d.titles[locale], ""
	switch {
	case strings.HasPrefix(d.group, "cycle_base"):
		title, body = str(p, "short"), str(p, "long")
	case d.group == "cycle_override":
		body = str(p, "message")
	default:
		body = str(p, "insight_message")
	}
	if a := str(p, "action"); a != "" {
		lead, ok := suggestion[locale]
		if !ok {
			lead = suggestion["en"]
		}
		body = strings.TrimSpace(body + " " + lead + a)
	}
	return rules.Variant{Title: title, Body: body}, title != "" && body != ""
}

// Languages are the languages the legacy defaults carry for every Def, sorted.
func Languages() []string {
	var out []string
	for code := range defaults()[Defs[0].group][Defs[0].item] {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}

// ContentJSON is the content column of d with base the default language's code.
func (d Def) ContentJSON(base string) (string, error) {
	b, ok := d.variant(base)
	if !ok {
		return "", fmt.Errorf("cycletips: no %s copy for %s", base, d.Key)
	}
	sc := struct {
		rules.Content
		Translations map[string]rules.Content `json:"translations,omitempty"`
	}{Content: rules.Content{A: b}, Translations: map[string]rules.Content{}}
	for _, code := range Languages() {
		if t, ok := d.variant(code); ok && code != base {
			sc.Translations[code] = rules.Content{A: t}
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

// TopicsJSON is the topics column of d ("" = NULL).
func (d Def) TopicsJSON() string {
	if len(d.Topics) == 0 {
		return ""
	}
	b, _ := json.Marshal(d.Topics)
	return string(b)
}
