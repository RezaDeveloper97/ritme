package rules

import (
	"slices"
	"time"
)

// Reasons a rule does not show (the admin-web translates them). Order = the order the engine checks them.
const (
	ReasonInactive        = "inactive"
	ReasonNotApproved     = "not_approved"
	ReasonMissingVariable = "missing_variable"
	ReasonConditions      = "conditions"
	ReasonExceptions      = "exceptions"
	ReasonWindow          = "window"
	ReasonAudience        = "audience"
	ReasonCooldown        = "cooldown"
	ReasonCap             = "cap"
)

// Verdict is one rule's evaluation in a simulation.
type Verdict struct {
	Rule     *Rule
	Eligible bool
	Reason   string // "" when eligible
	Variant  string
	Winner   bool
	// FailedGroups are the indexes of the condition groups that did not hold.
	FailedGroups []int
}

// Why names the first gate that stops the rule, mirroring Eligible plus the live path's renderability filter
// (a text whose {variable} has no value is skipped). "" means the rule may show.
func Why(r *Rule, facts Facts, userID uint64, now time.Time, h History) (string, error) {
	if !r.IsActive {
		return ReasonInactive, nil
	}
	if !r.Served() {
		return ReasonNotApproved, nil
	}
	if _, ok := RenderVariant(r.VariantContent(r.Assign(userID)), facts); !ok {
		return ReasonMissingVariable, nil
	}
	for _, g := range r.Groups {
		if !MatchGroup(g, facts) {
			return ReasonConditions, nil
		}
	}
	if slices.ContainsFunc(r.Exceptions, func(c Condition) bool { return Match(c, facts) }) {
		return ReasonExceptions, nil
	}
	if !InWindow(r, now) {
		return ReasonWindow, nil
	}
	if !r.InAudience(userID) {
		return ReasonAudience, nil
	}
	if r.Unlimited() || h == nil {
		return "", nil
	}
	if r.CooldownDays > 0 {
		last, ok, err := h.Last(r.ID, "impression")
		if err != nil {
			return "", err
		}
		if ok && now.Sub(last) < time.Duration(r.CooldownDays)*24*time.Hour {
			return ReasonCooldown, nil
		}
	}
	for _, cp := range r.EffectiveCaps() {
		n, err := h.Count(r.ID, "impression", now.Add(-capWindow(cp.Period, facts)))
		if err != nil {
			return "", err
		}
		if n >= cp.Count {
			return ReasonCap, nil
		}
	}
	return "", nil
}

// Simulate evaluates every rule of one placement in the engine's order (priority desc, id asc) and marks the winner:
// the first eligible one — exactly what Select returns over the renderable rules. It is pure (no writes).
func Simulate(all []Rule, facts Facts, userID uint64, now time.Time, h History) ([]Verdict, error) {
	cands := slices.Clone(all)
	slices.SortStableFunc(cands, func(a, b Rule) int {
		if a.Priority != b.Priority {
			return b.Priority - a.Priority
		}
		return int(a.ID) - int(b.ID) //nolint:gosec // ids are small
	})
	out := make([]Verdict, 0, len(cands))
	won := false
	for i := range cands {
		r := &cands[i]
		why, err := Why(r, facts, userID, now, h)
		if err != nil {
			return nil, err
		}
		v := Verdict{Rule: r, Eligible: why == "", Reason: why, Variant: r.Assign(userID)}
		if why == ReasonConditions {
			for gi, g := range r.Groups {
				if !MatchGroup(g, facts) {
					v.FailedGroups = append(v.FailedGroups, gi)
				}
			}
		}
		if v.Eligible && !won {
			v.Winner, won = true, true
		}
		out = append(out, v)
	}
	return out, nil
}
