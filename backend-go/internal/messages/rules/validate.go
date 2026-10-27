package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Variables are the placeholders a message text may use ({name}); each must be a fact the engine supplies.
// T-M9-38: pms_start_day and top_symptoms were removed — no fact ever filled them, so a text using them never rendered.
var Variables = []string{"name", "days_to_period", "days_late", "cycle_day", "pregnancy_week", "phase", "subphase"}

// CapPeriods are the valid cap periods.
var CapPeriods = []string{"day", "week", "cycle"}

// Input is the admin editor's payload.
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
}

var placeholder = regexp.MustCompile(`\{([a-z_]+)\}`)

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

// Validate returns field → message (English; admin-web translates by field) for everything the editor must satisfy.
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
	if !slices.Contains(Levels, in.Level) {
		add("level", "Unknown level.")
	}
	if !slices.Contains(Kinds, in.Kind) {
		add("kind", "Unknown kind.")
	} else if lo, hi := Band(in.Kind); in.Priority < lo || in.Priority > hi {
		add("priority", fmt.Sprintf("Priority for %s messages must be %d–%d.", in.Kind, lo, hi))
	}
	if !slices.Contains(Placements, in.Placement) {
		add("placement", "Unknown placement.")
	} else if in.Kind == KindProduct && slices.Contains(UrgentPlacements, in.Placement) {
		add("placement", "Product messages never run on urgent paths.")
	}
	if strings.TrimSpace(in.Content.A.Title) == "" || strings.TrimSpace(in.Content.A.Body) == "" {
		add("content.a", "Variant A needs a title and a body.")
	}
	if in.VariantsEnabled {
		if in.Content.B == nil || strings.TrimSpace(in.Content.B.Title) == "" || strings.TrimSpace(in.Content.B.Body) == "" {
			add("content.b", "Variant B needs a title and a body.")
		}
		if in.ABSplit != nil && (*in.ABSplit < 1 || *in.ABSplit > 99) {
			add("ab_split", "The A share must be 1–99.")
		}
	}
	// Level rules (the Admin_Messages four-level legend).
	switch in.Level {
	case LevelWorthFollowUp, LevelFollowSoon:
		if strings.TrimSpace(in.Content.Seen) == "" {
			add("content.seen", "«What we saw» is required for this level.")
		}
		if strings.TrimSpace(in.Content.Confidence) == "" {
			add("content.confidence", "«How sure we are» is required for this level.")
		}
		if in.Level == LevelFollowSoon {
			if in.Kind != KindSafety && in.Kind != KindClinical {
				add("level", "«Follow up soon» is only for safety or clinical messages.")
			}
			if strings.TrimSpace(in.Content.Contact) == "" {
				add("content.contact", "A contact button is required for «follow up soon».")
			}
		}
	case LevelSuggestion:
		if (in.Content.A.CTALabel == "") != (in.Content.A.CTATarget == "") {
			add("content.a", "A CTA needs both a label and a target.")
		}
	}
	if in.Kind == KindProduct && (in.CapCount == nil || *in.CapCount < 1) {
		add("cap_count", "Product messages need a frequency cap.")
	}
	if in.CapCount != nil {
		if *in.CapCount < 1 || *in.CapCount > 1000 {
			add("cap_count", "The cap must be 1–1000.")
		}
		if !slices.Contains(CapPeriods, in.CapPeriod) {
			add("cap_period", "The cap period must be day, week or cycle.")
		}
	}
	if in.CooldownDays < 0 || in.CooldownDays > 365 {
		add("cooldown_days", "Cooldown must be 0–365 days.")
	}
	if (in.HoursFrom == nil) != (in.HoursTo == nil) {
		add("hours_from", "Both ends of the hours window are required.")
	}
	for f, h := range map[string]*int{"hours_from": in.HoursFrom, "hours_to": in.HoursTo} {
		if h != nil && (*h < 0 || *h > 24) {
			add(f, "Hours must be 0–24.")
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
	if in.AudiencePercent != nil && (*in.AudiencePercent < 1 || *in.AudiencePercent > 100) {
		add("audience_percent", "The audience share must be 1–100.")
	}
	if len([]rune(in.Goal)) > 120 {
		add("goal", "The goal is too long.")
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
	texts := []string{in.Content.A.Title, in.Content.A.Body, in.Content.A.CTALabel}
	if in.Content.B != nil {
		texts = append(texts, in.Content.B.Title, in.Content.B.Body, in.Content.B.CTALabel)
	}
	for _, p := range Placeholders(texts...) {
		if !slices.Contains(Variables, p) {
			add("content.a", "Unknown variable {"+p+"}.")
		}
	}
	return errs
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
	if !a.EndsOn.IsZero() && !b.StartsOn.IsZero() && a.EndsOn.Before(b.StartsOn) {
		return false
	}
	if !b.EndsOn.IsZero() && !a.StartsOn.IsZero() && b.EndsOn.Before(a.StartsOn) {
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
