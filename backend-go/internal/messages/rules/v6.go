package rules

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Message model v6 (T-M10-02, SPEC §5.3/§5.4, D-160): the kind × level × slot matrices, the goal and topic catalogues,
// several caps, N weighted variants, a datetime schedule with several hour windows, and the legacy-column fallback.

// Spec level names (SPEC §5.3) accepted as input aliases of the stored levels (M-01: the stored values stay).
const (
	SpecLevelFollowUp = "follow_up" // = LevelWorthFollowUp
	SpecLevelUrgent   = "urgent"    // = LevelFollowSoon
)

// LevelAliases maps a spec level name onto the stored level.
var LevelAliases = map[string]string{SpecLevelFollowUp: LevelWorthFollowUp, SpecLevelUrgent: LevelFollowSoon}

// SpecLevel is the spec name of a stored level (info and suggestion are the same in both).
func SpecLevel(level string) string {
	for spec, stored := range LevelAliases {
		if stored == level {
			return spec
		}
	}
	return level
}

// ApprovalKinds are the kinds whose authoring and approval need the clinical capability (SPEC §5.3/§5.8).
var ApprovalKinds = []string{KindSafety, KindClinical, KindContent}

// LevelKinds is SPEC §5.3's level × kind table: the kinds a (stored) level may carry.
var LevelKinds = map[string][]string{
	LevelInfo:          {KindEngine, KindClinical, KindContent, KindProduct, KindMarketing},
	LevelSuggestion:    {KindEngine, KindContent, KindProduct},
	LevelWorthFollowUp: {KindEngine, KindClinical},
	LevelFollowSoon:    {KindSafety, KindClinical},
}

// SlotAlertsInbox is the alerts inbox slot (SPEC §5.4.1); `alerts` is its pre-v6 single-winner name.
const SlotAlertsInbox = "alerts_inbox"

// SlotKinds is SPEC §5.4's slot × kind table (selection skips a mismatch with `kind_not_allowed`, T-M10-04).
var SlotKinds = map[string][]string{
	"urgent_modal":    {KindSafety},
	"status_banner":   {KindEngine},
	"smart_tip":       {KindClinical},
	"daily_advice":    {KindContent},
	"daily_challenge": {KindProduct},
	"push":            Kinds,
	"modal":           {KindMarketing},
	SlotAlertsInbox:   {KindSafety, KindEngine, KindClinical, KindContent, KindProduct},
	"alerts":          {KindSafety, KindEngine, KindClinical, KindContent, KindProduct},
	"sms":             Kinds, // not a spec slot; kept as it was (any kind)
}

// KindAllowedIn reports whether kind may run in slot.
func KindAllowedIn(slot, kind string) bool { return slices.Contains(SlotKinds[slot], kind) }

// LevelAllowed reports whether level may carry kind.
func LevelAllowed(level, kind string) bool { return slices.Contains(LevelKinds[level], kind) }

// GoalEvents is the server-matched goal catalogue (M-17).
var GoalEvents = []string{"period_logged", "log_saved", "checkup_done", "plus_purchased", "challenge_joined"}

// TopicFertileWindow marks a message about the fertile window: removed under hormonal contraception (M-06).
const TopicFertileWindow = "fertile_window"

// Topics is the topic catalogue.
var Topics = []string{TopicFertileWindow}

// FertilityFacts are the facts (and text variables) that make a message a fertile-window message (M-06); the list
// lives in the fact registry (T-M10-03).
var FertilityFacts = facts.FertilityFacts

// fertilityWords mark a phase value about ovulation / the fertile window (phase = ovulation, subphase = fertile_peak…).
var fertilityWords = []string{"ovulat", "fertil"}

// phaseFacts are the facts whose values can name a fertile phase.
var phaseFacts = []string{"phase", "main_phase", "subphase"}

// MaxVariants is the most variants a rule may have.
const MaxVariants = 6

// Cap is one frequency cap: at most Count shows per Period (day | week | cycle).
type Cap struct {
	Count  int    `json:"count"`
	Period string `json:"period"`
}

// VariantWeight is one weighted variant key; its texts are Content.A / Content.B / Content.Variants[key].
type VariantWeight struct {
	Key    string `json:"key"`
	Weight int    `json:"weight"`
}

// HourWindow is a [from, to) user-local hour window; from > to wraps midnight.
type HourWindow [2]int

// EffectiveCaps are the rule's caps: the caps column, else the legacy cap_count / cap_period.
func (r *Rule) EffectiveCaps() []Cap {
	if r.Caps != nil {
		return r.Caps
	}
	if r.CapCount > 0 {
		return []Cap{{Count: r.CapCount, Period: r.CapPeriod}}
	}
	return nil
}

// EffectiveVariants are the rule's weighted variants: the variants column, else the legacy a/b split (a only when
// variants are off or there is no b text).
func (r *Rule) EffectiveVariants() []VariantWeight {
	if len(r.Variants) > 0 {
		return r.Variants
	}
	if r.VariantsEnabled && r.Content.B != nil {
		return []VariantWeight{{Key: "a", Weight: r.ABSplit}, {Key: "b", Weight: 100 - r.ABSplit}}
	}
	return []VariantWeight{{Key: "a", Weight: 100}}
}

