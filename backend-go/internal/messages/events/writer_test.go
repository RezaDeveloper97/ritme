package events_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/messages/events"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// fakeIns records the rows of every batch; block (when set) holds each insert until it is closed.
type fakeIns struct {
	mu    sync.Mutex
	rows  int
	calls int
	block chan struct{}
}

func (f *fakeIns) InsertMessageEventsBatch(_ context.Context, arg store.InsertMessageEventsBatchParams) error {
	if f.block != nil {
		<-f.block
	}
	var rows [][]any
	if err := json.Unmarshal([]byte(arg.Batch), &rows); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows += len(rows)
	f.calls++
	return nil
}

func (f *fakeIns) count() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rows, f.calls
}

func ev(i int) events.Event {
	return events.Event{RuleID: uint64(i + 1), UserID: 7, Version: 1, Variant: "a", Slot: "smart_tip", Event: events.Shown, //nolint:gosec // small
		At: time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)}
}

func fallbacks(t *testing.T) float64 {
	t.Helper()
	mfs, err := metrics.Registry().Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == "messages_events_sync_fallback_total" {
			return mf.GetMetric()[0].GetCounter().GetValue()
		}
	}
	return 0
}

// Shutdown drains the buffer: nothing queued is lost, and a write after Close is inserted synchronously.
func TestWriter_CloseDrains(t *testing.T) {
	ins := &fakeIns{}
	w := events.NewWriter(ins, nil, events.WithFlushEvery(time.Hour), events.WithBatch(10000))
	for i := range 1234 {
		w.Write(context.Background(), ev(i))
	}
	rows, _ := ins.count()
	assert.Zero(t, rows, "nothing flushed before the interval or the batch size")
	w.Close(context.Background())
	rows, calls := ins.count()
	assert.Equal(t, 1234, rows, "Close drained every queued row")
	assert.Equal(t, 13, calls, "multi-row inserts of at most 100 rows")
	before := fallbacks(t)
	w.Write(context.Background(), ev(1))
	rows, _ = ins.count()
	assert.Equal(t, 1235, rows, "a write after Close is synchronous")
	assert.InDelta(t, before+1, fallbacks(t), 0)
	w.Close(context.Background()) // idempotent
}

// A full buffer takes the synchronous path (never drops, counted).
func TestWriter_FullBufferIsSynchronous(t *testing.T) {
	ins := &fakeIns{block: make(chan struct{})}
	w := events.NewWriter(ins, nil, events.WithBuffer(4), events.WithBatch(1), events.WithFlushEvery(time.Hour))
	before := fallbacks(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The loop takes one row and blocks in its insert; 4 more fill the channel; the rest spill synchronously
		// (and block too, until the inserter is released).
		for i := range 10 {
			w.Write(context.Background(), ev(i))
		}
	}()
	require.Eventually(t, func() bool { return fallbacks(t) > before }, 2*time.Second, 5*time.Millisecond)
	close(ins.block)
	<-done
	w.Close(context.Background())
	rows, _ := ins.count()
	assert.Equal(t, 10, rows, "no row dropped")
	assert.GreaterOrEqual(t, fallbacks(t)-before, 1.0)
}

// The batch flushes on its size and on Flush.
func TestWriter_BatchAndFlush(t *testing.T) {
	ins := &fakeIns{}
	w := events.NewWriter(ins, nil, events.WithBatch(3), events.WithFlushEvery(time.Hour))
	defer w.Close(context.Background())
	for i := range 3 {
		w.Write(context.Background(), ev(i))
	}
	require.Eventually(t, func() bool { r, _ := ins.count(); return r == 3 }, time.Second, 5*time.Millisecond)
	w.Write(context.Background(), ev(4))
	w.Flush(context.Background())
	rows, _ := ins.count()
	assert.Equal(t, 4, rows)
}

func TestEncode(t *testing.T) {
	e := ev(0)
	e.Slot, e.Version, e.Variant = "", 0, `a"b`
	assert.Equal(t, `[[1,7,0,"a\"b","","impression","2026-10-21 13:30:00"]]`, events.Encode([]events.Event{e}),
		"Tehran wall clock; quotes escaped")
}
