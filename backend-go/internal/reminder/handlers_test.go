package reminder

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/reminder/store"
)

func TestIntParam(t *testing.T) {
	for in, want := range map[string]int64{"12": 12, " 12": 12, "12 ": 12, "12.9": 12, "-3": -3, "1e2": 100} {
		got, ok := intParam(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"abc", "12abc", "", "0x1A", "1e40"} {
		_, ok := intParam(in)
		assert.False(t, ok, in)
	}
}

func TestAssignDirtyLikeEloquent(t *testing.T) {
	now := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	row := store.Reminder{
		Type: "doctor", Title: "T", RecurrenceTime: sql.NullString{String: "09:00:00", Valid: true},
		StartsOn:    civildate.NullDate{Date: civildate.MustParse("2026-09-13"), Valid: true},
		ScheduledAt: sql.NullTime{Time: time.Date(2026, 10, 1, 11, 30, 0, 0, civildate.Tehran), Valid: true},
		IsActive:    true,
	}
	changed := func(k string, v any) bool {
		_, c, err := assign(&row, k, v, now)
		require.NoError(t, err, k)
		return c
	}
	assert.False(t, changed("type", "doctor"))
	assert.False(t, changed("is_active", "1"))
	assert.False(t, changed("starts_on", "2026-09-13"))
	assert.True(t, changed("starts_on", "2026-09-13 10:00"), "fromDateTime keeps the time of day")
	assert.False(t, changed("scheduled_at", "2026-10-01 11:30:00"))
	assert.False(t, changed("subtitle", nil))
	assert.True(t, changed("recurrence_time", "09:00"), `"09:00" !== "09:00:00"`)
	assert.True(t, changed("title", "U"))

	cast, _, err := assign(&row, "ends_on", "2026-09-23", now)
	require.NoError(t, err)
	b, err := jsonx.Marshal(cast, 0)
	require.NoError(t, err)
	assert.Equal(t, `"2026-09-22T20:30:00.000000Z"`, string(b))
	assert.Equal(t, civildate.MustParse("2026-09-23"), row.EndsOn.Date)
}
