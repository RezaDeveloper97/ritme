// Package rules is the rule engine of smart messages v2 (bloom B-N9-05, D-86): a message rule has a user level, a
// priority band, A/B variants, AND/OR condition groups with exceptions, caps, cooldown, hours, a start/end window and
// an audience share. The package is pure: facts and event history come in through Facts and History, so the same
// Select runs for a user's request, an admin preview and the conflict check.
package rules

import (
	"encoding/json"
	"hash/fnv"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Levels (what the user sees; independent of priority).
const (
	LevelInfo          = "info"
	LevelSuggestion    = "suggestion"
	LevelWorthFollowUp = "worth_followup"
	LevelFollowSoon    = "follow_soon"
)

// Kinds and their priority bands.
const (
	KindSafety   = "safety"   // 1000
	KindEngine   = "engine"   // 800–999
	KindClinical = "clinical" // 100–799
	KindProduct  = "product"  // 1–99
)

// Statuses.
const (
	StatusDraft    = "draft"
	StatusPending  = "pending_approval"
	StatusApproved = "approved"
)

// Placements. UrgentPlacements never carry product messages.
var Placements = []string{"status_banner", "smart_tip", "daily_advice", "daily_challenge", "push", "modal", "sms", "alerts", "urgent_modal"}

// UrgentPlacements are the urgent paths: product / marketing messages never run there.
var UrgentPlacements = []string{"urgent_modal", "alerts"}

// Levels and Kinds list the valid values.
var (
	Levels = []string{LevelInfo, LevelSuggestion, LevelWorthFollowUp, LevelFollowSoon}
	Kinds  = []string{KindSafety, KindEngine, KindClinical, KindProduct}
)

// Band is the inclusive priority range of a kind.
func Band(kind string) (lo, hi int) {
	switch kind {
	case KindSafety:
		return 1000, 1000
	case KindEngine:
		return 800, 999
	case KindClinical:
		return 100, 799
	}
	return 1, 99
}

// KindOfPriority is the kind a priority falls in ("" when out of range).
func KindOfPriority(p int) string {
	for _, k := range Kinds {
		if lo, hi := Band(k); p >= lo && p <= hi {
			return k
		}
	}
	return ""
}

// NeedsMedicalApproval reports whether a rule must be approved by the medical advisor before it is served.
func NeedsMedicalApproval(level, kind string) bool {
	return level == LevelWorthFollowUp || level == LevelFollowSoon || kind == KindSafety || kind == KindClinical
}

// Facts are the values conditions read: numbers (float64), strings, bools and []string.
type Facts map[string]any

// Condition is one comparison: fact <op> value. Ops: eq ne gt gte lt lte in nin between contains.
type Condition struct {
	Fact  string `json:"fact"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// Group is a set of conditions joined by Op ("and" | "or"); a rule's groups are joined by AND.
type Group struct {
	Op    string      `json:"op"`
	Items []Condition `json:"items"`
}

// Ops lists the valid comparison operators.
var Ops = []string{"eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "between", "contains"}

// Variant is one A/B text.
type Variant struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	CTALabel  string `json:"cta_label,omitempty"`
	CTATarget string `json:"cta_target,omitempty"`
}

// Content is a rule's message content. Seen and Confidence are the «what we saw» / «how sure we are» fields of the
// worth-follow-up and follow-soon levels; Contact is the call button of follow-soon.
type Content struct {
	A          Variant  `json:"a"`
	B          *Variant `json:"b,omitempty"`
	Seen       string   `json:"seen,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Contact    string   `json:"contact,omitempty"`
}

// Rule is the engine's view of a message_rules row.
type Rule struct {
	ID               uint64
	Level, Kind      string
	Priority         int
	Placement        string
	Content          Content
	VariantsEnabled  bool
	ABSplit          int // percent of users on A
	Groups           []Group
	Exceptions       []Condition
	CapCount         int    // 0 = no cap
	CapPeriod        string // day | week | cycle
	CooldownDays     int
	HoursFrom        int // -1 = no hour window
	HoursTo          int
	StartsOn, EndsOn civildate.Date // zero = open
	AudiencePercent  int
	Status           string
	IsActive         bool
}

// ParseGroups decodes a stored conditions column.
func ParseGroups(raw string) []Group {
	var g []Group
	if json.Unmarshal([]byte(raw), &g) != nil {
		return nil
	}
	return g
}

// ParseExceptions decodes a stored exceptions column.
func ParseExceptions(raw string) []Condition {
	var c []Condition
	if json.Unmarshal([]byte(raw), &c) != nil {
		return nil
	}
	return c
}

// ParseContent decodes a stored content column.
func ParseContent(raw string) Content {
	var c Content
	_ = json.Unmarshal([]byte(raw), &c)
	return c
}

// Served reports whether the rule may be shown at all: active, and a medical rule only while approved.
func (r *Rule) Served() bool {
	if !r.IsActive {
		return false
	}
	if NeedsMedicalApproval(r.Level, r.Kind) {
		return r.Status == StatusApproved
	}
	return true
}

// Unlimited reports whether caps and cooldown are skipped (safety messages are always shown first, no repeat cap).
func (r *Rule) Unlimited() bool { return r.Kind == KindSafety }

// bucket is a stable 0–99 bucket of (userID, ruleID, salt).
func bucket(userID, ruleID uint64, salt string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(ruleID, 10) + ":" + salt))
	return int(h.Sum32() % 100)
}

