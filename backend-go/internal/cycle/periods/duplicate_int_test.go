package periods

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// A same-start insert that loses the race on UNIQUE(user_id, period_start_date) (MySQL 1062, a
// writer outside the per-user lock) maps to errDuplicateStart, and the service answers from the
// committed row (idempotent 200) instead of a 500 (D-131).
func TestInsertDuplicateStart_MapsToExistingRow(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Sara', '09120000499', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id) //nolint:gosec // G115: test id
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, civildate.Tehran)
	start := civildate.MustParse("2026-10-01")

	svc := NewService(db)
	first, err := svc.Start(ctx, uid, start, now)
	require.NoError(t, err)

	row := store.CycleHistory{UserID: uid, PeriodStartDate: start, IsConfirmed: true, Source: "user_logged",
		CreatedAt: dbTime(now), UpdatedAt: dbTime(now)}
	_, err = insert(ctx, svc.q, row)
	require.ErrorIs(t, err, errDuplicateStart)

	got, err := svc.existingStart(ctx, uid, start, refuseStartOverlap)
	require.NoError(t, err)
	assert.Equal(t, first.Row.ID, got.Row.ID)

	_, err = svc.existingStart(ctx, uid, civildate.MustParse("2026-09-01"), refuseRangeOverlap)
	var r *Refusal
	require.ErrorAs(t, err, &r)
	assert.Equal(t, 422, r.Status)
	assert.Equal(t, "period_overlap", r.Code)
}
