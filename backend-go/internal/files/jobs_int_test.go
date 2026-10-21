package files_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/files"
)

// The scheduled files:sweep handler run twice leaves the same blobs: the orphan goes once, the referenced one stays
// (T-M9-11).
func TestJobs_SweepIsIdempotent(t *testing.T) {
	e := setup(t, false, nil)
	_, tok := e.user(t, "09120000011")
	kept := data(e.upload(t, tok, files.PurposeRecordDocument, pdf))
	gone := data(e.upload(t, tok, files.PurposeClaimDocument, pdf))
	var keptPath, gonePath string
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE id = ?`, kept["id"]).Scan(&keptPath))
	require.NoError(t, e.db.QueryRow(`SELECT path FROM files WHERE id = ?`, gone["id"]).Scan(&gonePath))
	_, err := e.db.Exec(`DELETE FROM files WHERE id = ?`, gone["id"])
	require.NoError(t, err)

	sweep := e.svc.SweepJob(func() time.Time { return time.Now().Add(2 * time.Hour) })
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(e.storage, filepath.FromSlash(rel)))
		return err == nil
	}
	for range 2 {
		require.NoError(t, sweep(context.Background(), []byte("{}")))
		assert.False(t, exists(gonePath), "the orphan is removed")
		assert.True(t, exists(keptPath), "a referenced blob stays")
	}
}
