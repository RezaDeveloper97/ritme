package care_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/care/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/config"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// pushed is one delivery the fake push service saw.
type pushed struct {
	endpoint string
	payload  map[string]string
}

// sendResult is one scripted answer of the fake push service: an HTTP status, or a transport error (timeout).
type sendResult struct {
	status int
	err    error
}

// fakePush is the push service: it records every attempt (safe for concurrent senders, T-M12-22), answers each
// endpoint from its script first (one entry per attempt), then from status, then 201.
type fakePush struct {
	mu     sync.Mutex
	got    []pushed
	status map[string]int          // endpoint → HTTP status (default 201)
	script map[string][]sendResult // endpoint → the answers of its next attempts, in order
	delay  time.Duration           // how long one send takes (widens the window concurrent ticks race in)
}

func (f *fakePush) send(_ context.Context, payload []byte, sub store.PushSubscription) (int, error) {
	var m map[string]string
	if err := json.Unmarshal(payload, &m); err != nil {
		return 0, err
	}
	f.mu.Lock()
	f.got = append(f.got, pushed{endpoint: sub.Endpoint, payload: m})
	var res *sendResult
	if q := f.script[sub.Endpoint]; len(q) > 0 {
		res, f.script[sub.Endpoint] = &q[0], q[1:]
	}
	s, ok := f.status[sub.Endpoint]
	delay := f.delay
	f.mu.Unlock()
	if delay > 0 {
		time.Sleep(delay) // a slow push service, so concurrent ticks overlap
	}
	switch {
	case res != nil:
		return res.status, res.err
	case ok:
		return s, nil
	}
	return 201, nil
}

// attempts counts the sends of the message tagged tag (to any endpoint when endpoint is "").
func (f *fakePush) attempts(endpoint, tag string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, g := range f.got {
		if g.payload["tag"] == tag && (endpoint == "" || g.endpoint == endpoint) {
			n++
		}
	}
	return n
}

// all is a copy of every attempt so far.
func (f *fakePush) all() []pushed {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.got)
}

func (e *env) pusher(f *fakePush) *care.Pusher {
	return care.NewPusher(e.db, profilestore.New(e.db), config.Push{VAPIDPublicKey: "x", VAPIDPrivateKey: "y"}, quiet).WithSender(f.send)
}

func (e *env) subscribe(t *testing.T, userID uint64, endpoint, locale string) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO push_subscriptions (user_id, endpoint, endpoint_hash, p256dh, auth, locale, created_at, updated_at)
		VALUES (?, ?, SHA2(?, 256), 'k', 'a', ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, userID, endpoint, endpoint, locale)
	require.NoError(t, err)
}

