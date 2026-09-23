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

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/platform/queue"
)

// The in-process asynq worker on a real Redis: a job that fails once is retried after the
// first backoff step and then succeeds.
func TestAsyncWorker_RetriesThenSucceeds(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR unset")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "ritme-go-test-" + hex.EncodeToString(suffix)

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
