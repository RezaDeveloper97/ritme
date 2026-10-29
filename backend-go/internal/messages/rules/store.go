package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
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
	// Message model v6 (T-M10-02): a NULL column keeps the legacy field (Effective* fall back to it).
	out.RuleKey, out.Version, out.GoalEvent, out.Locked = r.RuleKey.String, int(r.Version), r.GoalEvent.String, r.Locked
	if r.ApprovedVersion.Valid {
		out.ApprovedVersion = int(r.ApprovedVersion.Int32)
	}
	if v, ok := decodeJSON[[]string](nullJSON(r.Topics)); ok {
		out.Topics = v
	}
	if v, ok := decodeJSON[[]Cap](nullJSON(r.Caps)); ok {
		out.Caps = v
	}
	if v, ok := decodeJSON[[]VariantWeight](nullJSON(r.Variants)); ok {
		out.Variants = v
	}
	if v, ok := decodeJSON[[]HourWindow](nullJSON(r.HourWindows)); ok {
		out.HourWindows = v
	}
	if r.StartsAt.Valid {
		out.StartsAt = wallClock(r.StartsAt.Time)
	}
	if r.EndsAt.Valid {
		out.EndsAt = wallClock(r.EndsAt.Time)
	}
	return out
}

func nullJSON(j db.NullRawJSON) []byte {
	if !j.Valid {
		return nil
	}
	return j.V
}

// wallClock reads a DATETIME (Tehran wall clock, whatever location the driver attached) as Tehran time.
func wallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, civildate.Tehran)
}

// snapshot is the served part of a message_rule_versions snapshot (the admin JSON of the rule): the fields whose edit
// withdraws the approval (M-02). A key missing from an older snapshot keeps the head's value.
type snapshot struct {
	Level           *string         `json:"level"`
	Kind            *string         `json:"kind"`
	Placement       *string         `json:"placement"`
	Content         json.RawMessage `json:"content"`
	Conditions      json.RawMessage `json:"conditions"`
	Exceptions      json.RawMessage `json:"exceptions"`
	Topics          json.RawMessage `json:"topics"`
	Variants        json.RawMessage `json:"variants"`
	VariantsEnabled *bool           `json:"variants_enabled"`
	ABSplit         *uint8          `json:"ab_split"`
}

func rawColumn(raw json.RawMessage) db.NullRawJSON {
	if len(raw) == 0 || string(raw) == "null" {
		return db.NullRawJSON{}
	}
	return db.NullRawJSON{V: raw, Valid: true}
}

// ApplySnapshot overlays the served fields of an approved snapshot onto the head row and marks it approved, so the
// engine serves the approved version while the head waits for re-approval (M-03). Ops fields (priority, audience,
// schedule, hours, caps, cooldown) stay the head's: their edits never withdraw approval (M-02). ok is false when the
// snapshot cannot be read (the row must then not be served).
func ApplySnapshot(row *store.MessageRule, raw string) (store.MessageRule, bool) {
	var s snapshot
	if json.Unmarshal([]byte(raw), &s) != nil || len(s.Content) == 0 {
		return *row, false
	}
	out := *row
	if s.Level != nil {
		out.Level = *s.Level
	}
	if s.Kind != nil {
		out.Kind = *s.Kind
	}
	if s.Placement != nil {
		out.Placement = *s.Placement
	}
	out.Content = string(s.Content)
	if len(s.Conditions) > 0 {
		out.Conditions = string(s.Conditions)
	}
	if len(s.Exceptions) > 0 {
		out.Exceptions = string(s.Exceptions)
	}
	if s.Topics != nil {
		out.Topics = rawColumn(s.Topics)
	}
	if s.Variants != nil {
		out.Variants = rawColumn(s.Variants)
	}
	if s.VariantsEnabled != nil {
		out.VariantsEnabled = *s.VariantsEnabled
	}
	if s.ABSplit != nil {
		out.AbSplit = *s.ABSplit
	}
	out.Status = StatusApproved
	return out, true
}

// ServedRows turns the active rows into what users are served: a medical head waiting for re-approval is replaced by
// its approved_version snapshot (one ListServedMessageRuleSnapshots read); one whose snapshot is missing loses its
// approved version, so Served() drops it.
func ServedRows(ctx context.Context, q store.Querier, rows []store.MessageRule) ([]store.MessageRule, error) {
	var waiting bool
	for i := range rows {
		if pendingHead(&rows[i]) {
			waiting = true
			break
		}
	}
	if !waiting {
		return rows, nil
	}
	snaps, err := q.ListServedMessageRuleSnapshots(ctx)
	if err != nil {
		return nil, err
	}
	byRule := make(map[uint64]store.ListServedMessageRuleSnapshotsRow, len(snaps))
	for _, s := range snaps {
		byRule[s.RuleID] = s
	}
	out := make([]store.MessageRule, len(rows))
	copy(out, rows)
	for i := range out {
		r := &out[i]
		if !pendingHead(r) {
			continue
		}
		if r.ApprovedVersion.Int32 == int32(r.Version) { //nolint:gosec // versions are small
			r.Status = StatusApproved
			continue
		}
		s, ok := byRule[r.ID]
		if ok && int32(s.Version) == r.ApprovedVersion.Int32 { //nolint:gosec // versions are small
			if applied, ok := ApplySnapshot(r, s.Snapshot); ok {
				*r = applied
				continue
			}
		}
		r.ApprovedVersion = sql.NullInt32{}
	}
	return out, nil
}

// pendingHead reports whether a medical row's head is not approved but an approved version exists.
func pendingHead(r *store.MessageRule) bool {
	return r.Status != StatusApproved && r.ApprovedVersion.Valid && r.ApprovedVersion.Int32 > 0 &&
		NeedsMedicalApproval(r.Level, r.Kind)
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
		rows, err := q.ListActiveMessageRules(ctx, store.ListActiveMessageRulesParams{Placement: ""})
		if err != nil {
			return nil, err
		}
		return ServedRows(ctx, q, rows) // T-M10-02: the approved version of a head waiting for re-approval
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
