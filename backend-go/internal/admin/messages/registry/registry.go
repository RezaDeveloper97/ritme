package registry

import (
	"slices"
	"strconv"

	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// Item is one registered (group, item_key) with its payload schema.
type Item struct {
	Group        string
	Key          string
	Fields       []Field
	Placeholders []string // `{name}` placeholders the consumer fills (documentation for editors)
	// Typed is true for explicit schemas (pregnancy v2): PUT /messages/:id validates the merged
	// payload against Fields instead of the legacy shape-keeping merge, which would flatten
	// objects and lists of objects.
	Typed bool
}

// Group is a registered group and its items in order.
type Group struct {
	Name  string
	Items []Item
}

// Groups returns every registered group: the pregnancy v2 groups first, then the smart-message
// groups of the code fallback in export order.
func Groups() []Group { return groups }

// Lookup finds a registered item.
func Lookup(group, key string) (Item, bool) {
	for _, g := range groups {
		if g.Name != group {
			continue
		}
		for _, it := range g.Items {
			if it.Key == key {
				return it, true
			}
		}
		return Item{}, false
	}
	return Item{}, false
}

// GroupNames lists the registered group names.
func GroupNames() []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.Name)
	}
	return out
}

// Keys lists a group's item keys (nil for an unknown group).
func Keys(group string) []string {
	for _, g := range groups {
		if g.Name == group {
			out := make([]string, 0, len(g.Items))
			for _, it := range g.Items {
				out = append(out, it.Key)
			}
			return out
		}
	}
	return nil
}

var groups = buildGroups()

func buildGroups() []Group {
	out := []Group{weekTipGroup(), alertGroup(), setupGroup(), conditionNudgeGroup(), menopauseMessageGroup()}
	defaults, _ := phpval.Decode(content.DefaultsJSON())
	for _, name := range content.Groups() {
		if slices.ContainsFunc(out, func(g Group) bool { return g.Name == name }) {
			continue
		}
		g := Group{Name: name}
		for _, key := range content.Items(name) {
			g.Items = append(g.Items, Item{Group: name, Key: key, Fields: derive(exampleOf(defaults, name, key))})
		}
		out = append(out, g)
	}
	return out
}

// exampleOf is the first language's copy of a code-fallback entry (the export's shape; the
// language itself does not matter, every copy has the same keys).
func exampleOf(defaults any, group, key string) any {
	entry, _ := phpval.Get(defaults, group+"."+key)
	_, copies := phpval.Entries(entry)
	if len(copies) == 0 {
		return nil
	}
	return copies[0]
}

// derive builds a schema from an example payload: strings → text (empty example → nullable),
// arrays of strings → text_list, numbers → integer, booleans → boolean, objects → object.
func derive(example any) []Field {
	keys, vals := phpval.Entries(example)
	fields := make([]Field, 0, len(keys))
	for i, k := range keys {
		switch v := vals[i].(type) {
		case string:
			fields = append(fields, Field{Key: k, Kind: KindText, Nullable: v == ""})
		case nil:
			fields = append(fields, Field{Key: k, Kind: KindText, Nullable: true})
		case bool:
			fields = append(fields, Field{Key: k, Kind: KindBool})
		case int64, float64:
			fields = append(fields, Field{Key: k, Kind: KindInt, Min: 0, Max: 1_000_000})
		case []any:
			fields = append(fields, Field{Key: k, Kind: KindTextList})
		case phpval.Map:
			fields = append(fields, Field{Key: k, Kind: KindObject, Fields: derive(v)})
		}
	}
	return fields
}

// ---------------------------------------------------------------------------
// Pregnancy v2 groups (docs/pregnancy-v2/README.md § Stored shapes)

// Weeks of pregnancy v2.
const (
	MinWeek = 1
	MaxWeek = 42
)

func text(key string, maxLen int) Field { return Field{Key: key, Kind: KindText, MaxLen: maxLen} }

func optText(key string, maxLen int) Field {
	return Field{Key: key, Kind: KindText, MaxLen: maxLen, Nullable: true}
}

// WeekTipGroup is the smart tip of each week.
const WeekTipGroup = "pregnancy_week_tip"

func weekTipGroup() Group {
	fields := []Field{
		text("title", 255),
		text("body", 2000),
		{Key: "read_minutes", Kind: KindInt, Min: 1, Max: 60},
		{Key: "article_url", Kind: KindURL, Nullable: true},
	}
	g := Group{Name: WeekTipGroup}
	for w := MinWeek; w <= MaxWeek; w++ {
		g.Items = append(g.Items, Item{Group: WeekTipGroup, Key: strconv.Itoa(w), Fields: fields, Typed: true})
	}
	return g
}

// SetupGroup is the copy of the Setup screen.
const SetupGroup = "pregnancy_setup"