// EffectiveSchedule is the rule's [start, end) schedule in Tehran wall-clock time (zero = open): the datetime columns,
// else the legacy dates (start of starts_on, end of ends_on).
func (r *Rule) EffectiveSchedule() (start, end time.Time) {
	start, end = r.StartsAt, r.EndsAt
	if start.IsZero() && !r.StartsOn.IsZero() {
		start = r.StartsOn.TehranMidnight()
	}
	if end.IsZero() && !r.EndsOn.IsZero() {
		end = r.EndsOn.AddDays(1).TehranMidnight()
	}
	return start, end
}

// EffectiveHourWindows are the rule's hour windows: the hour_windows column, else the legacy hours_from / hours_to.
func (r *Rule) EffectiveHourWindows() []HourWindow {
	if r.HourWindows != nil {
		return r.HourWindows
	}
	if r.HoursFrom >= 0 && r.HoursTo >= 0 {
		return []HourWindow{{r.HoursFrom, r.HoursTo}}
	}
	return nil
}

// inHours reports whether hour h falls in the window.
func (w HourWindow) inHours(h int) bool {
	if w[0] <= w[1] {
		return h >= w[0] && h < w[1]
	}
	return h >= w[0] || h < w[1]
}

// UsesFertility reports whether the rule's conditions, exceptions or texts read a fertility fact or variable (M-06).
func (r *Rule) UsesFertility() bool {
	isFertile := func(c Condition) bool {
		if slices.Contains(FertilityFacts, c.Fact) {
			return true
		}
		if !slices.Contains(phaseFacts, c.Fact) {
			return false
		}
		vals := asList(c.Value)
		if s, ok := c.Value.(string); ok {
			vals = []any{s}
		}
		for _, v := range vals {
			s, _ := v.(string)
			for _, w := range fertilityWords {
				if strings.Contains(strings.ToLower(s), w) {
					return true
				}
			}
		}
		return false
	}
	for _, g := range r.Groups {
		if slices.ContainsFunc(g.Items, isFertile) {
			return true
		}
	}
	if slices.ContainsFunc(r.Exceptions, isFertile) {
		return true
	}
	return slices.ContainsFunc(Placeholders(r.Content.Texts()...), func(p string) bool {
		return slices.Contains(FertilityFacts, p)
	})
}

// Texts are every user-facing text of the content (all variants).
func (c *Content) Texts() []string {
	out := []string{c.A.Title, c.A.Body, c.A.CTALabel}
	if c.B != nil {
		out = append(out, c.B.Title, c.B.Body, c.B.CTALabel)
	}
	for _, k := range sortedKeys(c.Variants) {
		v := c.Variants[k]
		out = append(out, v.Title, v.Body, v.CTALabel)
	}
	return out
}

// AllVariants are every variant text with its key (a, b, then the extra keys in order).
func (c *Content) AllVariants() map[string]Variant {
	out := map[string]Variant{"a": c.A}
	if c.B != nil {
		out["b"] = *c.B
	}
	for k, v := range c.Variants {
		out[k] = v
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// VoidsApproval reports whether an edit from old to updated withdraws the medical approval (M-02): content (every
// variant text), conditions, exceptions, level, kind, slot, topics or the variant key set changed. Priority, audience,
// schedule, hour windows, caps, cooldown, weights and the admin title / goal keep it.
func VoidsApproval(old, updated *Rule) bool {
	if old.Level != updated.Level || old.Kind != updated.Kind || old.Placement != updated.Placement {
		return true
	}
	same := func(a, b any) bool {
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		return string(x) == string(y)
	}
	norm := func(c Content) Content {
		if c.Variants != nil && len(c.Variants) == 0 {
			c.Variants = nil
		}
		return c
	}
	groups := func(g []Group) []Group {
		if g == nil {
			return []Group{}
		}
		return g
	}
	conds := func(c []Condition) []Condition {
		if c == nil {
			return []Condition{}
		}
		return c
	}
	topics := func(t []string) []string {
		t = slices.Clone(t)
		slices.Sort(t)
		if t == nil {
			return []string{}
		}
		return t
	}
	keys := func(r *Rule) []string {
		var out []string
		for _, v := range r.EffectiveVariants() {
			out = append(out, v.Key)
		}
		slices.Sort(out)
		return out
	}
	return !same(norm(old.Content), norm(updated.Content)) || !same(groups(old.Groups), groups(updated.Groups)) ||
		!same(conds(old.Exceptions), conds(updated.Exceptions)) || !same(topics(old.Topics), topics(updated.Topics)) ||
		!same(keys(old), keys(updated))
}

// ScheduleLayout is the datetime format of starts_at / ends_at on the admin API (Tehran wall clock).
const ScheduleLayout = "2006-01-02 15:04:05"

var scheduleLayouts = []string{ScheduleLayout, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"}

// ParseSchedule reads a starts_at / ends_at value as Tehran wall-clock time (an RFC 3339 value with an offset is
// converted to Tehran).
func ParseSchedule(s string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(civildate.Tehran), true
	}
	for _, l := range scheduleLayouts {
		if t, err := time.ParseInLocation(l, s, civildate.Tehran); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func decodeJSON[T any](raw []byte) (T, bool) {
	var v T
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return v, false
	}
	return v, true
}
