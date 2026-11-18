package remindersms_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/remindersms"
)

// T-M12-21d — «یادآور پیامکی» on the user's own clock, like the Web Push sender (T-M12-21c): doses and quiet hours
// in her stored zone, the daily cap from her local midnight. A user without a stored zone keeps the Tehran clock.

func TestMain(m *testing.M) { testdb.Main(m) }

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

type env struct {
	db    *sql.DB
	users []uint64
}

func setup(t *testing.T) *env {
	t.Helper()
	return &env{db: testdb.New(t)}
}

// user is a Plus user (in the audience) with a mobile; zone "" = no stored zone; quietOn=false switches quiet hours off.
func (e *env) user(t *testing.T, mobile, zone string, quietOn bool) uint64 {
	t.Helper()
	res, err := e.db.Exec(`INSERT INTO users (name, mobile, created_at, updated_at) VALUES ('Test', ?, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, mobile)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	uid := uint64(id)
	if zone != "" {
		_, err = e.db.Exec(`INSERT INTO user_profiles (user_id, timezone, created_at, updated_at)
			VALUES (?, ?, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, uid, zone)
		require.NoError(t, err)
	}
	if !quietOn {
		_, err = e.db.Exec(`INSERT INTO notification_preferences (user_id, categories, quiet_hours_enabled, neutral_copy, created_at, updated_at)
			VALUES (?, '{}', 0, 0, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`, uid)
		require.NoError(t, err)
	}
	e.users = append(e.users, uid)
	return uid
}

