package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Variables are the placeholders a message text may use ({name}); each must be a fact the engine supplies.
// T-M9-38: pms_start_day and top_symptoms were removed — no fact ever filled them, so a text using them never rendered.
var Variables = []string{"name", "days_to_period", "days_late", "cycle_day", "pregnancy_week", "phase", "subphase"}

// CapPeriods are the valid cap periods.
var CapPeriods = []string{"day", "week", "cycle"}

// Input is the admin editor's payload. Type, Slot and Name are the spec names of Kind, Placement and Title (M-01),
// accepted as aliases; Normalize folds them (and the spec level names) onto the canonical fields.
type Input struct {
	Title           string      `json:"title"`
	Level           string      `json:"level"`
	Kind            string      `json:"kind"`
	Priority        int         `json:"priority"`
	Placement       string      `json:"placement"`
	Content         Content     `json:"content"`
	VariantsEnabled bool        `json:"variants_enabled"`
	ABSplit         *int        `json:"ab_split"`
	Conditions      []Group     `json:"conditions"`
	Exceptions      []Condition `json:"exceptions"`
	CapCount        *int        `json:"cap_count"`
	CapPeriod       string      `json:"cap_period"`
	CooldownDays    int         `json:"cooldown_days"`
	HoursFrom       *int        `json:"hours_from"`
	HoursTo         *int        `json:"hours_to"`
	StartsOn        string      `json:"starts_on"`
	EndsOn          string      `json:"ends_on"`
	AudiencePercent *int        `json:"audience_percent"`
	Goal            string      `json:"goal"`

	// Message model v6 (T-M10-02).
	RuleKey     string          `json:"rule_key"`
	Topics      []string        `json:"topics"`
	Caps        []Cap           `json:"caps"`
	Variants    []VariantWeight `json:"variants"`
	StartsAt    string          `json:"starts_at"`
	EndsAt      string          `json:"ends_at"`
	HourWindows []HourWindow    `json:"hour_windows"`
	GoalEvent   string          `json:"goal_event"`

	// Spec-name aliases (M-01).
	Type string `json:"type"`
	Slot string `json:"slot"`
	Name string `json:"name"`
}

// Fields are the editor fields Validate reports on, in the order the 422 lists them.
var Fields = []string{"title", "rule_key", "level", "kind", "priority", "placement", "content.a", "content.b",
	"content.variants", "content.seen", "content.confidence", "content.contact", "variants", "ab_split", "caps",
	"cap_count", "cap_period", "cooldown_days", "hours_from", "hours_to", "hour_windows", "starts_on", "ends_on",
	"starts_at", "ends_at", "audience_percent", "goal", "goal_event", "topics", "conditions", "exceptions"}

// Normalize folds the spec-name aliases onto the canonical fields (a canonical value wins over its alias).
func (in *Input) Normalize() {
	if in.Kind == "" {
		in.Kind = in.Type
	}
	if in.Placement == "" {
		in.Placement = in.Slot
	}
	if in.Title == "" {
		in.Title = in.Name
	}
	if l, ok := LevelAliases[in.Level]; ok {
		in.Level = l
	}
	in.Type, in.Slot, in.Name = "", "", ""
}

var (
	placeholder = regexp.MustCompile(`\{([a-z_]+)\}`)
	keyPattern  = regexp.MustCompile(`^[a-z0-9_.-]{1,64}$`)
	variantKey  = regexp.MustCompile(`^[a-z0-9_]{1,16}$`)
)

// Placeholders lists the {variables} used in texts.
func Placeholders(texts ...string) []string {
	var out []string
	for _, t := range texts {
		for _, m := range placeholder.FindAllStringSubmatch(t, -1) {
			if !slices.Contains(out, m[1]) {
				out = append(out, m[1])
			}
		}
	}
	return out
}

