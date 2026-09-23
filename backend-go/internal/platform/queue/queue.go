// Package queue runs background jobs on asynq over the shared Redis, with an in-process
// worker (the Go image has no separate `queue` container).
//
// Isolation: asynq cannot prefix its keys, so the Go queue is an asynq *queue name* derived
// from REDIS_PREFIX ("ritme-go:" → queue "ritme-go", keys "asynq:{ritme-go}:…"). Laravel's
// queue (PHP-serialized, `queues:default`) is never read.
//
// Mode "sync" (QUEUE_CONNECTION=sync, like Laravel's sync driver in the contract stack and
// local dev) runs the job inline, once, and returns its error to Dispatch's caller.
//
// Usage:
//
//	q := queue.New(rdb, queue.Options{Name: "ritme-go", Mode: queue.ModeFromEnv(), Logger: log})
//	q.Register(queue.Job{Type: "send_otp_sms", MaxTries: 3, Backoff: []time.Duration{5*time.Second, 15*time.Second}}, handler)
//	q.Start()            // on listen (no-op in sync mode)
//	defer q.Shutdown()   // on shutdown: waits for running jobs
//	err := q.Dispatch(ctx, "send_otp_sms", payload)
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Mode selects how Dispatch runs jobs.
type Mode string

const (
	// ModeAsync enqueues on Redis for the in-process asynq worker.
	ModeAsync Mode = "redis"
	// ModeSync runs the job inline in Dispatch (one try).
	ModeSync Mode = "sync"
)

// ModeFromEnv reads QUEUE_CONNECTION: "sync" → ModeSync, anything else → ModeAsync.
func ModeFromEnv() Mode {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("QUEUE_CONNECTION")), "sync") {
		return ModeSync
	}
	return ModeAsync
}

// NameFromPrefix turns a Redis key prefix into the asynq queue name ("ritme-go:" → "ritme-go").
func NameFromPrefix(prefix string) string {
	name := strings.Trim(prefix, ":")
	if name == "" {
		return "ritme-go"
	}
	return name
}

// Handler processes one job payload (the JSON Dispatch encoded). Returning an error fails
// this try; the job is retried until MaxTries is reached.
type Handler func(ctx context.Context, payload []byte) error

// Job describes a job type.
type Job struct {
	Type     string
	MaxTries int             // total tries incl. the first ($tries); < 1 means 1
	Backoff  []time.Duration // delay before retry n (1-based) = Backoff[min(n, len)-1] ($backoff)
	Timeout  time.Duration   // per try; 0 = 60s (queue:work --timeout default)
}

// OnFailed is called once when a job exhausted its tries (the job's failed() hook).
type OnFailed func(ctx context.Context, jobType string, payload []byte, err error)

// Options configure a Queue.
type Options struct {
	Name        string // asynq queue name (NameFromPrefix(REDIS_PREFIX))
	Mode        Mode
	Concurrency int // worker goroutines; 0 = 4
	Logger      *slog.Logger
}

// Queue dispatches and (in async mode) processes jobs.
type Queue struct {
	opts     Options
	rdb      *redis.Client
	client   *asynq.Client
	logger   *slog.Logger
	mu       sync.Mutex
	jobs     map[string]registered
	server   *asynq.Server
	started  bool
	onFailed OnFailed
}

type registered struct {
	job     Job
	handler Handler
}

// New builds a queue on rdb (shared connection; asynq never closes it). rdb may be nil
// only in ModeSync.
func New(rdb *redis.Client, opts Options) *Queue {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Name == "" {
		opts.Name = NameFromPrefix("")
	}
	if rdb == nil {
		opts.Mode = ModeSync
	}
	q := &Queue{opts: opts, rdb: rdb, logger: opts.Logger, jobs: map[string]registered{}}
	if opts.Mode != ModeSync {
		q.client = asynq.NewClientFromRedisClient(rdb)
	}
	return q
}

// Mode returns the effective mode.
func (q *Queue) Mode() Mode { return q.opts.Mode }

// OnFailed sets the exhausted-tries hook (both modes).
func (q *Queue) OnFailed(fn OnFailed) { q.onFailed = fn }