func setupGroup() Group {
	source := []Field{text("label", 120), text("hint", 500)}
	items := []struct {
		key          string
		fields       []Field
		placeholders []string
	}{
		{"welcome", []Field{
			text("title", 255), text("body", 1000),
			{Key: "benefits", Kind: KindTextList, MaxLen: 255, MaxItems: 6},
			text("primary", 120), text("secondary", 120),
		}, nil},
		{"dating", []Field{text("title", 255), text("body", 1000)}, nil},
		{"source_lmp", source, nil},
		{"source_ultrasound", source, nil},
		{"source_manual", source, nil},
		{"history", []Field{text("title", 255), text("body", 1000), text("disclaimer", 1000), text("skip", 120)}, nil},
		{"result", []Field{
			text("lead", 255), text("suffix", 255), text("due_label", 255), text("confidence", 255),
			text("range", 500), text("basis_lmp", 500), text("basis_ultrasound", 500), text("basis_manual", 500),
			text("primary", 120), text("secondary", 120),
		}, []string{"confidence", "range_from", "range_to", "date"}},
		{"due_disclaimer", []Field{text("title", 255), text("body", 1000)}, []string{"range_from", "range_to"}},
		// The pregnancy calendar's source note (design audit E2): the care-plan caveat, then the
		// dating-basis sentence of the user's source (T-M7-20).
		{"calendar_note", []Field{
			text("plan_note", 500), text("basis_lmp", 500), text("basis_ultrasound", 500), text("basis_manual", 500),
		}, nil},
		// The calm pregnancy exit of the mode switcher (B-N2-03, GET /profile/life-stage/loss-copy): the confirm
		// step, then the closing note. Unseeded — the app falls back to its bundle until an admin writes it.
		{"loss_exit", []Field{
			text("title", 255), text("body", 2000), text("confirm", 120), text("cancel", 120),
			text("done_title", 255), text("done_body", 2000), text("done_action", 120),
		}, nil},
	}
	g := Group{Name: SetupGroup}
	for _, it := range items {
		g.Items = append(g.Items, Item{Group: SetupGroup, Key: it.key, Fields: it.fields, Placeholders: it.placeholders, Typed: true})
	}
	return g
}

// ---------------------------------------------------------------------------
// Menopause (roadmap E02)

// MenopauseMessageGroup is the copy of the menopause reminders and alerts (CB-MENO-12,
// internal/messages/menomessages: Group, TextRules, TextKeys — kept equal by registry_test). Unseeded: the engine
// falls back to its embedded copy, per field, until an admin writes a row. The bleeding alert and the stage tips are
// catalog items (meno_alerts / meno_tips), edited in the catalog, not here.
const MenopauseMessageGroup = "menopause_message"

// MenopauseMessageRule is one rule (= item key) of MenopauseMessageGroup and its `{name}` placeholders.
type MenopauseMessageRule struct {
	Key          string
	Placeholders []string
}

// MenopauseMessageRules are the rules in display order.
var MenopauseMessageRules = []MenopauseMessageRule{
	{"checkup_overdue", []string{"checkup", "count"}},
	{"score_worsened", []string{"total", "previous", "delta"}},
	{"hrt_review", []string{"name", "date", "when", "days"}},
	{"checkup_due", []string{"checkup", "count"}},
}

func menopauseMessageGroup() Group {
	fields := []Field{text("title", 255), text("body", 1000), text("action", 120)}
	g := Group{Name: MenopauseMessageGroup}
	for _, r := range MenopauseMessageRules {
		g.Items = append(g.Items, Item{
			Group: MenopauseMessageGroup, Key: r.Key, Fields: fields, Placeholders: r.Placeholders, Typed: true,
		})
	}
	return g
}

// ---------------------------------------------------------------------------
// Condition programs (roadmap E05)

// ConditionNudgeGroup is the copy of the heavy pain / heavy bleeding nudges (CB-COND-06,
// internal/messages/conditionnudges: Group, Rules, TextKeys — kept equal by registry_test). Unseeded: the engine
// falls back to its embedded copy, per field, until an admin writes a row. `{days}` is the qualifying day count.
const ConditionNudgeGroup = "condition_nudge"

// ConditionNudgeRules are the nudge rule keys (= item keys) in display order.
var ConditionNudgeRules = []string{"heavy_pain", "heavy_bleeding"}

func conditionNudgeGroup() Group {
	fields := []Field{text("title", 255), text("body", 1000), text("action", 120), text("doctor_action", 120)}
	g := Group{Name: ConditionNudgeGroup}
	for _, k := range ConditionNudgeRules {
		g.Items = append(g.Items, Item{
			Group: ConditionNudgeGroup, Key: k, Fields: fields, Placeholders: []string{"days"}, Typed: true,
		})
	}
	return g
}
