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
