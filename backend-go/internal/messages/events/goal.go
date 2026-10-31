package events

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// GoalEvents is the server-matched goal catalogue (M-17), the same list as rules.GoalEvents (the feed tests pin it):
// this package stays free of the rules / facts imports, so the write paths that call Goal never import the engine.
var GoalEvents = []string{"period_logged", "log_saved", "checkup_done", "plus_purchased", "challenge_joined"}

// GoalWindow is how long after the last shown a domain event still counts as the message's goal (M-17).
const GoalWindow = 7 * 24 * time.Hour

// GoalQuerier is the goal read (store.Queries).
type GoalQuerier interface {
	ListUserGoalCandidates(ctx context.Context, arg store.ListUserGoalCandidatesParams) ([]store.ListUserGoalCandidatesRow, error)
}

// Goals matches the catalogue's domain events against the messages shown to the user (M-17).
type Goals struct {
	q      GoalQuerier
	w      *Writer
	marks  *Marks
	logger *slog.Logger
}

// NewGoals wires the matcher (marks may be nil: no Redis dedupe of goals still queued in the buffer).
func NewGoals(q GoalQuerier, w *Writer, marks *Marks, logger *slog.Logger) *Goals {
	if logger == nil {
		logger = slog.Default()
	}
	return &Goals{q: q, w: w, marks: marks, logger: logger}
}

// Match logs `goal` for every rule whose goal_event is event and that was shown to the user within GoalWindow
// before now, once per shown window (no goal row after the last shown). It returns the goals queued.
func (g *Goals) Match(ctx context.Context, userID uint64, event string, now time.Time) (int, error) {
	if userID == 0 || !slices.Contains(GoalEvents, event) {
		return 0, nil
	}
	rows, err := g.q.ListUserGoalCandidates(ctx, store.ListUserGoalCandidatesParams{
		UserID: userID, Since: sql.NullTime{Time: now.Add(-GoalWindow), Valid: true},
		GoalEvent: sql.NullString{String: event, Valid: true},
	})
	if err != nil {
		return 0, err
	}
	var evs []Event
	var keys []string
	for i := 0; i < len(rows); {
		r := rows[i]
		j := i
		for j < len(rows) && rows[j].RuleID == r.RuleID {
			j++
		}
		i = j
		// Newest first: a goal row on top means this shown window already has its goal.
		if r.Event != Shown || !r.CreatedAt.Valid {
			continue
		}
		evs = append(evs, Event{RuleID: r.RuleID, UserID: userID, Version: int(r.Version), Variant: r.Variant,
			Slot: r.Slot, Event: GoalEvent, At: now})
		keys = append(keys, GoalKey(userID, r.RuleID, r.CreatedAt.Time))
	}
	if len(evs) == 0 {
		return 0, nil
	}
	// A goal still in the buffer is not in the table yet: the mark keeps a second event of the same window out.
	if first, err := g.marks.First(ctx, GoalMarkTTL, keys...); err == nil {
		kept := evs[:0]
		for i, ev := range evs {
			if first[i] {
				kept = append(kept, ev)
			}
		}
		evs = kept
	}
	g.w.Write(ctx, evs...)
	return len(evs), nil
}

var defaultGoals atomic.Pointer[Goals]

// Install makes g the process matcher Goal uses (the messages routes do it at mount).
func Install(g *Goals) { defaultGoals.Store(g) }

// Goal is the hook the catalogue's write paths call after their commit (M-17): period_logged (cycle/periods),
// log_saved (healthlog), checkup_done (checkups), plus_purchased (plus), challenge_joined (challenges). It never fails
// the caller: without an installed matcher it does nothing, and an error is logged. The time is the request clock.
func Goal(ctx context.Context, userID uint64, event string) {
	g := defaultGoals.Load()
	if g == nil {
		return
	}
	now := clock.FromContext(ctx, clock.Real{}).Now()
	if _, err := g.Match(ctx, userID, event, now); err != nil {
		g.logger.WarnContext(ctx, "messages: goal match failed", "event", event, "error", err.Error())
	}
}
