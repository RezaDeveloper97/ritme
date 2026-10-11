package rules

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ritme/backend-go/internal/messages/store"
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