// Assign is the stable A/B assignment of a user: the same (user, rule) always gets the same variant.
func (r *Rule) Assign(userID uint64) string {
	if !r.VariantsEnabled || r.Content.B == nil {
		return "a"
	}
	if bucket(userID, r.ID, "ab") < r.ABSplit {
		return "a"
	}
	return "b"
}

// InAudience is the stable audience-share check (the share grows monotonically: a user inside 10% stays in at 50%).
func (r *Rule) InAudience(userID uint64) bool {
	return r.AudiencePercent >= 100 || bucket(userID, r.ID, "aud") < r.AudiencePercent
}

// VariantContent is the text of variant v.
func (r *Rule) VariantContent(v string) Variant {
	if v == "b" && r.Content.B != nil {
		return *r.Content.B
	}
	return r.Content.A
}

// History is a user's event history for the caps and the cooldown.
type History interface {
	// Count is the user's events of kind event on rule since the time.
	Count(rule uint64, event string, since time.Time) (int, error)
	// Last is the user's latest event of kind event on rule (ok false when none).
	Last(rule uint64, event string) (time.Time, bool, error)
}

// Choice is the engine's answer.
type Choice struct {
	Rule    *Rule
	Variant string
}

// Select picks the message for a placement: the highest priority eligible rule (safety first; ties by lowest id).
// now is the request time; the hour window and the dates read it in Tehran.
func Select(all []Rule, facts Facts, userID uint64, now time.Time, h History) (*Choice, error) {
	cands := slices.Clone(all)
	slices.SortStableFunc(cands, func(a, b Rule) int {
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		return int(a.ID) - int(b.ID) //nolint:gosec // ids are small
	})
	for i := range cands {
		r := &cands[i]
		ok, err := Eligible(r, facts, userID, now, h)
		if err != nil {
			return nil, err
		}
		if ok {
			return &Choice{Rule: r, Variant: r.Assign(userID)}, nil
		}
	}
	return nil, nil
}

// Eligible runs every gate of one rule: served, window, hours, audience, conditions, exceptions, cooldown, caps.
func Eligible(r *Rule, facts Facts, userID uint64, now time.Time, h History) (bool, error) {
	if !r.Served() {
		return false, nil
	}
	if !MatchRule(r, facts) {
		return false, nil
	}
	if !InWindow(r, now) || !r.InAudience(userID) {
		return false, nil
	}
	if r.Unlimited() || h == nil {
		return true, nil
	}
	if r.CooldownDays > 0 {
		last, ok, err := h.Last(r.ID, "impression")
		if err != nil {
			return false, err
		}
		if ok && now.Sub(last) < time.Duration(r.CooldownDays)*24*time.Hour {
			return false, nil
		}
	}
	if r.CapCount > 0 {
		n, err := h.Count(r.ID, "impression", now.Add(-capWindow(r, facts)))
		if err != nil {
			return false, err
		}
		if n >= r.CapCount {
			return false, nil
		}
	}
	return true, nil
}