// prefs stores the user's notification settings: categories JSON, quiet hours 23:00–08:00 on/off, «متن خنثی».
func (e *env) prefs(t *testing.T, userID uint64, categories string, quietOn, neutral bool) {
	t.Helper()
	_, err := e.db.Exec(`INSERT INTO notification_preferences (user_id, categories, quiet_hours_enabled, neutral_copy, created_at, updated_at)
		VALUES (?, ?, ?, ?, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, userID, categories, quietOn, neutral)
	require.NoError(t, err)
}

func (e *env) notice(t *testing.T, userID uint64, typ, action string, created time.Time, read bool) uint64 {
	t.Helper()
	var url, readAt any
	if action != "" {
		url = action
	}
	if read {
		readAt = created
	}
	res, err := e.db.Exec(`INSERT INTO user_notifications (user_id, type, title, body, action_url, read_at, created_at, updated_at)
		VALUES (?, ?, '{"fa":"همراه دعوتت را پذیرفت","en":"Your companion accepted"}', '{"fa":"متن","en":"Body"}', ?, ?, ?, ?)`,
		userID, typ, url, readAt, created, created)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint64(id)
}

func (e *env) handled(t *testing.T, id uint64) bool {
	t.Helper()
	var at sql.NullTime
	require.NoError(t, e.db.QueryRow(`SELECT pushed_at FROM user_notifications WHERE id = ?`, id).Scan(&at))
	return at.Valid
}

func TestTickNoticesPushesNewInboxRowsOnce(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	uid, _ := e.user(t, "09120000001")
	e.prefs(t, uid, `{}`, true, false)
	e.subscribe(t, uid, "https://fcm.googleapis.com/fcm/send/fa", "fa")
	e.subscribe(t, uid, "https://web.push.apple.com/en", "en")
	id := e.notice(t, uid, "companion", "/companions", at.Add(-time.Minute), false)
	plain := e.notice(t, uid, "postpartum", "https://evil.example/x", at.Add(-time.Minute), false)

	f := &fakePush{}
	p := e.pusher(f)
	sent, err := p.TickNotices(ctx, at)
	require.NoError(t, err)
	assert.Equal(t, 2, sent)
	require.Len(t, f.got, 4)
	assert.Equal(t, map[string]string{
		"title": "همراه دعوتت را پذیرفت", "body": "متن", "url": "/fa/companions", "tag": fmt.Sprintf("n:%d", id),
	}, f.got[0].payload)
	assert.Equal(t, map[string]string{
		"title": "Your companion accepted", "body": "Body", "url": "/en/companions", "tag": fmt.Sprintf("n:%d", id),
	}, f.got[1].payload)
	assert.Equal(t, "/fa/home", f.got[2].payload["url"], "an action that is not an in-app path opens the home")
	assert.True(t, e.handled(t, id))
	assert.True(t, e.handled(t, plain))

	sent, err = p.TickNotices(ctx, at.Add(30*time.Second))
	require.NoError(t, err)
	assert.Zero(t, sent, "a handled row is never announced again")
	assert.Len(t, f.got, 4)
}

func TestTickNoticesNeutralCopyIsTheDefault(t *testing.T) {
	e := setup(t)
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	uid, _ := e.user(t, "09120000002") // no settings row: «متن خنثی» on
	e.subscribe(t, uid, "https://fcm.googleapis.com/fcm/send/a", "fa")
	e.notice(t, uid, "learning", "/learn/unlocked/7", at.Add(-time.Minute), false)

	f := &fakePush{}
	sent, err := e.pusher(f).TickNotices(context.Background(), at)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, f.got, 1)
	assert.Equal(t, "اعلان تازه", f.got[0].payload["title"])
	assert.Equal(t, "یک اعلان تازه در ریتمی داری.", f.got[0].payload["body"])
	assert.Equal(t, "/fa/learn/unlocked/7", f.got[0].payload["url"])
}

func TestTickNoticesSilentRows(t *testing.T) {
	e := setup(t)
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	off, _ := e.user(t, "09120000003")
	e.prefs(t, off, `{"companion":false}`, false, false)
	e.subscribe(t, off, "https://fcm.googleapis.com/fcm/send/off", "fa")
	nosub, _ := e.user(t, "09120000004")

	ids := []uint64{
		e.notice(t, off, "companion", "/companions", at.Add(-time.Minute), false),               // category off
		e.notice(t, off, "family", "/family", at.Add(-time.Minute), false),                      // same switch
		e.notice(t, off, "learning", "/learn", at.Add(-time.Minute), true),                      // already read
		e.notice(t, off, "learning", "/learn", at.Add(-care.NoticeLookback-time.Minute), false), // stale
		e.notice(t, nosub, "learning", "/learn", at.Add(-time.Minute), false),                   // no browser
	}
	f := &fakePush{}
	sent, err := e.pusher(f).TickNotices(context.Background(), at)
	require.NoError(t, err)
	assert.Zero(t, sent)
	assert.Empty(t, f.got)
	for _, id := range ids {
		assert.True(t, e.handled(t, id), "row %d is handled without a push", id)
	}
}

func TestTickNoticesWaitsForTheEndOfQuietHours(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	night := time.Date(2026, 9, 23, 23, 30, 0, 0, civildate.Tehran)
	uid, _ := e.user(t, "09120000005")
	e.prefs(t, uid, `{}`, true, false)
	e.subscribe(t, uid, "https://fcm.googleapis.com/fcm/send/q", "fa")
	id := e.notice(t, uid, "message_push", "", night.Add(-time.Minute), false)

	f := &fakePush{}
	p := e.pusher(f)
	sent, err := p.TickNotices(ctx, night)
	require.NoError(t, err)
	assert.Zero(t, sent)
	assert.False(t, e.handled(t, id), "held, not dropped")

	sent, err = p.TickNotices(ctx, time.Date(2026, 9, 24, 8, 0, 30, 0, civildate.Tehran))
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, f.got, 1)
	assert.Equal(t, "/fa/home", f.got[0].payload["url"])
	assert.True(t, e.handled(t, id))
}

func TestTickNoticesDropsAGoneSubscription(t *testing.T) {
	e := setup(t)
	at := time.Date(2026, 9, 23, 10, 0, 0, 0, civildate.Tehran)
	uid, _ := e.user(t, "09120000006")
	e.subscribe(t, uid, "https://fcm.googleapis.com/fcm/send/gone", "fa")
	e.notice(t, uid, "learning", "/learn", at.Add(-time.Minute), false)

	f := &fakePush{status: map[string]int{"https://fcm.googleapis.com/fcm/send/gone": 410}}
	sent, err := e.pusher(f).TickNotices(context.Background(), at)
	require.NoError(t, err)
	assert.Zero(t, sent)
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE user_id = ?`, uid).Scan(&n))
	assert.Zero(t, n)
}

