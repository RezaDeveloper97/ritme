package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/admin/content/admintest"
	"github.com/ritme/backend-go/internal/metrics"
)

// The scheduled admin_metrics:snapshot handler run twice keeps one equal row for the day (T-M9-11).
func TestJobs_SnapshotIsIdempotent(t *testing.T) {
	e := admintest.New(t)
	seed(e, 12)
	job := metrics.New(e.DB, admintest.Quiet).RefreshJob(func() time.Time { return now })
	state := func() string {
		return e.String("SELECT CONCAT(`day`, '|', computed_at, '|', payload) FROM metric_snapshots")
	}
	require.NoError(t, job(t.Context(), []byte("{}")))
	first := state()
	require.NoError(t, job(t.Context(), []byte("{}")))
	assert.Equal(t, 1, e.Int("SELECT COUNT(*) FROM metric_snapshots"))
	assert.Equal(t, first, state())
}
