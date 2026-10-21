package queue_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/queue"
)

func TestRetryDelay_LaravelBackoff(t *testing.T) {
	b := []time.Duration{5 * time.Second, 15 * time.Second}
	// asynq passes the number of retries already done: 0 on the first failure.
	assert.Equal(t, 5*time.Second, queue.RetryDelay(b, 0))
	assert.Equal(t, 15*time.Second, queue.RetryDelay(b, 1))
	assert.Equal(t, 15*time.Second, queue.RetryDelay(b, 7))
	assert.Equal(t, 5*time.Second, queue.RetryDelay(nil, 3))
}

func TestNameFromPrefix(t *testing.T) {
	assert.Equal(t, "ritme-go", queue.NameFromPrefix("ritme-go:"))
	assert.Equal(t, "ritme-go", queue.NameFromPrefix(""))
}

func TestSync_RunsInlineOnceAndReportsFailure(t *testing.T) {
	q := queue.New(nil, queue.Options{Mode: queue.ModeSync})
	var runs, failed atomic.Int32
	q.OnFailed(func(context.Context, string, []byte, error) { failed.Add(1) })
	q.Register(queue.Job{Type: "j", MaxTries: 3}, func(_ context.Context, p []byte) error {
		runs.Add(1)
		if string(p) == `{"fail":true}` {
			return errors.New("boom")
		}
		return nil
	})
	require.NoError(t, q.Dispatch(context.Background(), "j", map[string]bool{"fail": false}))
	require.Error(t, q.Dispatch(context.Background(), "j", map[string]bool{"fail": true}))
	assert.Equal(t, int32(2), runs.Load())
	assert.Equal(t, int32(1), failed.Load())
	require.ErrorIs(t, q.Dispatch(context.Background(), "unknown", nil), queue.ErrUnknownJob)
}

func TestAsync_EnqueuesOnNamedQueueWithRetries(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	q := queue.New(rdb, queue.Options{Name: "ritme-go", Mode: queue.ModeAsync})
	q.Register(queue.Job{Type: "send_otp_sms", MaxTries: 3}, func(context.Context, []byte) error { return nil })
	require.NoError(t, q.Dispatch(context.Background(), "send_otp_sms", map[string]string{"mobile": "x"}))

	pending, err := rdb.LLen(context.Background(), "asynq:{ritme-go}:pending").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), pending)
}

func miniQueue(t *testing.T, opts queue.Options) (*queue.Queue, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	if opts.Name == "" {
		opts.Name = "ritme-go"
	}
	opts.Mode = queue.ModeAsync
	return queue.New(rdb, opts), mr, rdb
}

func noop(context.Context, []byte) error { return nil }

func TestPriorities_ThreeWeightedQueues(t *testing.T) {
	q, _, rdb := miniQueue(t, queue.Options{})
	assert.Equal(t, map[string]int{"ritme-go-critical": 6, "ritme-go": 3, "ritme-go-low": 1}, q.Queues())
	q.Register(queue.Job{Type: "otp", Priority: queue.Critical}, noop)
	q.Register(queue.Job{Type: "mail"}, noop)
	q.Register(queue.Job{Type: "sweep", Priority: queue.Low}, noop)
	ctx := context.Background()
	for _, typ := range []string{"otp", "mail", "sweep"} {
		require.NoError(t, q.Dispatch(ctx, typ, nil))
	}
	for _, name := range []string{"ritme-go-critical", "ritme-go", "ritme-go-low"} {
		n, err := rdb.LLen(ctx, "asynq:{"+name+"}:pending").Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), n, name)
	}
}

func TestUnique_DedupedWithinTTL(t *testing.T) {
	q, mr, rdb := miniQueue(t, queue.Options{})
	q.Register(queue.Job{Type: "snapshot", Priority: queue.Low}, noop)
	ctx := context.Background()
	payload := map[string]int{"user_id": 7}
	require.NoError(t, q.DispatchWith(ctx, "snapshot", payload, queue.Unique(time.Hour)))
	require.ErrorIs(t, q.DispatchWith(ctx, "snapshot", payload, queue.Unique(time.Hour)), queue.ErrDuplicate)
	require.NoError(t, q.DispatchWith(ctx, "snapshot", map[string]int{"user_id": 8}, queue.Unique(time.Hour)),
		"another payload is another task")

	mr.FastForward(time.Hour + time.Second) // the lock expired
	require.NoError(t, q.DispatchWith(ctx, "snapshot", payload, queue.Unique(time.Hour)))
	n, err := rdb.LLen(ctx, "asynq:{ritme-go-low}:pending").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestTaskID_NaturalKeyDeduped(t *testing.T) {
	q, _, _ := miniQueue(t, queue.Options{})
	q.Register(queue.Job{Type: "rollup"}, noop)
	ctx := context.Background()
	require.NoError(t, q.DispatchWith(ctx, "rollup", nil, queue.TaskID("rollup:2026-10-21")))
	require.ErrorIs(t, q.DispatchWith(ctx, "rollup", nil, queue.TaskID("rollup:2026-10-21")), queue.ErrDuplicate)
}

func TestDispatchInAt_GoToTheScheduledSet(t *testing.T) {
	q, _, rdb := miniQueue(t, queue.Options{})
	q.Register(queue.Job{Type: "later"}, noop)
	ctx := context.Background()
	require.NoError(t, q.DispatchIn(ctx, time.Hour, "later", nil))
	require.NoError(t, q.DispatchAt(ctx, time.Now().Add(2*time.Hour), "later", nil))
	pending, err := rdb.LLen(ctx, "asynq:{ritme-go}:pending").Result()
	require.NoError(t, err)
	assert.Zero(t, pending, "nothing is ready before its time")
	scheduled, err := rdb.ZCard(ctx, "asynq:{ritme-go}:scheduled").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(2), scheduled)
	require.ErrorIs(t, q.DispatchIn(ctx, time.Second, "nope", nil), queue.ErrUnknownJob)
}

