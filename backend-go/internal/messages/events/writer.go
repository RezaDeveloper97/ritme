// Package events is the message event log v6 (T-M10-06, D-163; SPEC §5.4 step 7, §5.6; DECISIONS.md M-12, M-13,
// M-17, M-19): the buffered writer every message event goes through, the once-a-day marks of server-logged `shown` and
// `eligible`, the server-matched goals and the rollup / retention jobs.
//
// Stored event names keep the pre-v6 spelling (M-01): shown = `impression`, clicked = `click`, dismissed = `dismiss`;
// `goal` and `eligible` are the same in both.
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Stored event names.
const (
	Shown     = "impression" // = rules.EventShown
	Dismissed = "dismiss"    // = rules.EventDismissed
	Clicked   = "click"
	GoalEvent = "goal"
	Eligible  = "eligible"
)

// Writer defaults (T-M10-06 scope 3).
const (
	DefaultBuffer     = 10000
	DefaultBatch      = 500
	DefaultFlushEvery = time.Second
	// chunk is the most rows one INSERT carries (InsertMessageEventsBatch walks its JSON array by index).
	chunk = 100
)

// Event is one row of message_events. Version 0 and Slot "" are stored as NULL (a legacy client event).
type Event struct {
	RuleID  uint64
	UserID  uint64
	Version int
	Variant string
	Slot    string
	Event   string
	At      time.Time
}

var (
	flushSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "messages_events_flush_seconds",
		Help:    "Duration of one flush of the message events buffer (every chunk of the batch).",
		Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
	})
	syncFallbacks = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_events_sync_fallback_total",
		Help: "Message events inserted synchronously because the buffer was full or the writer closed.",
	})
	writeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_events_write_errors_total",
		Help: "Message event rows that could not be inserted after a retry (logged, counted).",
	})
	depthMu sync.Mutex
	writers = map[*Writer]struct{}{}
)

func init() {
	metrics.Registry().MustRegister(flushSeconds, syncFallbacks, writeErrors,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "messages_events_queue_depth",
			Help: "Message events waiting in the buffer of this process.",
		}, func() float64 {
			depthMu.Lock()
			defer depthMu.Unlock()
			n := 0
			for w := range writers {
				n += len(w.ch)
			}
			return float64(n)
		}))
}

// Inserter is the multi-row insert (store.Queries).
type Inserter interface {
	InsertMessageEventsBatch(ctx context.Context, arg store.InsertMessageEventsBatchParams) error
}

// Writer buffers events in a channel and inserts them in multi-row INSERTs every FlushEvery or Batch rows. A full
// buffer (or a closed writer) inserts synchronously, so no event is ever dropped for lack of room; Close drains
// everything still queued. The flush loop starts on the first write.
type Writer struct {
	ins        Inserter
	logger     *slog.Logger
	ch         chan Event
	batch      int
	flushEvery time.Duration

	mu      sync.RWMutex // Write holds R while it sends; Close takes W to flip closed
	closed  bool
	start   sync.Once
	running atomic.Bool
	stop    chan struct{}
	done    chan struct{}
	flushes chan chan struct{}
}

// Option tunes a Writer.
type Option func(*Writer)

// WithBuffer sets the channel size.
func WithBuffer(n int) Option { return func(w *Writer) { w.ch = make(chan Event, max(n, 1)) } }

// WithBatch sets the rows that trigger a flush.
func WithBatch(n int) Option { return func(w *Writer) { w.batch = max(n, 1) } }

// WithFlushEvery sets the flush interval.
func WithFlushEvery(d time.Duration) Option { return func(w *Writer) { w.flushEvery = d } }

// NewWriter builds a writer over ins (logger nil = slog.Default()).
func NewWriter(ins Inserter, logger *slog.Logger, opts ...Option) *Writer {
	if logger == nil {
		logger = slog.Default()
	}
	w := &Writer{ins: ins, logger: logger, ch: make(chan Event, DefaultBuffer), batch: DefaultBatch,
		flushEvery: DefaultFlushEvery, stop: make(chan struct{}), done: make(chan struct{}),
		flushes: make(chan chan struct{})}
	for _, o := range opts {
		o(w)
	}
	depthMu.Lock()
	writers[w] = struct{}{}
	depthMu.Unlock()
	return w
}

