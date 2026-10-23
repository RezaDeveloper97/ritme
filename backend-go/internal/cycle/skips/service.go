// Package skips records and undoes «پریود نداشتم» — a skipped cycle (T-M9-29, spec §2.39.8, DECISIONS.md E-15,
// deviation D-125, Go-only).
//
// A skip is allowed only for the cycle the user is waiting on: by the v6 engine state for her local today
// (civildate.LocalToday, D-126), expected_start must equal today's predicted_next_period_start, today must be on or
// after it (any lateness, also beyond max_late) and no real period may have started since. The stored row is an
// engine input: the engine turns it into a virtual predicted_reference anchor, so the next prediction moves by one
// cycle and skipped_cycle_recorded is raised; deleting it restores the previous output.
//
// Both writes run in one transaction that takes the per-user lock (periods.LockUser, D-131) and bumps the engine
// revision (revision.Bump, T-M9-30). A refused write, a repeated skip and a delete of an unknown date bump nothing.
package skips

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/engineparams"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/cycle/periods"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Refusal codes of POST /cycle/skips (422 `code`).
const (
	CodeNotExpectedYet     = "not_expected_yet"    // today is before the predicted start
	CodePeriodStarted      = "period_started"      // a real period started on or after expected_start
	CodePredictionMismatch = "prediction_mismatch" // expected_start is not today's predicted next start
)

// Refusal is a refused skip (422 with Code).
type Refusal struct{ Code string }

func (r *Refusal) Error() string { return "skips: refused: " + r.Code }

// ErrNotFound is a delete of a date the user has no skip on.
var ErrNotFound = errors.New("skips: not found")

// Service owns the cycle_skips writes.
type Service struct{ db *sql.DB }

// NewService binds the service to the database.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Record stores the skip of expected for userID as of today (the user's local day). created is false when the skip
// already existed (idempotent repeat: nothing written, no bump).
func (s *Service) Record(ctx context.Context, userID uint64, expected, today civildate.Date) (created bool, err error) {
	err = s.tx(ctx, userID, func(q *store.Queries) (bool, error) {
		in, err := inputs.Load(ctx, q, userID, today)
		if err != nil {
			return false, fmt.Errorf("skips: load inputs: %w", err)
		}
		exp := engine.DayOf(expected)
		if slices.Contains(in.Skips, exp) {
			return false, nil
		}
		if code := Check(in, engineparams.For(today), exp, engine.DayOf(today)); code != "" {
			return false, &Refusal{Code: code}
		}
		n, err := q.InsertCycleSkip(ctx, store.InsertCycleSkipParams{UserID: userID, ExpectedStart: expected, Now: now(ctx)})
		if err != nil {
			return false, fmt.Errorf("skips: insert: %w", err)
		}
		created = n > 0
		return created, nil
	})
	return created, err
}

// Delete removes the skip of date for userID; ErrNotFound when she has none on that date.
func (s *Service) Delete(ctx context.Context, userID uint64, date civildate.Date) error {
	return s.tx(ctx, userID, func(q *store.Queries) (bool, error) {
		n, err := q.DeleteCycleSkip(ctx, store.DeleteCycleSkipParams{UserID: userID, ExpectedStart: date})
		if err != nil {
			return false, fmt.Errorf("skips: delete: %w", err)
		}
		if n == 0 {
			return false, ErrNotFound
		}
		return true, nil
	})
}

// Check is the §2.39.8 rule on the user's engine input: "" when a skip of expected is allowed today, else the
// refusal code. The skip must also be one the engine will match (it would otherwise be ignored and change nothing:
// e.g. a stale roll-forward anchor of a declared-only history), which is reported as prediction_mismatch.
func Check(in engine.Input, p engine.Params, expected, today engine.Day) string {
	st := engine.Prepare(in, p).Resolve(today, today)
	next := st.Anchors.PredictedNextPeriodStart
	if !next.IsSet() || next != expected {
		if realStartSince(in, expected, today) {
			return CodePeriodStarted
		}
		return CodePredictionMismatch
	}
	if today < expected {
		return CodeNotExpectedYet
	}
	with := in
	with.Skips = append(slices.Clone(in.Skips), expected)
	slices.Sort(with.Skips)
	if slices.Contains(engine.Prepare(with, p).Diagnostics().IgnoredSkips, expected) {
		return CodePredictionMismatch
	}
	return ""
}

// realStartSince reports whether a real (logged) period starts in [from, today].
func realStartSince(in engine.Input, from, today engine.Day) bool {
	for _, p := range in.Periods {
		if p.Start >= from && p.Start <= today {
			return true
		}
	}
	return false
}

// tx runs fn in a transaction holding the per-user lock (D-131); when fn reports a write, the engine revision is
// bumped once before the commit (T-M9-30). An error or a no-op rolls back.
func (s *Service) tx(ctx context.Context, userID uint64, fn func(q *store.Queries) (bool, error)) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("skips: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := periods.LockUser(ctx, tx, userID); err != nil {
		return err
	}
	wrote, err := fn(store.New(tx))
	if err != nil {
		return err
	}
	if !wrote {
		return nil
	}
	if err := revision.Bump(ctx, tx, userID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("skips: commit: %w", err)
	}
	return nil
}

// now is the request clock as the Tehran wall-clock the DB stores (created_at / updated_at).
func now(ctx context.Context) sql.NullTime {
	t := clock.FromContext(ctx, clock.Real{}).Now().In(civildate.Tehran)
	return sql.NullTime{Time: t.Truncate(1e9), Valid: true}
}
