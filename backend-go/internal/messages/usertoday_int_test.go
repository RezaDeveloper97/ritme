package messages_test

// T-M9-38b: the admin rule preview resolves the target user's day in her stored zone under v6 (D-126), Tehran's day
// under v1 or without a stored zone.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

func TestUserToday_StoredZone(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	uid := seedCycleUser(t, db, "09120006393", "non_ttc", "", civildate.MustParse("2026-10-10"))
	// 01:30 in Tehran on the 21st is still the 20th in Los Angeles.
	now := time.Date(2026, 10, 21, 1, 30, 0, 0, civildate.Tehran)

	v1 := messages.NewHandlers(db, clock.Real{})
	v6 := messages.NewHandlers(db, clock.Real{}).WithEngine(engineSource(t, db))

	got, err := v6.UserToday(ctx, uid, now)
	require.NoError(t, err)
	assert.Equal(t, civildate.MustParse("2026-10-21"), got, "no stored zone: Tehran")

	_, err = db.Exec(`UPDATE user_profiles SET timezone = 'America/Los_Angeles' WHERE user_id = ?`, uid)
	require.NoError(t, err)
	got, err = v6.UserToday(ctx, uid, now)
	require.NoError(t, err)
	assert.Equal(t, civildate.MustParse("2026-10-20"), got, "v6: her stored zone")

	got, err = v1.UserToday(ctx, uid, now)
	require.NoError(t, err)
	assert.Equal(t, civildate.MustParse("2026-10-21"), got, "v1: Tehran, unchanged")

	got, err = v6.UserToday(ctx, 999_999, now)
	require.NoError(t, err)
	assert.Equal(t, civildate.MustParse("2026-10-21"), got, "no profile: Tehran")
}
