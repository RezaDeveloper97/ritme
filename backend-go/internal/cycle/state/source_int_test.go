package state_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/state"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// Source (T-M9-36) reads the engine revision itself: a write that bumps it is visible to the next ForUser call, and a
// user without a user_revisions row is revision 0.
func TestSource_ForUserFollowsTheRevision(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	q := store.New(db)
	u := seedUser(t, db, 6)
	pr, _ := newProvider(t, q, &recordingKV{Client: testRedis(t)}, state.DefaultOptions())
	src := state.Source{P: pr, Revs: q}
	today := civildate.FromTime(now)

	s1, err := src.ForUser(ctx, u, today)
	require.NoError(t, err)
	assert.Len(t, s1.Periods(), 6)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	addPeriodRow(t, tx, u, today.AddDays(-2))
	require.NoError(t, revision.Bump(ctx, tx, u))
	require.NoError(t, tx.Commit())

	s2, err := src.ForUser(ctx, u, today)
	require.NoError(t, err)
	assert.Len(t, s2.Periods(), 7)

	none, err := src.ForUser(ctx, 987654321, today)
	require.NoError(t, err)
	assert.Empty(t, none.Periods())
}
