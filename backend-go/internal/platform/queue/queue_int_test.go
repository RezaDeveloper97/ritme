package queue_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/metrics"
	"github.com/ritme/backend-go/internal/platform/queue"
)

// The in-process asynq worker on a real Redis: a job that fails once is retried after the
// first backoff step and then succeeds.
func TestAsyncWorker_RetriesThenSucceeds(t *testing.T) {
	rdb, name := realRedis(t)

	q := queue.New(rdb, queue.Options{Name: name, Mode: queue.ModeAsync, Concurrency: 1})
	var runs atomic.Int32
	done := make(chan struct{})
	q.Register(queue.Job{Type: "flaky", MaxTries: 3, Backoff: []time.Duration{time.Second}}, func(context.Context, []byte) error {
		if runs.Add(1) == 1 {
			return errors.New("gateway down")
		}
		close(done)
		return nil
	})
	require.NoError(t, q.Start())
	t.Cleanup(q.Shutdown)
	require.NoError(t, q.Dispatch(context.Background(), "flaky", map[string]string{"x": "y"}))

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("job was not retried")
	}
	assert.Equal(t, int32(2), runs.Load())
}

// realRedis is a client on the test Redis and a fresh base queue name (its keys are deleted afterwards).
func realRedis(t *testing.T) (*redis.Client, string) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR unset")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "ritme-go-test-" + hex.EncodeToString(suffix)
	t.Cleanup(func() {
		ins := asynq.NewInspectorFromRedisClient(rdb)
		for _, p := range []queue.Priority{queue.Critical, queue.Default, queue.Low} {
			_ = ins.DeleteQueue(queue.NameFor(name, p), true)
		}
		_ = rdb.Del(context.Background(), name+":scheduler:leader").Err()
		_ = rdb.Close()
	})
	return rdb, name
}

// DispatchIn: the worker does not pick the task up before its time.
func TestAsyncWorker_DispatchInNotProcessedEarly(t *testing.T) {
	rdb, name := realRedis(t)
	q := queue.New(rdb, queue.Options{Name: name, Mode: queue.ModeAsync, Concurrency: 1})
	ran := make(chan time.Time, 1)
	q.Register(queue.Job{Type: "later", Priority: queue.Low}, func(context.Context, []byte) error {
		ran <- time.Now()
		return nil
	})
	require.NoError(t, q.Start())
	t.Cleanup(q.Shutdown)
	at := time.Now()
	require.NoError(t, q.DispatchIn(context.Background(), 3*time.Second, "later", nil))
	select {
	case got := <-ran:
		assert.GreaterOrEqual(t, got.Sub(at), 3*time.Second, "processed early")
	case <-time.After(20 * time.Second):
		t.Fatal("delayed job never ran")
	}
}

// gauge reads asynq_queue_size{queue,state} from the process registry.
func gauge(t *testing.T, queueName, state string) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != "asynq_queue_size" {
			continue
		}
		for _, m := range mf.GetMetric() {
			l := map[string]string{}
			for _, lp := range m.GetLabel() {
				l[lp.GetName()] = lp.GetValue()
			}
			if l["queue"] == queueName && l["state"] == state {
				return m.GetGauge().GetValue()
			}
		}
	}
	return -1
}

func counter(t *testing.T, name, queueName string) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "queue" && lp.GetValue() == queueName {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

// A task that always fails is retried (MaxTries), then archived: the job's OnFailed fires once and the archived gauge
// and counter move.
func TestAsyncWorker_FailingTaskRetriesThenArchives(t *testing.T) {
	rdb, name := realRedis(t)
	q := queue.New(rdb, queue.Options{Name: name, Mode: queue.ModeAsync, Concurrency: 1})
	var runs, failed atomic.Int32
	q.Register(queue.Job{
		Type: "doomed", Priority: queue.Critical, MaxTries: 2, Backoff: []time.Duration{time.Second},
		OnFailed: func(context.Context, []byte, error) { failed.Add(1) },
	}, func(context.Context, []byte) error {
		runs.Add(1)
		return errors.New("gateway down")
	})
	critical := queue.NameFor(name, queue.Critical)
	require.NoError(t, q.Start())
	t.Cleanup(q.Shutdown)
	before := counter(t, "asynq_archived_total", critical)
	require.NoError(t, q.Dispatch(context.Background(), "doomed", map[string]string{"x": "y"}))

	require.Eventually(t, func() bool { return gauge(t, critical, "archived") == 1 }, 20*time.Second, 100*time.Millisecond)
	assert.Equal(t, int32(2), runs.Load(), "first try + one retry")
	assert.Equal(t, int32(1), failed.Load())
	assert.InDelta(t, before+1, counter(t, "asynq_archived_total", critical), 0.001)
	assert.InDelta(t, 0, gauge(t, critical, "pending"), 0.001)
}

// Two processes with workers on one Redis: the scheduled entry runs once per tick, not once per process.
func TestScheduler_TwoProcessesOneRunPerTick(t *testing.T) {
	rdb, name := realRedis(t)
	var runs atomic.Int32
	newQ := func() *queue.Queue {
		q := queue.New(rdb, queue.Options{Name: name, Mode: queue.ModeAsync, Concurrency: 2, LeaderTTL: 900 * time.Millisecond})
		q.Register(queue.Job{Type: "tick", Priority: queue.Low}, func(context.Context, []byte) error {
			runs.Add(1)
			return nil
		})
		require.NoError(t, q.Scheduler().Register("@every 1s", "tick", nil))
		return q
	}
	a, b := newQ(), newQ()
	require.NoError(t, a.Start())
	require.NoError(t, b.Start())
	t.Cleanup(func() { a.Shutdown(); b.Shutdown() })
	require.Eventually(t, func() bool { return a.Scheduler().Leader() || b.Scheduler().Leader() }, 3*time.Second, 20*time.Millisecond)
	start := runs.Load()
	time.Sleep(4 * time.Second)
	n := runs.Load() - start
	assert.GreaterOrEqual(t, n, int32(3))
	assert.LessOrEqual(t, n, int32(5), "two schedulers would double the runs")
	assert.NotEqual(t, a.Scheduler().Leader(), b.Scheduler().Leader(), "exactly one leader")
}
