package care_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/care"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// T-M12-22 — a dose reminder and an inbox row reach each browser exactly once: under concurrent ticks of several
// processes (API + worker), after a passing failure of the push service (the claiming process retries it on its next
// ticks, a bounded number of times) and after a refusal for good (404 / 410 drops the subscription, never retried).

// morningDose is one 08:00 dose a day from 2026-09-01.
const morningDose = `{"title":"آهن","dose":"1","unit":"mg","form":"tablet","times":["08:00"],"starts_on":"2026-09-01"}`

// doseAt is the moment the 08:00 dose of 2026-09-24 is due (Tehran), plus the half-minute a tick lags.
var doseAt = time.Date(2026, 9, 24, 8, 0, 30, 0, civildate.Tehran)

func doseTag(medID uint64) string { return fmt.Sprintf("%d:2026-09-24:08:00", medID) }

// claims is how many reminder_pushes rows hold the dose (1 = handled, 0 = free for a later tick).
func (e *env) claims(t *testing.T, medID uint64) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM reminder_pushes WHERE reminder_id = ?`, medID).Scan(&n))
	return n
}

func (e *env) subscriptions(t *testing.T, userID uint64) int {
	t.Helper()
	var n int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE user_id = ?`, userID).Scan(&n))
	return n
}

// pushUser is a user with notification settings (no quiet hours, real text) and one browser.
func (e *env) pushUser(t *testing.T, mobile, endpoint string) (uint64, string) {
	t.Helper()
	uid, tok := e.user(t, mobile)
	e.prefs(t, uid, `{}`, false, false)
	e.subscribe(t, uid, endpoint, "fa")
	return uid, tok
}

func TestPush_ConcurrentTicksDeliverEachDoseAndNoticeOnce(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	var tags []string
	for i := range 3 {
		uid, tok := e.pushUser(t, fmt.Sprintf("0912000020%d", i), fmt.Sprintf("https://fcm.googleapis.com/fcm/send/c%d", i))
		tags = append(tags, doseTag(e.create(t, tok, morningDose)))
		for range 2 {
			tags = append(tags, fmt.Sprintf("n:%d", e.notice(t, uid, "learning", "/learn", doseAt.Add(-time.Minute), false)))
		}
	}

	f := &fakePush{delay: 5 * time.Millisecond}
	workers := []*care.Pusher{e.pusher(f), e.pusher(f)} // two processes sharing the database
	round := func(now time.Time) (doses, notices int) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		start := make(chan struct{})
		run := func(tick func(context.Context, time.Time) (int, error), total *int) {
			defer wg.Done()
			<-start
			n, err := tick(ctx, now)
			assert.NoError(t, err)
			mu.Lock()
			*total += n
			mu.Unlock()
		}
		for _, p := range workers {
			for range 2 { // and two overlapping ticks inside each process
				wg.Add(2)
				go run(p.Tick, &doses)
				go run(p.TickNotices, &notices)
			}
		}
		close(start)
		wg.Wait()
		return doses, notices
	}

	doses, notices := round(doseAt)
	assert.Equal(t, 3, doses, "each due dose is announced by one tick")
	assert.Equal(t, 6, notices, "each inbox row is announced by one tick")
	for _, tag := range tags {
		assert.Equal(t, 1, f.attempts("", tag), "%s is pushed exactly once", tag)
	}
	assert.Len(t, f.all(), len(tags))

	// The next round of the same workers finds nothing left.
	doses, notices = round(doseAt.Add(care.PushEvery))
	assert.Zero(t, doses)
	assert.Zero(t, notices)
	assert.Len(t, f.all(), len(tags), "no second push")
}

// transient are the answers of a push service that may work on the next try.
var transient = map[string]sendResult{
	"503":     {status: 503},
	"500":     {status: 500},
	"429":     {status: 429},
	"timeout": {err: context.DeadlineExceeded},
}

func TestTick_PassingFailureIsRetriedOnALaterTickAndDeliveredOnce(t *testing.T) {
	for name, fail := range transient {
		t.Run(name, func(t *testing.T) {
			e := setup(t)
			ctx := context.Background()
			const ep = "https://fcm.googleapis.com/fcm/send/flaky"
			_, tok := e.pushUser(t, "09120000210", ep)
			med := e.create(t, tok, morningDose)
			f := &fakePush{script: map[string][]sendResult{ep: {fail}}}
			p := e.pusher(f)

			sent, err := p.Tick(ctx, doseAt)
			require.NoError(t, err)
			assert.Zero(t, sent)
			assert.Equal(t, 1, f.attempts(ep, doseTag(med)))
			assert.Equal(t, 1, e.claims(t, med), "still claimed: the process that claimed it retries")
			other, err := e.pusher(f).Tick(ctx, doseAt.Add(care.PushEvery/2))
			require.NoError(t, err)
			assert.Zero(t, other, "another process never takes a claimed dose")

			sent, err = p.Tick(ctx, doseAt.Add(care.PushEvery))
			require.NoError(t, err)
			assert.Equal(t, 1, sent, "retried and delivered")
			assert.Equal(t, 2, f.attempts(ep, doseTag(med)))
			assert.Equal(t, 1, e.claims(t, med))

			sent, err = p.Tick(ctx, doseAt.Add(2*care.PushEvery))
			require.NoError(t, err)
			assert.Zero(t, sent)
			assert.Equal(t, 2, f.attempts(ep, doseTag(med)), "delivered once: never again")
		})
	}
}

