package rules

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/messages/facts"
)

// Conflict check v2 (T-M10-11, SPEC §5.4.1, gap 03 row 7 / §2.7). Two rules conflict when they run in the same slot
// (both kinds allowed there), their schedules, hour windows (minus the slot's quiet hours) and audiences overlap, and
// their conditions can hold for one user at once: for every fact either rule tests, the value sets intersect (numeric
// intervals, enum / value sets, set facts, booleans, "fact missing"). Conditions are expanded to a disjunctive normal
// form (AND across groups, an "or" group branches, an exception adds its negation); two rules overlap when one
// conjunction of each is jointly satisfiable, and that conjunction's per-fact intersection is the reported region.
// The expansion is bounded (maxDNF): past it, the extra branches are dropped, which can only over-report.

// maxDNF bounds the conjunctions a rule's conditions expand to.
const maxDNF = 256

// Conflict is one rule that can show to the same user, in the same slot, at the same time as the checked one.
type Conflict struct {
	RuleID   uint64
	Priority int
	Kind     string
	// Winner is "this", "other" or "tie" by the selection order (safety first, then priority, then the least recently
	// shown, then the lower id); "tie" only for an unsaved draft (id 0) of equal priority.
	Winner string
	// Safety is true when the other rule is a safety rule (it always shows first).
	Safety bool
	// Slot is the shared slot.
	Slot string
	// TieBreak is true when neither safety nor priority decides: the least recently shown wins, then the lower id
	// (Winner is the answer for a user who saw neither).
	TieBreak bool
	// CoShown is true when the slot shows more than one message (daily_advice, the inbox): both can show together and
	// Winner is only the order.
	CoShown bool
	Overlap Overlap
}

// Overlap is the region where two rules can both show.
type Overlap struct {
	// Start and End bound the common schedule (Tehran wall clock, End exclusive; zero = open).
	Start, End time.Time
	// Hours are the common user-local hour windows ([from, to), from > to wraps midnight); nil = every hour.
	Hours []HourWindow
	// AudiencePercent is the expected share of users inside both audience shares (the buckets are per rule).
	AudiencePercent float64
	// Facts is the per-fact intersection of the satisfiable conjunction, sorted by fact.
	Facts []FactRange
}

// Fact range kinds.
const (
	RangeNumber = "number"
	RangeValue  = "value"
	RangeSet    = "set"
)

// FactRange is the values of one fact inside an overlap.
type FactRange struct {
	Fact string
	Kind string // number | value | set
	// Absent is true when a user without the fact is inside; Present when a user with a value can be.
	Absent, Present bool
	// number: [Min, Max] (nil = unbounded); an int fact reports closed integer bounds.
	Min, Max         *float64
	MinOpen, MaxOpen bool
	// value: the allowed values (nil = any value not in Not).
	Values []string
	// Not are the excluded values (number, value and set).
	Not []string
	// set: members it must contain, and lists it must share one member with.
	Contains []string
	AnyOf    [][]string
}

// SlotOf is the slot a placement runs in (legacy `alerts` rows are inbox rows).
func SlotOf(placement string) string {
	if placement == "alerts" {
		return SlotAlertsInbox
	}
	return placement
}

// Overlaps reports whether a and b can both show to one user at one time, and where.
func Overlaps(a, b *Rule) (Overlap, bool) {
	return overlapOf(a, b, dnfOf(a), dnfOf(b))
}

func overlapOf(a, b *Rule, da, db [][]Condition) (Overlap, bool) {
	var o Overlap
	slot := SlotOf(a.Placement)
	s, known := Slots[slot]
	if SlotOf(b.Placement) != slot || !known || s.Reserved || !slices.Contains(s.Kinds, a.Kind) ||
		!slices.Contains(s.Kinds, b.Kind) {
		return o, false
	}
	as, ae := a.EffectiveSchedule()
	bs, be := b.EffectiveSchedule()
	o.Start, o.End = laterStart(as, bs), earlierEnd(ae, be)
	if !o.Start.IsZero() && !o.End.IsZero() && !o.End.After(o.Start) {
		return o, false
	}
	m := hourMask(a) & hourMask(b)
	if q := s.QuietHours; q != nil {
		m &^= windowMask(*q)
	}
	if m == 0 {
		return o, false
	}
	o.Hours = maskWindows(m)
	pa, pb := audienceOf(a), audienceOf(b)
	if pa <= 0 || pb <= 0 {
		return o, false
	}
	o.AudiencePercent = float64(pa) * float64(pb) / 100
	for _, ca := range da {
		for _, cb := range db {
			if doms, ok := solve(slices.Concat(ca, cb)); ok {
				o.Facts = ranges(doms)
				return o, true
			}
		}
	}
	return o, false
}

