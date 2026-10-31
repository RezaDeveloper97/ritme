package compiled

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ritme/backend-go/internal/messages/rules"
)

// Compiled conditions: rules.Match with the value side coerced once. The semantics are rules.Match's, exactly (the
// equivalence test runs both over random rules): a missing fact matches nothing except ne / nin; numbers compare as
// float64; a bool fact equals "true" / "false"; a set fact is "in" when one of its members is. Dates are compared as
// their Y-m-d strings (eq / ne), as the parsed path does.

type op uint8

const (
	opEq op = iota
	opNe
	opIn
	opNin
	opContains
	opBetween
	opGt
	opGte
	opLt
	opLte
)

var ops = map[string]op{"eq": opEq, "ne": opNe, "in": opIn, "nin": opNin, "contains": opContains,
	"between": opBetween, "gt": opGt, "gte": opGte, "lt": opLt, "lte": opLte}

// smallSet is the string side of in / nin: a slice up to 8 members, else a map.
type smallSet struct {
	list []string
	m    map[string]struct{}
}

func (s *smallSet) add(v string) {
	if s.m != nil {
		s.m[v] = struct{}{}
		return
	}
	if slices.Contains(s.list, v) {
		return
	}
	s.list = append(s.list, v)
	if len(s.list) > 8 {
		s.m = make(map[string]struct{}, len(s.list))
		for _, x := range s.list {
			s.m[x] = struct{}{}
		}
		s.list = nil
	}
}

func (s *smallSet) has(v string) bool {
	if s.m != nil {
		_, ok := s.m[v]
		return ok
	}
	return slices.Contains(s.list, v)
}

// scalar is a coerced value.
type scalar struct {
	num           float64
	str           string
	b             bool
	isNum, isBool bool
	isStr         bool
}

func scalarOf(v any) scalar {
	var s scalar
	if n, ok := toFloat(v); ok {
		s.num, s.isNum = n, true
	}
	switch x := v.(type) {
	case bool:
		s.b, s.isBool = x, true
	case string:
		s.str, s.isStr = x, true
	}
	return s
}

type cond struct {
	fact string
	op   op
	val  scalar // eq ne contains gt…lte

	// in / nin
	strs              smallSet
	nums              []float64
	hasTrue, hasFalse bool

	// between
	lo, hi  float64
	rangeOK bool
}

type group struct {
	or    bool
	items []cond
}

func compileCond(c rules.Condition) (cond, error) {
	o, ok := ops[c.Op]
	if !ok {
		return cond{}, fmt.Errorf("condition on %q: unknown op %q", c.Fact, c.Op)
	}
	out := cond{fact: c.Fact, op: o, val: scalarOf(c.Value)}
	switch o {
	case opIn, opNin:
		for _, w := range asList(c.Value) {
			s := scalarOf(w)
			if s.isNum {
				out.nums = append(out.nums, s.num)
			}
			if s.isBool {
				if s.b {
					out.hasTrue = true
				} else {
					out.hasFalse = true
				}
			}
			if s.isStr {
				out.strs.add(s.str)
			}
		}
	case opBetween:
		if l := asList(c.Value); len(l) == 2 {
			lo, ok1 := toFloat(l[0])
			hi, ok2 := toFloat(l[1])
			out.lo, out.hi, out.rangeOK = lo, hi, ok1 && ok2
		}
	}
	return out, nil
}

// equal is rules' equal(got, v) with v pre-coerced.
func (v *scalar) equal(got any) bool {
	if g, ok := toFloat(got); ok && v.isNum {
		return g == v.num
	}
	switch g := got.(type) {
	case bool:
		if v.isBool {
			return g == v.b
		}
		if v.isStr {
			return strconv.FormatBool(g) == v.str
		}
	case string:
		return v.isStr && g == v.str
	}
	return false
}

func (c *cond) inList(got any) bool {
	switch g := got.(type) {
	case []string:
		return slices.ContainsFunc(g, c.strs.has)
	case bool:
		if g {
			return c.hasTrue || c.strs.has("true")
		}
		return c.hasFalse || c.strs.has("false")
	case string:
		return c.strs.has(g)
	}
	if n, ok := toFloat(got); ok {
		return slices.Contains(c.nums, n)
	}
	return false
}

func (c *cond) match(f rules.Facts) bool {
	got := f[c.fact]
	switch c.op {
	case opEq:
		return c.val.equal(got)
	case opNe:
		return !c.val.equal(got)
	case opIn:
		return c.inList(got)
	case opNin:
		return !c.inList(got)
	case opContains:
		if l, ok := got.([]string); ok {
			return c.val.isStr && slices.Contains(l, c.val.str)
		}
		s, ok := got.(string)
		return ok && c.val.isStr && strings.Contains(s, c.val.str)
	case opBetween:
		g, ok := toFloat(got)
		return ok && c.rangeOK && g >= c.lo && g <= c.hi
	}
	g, ok := toFloat(got)
	if !ok || !c.val.isNum {
		return false
	}
	w := c.val.num
	switch c.op {
	case opGt:
		return g > w
	case opGte:
		return g >= w
	case opLt:
		return g < w
	}
	return g <= w
}

func (g *group) match(f rules.Facts) bool {
	if len(g.items) == 0 {
		return true
	}
	if g.or {
		for i := range g.items {
			if g.items[i].match(f) {
				return true
			}
		}
		return false
	}
	for i := range g.items {
		if !g.items[i].match(f) {
			return false
		}
	}
	return true
}

// toFloat and asList are rules' coercions (unexported there).
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
