package care_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M12-21c — doses, /care/upcoming and quiet hours on the user's own clock (user decision 2026-11-18, like M-08).
// A user without a stored zone keeps the Tehran clock (every other test in this package).

// storeZone gives the user a profile row with a stored zone (what usertz writes from X-Timezone).
func (e *env) storeZone(t *testing.T, userID uint64, zone string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO user_profiles (user_id, timezone, created_at, updated_at)
		VALUES (?, ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, userID, zone)
	require.NoError(t, err)
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	require.NoError(t, err)
	return loc
}

// The Web Push sender reads each user's stored zone: an 08:00 dose reaches a Berlin user at 08:00 Berlin (09:30
// Tehran), not at 08:00 Tehran (06:30 Berlin); a Tehran user's 08:00 dose is unchanged.
func TestTick_DoseOnTheUsersStoredZone(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	berlin := mustZone(t, "Europe/Berlin")

	ber, berTok := e.user(t, "09120000101")
	e.storeZone(t, ber, "Europe/Berlin")
	e.subscribe(t, ber, "https://fcm.googleapis.com/fcm/send/berlin", "fa")
	e.create(t, berTok, folic) // 08:00 + 20:00, from 2026-09-23
	thr, thrTok := e.user(t, "09120000102")
	e.subscribe(t, thr, "https://fcm.googleapis.com/fcm/send/tehran", "fa")
	e.create(t, thrTok, folic)

	f := &fakePush{}
	p := e.pusher(f)

	// 08:00:30 Tehran on the 24th = 06:30 Berlin: only the Tehran user's dose.
	sent, err := p.Tick(ctx, time.Date(2026, 9, 24, 8, 0, 30, 0, civildate.Tehran))
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, f.got, 1)
	assert.Equal(t, "https://fcm.googleapis.com/fcm/send/tehran", f.got[0].endpoint)

	// 08:00:30 Berlin on the 24th: the Berlin user's dose, keyed by her day and slot.
	sent, err = p.Tick(ctx, time.Date(2026, 9, 24, 8, 0, 30, 0, berlin))
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, f.got, 2)
	assert.Equal(t, "https://fcm.googleapis.com/fcm/send/berlin", f.got[1].endpoint)
	assert.Contains(t, f.got[1].payload["tag"], ":2026-09-24:08:00")

	// A second tick a minute later announces nothing again.
	sent, err = p.Tick(ctx, time.Date(2026, 9, 24, 8, 1, 30, 0, berlin))
	require.NoError(t, err)
	assert.Zero(t, sent)
}

// Quiet hours on her clock: 23:30 Tehran is 21:00 Berlin, outside a Berlin user's 23:00–08:00, so her inbox row goes
// out at once; a Tehran user's row waits.
func TestTickNotices_QuietHoursOnTheUsersStoredZone(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	night := time.Date(2026, 9, 23, 23, 30, 0, 0, civildate.Tehran)

	ber, _ := e.user(t, "09120000103")
	e.storeZone(t, ber, "Europe/Berlin")
	e.prefs(t, ber, `{}`, true, false)
	e.subscribe(t, ber, "https://fcm.googleapis.com/fcm/send/b", "fa")
	berRow := e.notice(t, ber, "message_push", "", night.Add(-time.Minute), false)
	thr, _ := e.user(t, "09120000104")
	e.prefs(t, thr, `{}`, true, false)
	e.subscribe(t, thr, "https://fcm.googleapis.com/fcm/send/t", "fa")
	thrRow := e.notice(t, thr, "message_push", "", night.Add(-time.Minute), false)

	f := &fakePush{}
	sent, err := e.pusher(f).TickNotices(ctx, night)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, f.got, 1)
	assert.Equal(t, "https://fcm.googleapis.com/fcm/send/b", f.got[0].endpoint)
	assert.True(t, e.handled(t, berRow))
	assert.False(t, e.handled(t, thrRow), "Tehran user: held until 08:00 Tehran")
}

// GET /care/upcoming for a Berlin user (stored zone, no header): the doses at 20:00 and 08:00 Berlin, the wall-clock
// strings on her clock; the same request with X-Timezone: Asia/Tehran gives the Tehran instants.
func TestUpcoming_OnTheUsersZone(t *testing.T) {
	e := setup(t)
	berlin := mustZone(t, "Europe/Berlin")
	uid, tok := e.user(t, "09120000105")
	e.storeZone(t, uid, "Europe/Berlin")
	e.prefs(t, uid, `{}`, true, false)
	e.create(t, tok, folic)

	// now = 2026-09-23 10:00 Tehran = 08:30 Berlin.
	res := e.do(t, "GET", "/api/v1/care/upcoming?hours=24", tok, "fa", "")
	require.Equal(t, 200, res.status, res.raw)
	assert.Equal(t, "2026-09-23 08:30:00", res.data()["generated_at"])
	items := res.data()["items"].([]any)
	require.Len(t, items, 2)
	first, second := items[0].(map[string]any), items[1].(map[string]any)
	assert.Equal(t, "2026-09-23 20:00:00", first["at"])
	assert.EqualValues(t, time.Date(2026, 9, 23, 20, 0, 0, 0, berlin).UnixMilli(), first["at_ms"])
	assert.Equal(t, "2026-09-24 08:00:00", second["at"])
	assert.EqualValues(t, time.Date(2026, 9, 24, 8, 0, 0, 0, berlin).UnixMilli(), second["at_ms"])
	assert.Equal(t, "2026-09-24", second["date"])
	assert.Equal(t, "08:00", second["slot"])
}

// The care day is her day: at 2026-09-23 10:00 Tehran it is still the 22nd in Los Angeles, so /care/today defaults to
// the 22nd and ticking the 23rd's dose is "in the future" for her; a Tehran user may tick it.
func TestTodayAndIntake_OnTheUsersDay(t *testing.T) {
	e := setup(t)
	la, laTok := e.user(t, "09120000106")
	e.storeZone(t, la, "America/Los_Angeles")
	_, thrTok := e.user(t, "09120000107")

	laMed := e.create(t, laTok, folic)
	thrMed := e.create(t, thrTok, folic)

	res := e.do(t, "GET", "/api/v1/care/today", laTok, "fa", "")
	require.Equal(t, 200, res.status, res.raw)
	assert.Equal(t, "2026-09-22", res.data()["date"])
	res = e.do(t, "GET", "/api/v1/care/today", thrTok, "fa", "")
	assert.Equal(t, "2026-09-23", res.data()["date"])

	res = e.do(t, "GET", medPath(laMed, ""), laTok, "fa", "")
	assert.Equal(t, "2026-09-22", res.data()["starts_on"], "a new medication starts on her day")

	res = e.do(t, "POST", medPath(laMed, "/intakes"), laTok, "fa", `{"date":"2026-09-23","slot":"08:00"}`)
	assert.Equal(t, 422, res.status, res.raw)
	assert.Contains(t, res.errors(), "date")
	res = e.do(t, "POST", medPath(laMed, "/intakes"), laTok, "fa", `{"date":"2026-09-22","slot":"08:00"}`)
	assert.Equal(t, 200, res.status, res.raw)
	res = e.do(t, "POST", medPath(thrMed, "/intakes"), thrTok, "fa", `{"date":"2026-09-23","slot":"08:00"}`)
	assert.Equal(t, 200, res.status, res.raw)
}