// winnerOf is the selection order between r and o (SelectSlot's sort without history).
func winnerOf(r, o *Rule) (winner string, tieBreak bool) {
	if rs, os := r.Kind == KindSafety, o.Kind == KindSafety; rs != os {
		if rs {
			return "this", false
		}
		return "other", false
	}
	switch {
	case r.Priority > o.Priority:
		return "this", false
	case r.Priority < o.Priority:
		return "other", false
	case r.ID == 0:
		return "tie", true
	case r.ID < o.ID:
		return "this", true
	}
	return "other", true
}

// FindConflicts lists the active rules of others that conflict with r (r itself, matched by id, is skipped).
func FindConflicts(r *Rule, others []Rule) []Conflict {
	mine := dnfOf(r)
	var out []Conflict
	for i := range others {
		o := &others[i]
		if (r.ID != 0 && o.ID == r.ID) || !o.IsActive {
			continue
		}
		ov, ok := overlapOf(r, o, mine, dnfOf(o))
		if !ok {
			continue
		}
		w, tie := winnerOf(r, o)
		s := Slots[SlotOf(r.Placement)]
		out = append(out, Conflict{RuleID: o.ID, Priority: o.Priority, Kind: o.Kind, Winner: w, Safety: o.Kind == KindSafety,
			Slot: s.Name, TieBreak: tie, CoShown: s.List || s.Capacity > 1, Overlap: ov})
	}
	return out
}

// ConflictPair is one conflicting pair of a rule set (A.ID < B.ID).
type ConflictPair struct {
	A, B   uint64
	Slot   string
	Winner uint64 // by the selection order, for a user who saw neither
	// Reason is what decides the winner: safety | priority | least_recently_shown (then the lower id).
	Reason  string
	CoShown bool
	Overlap Overlap
}

// Winner reasons of a ConflictPair.
const (
	WinBySafety   = "safety"
	WinByPriority = "priority"
	WinByRecency  = "least_recently_shown"
)

// ConflictPairs lists every conflicting pair among the active rules of all, by slot then ids.
func ConflictPairs(all []Rule) []ConflictPair {
	act := make([]*Rule, 0, len(all))
	for i := range all {
		if all[i].IsActive {
			act = append(act, &all[i])
		}
	}
	slices.SortStableFunc(act, func(x, y *Rule) int {
		if c := cmpStr(SlotOf(x.Placement), SlotOf(y.Placement)); c != 0 {
			return c
		}
		return cmpID(x.ID, y.ID)
	})
	dnfs := make([][][]Condition, len(act))
	for i, r := range act {
		dnfs[i] = dnfOf(r)
	}
	var out []ConflictPair
	for i, a := range act {
		for j := i + 1; j < len(act) && SlotOf(act[j].Placement) == SlotOf(a.Placement); j++ {
			b := act[j]
			ov, ok := overlapOf(a, b, dnfs[i], dnfs[j])
			if !ok {
				continue
			}
			w, tie := winnerOf(a, b)
			p := ConflictPair{A: a.ID, B: b.ID, Slot: SlotOf(a.Placement), Winner: a.ID, Reason: WinByPriority, Overlap: ov}
			if w == "other" {
				p.Winner = b.ID
			}
			switch {
			case tie:
				p.Reason = WinByRecency
			case (a.Kind == KindSafety) != (b.Kind == KindSafety):
				p.Reason = WinBySafety
			}
			s := Slots[p.Slot]
			p.CoShown = s.List || s.Capacity > 1
			out = append(out, p)
		}
	}
	return out
}

func cmpStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpID(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func laterStart(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.After(a)) {
		return b
	}
	return a
}

func earlierEnd(a, b time.Time) time.Time {
	if a.IsZero() || (!b.IsZero() && b.Before(a)) {
		return b
	}
	return a
}

func audienceOf(r *Rule) int {
	if r.Kind == KindSafety { // M-07: safety reaches everyone
		return 100
	}
	return min(r.AudiencePercent, 100)
}

const allHours uint32 = 1<<24 - 1

func windowMask(w HourWindow) uint32 {
	var m uint32
	for h := range 24 {
		if w.inHours(h) {
			m |= 1 << h
		}
	}
	return m
}

func hourMask(r *Rule) uint32 {
	ws := r.EffectiveHourWindows()
	if len(ws) == 0 {
		return allHours
	}
	var m uint32
	for _, w := range ws {
		m |= windowMask(w)
	}
	return m
}

