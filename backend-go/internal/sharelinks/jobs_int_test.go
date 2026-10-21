package sharelinks_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scheduled sharelinks:purge handler run twice leaves the same state (T-M9-11).
func TestJobs_PurgeIsIdempotent(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000001", true)
	for range 2 {
		r := e.do(t, http.MethodPost, "/api/v1/health-record/share-links", tok, map[string]any{"range": "1y", "sections": []string{"basics"}})
		require.Equal(t, http.StatusCreated, r.status, r.raw)
	}
	// One link expired long ago (row deleted), one just expired (report wiped).
	e.exec(t, "UPDATE health_share_links SET expires_at = '2026-08-01 09:00:00' WHERE user_id = ? ORDER BY id LIMIT 1", uid)
	e.exec(t, "UPDATE health_share_links SET expires_at = '2026-10-06 08:00:00' WHERE user_id = ? AND expires_at > '2026-09-01'", uid)

	purge := e.svc.PurgeJob(func() time.Time { return fixed })
	state := func() (rows, wiped int) {
		require.NoError(t, e.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(payload IS NULL), 0) FROM health_share_links WHERE user_id = ?", uid).
			Scan(&rows, &wiped))
		return rows, wiped
	}
	ctx := context.Background()
	require.NoError(t, purge(ctx, []byte("{}")))
	rows, wiped := state()
	assert.Equal(t, 1, rows, "the long-expired row is deleted")
	assert.Equal(t, 1, wiped, "the expired report is wiped")
	require.NoError(t, purge(ctx, []byte("{}")))
	rows2, wiped2 := state()
	assert.Equal(t, rows, rows2)
	assert.Equal(t, wiped, wiped2)
}
