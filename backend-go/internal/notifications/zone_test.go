package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M12-21c — the user's clock: WallTime's daylight-saving rule and quiet hours read in Preferences.Zone.

func TestWallTime(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	ny, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	utc := func(y int, m time.Month, d, hh, mm int) time.Time { return time.Date(y, m, d, hh, mm, 0, 0, time.UTC) }
	cases := []struct {
		name   string
		day    string
		minute int
		loc    *time.Location
		want   time.Time
	}{
		{"Tehran, plain", "2026-03-20", 8 * 60, civildate.Tehran, time.Date(2026, 3, 20, 8, 0, 0, 0, civildate.Tehran)},
		{"nil = Tehran", "2026-03-20", 23*60 + 59, nil, time.Date(2026, 3, 20, 23, 59, 0, 0, civildate.Tehran)},
		{"Berlin winter", "2026-03-28", 8 * 60, berlin, utc(2026, 3, 28, 7, 0)},
		{"Berlin summer", "2026-03-30", 8 * 60, berlin, utc(2026, 3, 30, 6, 0)},
		{"Berlin 01:59 before the gap", "2026-03-29", 1*60 + 59, berlin, utc(2026, 3, 29, 0, 59)},
		{"Berlin 02:00 in the gap → 03:00 CEST", "2026-03-29", 2 * 60, berlin, utc(2026, 3, 29, 1, 0)},
		{"Berlin 02:30 in the gap → 03:00 CEST", "2026-03-29", 2*60 + 30, berlin, utc(2026, 3, 29, 1, 0)},
		{"Berlin 03:00 right after the gap", "2026-03-29", 3 * 60, berlin, utc(2026, 3, 29, 1, 0)},
		{"Berlin 02:30 twice → the first (CEST)", "2026-10-25", 2*60 + 30, berlin, utc(2026, 10, 25, 0, 30)},
		{"Berlin 03:00 after the overlap", "2026-10-25", 3 * 60, berlin, utc(2026, 10, 25, 2, 0)},
		{"New York 02:30 in the gap", "2026-03-08", 2*60 + 30, ny, utc(2026, 3, 8, 7, 0)},
		{"New York 01:30 twice → EDT", "2026-11-01", 1*60 + 30, ny, utc(2026, 11, 1, 5, 30)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := WallTime(civildate.MustParse(c.day), c.minute, c.loc)
			assert.True(t, c.want.Equal(got), "got %v, want %v", got.UTC(), c.want.UTC())
			loc := c.loc
			if loc == nil {
				loc = civildate.Tehran
			}
			assert.Equal(t, loc, got.Location())
		})
	}
}

func TestQuietHours_OnTheUsersZone(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	tehran := Defaults()
	assert.Equal(t, civildate.Tehran, tehran.Location())
	p := tehran.InZone(berlin)
	assert.Equal(t, berlin, p.Location())
	assert.Nil(t, tehran.Zone, "InZone copies")

	// 23:30 Tehran = 21:00 Berlin (CEST): quiet for a Tehran user, not for a Berlin one.
	at := time.Date(2026, 9, 23, 23, 30, 0, 0, civildate.Tehran)
	assert.True(t, tehran.InQuietHours(at))
	assert.False(t, p.InQuietHours(at))
	// 07:00 Berlin = 08:30 Tehran: the other way round.
	at = time.Date(2026, 9, 24, 7, 0, 0, 0, berlin)
	assert.False(t, tehran.InQuietHours(at))
	assert.True(t, p.InQuietHours(at))
	d := Decide(p, Medications, at, time.Time{})
	assert.Equal(t, ReasonQuietHours, d.Reason)
	assert.True(t, time.Date(2026, 9, 24, 8, 0, 0, 0, berlin).Equal(d.DeferUntil))

	// The end inside the skipped hour: 02:30 on 2026-03-29 does not exist → the window ends at 03:00 CEST.
	gap := p
	gap.QuietStart, gap.QuietEnd = 23*60, 2*60+30
	night := time.Date(2026, 3, 29, 1, 30, 0, 0, berlin)
	require.True(t, gap.InQuietHours(night))
	assert.True(t, time.Date(2026, 3, 29, 3, 0, 0, 0, berlin).Equal(gap.QuietEndsAt(night)))
	// Over the 25-hour night the end is still 08:00 local.
	fall := time.Date(2026, 10, 24, 23, 30, 0, 0, berlin)
	assert.True(t, time.Date(2026, 10, 25, 8, 0, 0, 0, berlin).Equal(p.QuietEndsAt(fall)))
	assert.Equal(t, 9*time.Hour+30*time.Minute, p.QuietEndsAt(fall).Sub(fall))
}