// Rule is the engine rule the payload describes (id = the edited rule, 0 for a new one). Invalid parts are left
// zero: a draft may be incomplete while previewing.
func (in *Input) Rule(id uint64) Rule {
	r := Rule{
		ID: id, Level: in.Level, Kind: in.Kind, Priority: in.Priority, Placement: in.Placement, Content: in.Content,
		VariantsEnabled: in.VariantsEnabled, ABSplit: 50, Groups: in.Conditions, Exceptions: in.Exceptions,
		CooldownDays: in.CooldownDays, HoursFrom: -1, HoursTo: -1, AudiencePercent: 100,
		RuleKey: in.RuleKey, Topics: in.Topics, Caps: in.Caps, Variants: in.Variants, HourWindows: in.HourWindows,
		GoalEvent: in.GoalEvent,
	}
	if in.ABSplit != nil {
		r.ABSplit = *in.ABSplit
	}
	if in.AudiencePercent != nil {
		r.AudiencePercent = *in.AudiencePercent
	}
	if in.CapCount != nil {
		r.CapCount, r.CapPeriod = *in.CapCount, in.CapPeriod
	}
	if in.HoursFrom != nil && in.HoursTo != nil {
		r.HoursFrom, r.HoursTo = *in.HoursFrom, *in.HoursTo
	}
	if d, err := civildate.Parse(in.StartsOn); err == nil {
		r.StartsOn = d
	}
	if d, err := civildate.Parse(in.EndsOn); err == nil {
		r.EndsOn = d
	}
	if t, ok := ParseSchedule(in.StartsAt); ok {
		r.StartsAt = t
	}
	if t, ok := ParseSchedule(in.EndsAt); ok {
		r.EndsAt = t
	}
	return r
}

// targetsPlus reports whether a CTA target opens the paywall.
func targetsPlus(target string) bool {
	t := strings.ToLower(strings.TrimSpace(target))
	return t == "plus" || strings.HasPrefix(t, "/plus") || strings.Contains(t, "://plus") || strings.Contains(t, "/plus/") ||
		strings.HasSuffix(t, "/plus")
}

// Validate returns field → message (English; admin-web translates by field) for everything the editor must satisfy.
// Call Normalize first.
func (in *Input) Validate() map[string]string {
	errs := map[string]string{}
	add := func(f, m string) {
		if _, ok := errs[f]; !ok {
			errs[f] = m
		}
	}
	if t := strings.TrimSpace(in.Title); t == "" || len([]rune(t)) > 160 {
		add("title", "The title is required (max 160 characters).")
	}
	if in.RuleKey != "" && !keyPattern.MatchString(in.RuleKey) {
		add("rule_key", "The key is 1–64 characters: a–z, 0–9, «_», «.» or «-».")
	}
	levelOK, kindOK := slices.Contains(Levels, in.Level), slices.Contains(Kinds, in.Kind)
	if !levelOK {
		add("level", "Unknown level.")
	}
	if !kindOK {
		add("kind", "Unknown kind.")
	} else if lo, hi := Band(in.Kind); in.Priority < lo || in.Priority > hi {
		add("priority", fmt.Sprintf("Priority for %s messages must be %d–%d.", in.Kind, lo, hi))
	}
	if levelOK && kindOK && !LevelAllowed(in.Level, in.Kind) {
		add("level", fmt.Sprintf("The level %s is not allowed for %s messages.", SpecLevel(in.Level), in.Kind))
	}
	if !slices.Contains(Placements, in.Placement) {
		add("placement", "Unknown placement.")
	} else if kindOK && !KindAllowedIn(in.Placement, in.Kind) {
		add("placement", fmt.Sprintf("%s messages are not allowed in the %s slot.", in.Kind, in.Placement))
	}
	in.validateContent(add)
	in.validateVariants(add)
	in.validateFrequency(add)
	in.validateSchedule(add)
	if in.AudiencePercent != nil && (*in.AudiencePercent < 1 || *in.AudiencePercent > 100) {
		add("audience_percent", "The audience share must be 1–100.")
	}
	if in.Kind == KindSafety {
		// M-07: safety is exempt from caps, A/B and the audience share.
		if in.AudiencePercent != nil && *in.AudiencePercent != 100 {
			add("audience_percent", "Safety messages always reach the whole audience (100).")
		}
		if len(in.Caps) > 0 || in.CapCount != nil {
			add("caps", "Safety messages have no frequency cap.")
		}
	}
	if len([]rune(in.Goal)) > 120 {
		add("goal", "The goal is too long.")
	}
	if in.GoalEvent != "" && !slices.Contains(GoalEvents, in.GoalEvent) {
		add("goal_event", "Unknown goal event.")
	}
	for _, t := range in.Topics {
		if !slices.Contains(Topics, t) {
			add("topics", "Unknown topic «"+t+"».")
		}
	}
	checkCond := func(f string, c Condition) {
		if c.Fact == "" || !slices.Contains(Ops, c.Op) {
			add(f, "A condition needs a fact and a valid operator.")
		}
	}
	for _, g := range in.Conditions {
		if g.Op != "and" && g.Op != "or" {
			add("conditions", "A group is «and» or «or».")
		}
		for _, c := range g.Items {
			checkCond("conditions", c)
		}
	}
	for _, c := range in.Exceptions {
		checkCond("exceptions", c)
	}
	for _, p := range Placeholders(in.Content.Texts()...) {
		if !slices.Contains(Variables, p) {
			add("content.a", "Unknown variable {"+p+"}.")
		}
	}
	r := in.Rule(0)
	if r.UsesFertility() && !slices.Contains(in.Topics, TopicFertileWindow) {
		add("topics", "A message that reads fertile-window facts needs the «fertile_window» topic.")
	}
	return errs
}

