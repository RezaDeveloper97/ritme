package compiled

import (
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/messages/rules"
)

var noHistory = &rules.UserHistory{}

// Select is rules.SelectSlot over the compiled rules: the same winners, in the same order, with the same variants
// (the equivalence test runs both over random rule sets), without the drop list. now is the request time in the
// user's zone; h may be nil. It allocates once (the winners slice) and stops as soon as no later rule can win.
func (c *Compiled) Select(slot string, f rules.Facts, h *rules.UserHistory, userID uint64, now time.Time) []rules.Choice {
	s, list := c.slots[slot], c.BySlot[slot]
	if s == nil || len(list) == 0 {
		return nil
	}
	if h == nil {
		h = noHistory
	}
	hour := now.Hour()
	if q := s.QuietHours; q != nil && q.InHours(hour) {
		return []rules.Choice{} // §5.4.1: nothing in the quiet hours (the parsed path drops every candidate)
	}
	n := len(list)
	if !s.List {
		n = min(n, s.Capacity)
	}
	if n == 0 {
		return []rules.Choice{}
	}
	g := request{slot: s, f: f, h: h, now: now, hour: hour, userID: userID,
		needsInput: rules.FactTrue(f, "requires_user_input"), lowConf: f["confidence"] == "low",
		hormonal:   rules.FactTrue(f, "hormonal_contraception"),
		postSafety: !h.LastSafetyShown.IsZero() && now.Sub(h.LastSafetyShown) < rules.PostSafetyBlock}
	sh := h.Slots[s.Name]
	for _, sc := range s.SlotCaps {
		if sh.Of(sc.Period) >= sc.Count {
			g.slotCapped = true
			break
		}
	}
	out := make([]rules.Choice, 0, n)
	for _, r := range list {
		if len(out) == n && worseGroup(r, out[n-1].Rule) {
			break // pre-sorted: every later rule is in this group or a worse one
		}
		if !g.pass(r) {
			continue
		}
		out = g.insert(out, n, r)
	}
	return out
}

// worseGroup reports whether r's (safety, priority) group sorts after w's.
func worseGroup(r *Rule, w *rules.Rule) bool {
	if ws := w.Kind == rules.KindSafety; r.safety != ws {
		return ws
	}
	return r.priority < w.Priority
}

type request struct {
	slot                                      *rules.Slot
	f                                         rules.Facts
	h                                         *rules.UserHistory
	now                                       time.Time
	hour                                      int
	userID                                    uint64
	needsInput, lowConf, hormonal, postSafety bool
	slotCapped                                bool
}

// pass runs every gate of rules.SelectSlot (the drop reason is not needed, so the cheap gates go first).
func (g *request) pass(r *Rule) bool {
	if g.needsInput && !r.exemptInput || g.lowConf && r.lowConfSensitive || g.hormonal && r.fertile ||
		g.postSafety && r.postSafety {
		return false
	}
	if r.hasHours && r.hourMask&(1<<g.hour) == 0 {
		return false
	}
	if r.scheduled {
		loc := g.now.Location()
		if !r.start.IsZero() && g.now.Before(inZone(r.start, loc)) {
			return false
		}
		if !r.end.IsZero() && !g.now.Before(inZone(r.end, loc)) {
			return false
		}
	}
	if !r.safety {
		if g.slotCapped {
			return false
		}
		if !r.audienceAll && stableHash(g.userID, r.Src.ID, "aud")%100 >= r.audience {
			return false
		}
		if !g.caps(r) {
			return false
		}
	}
	for i := range r.groups {
		if !r.groups[i].match(g.f) {
			return false
		}
	}
	for i := range r.exceptions {
		if r.exceptions[i].match(g.f) {
			return false
		}
	}
	return true
}

// caps checks the rule's caps, its cooldown and the slot's per-message caps (slot caps are checked once per request).
func (g *request) caps(r *Rule) bool {
	if len(r.caps) == 0 && r.cooldown == 0 && len(g.slot.MessageCaps) == 0 {
		return true
	}
	rh := g.h.Rules[r.Src.ID]
	for _, c := range r.caps {
		if rh.CapCount(c.Period) >= c.Count {
			return false
		}
	}
	if r.cooldown > 0 {
		if last := rh.LastSeen(); !last.IsZero() && g.now.Sub(last) < r.cooldown {
			return false
		}
	}
	for _, c := range g.slot.MessageCaps {
		if rh.CapCount(c.Period) >= c.Count {
			return false
		}
	}
	return true
}

// insert places r among the winners (tie-break inside its group: least recently shown, then the lower id) and keeps
// at most n.
func (g *request) insert(out []rules.Choice, n int, r *Rule) []rules.Choice {
	last := g.h.Rules[r.Src.ID].LastShown
	i := len(out)
	for i > 0 {
		w := out[i-1].Rule
		if w.Priority != r.priority || (w.Kind == rules.KindSafety) != r.safety {
			break // an earlier group
		}
		wl := g.h.Rules[w.ID].LastShown
		if c := last.Compare(wl); c > 0 || c == 0 && r.Src.ID > w.ID {
			break
		}
		i--
	}
	if i >= n {
		return out
	}
	ch := rules.Choice{Rule: r.Src, Variant: g.variant(r)}
	if len(out) < n {
		out = append(out, rules.Choice{})
	}
	copy(out[i+1:], out[i:len(out)-1])
	out[i] = ch
	return out
}

// variant is rules.WeightedVariant (safety: the first key, M-07) without allocating.
func (g *request) variant(r *Rule) string {
	if r.safety || r.singleVariantOK {
		return r.safetyVariant
	}
	x := int(stableHash(g.userID, r.Src.ID, "ab") % uint32(r.variantSum)) //nolint:gosec // sum > 0 and small
	for _, v := range r.variants {
		if x < max(v.Weight, 0) {
			return v.Key
		}
		x -= max(v.Weight, 0)
	}
	return r.safetyVariant
}

// stableHash is rules.StableHash (fnv32a over "user:rule:salt") on a stack buffer.
func stableHash(userID, ruleID uint64, salt string) uint32 {
	var buf [48]byte
	b := strconv.AppendUint(buf[:0], userID, 10)
	b = append(b, ':')
	b = strconv.AppendUint(b, ruleID, 10)
	b = append(b, ':')
	h := uint32(2166136261)
	for _, c := range b {
		h ^= uint32(c)
		h *= 16777619
	}
	for i := 0; i < len(salt); i++ {
		h ^= uint32(salt[i])
		h *= 16777619
	}
	return h
}

// inZone reads a Tehran-stored wall clock as the same wall clock in loc (user-local schedules, M-08).
func inZone(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}
