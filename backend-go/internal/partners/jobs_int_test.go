package partners_test

import (
	"context"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scheduled partners:sync and partners:purge handlers run twice leave the same state (T-M9-11).
func TestJobs_SyncAndPurgeAreIdempotent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sid := e.service("home_blood_draw", true)
	sara := e.user("09120001001")
	moving, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	closed, err := e.svc.Create(ctx, sara, input(sid), at)
	require.NoError(t, err)
	require.NoError(t, e.svc.Cancel(ctx, sara, closed.ID, at))
	e.Exec("UPDATE partner_lab_requests SET partner_status = 'accepted' WHERE id = ?", moving.ID)

	now := func() time.Time { return at.Add(31 * 24 * time.Hour) }
	sync, purge := e.svc.SyncJob(now), e.svc.PurgeJob(now)
	state := func() string {
		return e.String("SELECT GROUP_CONCAT(CONCAT_WS(':', id, status, payload_enc IS NULL, IFNULL(purged_at, '-')) ORDER BY id) " +
			"FROM partner_lab_requests")
	}
	require.NoError(t, sync(ctx, []byte("{}")))
	require.NoError(t, purge(ctx, []byte("{}")))
	first := state()
	require.NoError(t, sync(ctx, []byte("{}")))
	require.NoError(t, purge(ctx, []byte("{}")))
	assert.Equal(t, first, state())
	assert.Equal(t, "accepted", e.String("SELECT status FROM partner_lab_requests WHERE id = ?", moving.ID))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM partner_lab_requests WHERE id = ? AND payload_enc IS NULL", closed.ID))
}
