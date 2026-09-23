package ojson

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// Placeholders written into goldens in place of volatile values.
const (
	IgnoredPlaceholder = "<ignored>"
	patternPrefix      = "<pattern:"
)

// Path is a location inside a JSON value: object keys and array indexes (decimal).
type Path []string

func (p Path) String() string {
	var b strings.Builder
	b.WriteString("$")
	for _, s := range p {
		if _, err := strconv.Atoi(s); err == nil {
			b.WriteString("[" + s + "]")
			continue
		}
		b.WriteString("." + s)
	}
	return b.String()
}

func (p Path) with(s string) Path {
	out := make(Path, len(p), len(p)+1)
	copy(out, p)
	return append(out, s)
}

// Rule is a compiled path selector. Syntax: dot-separated keys, `[n]` or `.n` for
// array indexes, `*` for exactly one key/index, `**` for any depth (including none):
// `data.access_token`, `data.items[*].id`, `**.created_at`.
type Rule struct {
	Raw  string
	segs []string
}

// ParseRule compiles a selector.
func ParseRule(s string) (Rule, error) {
	norm := strings.NewReplacer("[", ".", "]", "").Replace(strings.TrimSpace(s))
	norm = strings.TrimPrefix(strings.TrimPrefix(norm, "$"), ".")
	if norm == "" {
		return Rule{}, fmt.Errorf("empty path selector %q", s)
	}
	segs := strings.Split(norm, ".")
	for _, seg := range segs {
		if seg == "" {
			return Rule{}, fmt.Errorf("bad path selector %q", s)
		}
	}
	return Rule{Raw: s, segs: segs}, nil
}

// Match reports whether the rule selects path p.
func (r Rule) Match(p Path) bool { return matchSegs(r.segs, p) }

func matchSegs(rule []string, p Path) bool {
	if len(rule) == 0 {
		return len(p) == 0
	}
	if rule[0] == "**" {
		for i := 0; i <= len(p); i++ {
			if matchSegs(rule[1:], p[i:]) {
				return true
			}
		}
		return false
	}
	if len(p) == 0 {
		return false
	}
	if rule[0] != "*" && rule[0] != p[0] {
		return false
	}
	return matchSegs(rule[1:], p[1:])
}

// PatternRule requires the value at matching paths to be a scalar whose text matches Re.
type PatternRule struct {
	Rule
	Re *regexp.Regexp
}

// Rules is the volatile-field configuration for one step.
type Rules struct {
	Ignore   []Rule
	Patterns []PatternRule
}

// NewRules compiles ignore selectors and selector→regex patterns.
func NewRules(ignore []string, patterns map[string]string) (*Rules, error) {
	r := &Rules{}
	for _, s := range ignore {
		rule, err := ParseRule(s)
		if err != nil {
			return nil, err
		}
		r.Ignore = append(r.Ignore, rule)
	}
	for sel, expr := range patterns {
		rule, err := ParseRule(sel)
		if err != nil {
			return nil, err
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("pattern for %q: %w", sel, err)
		}
		r.Patterns = append(r.Patterns, PatternRule{Rule: rule, Re: re})
	}
	return r, nil
}

func (r *Rules) ignored(p Path) bool {
	if r == nil {
		return false
	}
	for _, rule := range r.Ignore {
		if rule.Match(p) {
			return true
		}
	}
	return false
}

func (r *Rules) pattern(p Path) *PatternRule {
	if r == nil {
		return nil
	}
	for i := range r.Patterns {
		if r.Patterns[i].Match(p) {
			return &r.Patterns[i]
		}
	}
	return nil
}

// Diff is one difference between the golden and the actual value.
type Diff struct {
	Path Path
	Msg  string
}

func (d Diff) String() string { return d.Path.String() + ": " + d.Msg }