// QUEUE_CONNECTION=sync: delays and uniqueness are ignored, the job runs inline now; the scheduler fires its entries
// from a local cron (and start-up entries once).
func TestSync_DelayedRunsInlineAndSchedulerFires(t *testing.T) {
	q := queue.New(nil, queue.Options{Mode: queue.ModeSync})
	var runs, ticks, boots atomic.Int32
	q.Register(queue.Job{Type: "later"}, func(context.Context, []byte) error { runs.Add(1); return nil })
	require.NoError(t, q.DispatchIn(context.Background(), time.Hour, "later", nil, queue.Unique(time.Hour)))
	require.NoError(t, q.DispatchIn(context.Background(), time.Hour, "later", nil, queue.Unique(time.Hour)))
	assert.Equal(t, int32(2), runs.Load())

	require.NoError(t, q.RegisterPeriodic(queue.Job{Type: "tick"},
		func(context.Context, []byte) error { ticks.Add(1); return nil }, "@every 1s"))
	require.NoError(t, q.RegisterPeriodic(queue.Job{Type: "boot"},
		func(context.Context, []byte) error { boots.Add(1); return nil }, "0 3 * * *", queue.RunAtStart()))
	require.NoError(t, q.Start())
	t.Cleanup(q.Shutdown)
	assert.True(t, q.Scheduler().Leader())
	assert.Eventually(t, func() bool { return ticks.Load() >= 1 && boots.Load() == 1 }, 3*time.Second, 50*time.Millisecond)
}

func TestScheduler_RejectsBadSpecAndUnknownJob(t *testing.T) {
	q := queue.New(nil, queue.Options{Mode: queue.ModeSync})
	require.ErrorIs(t, q.Scheduler().Register("* * * * *", "missing", nil), queue.ErrUnknownJob)
	q.Register(queue.Job{Type: "j"}, noop)
	require.Error(t, q.Scheduler().Register("every minute", "j", nil))
	require.NoError(t, q.Scheduler().Register("*/10 * * * *", "j", nil))
}

func TestNext_TehranWallClock(t *testing.T) {
	tehran, err := time.LoadLocation("Asia/Tehran")
	require.NoError(t, err)
	after := time.Date(2026, 10, 21, 2, 59, 0, 0, tehran).UTC()
	n, err := queue.Next("0 3 * * *", after)
	require.NoError(t, err)
	assert.True(t, n.Equal(time.Date(2026, 10, 21, 3, 0, 0, 0, tehran)))
}

// Two processes on one Redis: only the leader-key holder schedules, so an entry is enqueued once per tick; when the
// leader shuts down (releasing the key) the other one takes over.
func TestScheduler_LeaderKeyOneSchedulerAtATime(t *testing.T) {
	mr := miniredis.RunT(t)
	newQ := func() *queue.Queue {
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		t.Cleanup(func() { _ = rdb.Close() })
		q := queue.New(rdb, queue.Options{Name: "ritme-go", Mode: queue.ModeAsync, LeaderTTL: 600 * time.Millisecond})
		q.Register(queue.Job{Type: "tick", Priority: queue.Low}, noop)
		require.NoError(t, q.Scheduler().Register("@every 1s", "tick", nil))
		return q
	}
	a, b := newQ(), newQ()
	// Only the schedulers run here (the worker is exercised by the integration tests): Start would also start a
	// worker that drains the pending list we count.
	require.NoError(t, queue.StartSchedulerOnly(a))
	require.NoError(t, queue.StartSchedulerOnly(b))
	t.Cleanup(func() { queue.StopSchedulerOnly(a); queue.StopSchedulerOnly(b) })

	require.Eventually(t, func() bool { return a.Scheduler().Leader() != b.Scheduler().Leader() }, 2*time.Second, 20*time.Millisecond)
	leader, follower := a, b
	if b.Scheduler().Leader() {
		leader, follower = b, a
	}
	assert.True(t, mr.Exists("ritme-go:scheduler:leader"))

	time.Sleep(3200 * time.Millisecond)
	assert.False(t, follower.Scheduler().Leader(), "the follower never schedules while the key is held")
	pending, err := mr.List("asynq:{ritme-go-low}:pending")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(pending), 2)
	assert.LessOrEqual(t, len(pending), 4, "one enqueue per tick, not one per process")

	queue.StopSchedulerOnly(leader)
	assert.False(t, mr.Exists("ritme-go:scheduler:leader"), "the leader released the key")
	assert.Eventually(t, follower.Scheduler().Leader, 2*time.Second, 20*time.Millisecond, "the follower takes over")
}
