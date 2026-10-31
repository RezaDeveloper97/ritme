// Package feed is the message feed (T-M10-06, D-163; SPEC §5.4; gap 03 §2.9): every requested slot from one compiled
// rules snapshot (T-M10-05), one fact prefetch over the union of the slots, one history query, then server-side
// logging — `shown` once per (user, message, slot, user-local day) (M-19) and a sample of `eligible` (M-12) — through
// the buffered events writer.
package feed

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/messages/compiled"
	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// DefaultSlots are the screen slots the feed serves when ?slots= is absent, in response order; they are also the only
// slots ?slots= accepts and the ones whose `shown` the server logs.
var DefaultSlots = []string{"urgent_modal", "status_banner", "smart_tip", "daily_advice", "daily_challenge", "modal"}

// SlotModules maps a slot to the app module (internal/modules key) whose kill-switch empties it. No module owns a
// home slot yet; an entry here is all it takes.
var SlotModules = map[string]string{}

// EligibleTopK is how many evaluated candidates of a slot are logged as `eligible` (M-12).
const EligibleTopK = 5

// EligibleSamplePercent is the share of users whose product / marketing `eligible` rows are logged (M-12).
const EligibleSamplePercent = 10

// Modules reads the app module switches (modules.Service).
type Modules interface {
	Enabled(ctx context.Context, key string) bool
}

// FactsFunc builds the user's facts of need (the registry's Need, or the v1 builder).
type FactsFunc func(ctx context.Context, need []string) (rules.Facts, error)

// Service builds feeds.
type Service struct {
	q       store.Querier
	w       *events.Writer
	marks   *events.Marks
	modules Modules
	load    func(ctx context.Context) (*compiled.Compiled, error)
}

// New wires the service; marks and modules may be nil (no Redis → DB check; no switches → all on).
func New(q store.Querier, w *events.Writer, marks *events.Marks, modules Modules) *Service {
	return &Service{q: q, w: w, marks: marks, modules: modules,
		load: func(ctx context.Context) (*compiled.Compiled, error) { return compiled.Load(ctx, q) }}
}

// WithLoader replaces the compiled snapshot source (tests).
func (s *Service) WithLoader(load func(ctx context.Context) (*compiled.Compiled, error)) *Service {
	s.load = load
	return s
}

// Request is one feed build.
type Request struct {
	UserID uint64
	Slots  []string
	Locale string
	// Now is the request time in the user's zone (civildate.LocalNow): the local day of the shown marks.
	Now   time.Time
	Facts FactsFunc
	// Alerts adds the open inbox counts (the feed; /messages/selected leaves it off).
	Alerts bool
	// Inbox, when set, answers the counts from a row the request already read (the fact profile); ok = false falls
	// back to CountUserOpenInbox.
	Inbox func(ctx context.Context) (a Alerts, ok bool, err error)
}

// Slot is one slot's answer.
type Slot struct {
	Name    string
	Choices []rules.Choice
	Texts   []rules.Variant // rendered, in Choices order
}

// Alerts are the open inbox counts.
type Alerts struct{ Open, Urgent int64 }

// Result is a built feed.
type Result struct {
	Slots  []Slot
	Facts  rules.Facts
	Alerts Alerts
	Rev    uint64

	x *extras
}

// ValidSlot reports whether ?slots= may name slot.
func ValidSlot(slot string) bool { return slices.Contains(DefaultSlots, slot) }

// Build selects every slot of req and logs the server-side events.
func (s *Service) Build(ctx context.Context, req Request) (*Result, error) {
	comp, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	x := extrasOf(ctx, s.q, comp)
	need := slices.Concat(comp.Need(req.Slots...), compiled.SelectionFacts, x.placeholders(req.Slots))
	f, err := req.Facts(ctx, need)
	if err != nil {
		return nil, err
	}
	w := rules.CapWindows(req.Now, f)
	rows, err := s.q.ListUserMessageEventStats(ctx, store.ListUserMessageEventStatsParams{
		DaySince:   sql.NullTime{Time: w.Day, Valid: true},
		WeekSince:  sql.NullTime{Time: w.Week, Valid: true},
		CycleSince: sql.NullTime{Time: w.Cycle, Valid: true},
		UserID:     req.UserID, AllRules: true,
	})
	if err != nil {
		return nil, err
	}
	h := rules.BuildHistory(w, rows)
	res := &Result{Facts: f, Rev: comp.Rev, x: x, Slots: make([]Slot, 0, len(req.Slots))}
	var log []candidate
	sampled := Sampled(req.UserID)
	for _, name := range req.Slots {
		sl := Slot{Name: name, Choices: []rules.Choice{}}
		if m := SlotModules[name]; m != "" && s.modules != nil && !s.modules.Enabled(ctx, m) {
			res.Slots = append(res.Slots, sl)
			continue
		}
		winners := comp.Select(name, f, h, req.UserID, req.Now)
		texts, ok := x.render(winners, f, req.Locale)
		var parsedOut []rules.Choice
		if !ok || (ValidSlot(name) && len(x.parsed[name]) > len(winners)) {
			// A winner without a value for one of its variables is skipped (never a raw placeholder): select again
			// over the renderable rules. The parsed pass also yields the next eligible candidates.
			pool := x.parsed[name]
			if !ok {
				pool = x.renderable(name, f, req.Locale, req.UserID)
			}
			parsedOut = parsedTop(name, pool, f, h, req.UserID, req.Now)
			if ok { // the eligible extras must be renderable too
				parsedOut = slices.DeleteFunc(parsedOut, func(ch rules.Choice) bool {
					_, ok := rules.RenderVariant(x.text(ch.Rule, ch.Variant, req.Locale), f)
					return !ok
				})
			}
			if !ok {
				winners = parsedOut[:min(len(parsedOut), capacity(name, len(parsedOut)))]
				texts, _ = x.render(winners, f, req.Locale)
			}
		}
		if winners == nil {
			winners = []rules.Choice{}
		}
		sl.Choices, sl.Texts = winners, texts
		res.Slots = append(res.Slots, sl)
		if ValidSlot(name) {
			log = appendLog(log, name, winners, parsedOut, sampled)
		}
	}
	if req.Alerts {
		if res.Alerts, err = s.alerts(ctx, req); err != nil {
			return nil, err
		}
	}
	s.logEvents(ctx, req, rows, log)
	return res, nil
}