func TestTickNotices_PassingFailureIsRetriedOnALaterTickAndDeliveredOnce(t *testing.T) {
	for name, fail := range transient {
		t.Run(name, func(t *testing.T) {
			e := setup(t)
			ctx := context.Background()
			const ep = "https://fcm.googleapis.com/fcm/send/flaky"
			uid, _ := e.pushUser(t, "09120000211", ep)
			id := e.notice(t, uid, "learning", "/learn", doseAt.Add(-time.Minute), false)
			tag := fmt.Sprintf("n:%d", id)
			f := &fakePush{script: map[string][]sendResult{ep: {fail}}}
			p := e.pusher(f)

			sent, err := p.TickNotices(ctx, doseAt)
			require.NoError(t, err)
			assert.Zero(t, sent)
			assert.True(t, e.handled(t, id), "still claimed: the process that claimed it retries")
			other, err := e.pusher(f).TickNotices(ctx, doseAt.Add(care.PushEvery/2))
			require.NoError(t, err)
			assert.Zero(t, other, "another process never takes a claimed row")

			sent, err = p.TickNotices(ctx, doseAt.Add(care.PushEvery))
			require.NoError(t, err)
			assert.Equal(t, 1, sent, "retried and delivered")
			assert.True(t, e.handled(t, id))

			sent, err = p.TickNotices(ctx, doseAt.Add(2*care.PushEvery))
			require.NoError(t, err)
			assert.Zero(t, sent)
			assert.Equal(t, 2, f.attempts(ep, tag), "delivered once: never again")
		})
	}
}

// A push service that keeps failing is tried PushAttempts times, once per tick, then the dose / row stays claimed and
// silent: no retry storm for the rest of the lookback window.
func TestPush_PassingFailureRetriesAreBounded(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	const ep = "https://fcm.googleapis.com/fcm/send/down"
	uid, tok := e.pushUser(t, "09120000212", ep)
	med := e.create(t, tok, morningDose)
	id := e.notice(t, uid, "learning", "/learn", doseAt.Add(-time.Minute), false)
	down := make([]sendResult, 100)
	for i := range down {
		down[i] = sendResult{status: 503}
	}
	f := &fakePush{script: map[string][]sendResult{ep: down}}
	p := e.pusher(f)

	for i := range int(care.PushLookback / care.PushEvery) { // every tick of the whole dose window
		now := doseAt.Add(time.Duration(i) * care.PushEvery)
		sent, err := p.Tick(ctx, now)
		require.NoError(t, err)
		assert.Zero(t, sent)
		sent, err = p.TickNotices(ctx, now)
		require.NoError(t, err)
		assert.Zero(t, sent)
	}
	assert.Equal(t, care.PushAttempts, f.attempts(ep, doseTag(med)))
	assert.Equal(t, care.PushAttempts, f.attempts(ep, fmt.Sprintf("n:%d", id)))
	assert.Equal(t, 1, e.claims(t, med), "given up: claimed and silent")
	assert.True(t, e.handled(t, id), "given up: claimed and silent")
	assert.Equal(t, 1, e.subscriptions(t, uid), "a passing failure keeps the browser")
}