// Write queues evs. It never blocks on a full buffer and never drops: what does not fit is inserted now (counted in
// messages_events_sync_fallback_total).
func (w *Writer) Write(ctx context.Context, evs ...Event) {
	if len(evs) == 0 {
		return
	}
	w.startLoop()
	var spill []Event
	w.mu.RLock()
	if w.closed {
		spill = evs
	} else {
		for i, ev := range evs {
			select {
			case w.ch <- ev:
			default:
				spill = evs[i:]
			}
			if spill != nil {
				break
			}
		}
	}
	w.mu.RUnlock()
	if len(spill) > 0 {
		syncFallbacks.Add(float64(len(spill)))
		sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		w.insert(sctx, spill)
	}
}

// Flush inserts everything queued so far and returns when it is written (tests, and callers that must read their
// own events back).
func (w *Writer) Flush(ctx context.Context) {
	w.startLoop()
	if !w.running.Load() {
		return // closed before it ever ran: nothing can be queued
	}
	reply := make(chan struct{})
	select {
	case w.flushes <- reply:
	case <-w.done:
		return
	case <-ctx.Done():
		return
	}
	select {
	case <-reply:
	case <-ctx.Done():
	}
}

// Close stops the loop after draining the buffer (lifecycle shutdown) and waits for it (or ctx). Later writes are
// inserted synchronously. Idempotent.
func (w *Writer) Close(ctx context.Context) {
	w.mu.Lock()
	already := w.closed
	w.closed = true
	w.mu.Unlock()
	if already {
		return
	}
	w.start.Do(func() {}) // no loop starts after Close
	if w.running.Load() {
		close(w.stop)
		select {
		case <-w.done:
		case <-ctx.Done():
		}
	}
	depthMu.Lock()
	delete(writers, w)
	depthMu.Unlock()
}

func (w *Writer) startLoop() {
	w.start.Do(func() {
		w.running.Store(true)
		go w.loop()
	})
}

func (w *Writer) loop() {
	defer close(w.done)
	t := time.NewTicker(w.flushEvery)
	defer t.Stop()
	buf := make([]Event, 0, w.batch)
	flush := func() {
		if len(buf) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			w.insert(ctx, buf)
			cancel()
			buf = buf[:0]
		}
	}
	drain := func() {
		for {
			select {
			case ev := <-w.ch:
				buf = append(buf, ev)
				if len(buf) >= w.batch {
					flush()
				}
			default:
				flush()
				return
			}
		}
	}
	for {
		select {
		case ev := <-w.ch:
			buf = append(buf, ev)
			if len(buf) >= w.batch {
				flush()
			}
		case <-t.C:
			flush()
		case reply := <-w.flushes:
			drain()
			close(reply)
		case <-w.stop:
			drain()
			return
		}
	}
}

// insert writes evs in chunks; a failed chunk is retried once, then logged and counted.
func (w *Writer) insert(ctx context.Context, evs []Event) {
	if len(evs) == 0 {
		return
	}
	began := time.Now()
	defer func() { flushSeconds.Observe(time.Since(began).Seconds()) }()
	for i := 0; i < len(evs); i += chunk {
		part := evs[i:min(i+chunk, len(evs))]
		batch := Encode(part)
		err := w.ins.InsertMessageEventsBatch(ctx, store.InsertMessageEventsBatchParams{Batch: batch})
		if err != nil {
			err = w.ins.InsertMessageEventsBatch(ctx, store.InsertMessageEventsBatchParams{Batch: batch})
		}
		if err != nil {
			writeErrors.Add(float64(len(part)))
			w.logger.ErrorContext(ctx, "messages: events insert failed", "rows", len(part), "error", err.Error())
		}
	}
}

// Encode is the JSON array InsertMessageEventsBatch reads: [rule_id, user_id, version, variant, slot, event,
// created_at] per row, created_at as the Tehran wall clock (DB timestamps are Tehran).
func Encode(evs []Event) string {
	b := make([]byte, 0, 96*len(evs)+2)
	b = append(b, '[')
	for i, ev := range evs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '[')
		b = strconv.AppendUint(b, ev.RuleID, 10)
		b = append(b, ',')
		b = strconv.AppendUint(b, ev.UserID, 10)
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(max(ev.Version, 0)), 10)
		b = append(b, ',')
		b = appendString(b, ev.Variant)
		b = append(b, ',')
		b = appendString(b, ev.Slot)
		b = append(b, ',')
		b = appendString(b, ev.Event)
		b = append(b, ',', '"')
		b = ev.At.In(civildate.Tehran).AppendFormat(b, time.DateTime)
		b = append(b, '"', ']')
	}
	return string(append(b, ']'))
}

func appendString(b []byte, s string) []byte {
	q, _ := json.Marshal(s) // never fails for a string
	return append(b, q...)
}
