package challenges

import (
	"bytes"
	"encoding/json"
	"slices"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/messages/rules"
)

// Eligibility and contraindications (SPEC §5.5, D-167, gap 03 §2.8): both are rules over the message fact registry
// (T-M10-03), stored in challenge_programs.eligibility / .contraindications as
//
//	{"all_of": [cond…], "any_of": [[cond…]…], "exclude": [cond…]}
//
// the same three sections as the admin rule builder (all of = one AND group, each any_of entry = one OR group,
// exclude = the exceptions). A condition is a rules.Condition (fact, op, value) evaluated with rules.Match.
//
// The contraindications are evaluated first and always win: when they hold the user is refused with reason
// `contraindicated`, whatever the eligibility says. They fail closed: a stored value that does not parse, or a
// condition on a fact that the user's mode makes expected but that is unknown (a postpartum user without a birth date,
// a pregnant user without a pregnancy week), counts as holding.
//
// Legacy rows ({modes_include, modes_exclude, phases_include, phases_exclude}, bloom B-N9-07) are converted on the
// fly, no data migration: the includes become all_of `life_mode in […]` / `main_phase in […]`, the excludes become
// exclude conditions. That is exactly the old include / exclude semantics (an empty include allows everything, an
// exclude match refuses, a phase include refuses a user without a phase, a phase exclude never refuses her).

// Refusal reasons.
const (
	ReasonContraindicated = "contraindicated"
	ReasonNotEligible     = "not_eligible"
)

// Facts the challenges service computes itself (legacy-exact); every other name comes from the fact registry.
const (
	FactLifeMode  = "life_mode"
	FactMainPhase = "main_phase"
	FactMode      = "mode"
)

// Rule is an eligibility or contraindication rule. The zero value is "no rule".
type Rule struct {
	AllOf   []rules.Condition   `json:"all_of"`
	AnyOf   [][]rules.Condition `json:"any_of"`
	Exclude []rules.Condition   `json:"exclude"`
	// Broken marks a stored value that did not parse (contraindications then fail closed).
	Broken bool `json:"-"`
}

// Empty reports whether the rule has no condition at all.
func (r Rule) Empty() bool {
	if len(r.AllOf) > 0 || len(r.Exclude) > 0 {
		return false
	}
	return !slices.ContainsFunc(r.AnyOf, func(g []rules.Condition) bool { return len(g) > 0 })
}

// Normalized is r with nil sections as empty lists (the stored and echoed JSON).
func (r Rule) Normalized() Rule {
	out := Rule{AllOf: r.AllOf, AnyOf: make([][]rules.Condition, 0, len(r.AnyOf)), Exclude: r.Exclude}
	if out.AllOf == nil {
		out.AllOf = []rules.Condition{}
	}
	if out.Exclude == nil {
		out.Exclude = []rules.Condition{}
	}
	for _, g := range r.AnyOf {
		if len(g) > 0 {
			out.AnyOf = append(out.AnyOf, g)
		}
	}
	return out
}

// Conditions are every condition of the rule, in section order.
func (r Rule) Conditions() []rules.Condition {
	out := slices.Clone(r.AllOf)
	for _, g := range r.AnyOf {
		out = append(out, g...)
	}
	return append(out, r.Exclude...)
}

