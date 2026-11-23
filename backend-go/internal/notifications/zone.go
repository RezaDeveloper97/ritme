package notifications

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// The user's clock (T-M12-21c, like M-08 for messages): quiet hours, the timed reminders of internal/reminders and
// the medication doses of internal/care are wall-clock times of the user's own zone — «۸ صبح» is 08:00 where she
// is. Preferences.Zone carries that zone; nil is Asia/Tehran, so a user without a stored zone (and every sender
// that does not set one) behaves exactly as before. Requests take the zone from the usertz middleware
// (civildate.Zone(ctx): X-Timezone → stored → Tehran), background jobs from the stored user_profiles.timezone
// (usertz.Zones). Stored times stay wall-clock "HH:MM", so moving zones needs no data migration.

// Location is the zone the preferences' wall-clock times are read in (Asia/Tehran when none is set).
func (p Preferences) Location() *time.Location {
	if p.Zone == nil {
		return civildate.Tehran
	}
	return p.Zone
}

// InZone returns p read in loc (nil = Asia/Tehran).
func (p Preferences) InZone(loc *time.Location) Preferences {
	p.Zone = loc
	return p
}

// WallTime is the instant of the wall-clock time minute (minutes since midnight) on day in loc (nil = Tehran),
// defined on daylight-saving days so that every (day, minute) maps to exactly one instant:
//   - a time that exists once is that instant;
//   - a time that exists twice (the repeated hour when clocks go back, 02:30 on a 25-hour day) is the first one,
//     so a dose there fires once, not twice;
//   - a time that does not exist (inside the hour skipped when clocks go forward, 02:30 on a 23-hour day) is the
//     first valid instant after the gap — the transition itself (03:00 local) — so the dose is late, not lost.
func WallTime(day civildate.Date, minute int, loc *time.Location) time.Time {
	if loc == nil {
		loc = civildate.Tehran
	}
	h, m := minute/60, minute%60
	naive := time.Date(day.Year, day.Month, day.Day, h, m, 0, 0, time.UTC) // the wall clock read as if UTC
	var best time.Time
	for _, probe := range []time.Time{naive.Add(-24 * time.Hour), naive.Add(24 * time.Hour)} {
		_, offset := probe.In(loc).Zone()
		c := naive.Add(-time.Duration(offset) * time.Second)
		if l := c.In(loc); l.Year() == day.Year && l.Month() == day.Month && l.Day() == day.Day && l.Hour() == h && l.Minute() == m {
			if best.IsZero() || c.Before(best) {
				best = c
			}
		}
	}
	if !best.IsZero() {
		return best.In(loc)
	}
	// In the gap (the offset grew): read with the offset from before it, the wall clock lands just after the
	// transition, in the zone that starts there; that zone's start is the first valid instant.
	_, offset := naive.Add(-24 * time.Hour).In(loc).Zone()
	start, _ := naive.Add(-time.Duration(offset) * time.Second).In(loc).ZoneBounds()
	if start.IsZero() {
		return time.Date(day.Year, day.Month, day.Day, h, m, 0, 0, loc) // no transition known: Go's own reading
	}
	return start.In(loc)
}