// validateContent checks the texts and the level rules (SPEC §5.3 level table).
func (in *Input) validateContent(add func(f, m string)) {
	if strings.TrimSpace(in.Content.A.Title) == "" || strings.TrimSpace(in.Content.A.Body) == "" {
		add("content.a", "Variant A needs a title and a body.")
	}
	if in.VariantsEnabled && len(in.Variants) == 0 {
		if in.Content.B == nil || strings.TrimSpace(in.Content.B.Title) == "" || strings.TrimSpace(in.Content.B.Body) == "" {
			add("content.b", "Variant B needs a title and a body.")
		}
		if in.ABSplit != nil && (*in.ABSplit < 1 || *in.ABSplit > 99) {
			add("ab_split", "The A share must be 1–99.")
		}
	}
	all := in.Content.AllVariants()
	for _, k := range append([]string{"a", "b"}, sortedKeys(in.Content.Variants)...) {
		v, ok := all[k]
		if !ok {
			continue
		}
		f := "content." + k
		if k != "a" && k != "b" {
			f = "content.variants"
		}
		if (v.CTALabel == "") != (v.CTATarget == "") {
			add(f, "A CTA needs both a label and a target.")
		}
		if in.Level == LevelSuggestion && (v.CTALabel == "" || v.CTATarget == "") {
			add(f, "A suggestion needs exactly one CTA (a label and a target).")
		}
		if (in.Kind == KindSafety || in.Level == LevelFollowSoon) && targetsPlus(v.CTATarget) {
			add(f, "A safety or urgent message never links to Plus.")
		}
	}
	switch in.Level {
	case LevelWorthFollowUp, LevelFollowSoon:
		if strings.TrimSpace(in.Content.Seen) == "" {
			add("content.seen", "«What we saw» is required for this level.")
		}
		if strings.TrimSpace(in.Content.Confidence) == "" {
			add("content.confidence", "«How sure we are» is required for this level.")
		}
		if in.Level == LevelFollowSoon && strings.TrimSpace(in.Content.Contact) == "" {
			add("content.contact", "A call button (115 or the doctor) is required for «follow up soon».")
		}
	}
}