// Facts are the fact names the rule reads (sorted, unique).
func (r Rule) Facts() []string {
	var out []string
	for _, c := range r.Conditions() {
		out = append(out, c.Fact)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// legacyRule is the B-N9-07 include / exclude shape.
type legacyRule struct {
	ModesInclude  []string `json:"modes_include"`
	ModesExclude  []string `json:"modes_exclude"`
	PhasesInclude []string `json:"phases_include"`
	PhasesExclude []string `json:"phases_exclude"`
}

func inList(fact string, vals []string) rules.Condition {
	list := make([]any, len(vals))
	for i, v := range vals {
		list[i] = v
	}
	return rules.Condition{Fact: fact, Op: "in", Value: list}
}

// FromLegacy converts the include / exclude lists to a rule (an empty list adds no condition).
func FromLegacy(modesInclude, modesExclude, phasesInclude, phasesExclude []string) Rule {
	var r Rule
	if len(modesInclude) > 0 {
		r.AllOf = append(r.AllOf, inList(FactLifeMode, modesInclude))
	}
	if len(phasesInclude) > 0 {
		r.AllOf = append(r.AllOf, inList(FactMainPhase, phasesInclude))
	}
	if len(modesExclude) > 0 {
		r.Exclude = append(r.Exclude, inList(FactLifeMode, modesExclude))
	}
	if len(phasesExclude) > 0 {
		r.Exclude = append(r.Exclude, inList(FactMainPhase, phasesExclude))
	}
	return r
}

// ParseRule reads a stored rule: the v6 shape, or a legacy include / exclude object converted on the fly. NULL, JSON
// null and {} are no rule; anything else that does not parse is Broken (an eligibility then allows, as before; a
// contraindication refuses).
func ParseRule(raw db.NullRawJSON) Rule {
	if !raw.Valid {
		return Rule{}
	}
	b := bytes.TrimSpace(raw.V)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		return Rule{}
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		return Rule{Broken: true}
	}
	_, a := keys["all_of"]
	_, o := keys["any_of"]
	_, x := keys["exclude"]
	if a || o || x {
		var r Rule
		if err := json.Unmarshal(b, &r); err != nil {
			return Rule{Broken: true}
		}
		return r
	}
	var l legacyRule
	if err := json.Unmarshal(b, &l); err != nil {
		return Rule{Broken: true}
	}
	return FromLegacy(l.ModesInclude, l.ModesExclude, l.PhasesInclude, l.PhasesExclude)
}

// Allows reports whether the eligibility rule lets the user in (an empty or broken rule allows everyone).
func (r Rule) Allows(f rules.Facts) bool {
	if r.Broken {
		return true
	}
	for _, c := range r.AllOf {
		if !rules.Match(c, f) {
			return false
		}
	}
	for _, g := range r.AnyOf {
		if len(g) > 0 && !slices.ContainsFunc(g, func(c rules.Condition) bool { return rules.Match(c, f) }) {
			return false
		}
	}
	return !slices.ContainsFunc(r.Exclude, func(c rules.Condition) bool { return rules.Match(c, f) })
}

// ExpectedFacts are facts the user's mode makes expected: absent while she is in that mode means unknown, not "does
// not apply", so a contraindication on them fails closed.
var ExpectedFacts = map[string]string{
	"postpartum_week":     "postpartum",
	"pregnancy_week":      "pregnancy",
	"pregnancy_trimester": "pregnancy",
}

type tri int8

const (
	triFalse tri = iota
	triUnknown
	triTrue
)

// condTri evaluates one condition for a contraindication: unknown when the fact is expected but absent.
func condTri(c rules.Condition, f rules.Facts) tri {
	if _, present := f[c.Fact]; !present {
		if mode, ok := ExpectedFacts[c.Fact]; ok && inMode(f, mode) {
			return triUnknown
		}
	}
	if rules.Match(c, f) {
		return triTrue
	}
	return triFalse
}

// inMode reports whether the engine mode or the life mode is mode.
func inMode(f rules.Facts, mode string) bool {
	return f[FactMode] == mode || f[FactLifeMode] == mode
}

// Holds reports whether the contraindication applies to the user (three-valued: an unknown result counts as holding).
// An empty rule never holds; a broken one always does.
func (r Rule) Holds(f rules.Facts) bool {
	if r.Broken {
		return true
	}
	if r.Empty() {
		return false
	}
	res := triTrue
	for _, c := range r.AllOf {
		res = min(res, condTri(c, f))
	}
	for _, g := range r.AnyOf {
		if len(g) == 0 {
			continue
		}
		best := triFalse
		for _, c := range g {
			best = max(best, condTri(c, f))
		}
		res = min(res, best)
	}
	for _, c := range r.Exclude { // NOT(exclusion): true → false, unknown stays unknown
		res = min(res, triTrue-condTri(c, f))
	}
	return res != triFalse
}

// Decide applies the contraindications first, then the eligibility: "" = allowed, else the refusal reason.
func Decide(eligibility, contraindications Rule, f rules.Facts) string {
	if contraindications.Holds(f) {
		return ReasonContraindicated
	}
	if !eligibility.Allows(f) {
		return ReasonNotEligible
	}
	return ""
}

// ToLegacy is the include / exclude lists of a rule that only uses them (the shape FromLegacy writes); ok is false
// when the rule needs the v6 shape (other facts or operators, any_of groups). Older editors read these lists.
func ToLegacy(r Rule) (modesInclude, modesExclude, phasesInclude, phasesExclude []string, ok bool) {
	if len(slices.DeleteFunc(slices.Clone(r.AnyOf), func(g []rules.Condition) bool { return len(g) == 0 })) > 0 {
		return nil, nil, nil, nil, false
	}
	read := func(conds []rules.Condition) (modes, phases []string, ok bool) {
		for _, c := range conds {
			if c.Op != "in" {
				return nil, nil, false
			}
			var vals []string
			for _, v := range listOf(c.Value) {
				s, isStr := v.(string)
				if !isStr {
					return nil, nil, false
				}
				vals = append(vals, s)
			}
			switch {
			case c.Fact == FactLifeMode && modes == nil:
				modes = vals
			case c.Fact == FactMainPhase && phases == nil:
				phases = vals
			default:
				return nil, nil, false
			}
		}
		return modes, phases, true
	}
	mi, pi, ok1 := read(r.AllOf)
	me, pe, ok2 := read(r.Exclude)
	if !ok1 || !ok2 {
		return nil, nil, nil, nil, false
	}
	return mi, me, pi, pe, true
}

func listOf(v any) []any {
	switch l := v.(type) {
	case []any:
		return l
	case []string:
		out := make([]any, len(l))
		for i, s := range l {
			out[i] = s
		}
		return out
	}
	return nil
}
