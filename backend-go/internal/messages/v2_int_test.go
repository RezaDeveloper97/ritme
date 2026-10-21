package messages_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

// T-M9-03: BuildFacts — the /messages/selected facts and the admin rule preview (admin/messages Preview calls it) —
// gives a pregnant user her gestational week (LMP 70 days ago → week 11), not a negative number.
func TestBuildFacts_PregnancyWeek(t *testing.T) {
	db := testdb.New(t)
	today := civildate.MustParse("2026-10-21")
	res, err := db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Mina', '09120006031', '2026-09-01 09:00:00', '2026-09-01 09:00:00')`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO pregnancy_profiles (user_id, pregnancy_mode, cycle_mode, age_source, lmp_date, onboarding_completed, created_at, updated_at)
		VALUES (?, 1, 0, 'lmp', ?, 1, '2026-09-01 09:00:00', '2026-09-01 09:00:00')`, uid, today.AddDays(-70).String())
	require.NoError(t, err)

	facts, err := messages.BuildFacts(context.Background(), messages.NewHandlers(db, clock.Real{}), uint64(uid), today) //nolint:gosec // test id
	require.NoError(t, err)
	assert.Equal(t, "pregnancy", facts["mode"])
	assert.Equal(t, 11.0, facts["pregnancy_week"])
}