// validateVariants checks the weighted variants (keys unique, weights > 0, at most MaxVariants, a text per key).
func (in *Input) validateVariants(add func(f, m string)) {
	if in.Kind == KindSafety && (len(in.Variants) > 1 || (len(in.Variants) == 0 && in.VariantsEnabled)) {
		add("variants", "Safety messages have a single variant.")
	}
	if len(in.Variants) == 0 {
		return
	}
	if len(in.Variants) > MaxVariants {
		add("variants", fmt.Sprintf("At most %d variants.", MaxVariants))
	}
	all := in.Content.AllVariants()
	seen := map[string]bool{}
	for _, v := range in.Variants {
		switch {
		case !variantKey.MatchString(v.Key):
			add("variants", "A variant key is 1–16 characters: a–z, 0–9 or «_».")
		case seen[v.Key]:
			add("variants", "Variant keys must be unique.")
		case v.Weight < 1 || v.Weight > 1000:
			add("variants", "A variant weight must be 1–1000.")
		}
		seen[v.Key] = true
		if t, ok := all[v.Key]; !ok || strings.TrimSpace(t.Title) == "" || strings.TrimSpace(t.Body) == "" {
			add("content.variants", "Variant «"+v.Key+"» needs a title and a body.")
		}
	}
}

// validateFrequency checks the caps (new list and legacy single cap) and the cooldown.
func (in *Input) validateFrequency(add func(f, m string)) {
	if in.CapCount != nil {
		if *in.CapCount < 1 || *in.CapCount > 1000 {
			add("cap_count", "The cap must be 1–1000.")
		}
		if !slices.Contains(CapPeriods, in.CapPeriod) {
			add("cap_period", "The cap period must be day, week or cycle.")
		}
	}
	periods := map[string]bool{}
	for _, c := range in.Caps {
		switch {
		case c.Count < 1 || c.Count > 1000:
			add("caps", "A cap count must be 1–1000.")
		case !slices.Contains(CapPeriods, c.Period):
			add("caps", "A cap period must be day, week or cycle.")
		case periods[c.Period]:
			add("caps", "One cap per period.")
		}
		periods[c.Period] = true
	}
	if in.Kind == KindProduct || in.Kind == KindMarketing {
		dayOrWeek := periods["day"] || periods["week"]
		if len(in.Caps) == 0 && in.CapCount != nil && (in.CapPeriod == "day" || in.CapPeriod == "week") {
			dayOrWeek = true
		}
		if !dayOrWeek {
			add("caps", "Product and marketing messages need a daily or weekly cap.")
		}
	}
	if in.CooldownDays < 0 || in.CooldownDays > 365 {
		add("cooldown_days", "Cooldown must be 0–365 days.")
	}
}

// validateSchedule checks the legacy dates / hours and the v6 datetimes / hour windows.
func (in *Input) validateSchedule(add func(f, m string)) {
	if (in.HoursFrom == nil) != (in.HoursTo == nil) {
		add("hours_from", "Both ends of the hours window are required.")
	}
	for f, h := range map[string]*int{"hours_from": in.HoursFrom, "hours_to": in.HoursTo} {
		if h != nil && (*h < 0 || *h > 24) {
			add(f, "Hours must be 0–24.")
		}
	}
	if len(in.HourWindows) > 6 {
		add("hour_windows", "At most 6 hour windows.")
	}
	for _, w := range in.HourWindows {
		if w[0] < 0 || w[0] > 24 || w[1] < 0 || w[1] > 24 || w[0] == w[1] {
			add("hour_windows", "An hour window is [from, to] with hours 0–24 and from ≠ to.")
		}
	}
	var starts, ends civildate.Date
	var err error
	if in.StartsOn != "" {
		if starts, err = civildate.Parse(in.StartsOn); err != nil {
			add("starts_on", "Invalid date (Y-m-d).")
		}
	}
	if in.EndsOn != "" {
		if ends, err = civildate.Parse(in.EndsOn); err != nil {
			add("ends_on", "Invalid date (Y-m-d).")
		} else if !starts.IsZero() && ends.Before(starts) {
			add("ends_on", "The end is before the start.")
		}
	}
	var startAt, endAt time.Time
	var ok bool
	if in.StartsAt != "" {
		if startAt, ok = ParseSchedule(in.StartsAt); !ok {
			add("starts_at", "Invalid date and time (Y-m-d H:i:s).")
		}
	}
	if in.EndsAt != "" {
		if endAt, ok = ParseSchedule(in.EndsAt); !ok {
			add("ends_at", "Invalid date and time (Y-m-d H:i:s).")
		} else if !startAt.IsZero() && !endAt.After(startAt) {
			add("ends_at", "The end is not after the start.")
		}
	}
}