// maskWindows turns an hour mask into [from, to) windows (a run through midnight is one wrapping window); nil = all.
func maskWindows(m uint32) []HourWindow {
	if m == allHours {
		return nil
	}
	var out []HourWindow
	for h := 0; h < 24; {
		if m&(1<<h) == 0 {
			h++
			continue
		}
		from := h
		for h < 24 && m&(1<<h) != 0 {
			h++
		}
		out = append(out, HourWindow{from, h})
	}
	if n := len(out); n > 1 && out[0][0] == 0 && out[n-1][1] == 24 {
		out[0] = HourWindow{out[n-1][0], out[0][1]}
		out = out[:n-1]
	}
	return out
}

// dnfOf expands r's groups and exceptions into conjunctions of atoms (ops of Match plus "absent" and "ncontains").
func dnfOf(r *Rule) [][]Condition {
	conj := [][]Condition{{}}
	branch := func(alts []Condition) {
		if len(alts) == 0 || len(conj)*len(alts) > maxDNF {
			return // always true, or past the bound: dropped (over-reports)
		}
		next := make([][]Condition, 0, len(conj)*len(alts))
		for _, c := range conj {
			for _, a := range alts {
				next = append(next, append(slices.Clip(c), a))
			}
		}
		conj = next
	}
	for _, g := range r.Groups {
		if len(g.Items) == 0 {
			continue
		}
		if g.Op == "or" {
			branch(g.Items)
			continue
		}
		for i := range conj {
			conj[i] = append(slices.Clip(conj[i]), g.Items...)
		}
	}
	for _, e := range r.Exceptions {
		branch(negate(e))
	}
	return conj
}

// negate is the alternatives of NOT c under Match's semantics (a missing fact fails every op but ne / nin); nil when
// NOT c always holds (an op Match never satisfies).
func negate(c Condition) []Condition {
	n := func(op string, v any) Condition { return Condition{Fact: c.Fact, Op: op, Value: v} }
	absent := n("absent", nil)
	switch c.Op {
	case "eq":
		return []Condition{n("ne", c.Value)}
	case "ne":
		return []Condition{n("eq", c.Value)}
	case "in":
		return []Condition{n("nin", c.Value)}
	case "nin":
		return []Condition{n("in", c.Value)}
	case "contains":
		return []Condition{n("ncontains", c.Value)}
	case "gt":
		return []Condition{n("lte", c.Value), absent}
	case "gte":
		return []Condition{n("lt", c.Value), absent}
	case "lt":
		return []Condition{n("gte", c.Value), absent}
	case "lte":
		return []Condition{n("gt", c.Value), absent}
	case "between":
		if l := asList(c.Value); len(l) == 2 {
			return []Condition{n("lt", l[0]), n("gt", l[1]), absent}
		}
	}
	return nil
}

// dom is what one fact may be under a conjunction.
type dom struct {
	kind     string
	isInt    bool
	universe []string // known values (enum / bool), nil = open
	absentOK bool     // a missing fact satisfies every atom so far
	noValue  bool     // no present value satisfies the atoms (an "absent" atom, or an op that never matches)

	// number
	lo, hi         float64
	loOpen, hiOpen bool
	points         []float64 // nil = any in [lo, hi]
	notNum         []float64

	// value and set
	allowed map[string]bool // value: nil = any
	not     map[string]bool
	req     map[string]bool // set: must contain
	anyOf   [][]string      // set: must share one member with each
}

func newDom(fact string, atoms []Condition) *dom {
	d := &dom{absentOK: true, lo: math.Inf(-1), hi: math.Inf(1), not: map[string]bool{}, req: map[string]bool{}}
	if def, ok := facts.Lookup(fact); ok {
		switch {
		case def.Type.Numeric():
			d.kind, d.isInt = RangeNumber, def.Type == facts.TypeInt
		case def.Type == facts.TypeSet:
			d.kind, d.universe = RangeSet, def.Enum
		case def.Type == facts.TypeBool:
			d.kind, d.universe = RangeValue, []string{"false", "true"}
		default:
			d.kind, d.universe = RangeValue, def.Enum
		}
		return d
	}
	d.kind = RangeValue // not in the registry: numeric when an atom compares numbers
	for _, a := range atoms {
		if a.Fact != fact {
			continue
		}
		_, num := toFloat(a.Value)
		if num || slices.Contains([]string{"gt", "gte", "lt", "lte", "between"}, a.Op) {
			d.kind = RangeNumber
		}
	}
	return d
}

func valueKey(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	}
	if f, ok := toFloat(v); ok {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return fmt.Sprint(v)
}

func (d *dom) tightenLo(x float64, open bool) {
	if x > d.lo || (x == d.lo && open) {
		d.lo, d.loOpen = x, open
	}
}