// alerts is the open inbox counts: off req.Inbox when it has them, else one CountUserOpenInbox.
func (s *Service) alerts(ctx context.Context, req Request) (Alerts, error) {
	if req.Inbox != nil {
		a, ok, err := req.Inbox(ctx)
		if err != nil || ok {
			return a, err
		}
	}
	a, err := s.q.CountUserOpenInbox(ctx, req.UserID)
	if err != nil {
		return Alerts{}, err
	}
	return Alerts{Open: a.OpenCount, Urgent: a.UrgentCount}, nil
}

// capacity is the winners a slot keeps.
func capacity(slot string, n int) int {
	s := rules.Slots[slot]
	if s.List {
		return n
	}
	return s.Capacity
}

// parsedTop is the parsed selection (rules.SelectSlot) with the capacity-dropped candidates after the winners, in
// selection order: the evaluated top of the slot.
func parsedTop(slot string, pool []rules.Rule, f rules.Facts, h *rules.UserHistory, userID uint64, now time.Time) []rules.Choice {
	winners, dropped := rules.SelectSlot(slot, pool, f, h, userID, now)
	out := slices.Clone(winners)
	for _, d := range dropped {
		if len(out) >= EligibleTopK {
			break
		}
		if d.Reason != rules.ReasonCapacity {
			continue
		}
		i := slices.IndexFunc(pool, func(r rules.Rule) bool { return r.ID == d.RuleID })
		if i >= 0 {
			out = append(out, rules.Choice{Rule: &pool[i], Variant: pool[i].Assign(userID)})
		}
	}
	return out
}

// Sampled reports whether the user's product / marketing `eligible` rows are logged: a stable per-user hash.
func Sampled(userID uint64) bool {
	return rules.HashOf("eligible:"+strconv.FormatUint(userID, 10))%100 < EligibleSamplePercent
}

// candidate is one event the build may log.
type candidate struct {
	slot  string
	ch    rules.Choice
	event string
}

// appendLog adds the shown winners and the eligible top-K (product / marketing only for sampled users).
func appendLog(out []candidate, slot string, winners, top []rules.Choice, sampled bool) []candidate {
	for _, ch := range winners {
		out = append(out, candidate{slot: slot, ch: ch, event: events.Shown})
	}
	if len(top) == 0 {
		top = winners
	}
	for i, ch := range top {
		if i >= EligibleTopK {
			break
		}
		if (ch.Rule.Kind == rules.KindProduct || ch.Rule.Kind == rules.KindMarketing) && !sampled {
			continue
		}
		out = append(out, candidate{slot: slot, ch: ch, event: events.Eligible})
	}
	return out
}

// logEvents writes each candidate once per user-local day: the Redis mark decides; without Redis the day's history
// rows (already read) do.
func (s *Service) logEvents(ctx context.Context, req Request, rows []store.ListUserMessageEventStatsRow, cands []candidate) {
	if len(cands) == 0 || s.w == nil {
		return
	}
	day := civildate.FromTime(req.Now).String()
	keys := make([]string, len(cands))
	for i, c := range cands {
		kind := "shown"
		if c.event == events.Eligible {
			kind = "eligible"
		}
		keys[i] = events.DayKey(kind, req.UserID, c.ch.Rule.ID, c.slot, day)
	}
	first, err := s.marks.First(ctx, events.MarkTTL, keys...)
	if err != nil {
		if !errors.Is(err, events.ErrNoMarks) {
			markErrors.Inc()
		}
		first = dbFirst(rows, cands)
	}
	evs := make([]events.Event, 0, len(cands))
	for i, c := range cands {
		if !first[i] {
			continue
		}
		evs = append(evs, events.Event{RuleID: c.ch.Rule.ID, UserID: req.UserID, Version: ServedVersion(c.ch.Rule),
			Variant: c.ch.Variant, Slot: c.slot, Event: c.event, At: req.Now})
	}
	s.w.Write(ctx, evs...)
}

// dbFirst is the Redis-down check: an event is first when the history has no row of it in this slot today.
func dbFirst(rows []store.ListUserMessageEventStatsRow, cands []candidate) []bool {
	type key struct {
		rule        uint64
		slot, event string
	}
	today := map[key]bool{}
	for _, r := range rows {
		if r.DayCount > 0 {
			ev := r.Event
			if ev == "shown" {
				ev = events.Shown
			}
			today[key{r.RuleID, r.Slot, ev}] = true
		}
	}
	out := make([]bool, len(cands))
	for i, c := range cands {
		k := key{c.ch.Rule.ID, c.slot, c.event}
		out[i] = !today[k]
		today[k] = true
	}
	return out
}

// ServedVersion is the version a user sees: the approved version of a medical rule, else the head.
func ServedVersion(r *rules.Rule) int {
	if rules.NeedsMedicalApproval(r.Level, r.Kind) && r.ApprovedVersion > 0 {
		return r.ApprovedVersion
	}
	return r.Version
}