// Render fills the {variables} of a text from facts; ok is false when a variable has no value (the rule is then
// skipped for that user instead of showing a raw placeholder).
func Render(s string, facts Facts) (string, bool) {
	ok := true
	out := placeholder.ReplaceAllStringFunc(s, func(m string) string {
		v, present := facts[m[1:len(m)-1]]
		if !present {
			ok = false
			return m
		}
		if l, isList := v.([]string); isList {
			return strings.Join(l, "، ")
		}
		if f, isNum := toFloat(v); isNum {
			return fmt.Sprintf("%g", f)
		}
		return fmt.Sprint(v)
	})
	return out, ok
}

// RenderVariant renders every text of a variant.
func RenderVariant(v Variant, facts Facts) (Variant, bool) {
	var a, b, c bool
	v.Title, a = Render(v.Title, facts)
	v.Body, b = Render(v.Body, facts)
	v.CTALabel, c = Render(v.CTALabel, facts)
	return v, a && b && c
}

// Conflict is one overlapping rule found by the conflict check.
type Conflict struct {
	RuleID   uint64
	Priority int
	Kind     string
	// Winner is "this", "other" or "tie" (equal priority: the lower id wins).
	Winner string
	// Safety is true when the other rule is a safety rule (it always shows first).
	Safety bool
}

func factsOf(r *Rule) map[string]bool {
	m := map[string]bool{}
	for _, g := range r.Groups {
		for _, c := range g.Items {
			m[c.Fact] = true
		}
	}
	return m
}

func windowsOverlap(a, b *Rule) bool {
	as, ae := a.EffectiveSchedule()
	bs, be := b.EffectiveSchedule()
	if !ae.IsZero() && !bs.IsZero() && !ae.After(bs) {
		return false
	}
	if !be.IsZero() && !as.IsZero() && !be.After(as) {
		return false
	}
	return true
}

// FindConflicts lists the other rules of the same placement that can show to the same users at the same time:
// overlapping windows and conditions that read a common fact (or one side has no conditions). It is a heuristic
// warning for the editor, not a proof — the engine's priority order still decides.
func FindConflicts(r *Rule, others []Rule) []Conflict {
	mine := factsOf(r)
	var out []Conflict
	for i := range others {
		o := &others[i]
		if o.ID == r.ID || o.Placement != r.Placement || !o.IsActive || !windowsOverlap(r, o) {
			continue
		}
		theirs := factsOf(o)
		shared := len(mine) == 0 || len(theirs) == 0
		for f := range mine {
			if theirs[f] {
				shared = true
			}
		}
		if !shared {
			continue
		}
		w := "tie"
		switch {
		case r.Priority > o.Priority:
			w = "this"
		case r.Priority < o.Priority:
			w = "other"
		case r.ID != 0 && r.ID < o.ID:
			w = "this"
		case r.ID != 0:
			w = "other"
		}
		out = append(out, Conflict{RuleID: o.ID, Priority: o.Priority, Kind: o.Kind, Winner: w, Safety: o.Kind == KindSafety})
	}
	return out
}