// Mask replaces volatile values in v (in place) with placeholders so a golden is
// deterministic. Values under pattern rules must match their regex; mismatches are
// returned as diffs (the recording is suspicious).
func Mask(v *Value, rules *Rules) []Diff {
	var out []Diff
	var walk func(v *Value, p Path) *Value
	walk = func(v *Value, p Path) *Value {
		if rules.ignored(p) {
			return NewString(IgnoredPlaceholder)
		}
		if pr := rules.pattern(p); pr != nil {
			if msg := checkPattern(v, pr); msg != "" {
				out = append(out, Diff{Path: p, Msg: msg})
			}
			return NewString(patternPrefix + pr.Re.String() + ">")
		}
		switch v.Kind {
		case Array:
			for i := range v.Arr {
				v.Arr[i] = walk(v.Arr[i], p.with(strconv.Itoa(i)))
			}
		case Object:
			for i, k := range v.Keys {
				v.Vals[i] = walk(v.Vals[i], p.with(k))
			}
		}
		return v
	}
	walk(v, Path{})
	return out
}

func checkPattern(v *Value, pr *PatternRule) string {
	if v.Kind == Array || v.Kind == Object {
		return fmt.Sprintf("pattern %s expects a scalar, got %s", pr.Re, v.Kind)
	}
	if !pr.Re.MatchString(v.Scalar()) {
		return fmt.Sprintf("value %s does not match pattern %s", describe(v), pr.Re)
	}
	return ""
}

// Compare diffs the golden want against got under rules. Order of object keys is
// ignored; everything else is strict (see the package comment).
func Compare(want, got *Value, rules *Rules) []Diff {
	var out []Diff
	compare(want, got, Path{}, rules, &out)
	return out
}

func compare(want, got *Value, p Path, rules *Rules, out *[]Diff) {
	if rules.ignored(p) {
		return
	}
	if pr := rules.pattern(p); pr != nil {
		if msg := checkPattern(got, pr); msg != "" {
			*out = append(*out, Diff{Path: p, Msg: msg})
		}
		return
	}
	if want.Kind != got.Kind {
		*out = append(*out, Diff{Path: p, Msg: fmt.Sprintf("type: want %s %s, got %s %s",
			want.Kind, describe(want), got.Kind, describe(got))})
		return
	}
	switch want.Kind {
	case Null:
	case Bool:
		if want.Bool != got.Bool {
			*out = append(*out, Diff{Path: p, Msg: fmt.Sprintf("want %v, got %v", want.Bool, got.Bool)})
		}
	case String:
		if want.Str != got.Str {
			*out = append(*out, Diff{Path: p, Msg: fmt.Sprintf("want %s, got %s", describe(want), describe(got))})
		}
	case Number:
		if !NumbersEqual(want.Num, got.Num) {
			*out = append(*out, Diff{Path: p, Msg: fmt.Sprintf("want number %s, got %s", want.Num, got.Num)})
		}
	case Array:
		if len(want.Arr) != len(got.Arr) {
			*out = append(*out, Diff{Path: p, Msg: fmt.Sprintf("array length: want %d, got %d", len(want.Arr), len(got.Arr))})
		}
		for i := 0; i < len(want.Arr) && i < len(got.Arr); i++ {
			compare(want.Arr[i], got.Arr[i], p.with(strconv.Itoa(i)), rules, out)
		}
	case Object:
		for i, k := range want.Keys {
			g := got.Get(k)
			if g == nil {
				*out = append(*out, Diff{Path: p.with(k), Msg: "missing (want " + describe(want.Vals[i]) + ")"})
				continue
			}
			compare(want.Vals[i], g, p.with(k), rules, out)
		}
		for i, k := range got.Keys {
			if want.Get(k) == nil {
				*out = append(*out, Diff{Path: p.with(k), Msg: "unexpected key (got " + describe(got.Vals[i]) + ")"})
			}
		}
	}
}

// NumbersEqual compares two JSON number literals: same literal kind (integer vs.
// fraction/exponent) and the same numeric value.
func NumbersEqual(a, b string) bool {
	if a == b {
		return true
	}
	if isIntLiteral(a) != isIntLiteral(b) {
		return false
	}
	fa, ok1 := new(big.Float).SetPrec(256).SetString(a)
	fb, ok2 := new(big.Float).SetPrec(256).SetString(b)
	return ok1 && ok2 && fa.Cmp(fb) == 0
}

func isIntLiteral(s string) bool { return !strings.ContainsAny(s, ".eE") }

func describe(v *Value) string {
	s := string(Encode(v, ""))
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
}
