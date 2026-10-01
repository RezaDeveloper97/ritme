package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const menopauseCheckupsVersion int64 = 24 // 00024_activate_menopause_checkups.sql

// CB-MENO-01b: the nine menopause checkups are active at head, inactive again after Down, and no other row changes.
func TestActivateMenopauseCheckupsMigration(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(q string) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, q).Scan(&n))
		return n
	}
	const meno = "SELECT COUNT(*) FROM checkup_types WHERE user_id IS NULL AND `key` LIKE 'meno\\_%' AND is_active = 1"
	const shared = "SELECT COUNT(*) FROM checkup_types WHERE user_id IS NULL AND `key` NOT LIKE 'meno\\_%' AND is_active = 1"

	assert.Equal(t, 9, count(meno))
	sharedActive := count(shared)
	assert.Positive(t, sharedActive)

	_, err = p.DownTo(ctx, menopauseCheckupsVersion-1)
	require.NoError(t, err)
	assert.Zero(t, count(meno))
	assert.Equal(t, sharedActive, count(shared), "shared rows untouched")

	_, err = p.UpTo(ctx, menopauseCheckupsVersion)
	require.NoError(t, err)
	assert.Equal(t, 9, count(meno))
	assert.Equal(t, sharedActive, count(shared))
}