func (d *dom) tightenHi(x float64, open bool) {
	if x < d.hi || (x == d.hi && open) {
		d.hi, d.hiOpen = x, open
	}
}

func (d *dom) intersectValues(vals []string) {
	next := map[string]bool{}
	for _, v := range vals {
		if d.allowed == nil || d.allowed[v] {
			next[v] = true
		}
	}
	d.allowed = next
}

func (d *dom) intersectPoints(ps []float64) {
	if d.points == nil {
		d.points = slices.Clone(ps)
		if d.points == nil {
			d.points = []float64{}
		}
		return
	}
	d.points = slices.DeleteFunc(d.points, func(p float64) bool { return !slices.Contains(ps, p) })
}

// apply narrows d by one atom.
func (d *dom) apply(a Condition) {
	if a.Op == "absent" { // a missing fact satisfies it, no value does
		d.noValue = true
		return
	}
	if a.Op != "ne" && a.Op != "nin" && a.Op != "ncontains" {
		d.absentOK = false
	}
	switch d.kind {
	case RangeNumber:
		d.applyNumber(a)
	case RangeSet:
		d.applySet(a)
	default:
		d.applyValue(a)
	}
}

func (d *dom) applyNumber(a Condition) {
	x, isNum := toFloat(a.Value)
	nums := func() []float64 {
		var out []float64
		for _, v := range asList(a.Value) {
			if f, ok := toFloat(v); ok {
				out = append(out, f)
			}
		}
		return out
	}
	switch a.Op {
	case "eq":
		if !isNum {
			d.noValue = true
			return
		}
		d.intersectPoints([]float64{x})
	case "ne":
		if isNum {
			d.notNum = append(d.notNum, x)
		}
	case "in":
		d.intersectPoints(nums())
	case "nin":
		d.notNum = append(d.notNum, nums()...)
	case "gt", "gte", "lt", "lte":
		if !isNum {
			d.noValue = true
			return
		}
		switch a.Op {
		case "gt":
			d.tightenLo(x, true)
		case "gte":
			d.tightenLo(x, false)
		case "lt":
			d.tightenHi(x, true)
		default:
			d.tightenHi(x, false)
		}
	case "between":
		l := asList(a.Value)
		if len(l) != 2 {
			d.noValue = true
			return
		}
		lo, ok1 := toFloat(l[0])
		hi, ok2 := toFloat(l[1])
		if !ok1 || !ok2 {
			d.noValue = true
			return
		}
		d.tightenLo(lo, false)
		d.tightenHi(hi, false)
	case "contains":
		d.noValue = true // Match: a number never contains anything
	}
}

func (d *dom) applyValue(a Condition) {
	switch a.Op {
	case "eq":
		d.intersectValues([]string{valueKey(a.Value)})
	case "ne":
		d.not[valueKey(a.Value)] = true
	case "in":
		var vs []string
		for _, v := range asList(a.Value) {
			vs = append(vs, valueKey(v))
		}
		d.intersectValues(vs)
	case "nin":
		for _, v := range asList(a.Value) {
			d.not[valueKey(v)] = true
		}
	case "gt", "gte", "lt", "lte", "between":
		d.noValue = true // Match compares numbers only
	}
	// contains / ncontains on a text value is a substring test: not narrowed (over-reports).
}

func (d *dom) applySet(a Condition) {
	switch a.Op {
	case "contains":
		d.req[valueKey(a.Value)] = true
	case "ncontains":
		d.not[valueKey(a.Value)] = true
	case "in":
		var vs []string
		for _, v := range asList(a.Value) {
			vs = append(vs, valueKey(v))
		}
		d.anyOf = append(d.anyOf, vs)
	case "nin":
		for _, v := range asList(a.Value) {
			d.not[valueKey(v)] = true
		}
	case "eq", "gt", "gte", "lt", "lte", "between":
		d.noValue = true // Match never matches a list with these
	}
}

// intBounds are the closed integer bounds of an int fact.
func (d *dom) intBounds() (lo, hi float64) {
	lo, hi = math.Ceil(d.lo), math.Floor(d.hi)
	if d.loOpen && lo == d.lo {
		lo++
	}
	if d.hiOpen && hi == d.hi {
		hi--
	}
	return lo, hi
}

func (d *dom) inNumRange(p float64) bool {
	if d.isInt && p != math.Trunc(p) {
		return false
	}
	if p < d.lo || (p == d.lo && d.loOpen) || p > d.hi || (p == d.hi && d.hiOpen) {
		return false
	}
	return !slices.Contains(d.notNum, p)
}