func TestUpcomingListsDosesAndAppointmentReminders(t *testing.T) {
	e := setup(t)
	uid, tok := e.user(t, "09120000007")
	e.prefs(t, uid, `{}`, true, false)
	require.Equal(t, 201, e.do(t, "POST", "/api/v1/care/medications", tok, "fa", folic).status)
	res := e.do(t, "POST", "/api/v1/care/appointments", tok, "fa",
		`{"kind":"in_person","topic":"ultrasound","title":"سونوگرافی NT","scheduled_at":"2026-09-30 10:30:00","remind_before":"1d"}`)
	require.Equal(t, 201, res.status, res.raw)
	apptID := uint64(res.body["data"].(map[string]any)["id"].(float64))

	res = e.do(t, "GET", "/api/v1/care/upcoming?hours=168", tok, "fa", "")
	require.Equal(t, 200, res.status, res.raw)
	items := res.body["data"].(map[string]any)["items"].([]any)
	var appts []map[string]any
	last := float64(0)
	for _, it := range items {
		m := it.(map[string]any)
		assert.GreaterOrEqual(t, m["at_ms"].(float64), last, "soonest first")
		last = m["at_ms"].(float64)
		if m["kind"] == care.UpcomingAppointment {
			appts = append(appts, m)
		} else {
			assert.Equal(t, care.UpcomingMedication, m["kind"])
		}
	}
	assert.Greater(t, len(items), len(appts), "doses are still listed")
	require.Len(t, appts, 1)
	a := appts[0]
	assert.Equal(t, fmt.Sprintf("%d:2026-09-30:10:30", apptID), a["id"])
	assert.Equal(t, "2026-09-29 10:30:00", a["at"], "the visit − remind_before")
	assert.Equal(t, "یادآور نوبت پزشک", a["title"])
	assert.Equal(t, "سونوگرافی NT · فردا ساعت 10:30", a["body"])
	assert.Equal(t, "/fa/reminders", a["url"])

	_, err := e.db.Exec(`UPDATE notification_preferences SET categories = '{"appointments":false}' WHERE user_id = ?`, uid)
	require.NoError(t, err)
	res = e.do(t, "GET", "/api/v1/care/upcoming?hours=168", tok, "fa", "")
	require.Equal(t, 200, res.status, res.raw)
	for _, it := range res.body["data"].(map[string]any)["items"].([]any) {
		assert.Equal(t, care.UpcomingMedication, it.(map[string]any)["kind"], "«نوبت پزشک» is off")
	}
}
