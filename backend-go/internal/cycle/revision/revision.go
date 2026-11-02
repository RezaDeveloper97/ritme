// Package revision bumps the engine v6 per-user revisions (T-M9-30, docs/engine-v6/README.md §3, gap 07 B7,
// gap 02 §7): every write to an engine-input table bumps user_revisions.engine_rev (and analysis_rev) inside the
// same transaction, so a cache keyed on the revision can never serve data the user has since changed.
//
// The rule is "the transaction owner bumps, once": a writer that opens its own transaction calls Bump before the
// commit; a helper that runs on its caller's transaction (healthlog.NewService(tx), profile.Service bound to a tx,
// pregnancy.saveProfile on a tx-bound querier, periods.RecomputeCycleLengths / Reanchor) never bumps, so one write
// request moves the revision by exactly 1. A refused or rolled-back write never bumps (the bump rolls back with it).
//
// manifest.go lists every sqlc write query on an engine-input table with the integration test that proves its bump
// (or why it is exempt); TestManifest_CoversEveryEngineWrite keeps it complete. AnalysisManifest does the same for the
// analysis-only tables (T-M10-28: vitals, labs, EPDS, recovery, baby and pregnancy logs), whose writers call
// BumpAnalysis / AnalysisTx; TestAnalysisManifest_CoversEveryAnalysisWrite keeps it complete.
//
// calculation_version / profile.MarkRecalculated stay the client-visible "recalculated" signal (unchanged).
// Reading the revision (caches, ETags) is T-M9-31.
//
// Every bump on a user request also marks the user's log day (user_activity_days.logged, T-M10-30) through the marker
// the auth guard put on the context; the mark is queued for after the request and never fails or slows the write.
// A bump inside a transaction that later rolls back still marks the day (the writer is fire-and-forget); refused
// writes are rejected before they reach Bump.
package revision

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/auth/activity"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// Bump is an engine-input write of userID: engine_rev and analysis_rev + 1 (the first bump creates the row at
// 1 / 1). db is the caller's transaction (any domain's DBTX: *sql.Tx satisfies them all).
func Bump(ctx context.Context, db store.DBTX, userID uint64) error {
	if err := store.New(db).BumpUserRevisions(ctx, store.BumpUserRevisionsParams{UserID: userID, Now: now(ctx)}); err != nil {
		return fmt.Errorf("revision: bump: %w", err)
	}
	activity.MarkLogged(ctx, userID)
	return nil
}

// BumpAnalysis is an analysis-only write of userID (vitals, labs, weight, EPDS — T-M10-28): analysis_rev + 1,
// engine_rev unchanged.
func BumpAnalysis(ctx context.Context, db store.DBTX, userID uint64) error {
	if err := store.New(db).BumpAnalysisRevision(ctx, store.BumpAnalysisRevisionParams{UserID: userID, Now: now(ctx)}); err != nil {
		return fmt.Errorf("revision: bump analysis: %w", err)
	}
	activity.MarkLogged(ctx, userID)
	return nil
}

// AnalysisTx runs fn in a transaction on db and bumps userID's analysis_rev before the commit (T-M10-28): the owner
// helper of the analysis-only writers that had no transaction of their own (vitals, labs, EPDS, recovery log, baby
// logs, pregnancy logs). An error from fn rolls the write and the bump back.
func AnalysisTx(ctx context.Context, db *sql.DB, userID uint64, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("revision: begin: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := BumpAnalysis(ctx, tx, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("revision: commit: %w", err)
	}
	return nil
}

// now is the request clock (the X-Test-Now header in tests) as the Tehran wall-clock the DB stores.
func now(ctx context.Context) sql.NullTime {
	t := clock.FromContext(ctx, clock.Real{}).Now()
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
