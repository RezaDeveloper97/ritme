// Package facts is the message fact registry v6 (T-M10-03, SPEC §5.3, D-161, DECISIONS.md M-06, M-14): every fact a
// message rule may test, typed and described, and the lazy producers that compute them for one user and request.
//
// Each group file (engine.go, account.go, logs.go, vitals.go, pregnancy.go, postpartum.go, contraception.go) registers
// its definitions and its producer from init() through Register / RegisterGroup; a module task adds its own file the
// same way, there is no central list. A request builds a Facts: a producer group runs at most once (memoised), and
// only when a fact of it is needed (Need), so a home render costs one engine state load plus the few small queries of
// the groups its rules reference.
//
// A fact whose input is missing is absent (never a guessed value): a condition on it then matches nothing except ne /
// nin (rules.Match). Fertility facts honour limits.fertility_suppressed (engine.go). The package never imports admin
// or http; producers read through the small read-only Sources interfaces, implemented by messages.StoreSource.
package facts

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Type is a fact's value type.
type Type string

// The fact types. Values are float64 (int, float), bool, string (enum, date as Y-m-d) and []string (set).
const (
	TypeInt   Type = "int"
	TypeFloat Type = "float"
	TypeBool  Type = "bool"
	TypeEnum  Type = "enum"
	TypeSet   Type = "set"
	TypeDate  Type = "date"
)

// Numeric reports whether t compares as a number (between, gt, gte, lt, lte).
func (t Type) Numeric() bool { return t == TypeInt || t == TypeFloat }

// Def is one registered fact.
type Def struct {
	Name string
	Type Type
	// Enum is the closed value set of an enum fact or of a set fact's members; nil = open (e.g. locale: the languages
	// table is data, not code).
	Enum []string
	Unit string
	// Group is the producer group that computes the fact.
	Group string
	// Description is the editor help text keyed by language code (falls back to the default language in the editor).
	Description map[string]string
	// MemberSafe marks a fact a family member's fact set may expose (M11 member projections).
	MemberSafe bool
	// Pending names the task that fills the fact; until then it always holds its default (segment: empty).
	Pending string
	// Variable marks a text variable that is not a condition fact (name): produced for rendering, rejected in conditions.
	Variable bool
}

// Values are computed facts by name.
type Values map[string]any

// Producer computes every fact of its group for f's user. It may read other groups through f.Need (memoised).
type Producer func(ctx context.Context, f *Facts) (Values, error)

var (
	defs      = map[string]Def{}
	producers = map[string]Producer{}
)

// Register adds fact definitions (from a group file's init). A duplicate name panics: two files must not define one
// fact two ways.
func Register(ds ...Def) {
	for _, d := range ds {
		if _, dup := defs[d.Name]; dup {
			panic("facts: duplicate fact " + d.Name)
		}
		if d.Group == "" {
			panic("facts: fact " + d.Name + " has no group")
		}
		defs[d.Name] = d
	}
}

// RegisterGroup adds the producer of a group.
func RegisterGroup(group string, p Producer) {
	if _, dup := producers[group]; dup {
		panic("facts: duplicate group " + group)
	}
	producers[group] = p
}

// Lookup returns the definition of name.
func Lookup(name string) (Def, bool) {
	d, ok := defs[name]
	return d, ok
}

// Defs are every registered definition in a stable order (by name).
func Defs() []Def {
	out := make([]Def, 0, len(defs))
	for _, d := range defs {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Groups are the registered producer groups, sorted.
func Groups() []string {
	out := make([]string, 0, len(producers))
	for g := range producers {
		out = append(out, g)
	}
	slices.Sort(out)
	return out
}

// FertilityFacts are the facts (and text variables) that make a message a fertile-window message: the validator then
// requires the fertile_window topic, and selection removes it under hormonal contraception (M-06). It also lists
// names that are not registered facts, so an older rule or text using them is still recognised.
var FertilityFacts = []string{"is_fertile_window", "fertile_window", "is_ovulation_day", "ovulation_day",
	"days_to_ovulation", "fertility_level", "fertile_window_start", "fertile_window_end"}

// Sources are the read-only module views the producers read for one user and request (nil = that module has
// nothing, so its facts are absent).
type Sources struct {
	Engine        EngineSource
	Account       AccountSource
	Logs          LogSource
	Vitals        VitalsSource
	Pregnancy     PregnancySource
	Postpartum    PostpartumSource
	Contraception ContraceptionSource
}

// Facts are one user's facts for one request: lazy and memoised per producer group.
type Facts struct {
	src    Sources
	today  civildate.Date
	locale string

	vals Values
	done map[string]bool
	errs map[string]error

	engine        *engineMemo
	accountRow    *Account
	accountLoaded bool
}

// New returns the lazy facts of a user on today (the user-local day) for a request in locale ("" = unknown).
func New(src Sources, today civildate.Date, locale string) *Facts {
	return &Facts{src: src, today: today, locale: locale, vals: Values{}, done: map[string]bool{}, errs: map[string]error{}}
}

// Today is the target day (LocalToday).
func (f *Facts) Today() civildate.Date { return f.today }

// GroupOf is the producer group of a registered fact name ("" when unknown).
func GroupOf(name string) string {
	if d, ok := defs[name]; ok {
		return d.Group
	}
	return ""
}

// Need computes the groups of names (unknown names are ignored) in one pass: each group runs once per request.
func (f *Facts) Need(ctx context.Context, names ...string) error {
	var groups []string
	for _, n := range names {
		if g := GroupOf(n); g != "" && !slices.Contains(groups, g) {
			groups = append(groups, g)
		}
	}
	slices.Sort(groups)
	for _, g := range groups {
		if err := f.group(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

// All computes every group.
func (f *Facts) All(ctx context.Context) error {
	for _, g := range Groups() {
		if err := f.group(ctx, g); err != nil {
			return err
		}
	}
	return nil
}

// group runs one producer group (memoised, errors included).
func (f *Facts) group(ctx context.Context, g string) error {
	if f.done[g] {
		return f.errs[g]
	}
	f.done[g] = true // set first: a producer reading its own group sees the memo, never recurses
	p, ok := producers[g]
	if !ok {
		return nil
	}
	v, err := p(ctx, f)
	if err != nil {
		f.errs[g] = fmt.Errorf("facts: %s: %w", g, err)
		return f.errs[g]
	}
	for k, x := range v {
		f.vals[k] = x
	}
	return nil
}

// Get returns one fact (computing its group); ok is false when the fact is absent.
func (f *Facts) Get(ctx context.Context, name string) (any, bool, error) {
	if err := f.Need(ctx, name); err != nil {
		return nil, false, err
	}
	v, ok := f.vals[name]
	return v, ok, nil
}

// Values are the facts computed so far (a copy).
func (f *Facts) Values() Values {
	out := make(Values, len(f.vals))
	for k, v := range f.vals {
		out[k] = v
	}
	return out
}

// value reads a computed fact (no computing).
func (f *Facts) value(name string) (any, bool) {
	v, ok := f.vals[name]
	return v, ok
}

// desc is a Description of the fa and en seed texts (the editor falls back to the default language for others).
func desc(fa, en string) map[string]string {
	return map[string]string{"fa": fa, "en": en}
}

// sortedKeys of a set.
func setOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// IsDate reports whether s is a Y-m-d date.
func IsDate(s string) bool {
	_, err := civildate.Parse(strings.TrimSpace(s))
	return err == nil
}