// Register adds a job type. Register all jobs before Start.
func (q *Queue) Register(job Job, h Handler) {
	if job.Type == "" || h == nil {
		panic("queue: Register needs a job type and a handler")
	}
	if job.MaxTries < 1 {
		job.MaxTries = 1
	}
	if job.Timeout <= 0 {
		job.Timeout = 60 * time.Second
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, dup := q.jobs[job.Type]; dup {
		panic(fmt.Sprintf("queue: job %q registered twice", job.Type))
	}
	q.jobs[job.Type] = registered{job: job, handler: h}
}

// ErrUnknownJob is returned by Dispatch for an unregistered job type.
var ErrUnknownJob = errors.New("queue: unknown job type")

// Dispatch encodes payload as JSON and enqueues it (async) or runs it inline (sync).
func (q *Queue) Dispatch(ctx context.Context, jobType string, payload any) error {
	q.mu.Lock()
	reg, ok := q.jobs[jobType]
	q.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownJob, jobType)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("queue: encode %s: %w", jobType, err)
	}
	if q.opts.Mode == ModeSync {
		if err := reg.handler(ctx, body); err != nil {
			q.failed(ctx, jobType, body, err)
			return fmt.Errorf("queue: %s: %w", jobType, err)
		}
		return nil
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(jobType, body),
		asynq.Queue(q.opts.Name),
		asynq.MaxRetry(reg.job.MaxTries-1),
		asynq.Timeout(reg.job.Timeout),
	)
	if err != nil {
		return fmt.Errorf("queue: enqueue %s: %w", jobType, err)
	}
	return nil
}

// RetryDelay is the delay before retry n (asynq passes the number of retries done so far,
// 0 on the first failure): Backoff[min(n, len-1)], or 5s when no backoff is set.
func RetryDelay(backoff []time.Duration, n int) time.Duration {
	if len(backoff) == 0 {
		return 5 * time.Second
	}
	return backoff[min(max(n, 0), len(backoff)-1)]
}

// Start launches the in-process worker (async mode only; idempotent).
func (q *Queue) Start() error {
	if q.opts.Mode == ModeSync {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.started {
		return nil
	}
	conc := q.opts.Concurrency
	if conc <= 0 {
		conc = 4
	}
	q.server = asynq.NewServerFromRedisClient(q.rdb, asynq.Config{
		Concurrency: conc,
		Queues:      map[string]int{q.opts.Name: 1},
		RetryDelayFunc: func(n int, _ error, t *asynq.Task) time.Duration {
			q.mu.Lock()
			reg := q.jobs[t.Type()]
			q.mu.Unlock()
			return RetryDelay(reg.job.Backoff, n)
		},
		Logger:          slogAdapter{q.logger},
		LogLevel:        asynq.WarnLevel,
		ShutdownTimeout: 10 * time.Second,
	})
	mux := asynq.NewServeMux()
	for typ, reg := range q.jobs {
		mux.HandleFunc(typ, func(ctx context.Context, t *asynq.Task) error {
			err := reg.handler(ctx, t.Payload())
			if err != nil {
				retried, _ := asynq.GetRetryCount(ctx)
				maxRetry, _ := asynq.GetMaxRetry(ctx)
				if retried >= maxRetry {
					q.failed(ctx, typ, t.Payload(), err)
				}
			}
			return err
		})
	}
	if err := q.server.Start(mux); err != nil {
		return fmt.Errorf("queue: start worker: %w", err)
	}
	q.started = true
	return nil
}

// Shutdown stops the worker, letting running jobs finish (up to 10s).
func (q *Queue) Shutdown() {
	q.mu.Lock()
	srv, started := q.server, q.started
	q.started = false
	q.mu.Unlock()
	if started && srv != nil {
		srv.Shutdown()
	}
}

func (q *Queue) failed(ctx context.Context, jobType string, payload []byte, err error) {
	if q.onFailed != nil {
		q.onFailed(ctx, jobType, payload, err)
	}
}

// slogAdapter routes asynq's logs to slog.
type slogAdapter struct{ l *slog.Logger }

func (a slogAdapter) Debug(args ...any) {
	a.l.Debug(fmt.Sprint(args...), slog.String("component", "asynq"))
}
func (a slogAdapter) Info(args ...any) {
	a.l.Info(fmt.Sprint(args...), slog.String("component", "asynq"))
}
func (a slogAdapter) Warn(args ...any) {
	a.l.Warn(fmt.Sprint(args...), slog.String("component", "asynq"))
}
func (a slogAdapter) Error(args ...any) {
	a.l.Error(fmt.Sprint(args...), slog.String("component", "asynq"))
}
func (a slogAdapter) Fatal(args ...any) {
	a.l.Error(fmt.Sprint(args...), slog.String("component", "asynq"), slog.Bool("fatal", true))
	os.Exit(1)
}
