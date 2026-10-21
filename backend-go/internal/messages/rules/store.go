package rules

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
)

// FromRow maps a message_rules row onto the engine's Rule.
func FromRow(r *store.MessageRule) Rule {
	out := Rule{
		ID: r.ID, Level: r.Level, Kind: r.Kind, Priority: int(r.Priority), Placement: r.Placement,
		Content: ParseContent(r.Content), VariantsEnabled: r.VariantsEnabled, ABSplit: int(r.AbSplit),
		Groups: ParseGroups(r.Conditions), Exceptions: ParseExceptions(r.Exceptions),
		CooldownDays: int(r.CooldownDays), HoursFrom: -1, HoursTo: -1,
		AudiencePercent: int(r.AudiencePercent), Status: r.Status, IsActive: r.IsActive,
	}
	if r.CapCount.Valid {
		out.CapCount, out.CapPeriod = int(r.CapCount.Int16), r.CapPeriod.String
	}
	if r.HoursFrom.Valid && r.HoursTo.Valid {
		out.HoursFrom, out.HoursTo = int(r.HoursFrom.Int16), int(r.HoursTo.Int16)
	}
	if r.StartsOn.Valid {
		out.StartsOn = r.StartsOn.Date
	}
	if r.EndsOn.Valid {
		out.EndsOn = r.EndsOn.Date
	}
	return out
}

// DBHistory is the History of one user backed by message_events.
type DBHistory struct {
	Ctx    context.Context //nolint:containedctx // adapter for the pure engine, scoped to one request
	Q      store.Querier
	UserID uint64
}

// Count implements History.
func (h DBHistory) Count(rule uint64, event string, since time.Time) (int, error) {
	n, err := h.Q.CountUserMessageEvents(h.Ctx, store.CountUserMessageEventsParams{
		UserID: h.UserID, RuleID: rule, Event: event, CreatedAt: sql.NullTime{Time: since, Valid: true},
	})
	return int(n), err
}

// Last implements History.
func (h DBHistory) Last(rule uint64, event string) (time.Time, bool, error) {
	t, err := h.Q.LastUserMessageEvent(h.Ctx, store.LastUserMessageEventParams{UserID: h.UserID, RuleID: rule, Event: event})
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !t.Valid) {
		return time.Time{}, false, nil
	}
	return t.Time, err == nil, err
}

// Prefetch implements Prefetcher: one ListUserMessageEventStats read for every candidate (T-M9-13).
func (h DBHistory) Prefetch(rules []uint64, windows [3]time.Time) (History, error) {
	rows, err := h.Q.ListUserMessageEventStats(h.Ctx, store.ListUserMessageEventStatsParams{
		DaySince:   sql.NullTime{Time: windows[0], Valid: true},
		WeekSince:  sql.NullTime{Time: windows[1], Valid: true},
		CycleSince: sql.NullTime{Time: windows[2], Valid: true},
		UserID:     h.UserID, RuleIds: rules,
	})
	if err != nil {
		return nil, err
	}
	return NewStats(h, rules, windows, rows), nil
}

// EventStat is one rule's history of one event kind.
type EventStat struct {
	Last   time.Time // zero = never
	Counts [3]int    // since windows[0], [1], [2]
}

// Stats is a prefetched History: answers for the prefetched rules come from memory; anything else (another rule, a
// window it was not prefetched for) falls through to the underlying History.
type Stats struct {
	fallback History
	windows  [3]time.Time
	known    map[uint64]bool
	byRule   map[uint64]map[string]EventStat
}

// NewStats builds a Stats from the grouped rows (rules = the prefetched ids; a rule without rows has no history).
func NewStats(fallback History, rules []uint64, windows [3]time.Time, rows []store.ListUserMessageEventStatsRow) *Stats {
	s := &Stats{fallback: fallback, windows: windows, known: make(map[uint64]bool, len(rules)),
		byRule: make(map[uint64]map[string]EventStat, len(rows))}
	for _, id := range rules {
		s.known[id] = true
	}
	for _, r := range rows {
		m := s.byRule[r.RuleID]
		if m == nil {
			m = map[string]EventStat{}
			s.byRule[r.RuleID] = m
		}
		m[r.Event] = EventStat{Last: r.LastAt, Counts: [3]int{int(r.DayCount), int(r.WeekCount), int(r.CycleCount)}}
	}
	return s
}

// Count implements History.
func (s *Stats) Count(rule uint64, event string, since time.Time) (int, error) {
	if s.known[rule] {
		for i, w := range s.windows {
			if w.Equal(since) {
				return s.byRule[rule][event].Counts[i], nil
			}
		}
	}
	if s.fallback == nil {
		return 0, nil
	}
	return s.fallback.Count(rule, event, since)
}

// Last implements History.
func (s *Stats) Last(rule uint64, event string) (time.Time, bool, error) {
	if s.known[rule] {
		st := s.byRule[rule][event]
		return st.Last, !st.Last.IsZero(), nil
	}
	if s.fallback == nil {
		return time.Time{}, false, nil
	}
	return s.fallback.Last(rule, event)
}

// Table is the table the active-rules snapshot reads; admin writers publish it (configcache.Changed) after commit.
const Table = "message_rules"

// active is the process-wide snapshot of every active message_rules row (raw rows: compiling them is T-M10-05).
var active = configcache.New[[]store.MessageRule](Table, []string{Table}, nil)

// ActiveRows is ListActiveMessageRules from the process snapshot: one query per process until an admin write
// publishes message_rules (or the 60 s TTL ends). Placement "" = every placement; order is priority DESC, id ASC.
// The rows are shared: callers must not modify them.
func ActiveRows(ctx context.Context, q store.Querier, placement string) ([]store.MessageRule, error) {
	all, err := active.GetWith(ctx, func(ctx context.Context) ([]store.MessageRule, error) {
		return q.ListActiveMessageRules(ctx, store.ListActiveMessageRulesParams{Placement: ""})
	})
	if err != nil || placement == "" {
		return all, err
	}
	out := make([]store.MessageRule, 0, len(all))
	for i := range all {
		if all[i].Placement == placement {
			out = append(out, all[i])
		}
	}
	return out, nil
}
