package modules_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ritme/backend-go/internal/modules"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// The switches come from the in-process snapshot (no Redis GET per request); Set publishes app_modules, so every
// Service of the process — the gate's included — sees the change on its next read.
func TestStates_SnapshotAndSet(t *testing.T) {
	configcache.SetEnabled(true)
	t.Cleanup(func() { configcache.SetEnabled(false) })
	db := testdb.New(t)
	configcache.Default.Invalidate(modules.Table)
	ctx := context.Background()
	quiet := slog.New(slog.NewJSONHandler(io.Discard, nil))
	gate, admin := modules.NewService(db, nil, quiet), modules.NewService(db, nil, quiet)

	assert.True(t, gate.Enabled(ctx, modules.TTC))
	_, err := db.Exec("UPDATE app_modules SET enabled = 0 WHERE `key` = 'teen'")
	assert.NoError(t, err)
	assert.True(t, gate.Enabled(ctx, modules.Teen), "a direct DB edit waits for the TTL")

	assert.NoError(t, admin.Set(ctx, map[string]bool{modules.TTC: false}, time.Now()))
	assert.False(t, gate.Enabled(ctx, modules.TTC), "another Service of the process: next read")
	assert.False(t, gate.Enabled(ctx, modules.Teen))
}
