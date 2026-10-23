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
// (or why it is exempt); TestManifest_CoversEveryEngineWrite keeps it complete.
//
// calculation_version / profile.MarkRecalculated stay the client-visible "recalculated" signal (unchanged).
// Reading the revision (caches, ETags) is T-M9-31.
package revision

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	return nil
}

// BumpAnalysis is an analysis-only write of userID (vitals, labs, weight, EPDS — T-M10-28): analysis_rev + 1,
// engine_rev unchanged.
func BumpAnalysis(ctx context.Context, db store.DBTX, userID uint64) error {
	if err := store.New(db).BumpAnalysisRevision(ctx, store.BumpAnalysisRevisionParams{UserID: userID, Now: now(ctx)}); err != nil {
		return fmt.Errorf("revision: bump analysis: %w", err)
	}
	return nil
}

// now is the request clock (the X-Test-Now header in tests) as the Tehran wall-clock the DB stores.
func now(ctx context.Context) sql.NullTime {
	t := clock.FromContext(ctx, clock.Real{}).Now()
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