// med is a daily medication with the given wall-clock slots, from 2026-01-01.
func (e *env) med(t *testing.T, userID uint64, times string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO reminders (user_id, type, title, recurrence, starts_on, is_active, meta, created_at, updated_at)
		VALUES (?, 'medication', 'آهن', 'daily', '2026-01-01', 1, ?, '2026-01-01 09:00:00', '2026-01-01 09:00:00')`,
		userID, `{"v":1,"form":"tablet","times":`+times+`,"notify":true}`)
	require.NoError(t, err)
}

func (e *env) worker(f *remindersms.Fake, dailyCap int) *remindersms.Worker {
	audience := func(context.Context, time.Time) ([]uint64, error) { return e.users, nil }
	return remindersms.NewWorker(e.db, profilestore.New(e.db), audience, f, func(context.Context) string { return "fa" }, dailyCap, quiet)
}

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}

type got struct {
	mobile, clock string
	tick          time.Time
}

// A Berlin user's 08:00 dose is texted at 08:00 Berlin, exactly once, on the day clocks go forward (2026-03-29) and on
// the day they go back (2026-10-25); her 02:30 dose — skipped hour in March, repeated hour in October — once too. A
// Tehran user (no stored zone) keeps 08:00 Tehran. The worker ticks every 5 minutes through her whole local day.
func TestTick_BerlinDoseAtLocalEightOnceAcrossDST(t *testing.T) {
	berlin := zone(t, "Europe/Berlin")
	for _, day := range []string{"2026-03-29", "2026-10-25"} {
		t.Run(day, func(t *testing.T) {
			e := setup(t)
			ctx := context.Background()
			ber := e.user(t, "09120000201", "Europe/Berlin", false)
			e.med(t, ber, `["02:30","08:00"]`)
			thr := e.user(t, "09120000202", "", false)
			e.med(t, thr, `["08:00"]`)

			f := remindersms.NewFake(quiet)
			w := e.worker(f, 10)
			d := civildate.MustParse(day)
			start := time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, berlin)
			end := start.AddDate(0, 0, 1) // 23 or 25 hours later
			var sms []got
			for now := start.Add(30 * time.Second); now.Before(end); now = now.Add(5 * time.Minute) {
				_, err := w.Tick(ctx, now)
				require.NoError(t, err)
				for _, s := range f.Drain() {
					sms = append(sms, got{mobile: s.Mobile, clock: s.SMS.Clock, tick: now})
				}
			}

			var berlinSMS, tehranEight []got
			for _, s := range sms {
				switch {
				case s.mobile == "09120000201":
					berlinSMS = append(berlinSMS, s)
				case s.mobile == "09120000202" && s.clock == "08:00":
					tehranEight = append(tehranEight, s)
				}
			}
			require.Len(t, berlinSMS, 2, "Berlin: one SMS per dose, never doubled or lost")
			byClock := map[string]time.Time{}
			for _, s := range berlinSMS {
				byClock[s.clock] = s.tick.In(berlin)
			}
			eight, ok := byClock["08:00"]
			require.True(t, ok)
			assert.Equal(t, 8, eight.Hour(), "the 08:00 dose at 08:00 Berlin, got %s", eight)
			assert.Less(t, eight.Minute(), 10)
			night, ok := byClock["02:30"]
			require.True(t, ok)
			if day == "2026-03-29" { // 02:30 does not exist: at the transition, 03:00 CEST
				assert.Equal(t, 3, night.Hour(), "got %s", night)
				assert.Less(t, night.Minute(), 10)
			} else { // 02:30 happens twice: the first one, still CEST
				assert.Equal(t, 2, night.Hour(), "got %s", night)
				_, off := night.Zone()
				assert.Equal(t, 2*3600, off, "first occurrence (CEST), got %s", night)
			}

			require.Len(t, tehranEight, 1, "Tehran: once")
			tt := tehranEight[0].tick.In(civildate.Tehran)
			assert.Equal(t, 8, tt.Hour(), "Tehran user unchanged: 08:00 Tehran, got %s", tt)
			assert.Less(t, tt.Minute(), 10)
		})
	}
}

// The daily cap (1) counts from the user's local midnight. On 2026-09-23 (Berlin CEST = Tehran − 1:30), in time order:
//
//	Tehran 23:00            Tehran user 23:00 dose          → sent
//	Tehran 23:40 = Ber 22:10 Berlin user 22:10 dose          → sent
//	Tehran 00:10 = Ber 22:40 Tehran 00:10 (her new day)      → sent; Berlin 22:40 (still her 23rd) → capped
//	Tehran 00:20             Tehran 00:20                    → capped, as before
//	Tehran 01:40 = Ber 00:10 Berlin 00:10 (her 24th)         → sent
func TestTick_DailyCapResetsAtLocalMidnight(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	ber := e.user(t, "09120000203", "Europe/Berlin", false)
	e.med(t, ber, `["22:10","22:40","00:10"]`)
	thr := e.user(t, "09120000204", "", false)
	e.med(t, thr, `["23:00","00:10","00:20"]`)

	f := remindersms.NewFake(quiet)
	w := e.worker(f, 1)
	tick := func(at time.Time) []string {
		t.Helper()
		_, err := w.Tick(ctx, at)
		require.NoError(t, err)
		out := []string{}
		for _, s := range f.Drain() {
			out = append(out, s.Mobile+" "+s.SMS.Clock)
		}
		return out
	}
	tehran := func(day, h, m int) time.Time { return time.Date(2026, 9, day, h, m, 30, 0, civildate.Tehran) }

	assert.Equal(t, []string{"09120000204 23:00"}, tick(tehran(23, 23, 0)))
	assert.Equal(t, []string{"09120000203 22:10"}, tick(tehran(23, 23, 40)))
	assert.Equal(t, []string{"09120000204 00:10"}, tick(tehran(24, 0, 10)), "Berlin 22:40 capped: still her 23rd")
	assert.Equal(t, []string{}, tick(tehran(24, 0, 20)), "Tehran: capped")
	assert.Equal(t, []string{"09120000203 00:10"}, tick(tehran(24, 1, 40)), "her 24th: a new day")
}

// Quiet hours on her clock: 22:30 Berlin is 00:00 Tehran — inside Tehran's default 23:00–08:00, outside hers.
func TestTick_QuietHoursOnTheUsersZone(t *testing.T) {
	e := setup(t)
	berlin := zone(t, "Europe/Berlin")
	ber := e.user(t, "09120000205", "Europe/Berlin", true)
	e.med(t, ber, `["22:30"]`)

	f := remindersms.NewFake(quiet)
	sent, err := e.worker(f, 10).Tick(context.Background(), time.Date(2026, 9, 23, 22, 30, 30, 0, berlin))
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	if s := f.Drain(); assert.Len(t, s, 1) {
		assert.Equal(t, "22:30", s[0].SMS.Clock)
	}
}
