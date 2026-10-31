package messages

import (
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/admin/httpadmin"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Conflict check v2 (T-M10-11, rules/conflicts.go): GET /messages/rules/conflicts?slot= lists every conflicting pair
// of the active rules — same slot, overlapping schedule, hours and audience, and condition value sets that intersect
// for every fact — with the overlapping region and the winner by the selection order (safety always wins). The
// per-rule check of the editor (POST /messages/rules/conflicts, GET /messages/rules/:id) runs the same v2 check.

// ConflictPairs is GET /messages/rules/conflicts.
func (h *Handlers) ConflictPairs(c fiber.Ctx) error {
	slot := c.Query("slot")
	if slot != "" && !slices.Contains(SimulateSlots, slot) && slot != "alerts" {
		return httpadmin.FieldError("slot", "The slot is invalid.")
	}
	by, titles, err := h.activeByPlacement(c, nil)
	if err != nil {
		return err
	}
	var all []rules.Rule
	for _, p := range rules.Placements {
		if slot == "" || rules.SlotOf(p) == rules.SlotOf(slot) {
			all = append(all, by[p]...)
		}
	}
	pairs := rules.ConflictPairs(all)
	kinds := make(map[uint64]*rules.Rule, len(all))
	for i := range all {
		kinds[all[i].ID] = &all[i]
	}
	side := func(id uint64) *jsonx.OrderedMap {
		r := kinds[id]
		return jsonx.Obj("rule_id", id, "title", titles[id], "kind", r.Kind, "priority", r.Priority)
	}
	items := make([]*jsonx.OrderedMap, 0, len(pairs))
	for _, p := range pairs {
		items = append(items, jsonx.Obj("slot", p.Slot, "a", side(p.A), "b", side(p.B), "winner", p.Winner,
			"decided_by", p.Reason, "co_shown", p.CoShown, "overlap", overlapJSON(&p.Overlap)))
	}
	return httpadmin.OK(c, jsonx.Obj("count", len(items), "conflicts", items))
}

// ConflictV2JSON is one conflict of the editor's check with its v2 fields (slot, tie_break, co_shown, overlap) after
// the pre-v2 ones; rules.go's conflictJSON can switch to it (T-M10-11 report).
func ConflictV2JSON(c *rules.Conflict, title string) *jsonx.OrderedMap {
	return jsonx.Obj("rule_id", c.RuleID, "title", title, "priority", c.Priority, "kind", c.Kind, "winner", c.Winner,
		"safety", c.Safety, "slot", c.Slot, "tie_break", c.TieBreak, "co_shown", c.CoShown, "overlap", overlapJSON(&c.Overlap))
}

func overlapTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(rules.ScheduleLayout)
}

func overlapJSON(o *rules.Overlap) *jsonx.OrderedMap {
	var hours any
	if o.Hours != nil {
		hs := make([][2]int, len(o.Hours))
		for i, w := range o.Hours {
			hs[i] = w
		}
		hours = hs
	}
	fs := make([]*jsonx.OrderedMap, 0, len(o.Facts))
	for i := range o.Facts {
		fs = append(fs, factRangeJSON(&o.Facts[i]))
	}
	return jsonx.Obj("starts_at", overlapTime(o.Start), "ends_at", overlapTime(o.End), "hours", hours,
		"audience_percent", o.AudiencePercent, "facts", fs)
}

func factRangeJSON(f *rules.FactRange) *jsonx.OrderedMap {
	o := jsonx.Obj("fact", f.Fact, "kind", f.Kind, "absent", f.Absent, "present", f.Present)
	strs := func(v []string) []string {
		if v == nil {
			return []string{}
		}
		return v
	}
	switch f.Kind {
	case rules.RangeNumber:
		var lo, hi any
		if f.Min != nil {
			lo = *f.Min
		}
		if f.Max != nil {
			hi = *f.Max
		}
		o.Set("min", lo)
		o.Set("max", hi)
		o.Set("min_open", f.MinOpen)
		o.Set("max_open", f.MaxOpen)
		o.Set("not", strs(f.Not))
	case rules.RangeSet:
		anyOf := make([][]string, 0, len(f.AnyOf))
		for _, l := range f.AnyOf {
			anyOf = append(anyOf, strs(l))
		}
		o.Set("contains", strs(f.Contains))
		o.Set("any_of", anyOf)
		o.Set("not", strs(f.Not))
	default:
		var vals any
		if f.Values != nil {
			vals = f.Values
		}
		o.Set("values", vals)
		o.Set("not", strs(f.Not))
	}
	return o
}