func capWindow(r *Rule, facts Facts) time.Duration {
	days := 1
	switch r.CapPeriod {
	case "week":
		days = 7
	case "cycle":
		days = 28
		if n, ok := toFloat(facts["cycle_length"]); ok && n >= 15 && n <= 90 {
			days = int(n)
		}
	}
	return time.Duration(days) * 24 * time.Hour
}

// InWindow checks the start/end dates and the hour window (Tehran time; an overnight window wraps midnight).
func InWindow(r *Rule, now time.Time) bool {
	today := civildate.InTehran(now)
	if !r.StartsOn.IsZero() && today.Before(r.StartsOn) {
		return false
	}
	if !r.EndsOn.IsZero() && today.After(r.EndsOn) {
		return false
	}
	if r.HoursFrom >= 0 && r.HoursTo >= 0 {
		hr := now.In(civildate.Tehran).Hour()
		if r.HoursFrom <= r.HoursTo {
			return hr >= r.HoursFrom && hr < r.HoursTo
		}
		return hr >= r.HoursFrom || hr < r.HoursTo
	}
	return true
}

// MatchRule evaluates the groups (AND across groups, each group by its own op) and then the exceptions: any exception
// that holds removes the rule. A rule without groups matches every user.
func MatchRule(r *Rule, facts Facts) bool {
	for _, g := range r.Groups {
		if !MatchGroup(g, facts) {
			return false
		}
	}
	return !slices.ContainsFunc(r.Exceptions, func(c Condition) bool { return Match(c, facts) })
}

// MatchGroup evaluates one group (an empty group holds).
func MatchGroup(g Group, facts Facts) bool {
	if len(g.Items) == 0 {
		return true
	}
	if g.Op == "or" {
		return slices.ContainsFunc(g.Items, func(c Condition) bool { return Match(c, facts) })
	}
	return !slices.ContainsFunc(g.Items, func(c Condition) bool { return !Match(c, facts) })
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func asList(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case []string:
		out := make([]any, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	case string:
		var out []any
		for _, p := range strings.Split(l, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return nil
}

func equal(a, b any) bool {
	if x, ok := toFloat(a); ok {
		if y, ok := toFloat(b); ok {
			return x == y
		}
	}
	if x, ok := a.(bool); ok {
		if y, ok := b.(bool); ok {
			return x == y
		}
		if s, ok := b.(string); ok {
			return strconv.FormatBool(x) == s
		}
	}
	if s, ok := a.(string); ok {
		if t, ok := b.(string); ok {
			return s == t
		}
	}
	return false
}

// Match evaluates one condition. A missing fact matches nothing except ne / nin (unknown is not equal to anything).
func Match(c Condition, facts Facts) bool {
	got, present := facts[c.Fact]
	switch c.Op {
	case "eq":
		return present && equal(got, c.Value)
	case "ne":
		return !present || !equal(got, c.Value)
	case "in", "nin":
		in := false
		if present {
			in = slices.ContainsFunc(asList(c.Value), func(w any) bool { return equal(got, w) })
			// a list fact is "in" when any of its members is.
			if l, ok := got.([]string); ok {
				in = slices.ContainsFunc(l, func(s string) bool {
					return slices.ContainsFunc(asList(c.Value), func(w any) bool { return equal(s, w) })
				})
			}
		}
		return in == (c.Op == "in")
	case "contains":
		if l, ok := got.([]string); ok {
			return slices.ContainsFunc(l, func(s string) bool { return equal(s, c.Value) })
		}
		s, ok := got.(string)
		w, ok2 := c.Value.(string)
		return ok && ok2 && strings.Contains(s, w)
	case "between":
		g, ok := toFloat(got)
		l := asList(c.Value)
		if !present || !ok || len(l) != 2 {
			return false
		}
		lo, ok1 := toFloat(l[0])
		hi, ok2 := toFloat(l[1])
		return ok1 && ok2 && g >= lo && g <= hi
	case "gt", "gte", "lt", "lte":
		g, ok := toFloat(got)
		w, ok2 := toFloat(c.Value)
		if !present || !ok || !ok2 {
			return false
		}
		switch c.Op {
		case "gt":
			return g > w
		case "gte":
			return g >= w
		case "lt":
			return g < w
		}
		return g <= w
	}
	return false
}
