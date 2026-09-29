package migrations_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	platformdb "github.com/ritme/backend-go/internal/platform/db"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

const pregnancyV2Version int64 = 5 // 00005_pregnancy_v2.sql

// Review #14 (T-M2-34): rolling back 00005 removes only the untouched seed rows of its
// message_contents groups; admin-edited and admin-added rows survive, and Up runs again.
func TestPregnancyV2Down_KeepsAdminCopy(t *testing.T) {
	ctx := context.Background()
	db := testdb.New(t)
	p, err := platformdb.NewMigrator(db)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = p.Up(context.Background()) })

	count := func(where string, args ...any) int {
		var n int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_contents WHERE "+where, args...).Scan(&n))
		return n
	}
	groups := "`group` IN ('pregnancy_week_tip', 'pregnancy_alert', 'pregnancy_setup')"
	seeded := count(groups)
	require.Positive(t, seeded)

	// An admin edit (bumps updated_at) and an admin-added locale row.
	_, err = db.ExecContext(ctx, "UPDATE message_contents SET payload = JSON_SET(payload, '$.title', 'Edited'), "+
		"updated_at = created_at + INTERVAL 1 DAY WHERE `group` = 'pregnancy_week_tip' AND item_key = '12' AND locale = 'en'")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO message_contents (`group`, item_key, locale, label, payload, created_at, updated_at) "+
		"VALUES ('pregnancy_week_tip', '12', 'ar', 'pregnancy_week_tip / 12', '{\"title\":\"ar\"}', NOW(), NOW())")
	require.NoError(t, err)

	_, err = p.DownTo(ctx, pregnancyV2Version-1)
	require.NoError(t, err)
	assert.Equal(t, 2, count(groups), "only the edited and the admin-added row remain")
	assert.Equal(t, 1, count("`group` = 'pregnancy_week_tip' AND item_key = '12' AND locale = 'en' AND JSON_VALUE(payload, '$.title') = 'Edited'"))

	_, err = p.Up(ctx)
	require.NoError(t, err)
	assert.Equal(t, seeded+1, count(groups), "Up re-seeds the defaults around the kept rows")
	assert.Equal(t, 1, count("`group` = 'pregnancy_week_tip' AND item_key = '12' AND locale = 'en' AND JSON_VALUE(payload, '$.title') = 'Edited'"))
}
