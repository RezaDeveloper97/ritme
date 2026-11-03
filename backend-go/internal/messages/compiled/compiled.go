// Package compiled holds the message rules compiled for selection (T-M10-05, gap 03 §4.1–4.2): conditions parsed and
// coerced once (numbers as float64, enum / set lists as small sets), hour windows as a 24-bit mask, the served content
// resolved (the approved snapshot of a head waiting for re-approval, T-M10-02, via rules.ActiveRows), every rule
// filed under each slot it may run in (rules.Slots: placement + allowed kind + served) and pre-sorted by the
// selection order. Select then does no parsing, no sorting and at most one allocation per slot.
//
// Compiled is immutable once built: a reload builds a new one and the cache swaps it in atomically (cache.go).
package compiled

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
)

// FactSet is a sorted set of fact names (what facts.Facts.Need takes).
type FactSet []string

// Has reports whether name is in the set.
func (s FactSet) Has(name string) bool {
	_, ok := slices.BinarySearch(s, name)
	return ok
}

// SelectionFacts are the facts selection itself reads besides the rules' conditions: the §5.4.1 invariants and the
// cap windows (rules.CapWindows). The feed prefetches them with the slots' NeedFacts.
var SelectionFacts = FactSet{"confidence", "current_period_start", "cycle_length", "hormonal_contraception",
	"requires_user_input"}

// Compiled is one compiled snapshot of the served message rules.
type Compiled struct {
	// BySlot are the candidates of each slot of rules.Slots, in the static selection order (safety, priority DESC,
	// id ASC); the per-user tie-break (least recently shown) is applied by Select.
	BySlot map[string][]*Rule `json:"-"`
	// NeedFacts are the facts the conditions and exceptions of each slot's rules read (T-M10-03 Need).
	NeedFacts map[string]FactSet `json:"-"`
	// Rev is the cache revision that built this snapshot (1, 2, … per process; 0 = built outside a Cache). It is
	// returned in feed responses for debugging (T-M10-06).
	Rev uint64 `json:"-"`
	// Fingerprint digests the source rows: equal content → equal fingerprint, in every process.
	Fingerprint string
	// Errors are the rows that failed to compile and were skipped.
	Errors []RuleError `json:"-"`
	// Rows are the source rows a Cache compiled this snapshot from (nil outside a Cache); shared, read-only. The feed
	// reads the translations off them instead of a second rows read (T-M9-49b).
	Rows []store.MessageRule `json:"-"`

	slots map[string]*rules.Slot // the slot table at compile time (a pointer per slot: Select copies nothing)
}

// RuleError is one row that did not compile.
type RuleError struct {
	RuleID uint64
	Err    error
}

func (e RuleError) Error() string { return fmt.Sprintf("message rule %d: %v", e.RuleID, e.Err) }

// ErrSnapshot is a failure of the whole snapshot (the cache keeps serving the previous one).
var ErrSnapshot = errors.New("compiled message rules: snapshot rejected")

// Rule is one compiled rule. Src is the parsed rule (content, variants, admin fields) the winners point at.
type Rule struct {
	Src *rules.Rule

	safety, exemptInput, lowConfSensitive, fertile, postSafety bool
	priority                                                   int
	audienceAll                                                bool
	audience                                                   uint32

	scheduled       bool
	start, end      time.Time // Tehran wall clock (EffectiveSchedule); zero = open
	hourMask        uint32    // bit h = hour h allowed
	hasHours        bool
	groups          []group
	exceptions      []cond
	caps            []rules.Cap
	cooldown        time.Duration
	variants        []rules.VariantWeight
	variantSum      int
	safetyVariant   string
	singleVariantOK bool // no hashing needed: one variant or no positive weight
}