func (d *dom) valueOK(v string) bool {
	return !d.not[v] && (d.universe == nil || slices.Contains(d.universe, v))
}

// hasValue reports whether some present value satisfies the atoms.
func (d *dom) hasValue() bool {
	if d.noValue {
		return false
	}
	switch d.kind {
	case RangeNumber:
		if d.points != nil {
			return slices.ContainsFunc(d.points, d.inNumRange)
		}
		if d.isInt {
			lo, hi := d.intBounds()
			if lo > hi {
				return false
			}
			if hi-lo > 1e6 {
				return true
			}
			excluded := 0
			for _, p := range slices.Compact(slices.Sorted(slices.Values(d.notNum))) {
				if p >= lo && p <= hi && p == math.Trunc(p) {
					excluded++
				}
			}
			return hi-lo+1 > float64(excluded)
		}
		if d.lo < d.hi {
			return true
		}
		return d.lo == d.hi && !d.loOpen && !d.hiOpen && !slices.Contains(d.notNum, d.lo)
	case RangeSet:
		for v := range d.req {
			if !d.valueOK(v) {
				return false
			}
		}
		for _, l := range d.anyOf {
			if !slices.ContainsFunc(l, d.valueOK) {
				return false
			}
		}
		return true
	}
	if d.allowed != nil {
		for v := range d.allowed {
			if d.valueOK(v) {
				return true
			}
		}
		return false
	}
	if d.universe != nil {
		return slices.ContainsFunc(d.universe, d.valueOK)
	}
	return true
}

// solve builds the per-fact domains of a conjunction; ok is false when one fact has no possible value.
func solve(atoms []Condition) (map[string]*dom, bool) {
	doms := map[string]*dom{}
	for _, a := range atoms {
		d := doms[a.Fact]
		if d == nil {
			d = newDom(a.Fact, atoms)
			doms[a.Fact] = d
		}
		d.apply(a)
	}
	for _, d := range doms {
		if !d.absentOK && !d.hasValue() {
			return nil, false
		}
	}
	return doms, true
}

func sortedSet(m map[string]bool, keep func(string) bool) []string {
	out := []string{}
	for k := range m {
		if keep == nil || keep(k) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// ranges reports the domains, sorted by fact.
func ranges(doms map[string]*dom) []FactRange {
	names := make([]string, 0, len(doms))
	for n := range doms {
		names = append(names, n)
	}
	slices.Sort(names)
	out := make([]FactRange, 0, len(names))
	for _, n := range names {
		d := doms[n]
		fr := FactRange{Fact: n, Kind: d.kind, Absent: d.absentOK, Present: d.hasValue(), Not: sortedSet(d.not, nil)}
		switch d.kind {
		case RangeNumber:
			fr.Not = nil
			lo, hi, loOpen, hiOpen := d.lo, d.hi, d.loOpen, d.hiOpen
			if d.isInt {
				lo, hi = d.intBounds()
				loOpen, hiOpen = false, false
			}
			if d.points != nil {
				ps := slices.DeleteFunc(slices.Clone(d.points), func(p float64) bool { return !d.inNumRange(p) })
				slices.Sort(ps)
				lo, hi, loOpen, hiOpen = math.Inf(-1), math.Inf(1), false, false
				if len(ps) > 0 {
					lo, hi = ps[0], ps[len(ps)-1]
				}
			}
			if !math.IsInf(lo, 0) {
				fr.Min = &lo
			}
			if !math.IsInf(hi, 0) {
				fr.Max = &hi
			}
			fr.MinOpen, fr.MaxOpen = loOpen, hiOpen
			for _, p := range slices.Compact(slices.Sorted(slices.Values(d.notNum))) {
				if (fr.Min == nil || p >= *fr.Min) && (fr.Max == nil || p <= *fr.Max) {
					fr.Not = append(fr.Not, valueKey(p))
				}
			}
		case RangeSet:
			fr.Contains = sortedSet(d.req, nil)
			for _, l := range d.anyOf {
				keep := slices.DeleteFunc(slices.Clone(l), func(v string) bool { return !d.valueOK(v) })
				slices.Sort(keep)
				fr.AnyOf = append(fr.AnyOf, slices.Compact(keep))
			}
		default:
			switch {
			case d.allowed != nil:
				fr.Values = sortedSet(d.allowed, d.valueOK)
				fr.Not = nil
			case d.universe != nil:
				fr.Values = slices.DeleteFunc(slices.Clone(d.universe), func(v string) bool { return !d.valueOK(v) })
				fr.Not = nil
			}
		}
		out = append(out, fr)
	}
	return out
}
