package manager

import (
	"fmt"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Log is one daily_health_logs row as the message system sees it: the Eloquent model
// (extractSymptoms) or its toArray() (PatternLayer). Attr of a column that was not loaded
// reads nil, exactly like PHP reading an attribute the table does not have — which is what
// most of the reads are (deviation D-05, preserved). LoadedLogColumns lists what a Source must
// load; dead_fields_test.go proves every other field read here is absent from the schema.
type Log struct{ attrs map[string]any }

// LoadedLogColumns are the daily_health_logs columns the message system reads that exist.
var LoadedLogColumns = []string{"energy_level", "sleep_quality"}

// NewLog wraps the attributes of one log (column → value after the model casts; NULL = nil).
func NewLog(attrs map[string]any) Log { return Log{attrs: attrs} }

// Attr is `$log->$name` / `$log[$name] ?? null`.
func (l Log) Attr(name string) any { return l.attrs[name] }

// truthy is PHP `if ($log->name)`.
func (l Log) truthy(name string) bool { return phpval.Truthy(l.attrs[name]) }

// isString is PHP `$log->name === $s`.
func (l Log) isString(name, s string) bool {
	v, ok := l.attrs[name].(string)
	return ok && v == s
}

// Context is MessageContext (backend/app/Services/MessageSystem/Core/MessageContext.php).
// Nullable PHP fields are pointers (nil = null); phase and subphase use "" for null.
type Context struct {
	Locale string
	Date   civildate.Date
	Mode   enums.MessageMode

	UserGoal         string
	SubscriptionType string

	// Cycle mode.
	CyclePhase            string
	CycleSubphase         string
	CycleDay              *int
	CycleLength           *int
	IsFertileWindow       bool
	IsPmsWindow           bool
	EstimatedOvulationDay *int

	// Pregnancy mode (0-based gestational weeks, as the manager reads them).
	PregnancyWeek *int
	PregnancyDay  *int
	Trimester     *int
	DueDate       *civildate.Date

	DailyLog   *Log
	Symptoms   []string
	RecentLogs []Log
}

// IsTTC is MessageContext::isTTC().
func (c *Context) IsTTC() bool { return c.UserGoal == "ttc" }

// IsPremium is MessageContext::isPremium().
func (c *Context) IsPremium() bool { return c.SubscriptionType == "premium" }

// IsCycleMode is MessageContext::isCycleMode().
func (c *Context) IsCycleMode() bool { return c.Mode == enums.MessageModeCycle }

// IsPregnancyMode is MessageContext::isPregnancyMode().
func (c *Context) IsPregnancyMode() bool { return c.Mode == enums.MessageModePregnancy }

// HasSymptom is in_array($s, $context->symptoms).
func (c *Context) HasSymptom(s ...string) bool {
	for _, want := range s {
		for _, have := range c.Symptoms {
			if have == want {
				return true
			}
		}
	}
	return false
}

// trimesterOr1 is `$context->trimester ?? 1`.
func (c *Context) trimesterOr1() int {
	if c.Trimester == nil {
		return 1
	}
	return *c.Trimester
}

// GestationalAgeString is MessageContext::getGestationalAgeString() (hard-coded fa / English).
func (c *Context) GestationalAgeString() any {
	if !c.IsPregnancyMode() || c.PregnancyWeek == nil {
		return nil
	}
	day := 0
	if c.PregnancyDay != nil {
		day = *c.PregnancyDay
	}
	if c.Locale == "fa" {
		return fmt.Sprintf("هفته %d و روز %d", *c.PregnancyWeek, day)
	}
	return fmt.Sprintf("Week %d, Day %d", *c.PregnancyWeek, day)
}

// ExtractSymptoms is MessageManager::extractSymptoms (MessageManager.php:289), dead reads
// included: of these fields only energy_level and sleep_quality exist, and sleep_quality
// stores good/medium/bad, so on real data only "low_energy" can fire (D-05).
func ExtractSymptoms(log *Log) []string {
	if log == nil {
		return []string{}
	}
	l := *log
	var s []string
	add := func(cond bool, name string) {
		if cond {
			s = append(s, name)
		}
	}
	// Pain
	add(l.truthy("has_cramps"), "cramps")
	add(l.truthy("has_headache"), "headache")
	add(l.truthy("has_backache"), "backache")
	add(l.truthy("has_breast_tenderness"), "breast_tenderness")
	// Mood
	add(l.isString("mood", "sad") || l.isString("mood", "depressed"), "mood_sad")
	add(l.isString("mood", "anxious") || l.isString("mood", "stressed"), "mood_anxious")
	add(l.isString("mood", "angry") || l.isString("mood", "irritable"), "mood_angry")
	// Flow
	add(l.isString("flow_intensity", "heavy"), "heavy_flow")
	add(l.isString("flow_intensity", "light"), "light_flow")
	// Energy & sleep
	add(l.isString("energy_level", "low") || l.isString("energy_level", "very_low"), "low_energy")
	add(l.isString("sleep_quality", "poor") || l.isString("sleep_quality", "very_poor"), "poor_sleep")
	// Other
	add(l.truthy("has_bloating"), "bloating")
	add(l.truthy("has_nausea"), "nausea")
	add(l.truthy("has_fatigue"), "fatigue")
	add(l.truthy("has_acne"), "acne")
	if s == nil {
		return []string{}
	}
	return s
}

// symptomLogFields are every daily-log field extractSymptoms and PatternLayer read (for the
// dead-field test).
var symptomLogFields = []string{
	"has_cramps", "has_headache", "has_backache", "has_breast_tenderness", "mood", "flow_intensity",
	"energy_level", "sleep_quality", "has_bloating", "has_nausea", "has_fatigue", "has_acne",
	// PatternLayer only
	"has_sad", "has_angry", "cramp_severity",
}