// A refusal for good is never retried: 404 / 410 drop the subscription, any other 4xx keeps it.
func TestPush_PermanentRefusalIsNotRetried(t *testing.T) {
	for _, tc := range []struct {
		status   int
		keepsSub bool
	}{{410, false}, {404, false}, {400, true}, {403, true}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			e := setup(t)
			ctx := context.Background()
			// One user per channel, so the dose's 410 does not leave the inbox row without a browser.
			const epDose, epNotice = "https://fcm.googleapis.com/fcm/send/refused-dose", "https://fcm.googleapis.com/fcm/send/refused-inbox"
			doseUser, tok := e.pushUser(t, "09120000213", epDose)
			med := e.create(t, tok, morningDose)
			inboxUser, _ := e.pushUser(t, "09120000216", epNotice)
			id := e.notice(t, inboxUser, "learning", "/learn", doseAt.Add(-time.Minute), false)
			f := &fakePush{status: map[string]int{epDose: tc.status, epNotice: tc.status}}
			p := e.pusher(f)

			for i := range 4 {
				now := doseAt.Add(time.Duration(i) * care.PushEvery)
				sent, err := p.Tick(ctx, now)
				require.NoError(t, err)
				assert.Zero(t, sent)
				sent, err = p.TickNotices(ctx, now)
				require.NoError(t, err)
				assert.Zero(t, sent)
			}
			assert.Equal(t, 1, f.attempts(epDose, doseTag(med)), "one try")
			assert.Equal(t, 1, f.attempts(epNotice, fmt.Sprintf("n:%d", id)), "one try")
			assert.Equal(t, 1, e.claims(t, med))
			assert.True(t, e.handled(t, id))
			want := 0
			if tc.keepsSub {
				want = 1
			}
			assert.Equal(t, want, e.subscriptions(t, doseUser), "404 / 410 drop the browser, other refusals keep it")
			assert.Equal(t, want, e.subscriptions(t, inboxUser), "404 / 410 drop the browser, other refusals keep it")
		})
	}
}

// One browser accepted the push, another failed for the moment: the reminder counts as delivered and is not
// retried, so the browser that has it never gets it twice.
func TestPush_OneBrowserAcceptedIsDelivered(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	const ok, flaky = "https://fcm.googleapis.com/fcm/send/ok", "https://web.push.apple.com/flaky"
	uid, tok := e.pushUser(t, "09120000214", ok)
	e.subscribe(t, uid, flaky, "en")
	med := e.create(t, tok, morningDose)
	id := e.notice(t, uid, "learning", "/learn", doseAt.Add(-time.Minute), false)
	f := &fakePush{script: map[string][]sendResult{flaky: {{status: 503}, {status: 503}}}}
	p := e.pusher(f)

	for i := range 3 {
		now := doseAt.Add(time.Duration(i) * care.PushEvery)
		_, err := p.Tick(ctx, now)
		require.NoError(t, err)
		_, err = p.TickNotices(ctx, now)
		require.NoError(t, err)
	}
	for _, tag := range []string{doseTag(med), fmt.Sprintf("n:%d", id)} {
		assert.Equal(t, 1, f.attempts(ok, tag), "%s: once on the browser that accepted it", tag)
		assert.Equal(t, 1, f.attempts(flaky, tag), "%s: not retried", tag)
	}
}

// The two channels are separate by design (docs/care-reminders/README.md, «Web Push for the whole inbox»): a dose is
// announced by Tick only — it never writes an inbox row — and an inbox row by TickNotices only, each with its own
// tag, so a dose and an inbox row about the same medication are two pushes, each exactly once; the «داروها» switch
// silences the dose, not the inbox row (an untyped kind has no switch of its own).
func TestPush_DoseAndInboxRowAreSeparateChannels(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	const ep = "https://fcm.googleapis.com/fcm/send/both"
	uid, tok := e.pushUser(t, "09120000215", ep)
	med := e.create(t, tok, morningDose)
	id := e.notice(t, uid, "message_push", "/reminders", doseAt.Add(-time.Minute), false)
	var inbox int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, uid).Scan(&inbox))

	f := &fakePush{}
	p := e.pusher(f)
	for i := range 2 {
		now := doseAt.Add(time.Duration(i) * care.PushEvery)
		_, err := p.Tick(ctx, now)
		require.NoError(t, err)
		_, err = p.TickNotices(ctx, now)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, f.attempts(ep, doseTag(med)), "the dose, once")
	assert.Equal(t, 1, f.attempts(ep, fmt.Sprintf("n:%d", id)), "the inbox row, once")
	assert.Len(t, f.all(), 2)
	var after int
	require.NoError(t, e.db.QueryRow(`SELECT COUNT(*) FROM user_notifications WHERE user_id = ?`, uid).Scan(&after))
	assert.Equal(t, inbox, after, "a dose push writes no inbox row")

	// «داروها» off: the next day's dose is silent, a new inbox row still goes.
	_, err := e.db.Exec(`UPDATE notification_preferences SET categories = '{"medications":false}' WHERE user_id = ?`, uid)
	require.NoError(t, err)
	next := doseAt.AddDate(0, 0, 1)
	id2 := e.notice(t, uid, "message_push", "/reminders", next.Add(-time.Minute), false)
	sent, err := e.pusher(f).Tick(ctx, next)
	require.NoError(t, err)
	assert.Zero(t, sent)
	sent, err = e.pusher(f).TickNotices(ctx, next)
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	assert.Equal(t, 1, f.attempts(ep, fmt.Sprintf("n:%d", id2)))
	assert.Len(t, f.all(), 3)
}