// Need is the union of the NeedFacts of slots (sorted, unique; unknown slots add nothing).
func (c *Compiled) Need(slots ...string) FactSet {
	var out FactSet
	for _, s := range slots {
		out = append(out, c.NeedFacts[s]...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Compile compiles the served rows (rules.ActiveRows: active, with the approved snapshot already applied). A row that
// does not compile is skipped and listed in Errors; the snapshot fails as a whole (ErrSnapshot) when two rows share
// an id or when there were rows and none compiled.
func Compile(rows []store.MessageRule) (*Compiled, error) {
	parsed := make([]rules.Rule, 0, len(rows))
	var errs []RuleError
	seen := make(map[uint64]bool, len(rows))
	for i := range rows {
		row := &rows[i]
		if seen[row.ID] {
			return nil, fmt.Errorf("%w: duplicate rule id %d", ErrSnapshot, row.ID)
		}
		seen[row.ID] = true
		if err := checkRow(row); err != nil {
			errs = append(errs, RuleError{RuleID: row.ID, Err: err})
			continue
		}
		parsed = append(parsed, rules.FromRow(row))
	}
	c, ruleErrs := compileRules(parsed)
	errs = append(errs, ruleErrs...)
	if len(rows) > 0 && len(errs) == len(rows) {
		return nil, fmt.Errorf("%w: none of %d rules compiled (first: %s)", ErrSnapshot, len(rows), errs[0].Error())
	}
	c.Errors = errs
	c.Fingerprint = fingerprint(rows)
	return c, nil
}

// CompileRules compiles already parsed rules (admin previews and tests); rules is not modified and must outlive the
// result (winners point into it). Rules that fail to compile are skipped and listed in Errors.
func CompileRules(rs []rules.Rule) *Compiled {
	c, errs := compileRules(rs)
	c.Errors = errs
	return c
}

func compileRules(rs []rules.Rule) (*Compiled, []RuleError) {
	c := &Compiled{BySlot: map[string][]*Rule{}, NeedFacts: map[string]FactSet{}, slots: map[string]*rules.Slot{}}
	for name, s := range rules.Slots {
		c.slots[name] = &s
	}
	var errs []RuleError
	for i := range rs {
		src := &rs[i]
		cr, err := compileRule(src)
		if err != nil {
			errs = append(errs, RuleError{RuleID: src.ID, Err: err})
			continue
		}
		if !src.Served() {
			continue // inactive / not approved: never a candidate
		}
		for name, s := range rules.Slots {
			if s.Reserved || !rules.CandidateOf(name, src.Placement) || !slices.Contains(s.Kinds, src.Kind) {
				continue
			}
			c.BySlot[name] = append(c.BySlot[name], cr)
		}
	}
	for name, list := range c.BySlot {
		slices.SortFunc(list, func(a, b *Rule) int {
			if a.safety != b.safety {
				if a.safety {
					return -1
				}
				return 1
			}
			if a.priority != b.priority {
				return b.priority - a.priority
			}
			switch {
			case a.Src.ID < b.Src.ID:
				return -1
			case a.Src.ID > b.Src.ID:
				return 1
			}
			return 0
		})
		var need FactSet
		for _, r := range list {
			need = r.appendFacts(need)
		}
		slices.Sort(need)
		c.NeedFacts[name] = slices.Compact(need)
	}
	return c, errs
}

func (r *Rule) appendFacts(out FactSet) FactSet {
	for _, g := range r.groups {
		for _, it := range g.items {
			out = append(out, it.fact)
		}
	}
	for _, it := range r.exceptions {
		out = append(out, it.fact)
	}
	return out
}

func compileRule(src *rules.Rule) (cr *Rule, err error) {
	defer func() {
		if p := recover(); p != nil {
			cr, err = nil, fmt.Errorf("panic: %v", p)
		}
	}()
	cr = &Rule{
		Src: src, safety: src.Kind == rules.KindSafety, priority: src.Priority,
		exemptInput:      src.Kind == rules.KindSafety || src.Kind == rules.KindEngine,
		lowConfSensitive: src.Kind == rules.KindClinical && !src.TestsFact("confidence"),
		fertile:          src.IsFertileWindow(),
		postSafety:       src.Kind == rules.KindProduct || src.Kind == rules.KindMarketing,
		audienceAll:      src.AudiencePercent >= 100,
		caps:             src.EffectiveCaps(),
	}
	if src.AudiencePercent > 0 {
		cr.audience = uint32(src.AudiencePercent) //nolint:gosec // < 100 here
	}
	cr.start, cr.end = src.EffectiveSchedule()
	cr.scheduled = !cr.start.IsZero() || !cr.end.IsZero()
	if ws := src.EffectiveHourWindows(); len(ws) > 0 {
		cr.hasHours = true
		for h := range 24 {
			if slices.ContainsFunc(ws, func(w rules.HourWindow) bool { return w.InHours(h) }) {
				cr.hourMask |= 1 << h
			}
		}
	}
	if src.CooldownDays > 0 {
		cr.cooldown = time.Duration(src.CooldownDays) * 24 * time.Hour
	}
	for _, g := range src.Groups {
		cg := group{or: g.Op == "or"}
		for _, it := range g.Items {
			cc, err := compileCond(it)
			if err != nil {
				return nil, err
			}
			cg.items = append(cg.items, cc)
		}
		cr.groups = append(cr.groups, cg)
	}
	for _, it := range src.Exceptions {
		cc, err := compileCond(it)
		if err != nil {
			return nil, err
		}
		cr.exceptions = append(cr.exceptions, cc)
	}
	cr.variants = src.EffectiveVariants()
	cr.safetyVariant = cr.variants[0].Key
	for _, v := range cr.variants {
		cr.variantSum += max(v.Weight, 0)
	}
	cr.singleVariantOK = cr.variantSum <= 0 || len(cr.variants) == 1
	return cr, nil
}

// checkRow rejects a row whose stored JSON does not parse (rules.FromRow would silently read it as empty, which for
// conditions means "every user").
func checkRow(r *store.MessageRule) error {
	cols := []struct {
		name string
		raw  []byte
		into any
	}{
		{"content", []byte(r.Content), &rules.Content{}},
		{"conditions", []byte(r.Conditions), &[]rules.Group{}},
		{"exceptions", []byte(r.Exceptions), &[]rules.Condition{}},
	}
	if r.Topics.Valid {
		cols = append(cols, struct {
			name string
			raw  []byte
			into any
		}{"topics", r.Topics.V, &[]string{}})
	}
	if r.Caps.Valid {
		cols = append(cols, struct {
			name string
			raw  []byte
			into any
		}{"caps", r.Caps.V, &[]rules.Cap{}})
	}
	if r.Variants.Valid {
		cols = append(cols, struct {
			name string
			raw  []byte
			into any
		}{"variants", r.Variants.V, &[]rules.VariantWeight{}})
	}
	if r.HourWindows.Valid {
		cols = append(cols, struct {
			name string
			raw  []byte
			into any
		}{"hour_windows", r.HourWindows.V, &[]rules.HourWindow{}})
	}
	for _, c := range cols {
		s := strings.TrimSpace(string(c.raw))
		if s == "" || s == "null" {
			if c.name == "content" {
				return errors.New("content: empty")
			}
			continue
		}
		if err := json.Unmarshal(c.raw, c.into); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
	}
	return nil
}

// fingerprint is SHA-256 of the rows' JSON (process independent).
func fingerprint(rows []store.MessageRule) string {
	b, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:12])
}
