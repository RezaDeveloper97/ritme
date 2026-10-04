package registry

import (
	"github.com/ritme/backend-go/internal/enums"
)

// AlertGroup holds the v2 alert rules (one row per rule and locale) and the level legend.
const AlertGroup = "pregnancy_alert"

// LegendKey is the non-rule item of AlertGroup (the level legend of the Alerts screen).
const LegendKey = "legend"

// AlertLevels are the four v2 levels (info → urgent).
var AlertLevels = []string{"info", "suggestion", "follow_up", "urgent"}

// AlertActions are the action keys an alert card can offer (the app maps each to a behaviour).
var AlertActions = []string{"ack", "add_to_visit_note", "log_weight", "open_week", "call"}

// LogSymptoms are the symptom toggles of the Log screen (severity mild|moderate|severe).
var LogSymptoms = []string{
	"nausea", "vomiting", "fatigue", "headache", "back_pain", "breast_pain", "heartburn", "constipation", "spotting",
}

// CriticalSymptoms are the v1 critical symptoms (internal/pregnancy/alerts).
var CriticalSymptoms = []string{"spotting", "bleeding", "fluid_leakage", "severe_sudden_pain"}

// Behaviour keys of an alert payload: they drive the engine and are kept identical in every
// locale's row (the engine reads them from the default-language row).
var AlertBehaviourKeys = []string{"enabled", "level", "window_days", "params"}

// AlertTextKeys are the per-locale texts of an alert payload.
var AlertTextKeys = []string{"title", "what_we_saw", "how_sure", "advice", "actions", "contact"}

// AlertRule is one detector of the rule registry with its typed params.
type AlertRule struct {
	Key          string
	Params       []Field
	Placeholders []string
}

func intParam(key string, lo, hi int) Field { return Field{Key: key, Kind: KindInt, Min: lo, Max: hi} }

func fetalStatuses() []string {
	out := []string{}
	for _, s := range enums.FetalMovementStatusCases() {
		out = append(out, string(s))
	}
	return out
}

// AlertRules are the v2 rules in display order (params per docs/pregnancy-v2/README.md; the
// detectors are T-M7-04's). Placeholders are the `{name}`s their texts may use.
var AlertRules = []AlertRule{
	{Key: "vomiting_streak", Params: []Field{
		intParam("min_streak_days", 2, 14), intParam("severe_min_count", 0, 14),
	}, Placeholders: []string{"days", "severe_count"}},
	{Key: "severe_symptom_count", Params: []Field{
		intParam("min_count", 1, 50),
		{Key: "symptoms", Kind: KindEnumList, Values: LogSymptoms, MinItems: 1},
	}, Placeholders: []string{"count"}},
	{Key: "critical_symptom", Params: []Field{
		{Key: "symptoms", Kind: KindEnumList, Values: CriticalSymptoms, MinItems: 1},
		intParam("spotting_until_week", 0, MaxWeek),
	}, Placeholders: []string{"symptom"}},
	// from_weekday: 0-based day of the pregnancy week from which the rule fires (review #11); null
	// or missing = the engine default (pregnancyalerts.WeightMissingFromWeekday).
	{Key: "weight_missing_week", Params: []Field{
		intParam("from_week", MinWeek, MaxWeek),
		{Key: "from_weekday", Kind: KindInt, Min: 0, Max: 6, Nullable: true},
	}, Placeholders: []string{"week"}},
	{Key: "week_entered", Params: []Field{}, Placeholders: []string{"week", "basis"}},
	{Key: "bp_high", Params: []Field{
		intParam("systolic_min", 90, 200), intParam("diastolic_min", 50, 130),
	}, Placeholders: []string{"systolic", "diastolic"}},
	{Key: "sugar_high", Params: []Field{
		intParam("fasting_max", 60, 200), intParam("post_meal_max", 80, 300),
	}, Placeholders: []string{"fasting", "post_meal"}},
	{Key: "fetal_movement", Params: []Field{
		intParam("from_week", 12, MaxWeek),
		{Key: "statuses", Kind: KindEnumList, Values: fetalStatuses(), MinItems: 1},
	}, Placeholders: []string{"status"}},
	// 5-1-1 of the contraction timer (bloom B-N5-03, internal/pregnancy/labor): average interval at most
	// interval_max_minutes, average duration at least duration_min_seconds, sustained for run_minutes. {interval} and
	// {duration} are m:ss.
	{Key: "contractions_511", Params: []Field{
		intParam("interval_max_minutes", 2, 15), intParam("duration_min_seconds", 20, 120), intParam("run_minutes", 20, 180),
	}, Placeholders: []string{"count", "minutes", "interval", "duration"}},
}

// FindAlertRule looks a rule up by key.
func FindAlertRule(key string) (AlertRule, bool) {
	for _, r := range AlertRules {
		if r.Key == key {
			return r, true
		}
	}
	return AlertRule{}, false
}

// Window days of a rule.
const (
	MinWindowDays = 1
	MaxWindowDays = 30
)

// AlertBehaviourFields are the behaviour fields of a rule.
func AlertBehaviourFields(r AlertRule) []Field {
	return []Field{
		{Key: "enabled", Kind: KindBool},
		{Key: "level", Kind: KindEnum, Values: AlertLevels},
		intParam("window_days", MinWindowDays, MaxWindowDays),
		{Key: "params", Kind: KindObject, Fields: r.Params},
	}
}

// AlertTextFields are the per-locale text fields of every rule.
func AlertTextFields() []Field {
	return []Field{
		text("title", 255),
		text("what_we_saw", 500),
		text("how_sure", 500),
		text("advice", 2000),
		{Key: "actions", Kind: KindObjectList, MaxItems: 4, Fields: []Field{
			{Key: "key", Kind: KindEnum, Values: AlertActions},
			text("label", 60),
		}},
		optText("contact", 500),
	}
}

// AlertFields is the full stored payload of a rule row, in stored order.
func AlertFields(r AlertRule) []Field {
	return append(AlertBehaviourFields(r), AlertTextFields()...)
}

func legendFields() []Field {
	level := Field{Kind: KindObject, Fields: []Field{text("label", 60), text("description", 500)}}
	levels := make([]Field, 0, len(AlertLevels))
	for _, l := range AlertLevels {
		f := level
		f.Key = l
		levels = append(levels, f)
	}
	return []Field{
		text("window_note", 255),
		text("title", 255),
		{Key: "levels", Kind: KindObject, Fields: levels},
		text("disclaimer", 1000),
	}
}

func alertGroup() Group {
	g := Group{Name: AlertGroup}
	for _, r := range AlertRules {
		g.Items = append(g.Items, Item{
			Group: AlertGroup, Key: r.Key, Fields: AlertFields(r), Placeholders: r.Placeholders, Typed: true,
		})
	}
	g.Items = append(g.Items, Item{
		Group: AlertGroup, Key: LegendKey, Fields: legendFields(), Placeholders: []string{"days"}, Typed: true,
	})
	return g
}
