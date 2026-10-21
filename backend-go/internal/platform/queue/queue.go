// Package queue runs background jobs on asynq over the shared Redis (T-M9-11, docs/engine-v6/jobs.md).
//
// Isolation: asynq cannot prefix its keys, so the Go queues are asynq *queue names* derived from REDIS_PREFIX
// ("ritme-go:" → base name "ritme-go"). Three weighted queues share that base name:
//
//	critical  "<base>-critical"  weight 6  (OTP SMS)
//	default   "<base>"           weight 3  (the pre-v2 queue name, so tasks enqueued before a deploy still run)
//	low       "<base>-low"       weight 1  (sweeps, purges, snapshots)
//
// Laravel's queue (PHP-serialized, `queues:default`) is never read.
//
// Mode "sync" (QUEUE_CONNECTION=sync, like Laravel's sync driver in the contract stack and local dev) runs the job
// inline, once, and returns its error to the caller; delays and uniqueness are ignored, and the Scheduler fires its
// entries from an in-process cron instead of Redis.
//
// Usage:
//
//	q := queue.New(rdb, queue.Options{Name: "ritme-go", Mode: queue.ModeFromEnv(), Logger: log})
//	q.Register(queue.Job{Type: "send_otp_sms", Priority: queue.Critical, MaxTries: 3,
//		Backoff: []time.Duration{5*time.Second, 15*time.Second}}, handler)
//	q.Scheduler().Register("*/10 * * * *", "partners:sync", nil, queue.Unique(9*time.Minute))
//	q.Start()            // on listen: worker + scheduler (no worker in sync mode)
//	defer q.Shutdown()   // on shutdown: waits for running jobs
//	err := q.Dispatch(ctx, "send_otp_sms", payload)
//	err = q.DispatchIn(ctx, time.Minute, "x", payload, queue.Unique(time.Hour))
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
	// ModeAsync enqueues on Redis for the asynq worker.
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

// NameFromPrefix turns a Redis key prefix into the base queue name ("ritme-go:" → "ritme-go").
func NameFromPrefix(prefix string) string {
	name := strings.Trim(prefix, ":")
	if name == "" {
		return "ritme-go"
	}
	return name
}

// Priority picks one of the three weighted queues.
type Priority int

const (
	// Default is weight 3 (the zero value).
	Default Priority = iota
	// Critical is weight 6: user-facing latency (OTP).
	Critical
	// Low is weight 1: periodic housekeeping.
	Low
)

// Queue weights (asynq picks a queue with probability weight / sum).
const (
	WeightCritical = 6
	WeightDefault  = 3
	WeightLow      = 1
)

func (p Priority) String() string {
	switch p {
	case Critical:
		return "critical"
	case Low:
		return "low"
	default:
		return "default"
	}
}

// NameFor is the asynq queue of priority p under base name.
func NameFor(base string, p Priority) string {
	switch p {
	case Critical:
		return base + "-critical"
	case Low:
		return base + "-low"
	default:
		return base
	}
}

// Handler processes one job payload (the JSON Dispatch encoded). Returning an error fails this try; the job is
// retried until MaxTries is reached.
type Handler func(ctx context.Context, payload []byte) error

// Job describes a job type.
type Job struct {
	Type     string
	Priority Priority
	MaxTries int             // total tries incl. the first ($tries); < 1 means 1
	Backoff  []time.Duration // delay before retry n (1-based) = Backoff[min(n, len)-1] ($backoff)
	Timeout  time.Duration   // per try; 0 = 60s (queue:work --timeout default)
	// OnFailed is called once when this job exhausted its tries (the job's failed() hook), before the queue-wide one.
	OnFailed func(ctx context.Context, payload []byte, err error)
}

// OnFailed is called once when a job exhausted its tries (any job type).
type OnFailed func(ctx context.Context, jobType string, payload []byte, err error)

// Options configure a Queue.
type Options struct {
	Name        string // base queue name (NameFromPrefix(REDIS_PREFIX))
	Mode        Mode
	Concurrency int // worker goroutines; 0 = 4
	Logger      *slog.Logger
	// LeaderTTL is the scheduler leader key's lifetime; it is renewed every LeaderTTL/3. 0 = 30s.
	LeaderTTL time.Duration
}

// Queue dispatches and (in async mode) processes jobs; Scheduler() fires periodic ones.
type Queue struct {
	opts      Options
	rdb       *redis.Client
	client    *asynq.Client
	logger    *slog.Logger
	mu        sync.Mutex
	jobs      map[string]registered
	server    *asynq.Server
	started   bool
	onFailed  OnFailed
	scheduler *Scheduler
}

type registered struct {
	job     Job
	handler Handler
}

// New builds a queue on rdb (shared connection; asynq never closes it). rdb may be nil only in ModeSync.
func New(rdb *redis.Client, opts Options) *Queue {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Name == "" {
		opts.Name = NameFromPrefix("")
	}
	if opts.LeaderTTL <= 0 {
		opts.LeaderTTL = 30 * time.Second
	}
	if rdb == nil {
		opts.Mode = ModeSync
	}
	q := &Queue{opts: opts, rdb: rdb, logger: opts.Logger, jobs: map[string]registered{}}
	if opts.Mode != ModeSync {
		q.client = asynq.NewClientFromRedisClient(rdb)
	}
	q.scheduler = newScheduler(q)
	return q
}

// Mode returns the effective mode.
func (q *Queue) Mode() Mode { return q.opts.Mode }

// Name returns the base queue name.
func (q *Queue) Name() string { return q.opts.Name }

// Queues returns the three asynq queue names with their weights.
func (q *Queue) Queues() map[string]int {
	return map[string]int{
		NameFor(q.opts.Name, Critical): WeightCritical,
		NameFor(q.opts.Name, Default):  WeightDefault,
		NameFor(q.opts.Name, Low):      WeightLow,
	}
}

// Scheduler returns the queue's periodic scheduler (started and stopped with the queue).
func (q *Queue) Scheduler() *Scheduler { return q.scheduler }

// OnFailed sets the exhausted-tries hook for every job type (both modes).
func (q *Queue) OnFailed(fn OnFailed) { q.onFailed = fn }

// Register adds a job type. It may be called after Start (the worker looks handlers up per task).
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

func (q *Queue) lookup(jobType string) (registered, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	reg, ok := q.jobs[jobType]
	return reg, ok
}

// ErrUnknownJob is returned by Dispatch for an unregistered job type.
var ErrUnknownJob = errors.New("queue: unknown job type")

// ErrDuplicate is returned when a Unique job is still locked or a TaskID is already taken. The job was not enqueued
// again; callers that only want "at most one" treat it as success.
var ErrDuplicate = errors.New("queue: duplicate task")

// Option tunes one dispatch (or a scheduler entry).
type Option func(*dispatchOpts)

type dispatchOpts struct {
	unique     time.Duration
	taskID     string
	processAt  time.Time
	runAtStart bool
}

// Unique drops the dispatch (ErrDuplicate) while a task with the same type, payload and queue is still locked: until
// it succeeds or ttl passes. Ignored in sync mode.
func Unique(ttl time.Duration) Option { return func(o *dispatchOpts) { o.unique = ttl } }

// TaskID gives the task an explicit id (a natural key): a second dispatch with the same id answers ErrDuplicate while
// the first is pending, running or retrying. Ignored in sync mode.
func TaskID(id string) Option { return func(o *dispatchOpts) { o.taskID = id } }

// RunAtStart makes a scheduler entry also fire once when this process becomes the scheduling leader (the old
// tickers ran at start-up). Ignored by Dispatch.
func RunAtStart() Option { return func(o *dispatchOpts) { o.runAtStart = true } }

func collect(opts []Option) dispatchOpts {
	var o dispatchOpts
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o
}

// Dispatch encodes payload as JSON and enqueues it (async) or runs it inline (sync).
func (q *Queue) Dispatch(ctx context.Context, jobType string, payload any) error {
	return q.DispatchWith(ctx, jobType, payload)
}

// DispatchWith is Dispatch with options (Unique, TaskID).
func (q *Queue) DispatchWith(ctx context.Context, jobType string, payload any, opts ...Option) error {
	return q.dispatch(ctx, jobType, payload, collect(opts))
}

// DispatchIn enqueues the job to run no earlier than d from now (sync mode: inline, now — like Laravel's sync driver).
func (q *Queue) DispatchIn(ctx context.Context, d time.Duration, jobType string, payload any, opts ...Option) error {
	o := collect(opts)
	o.processAt = time.Now().Add(d)
	return q.dispatch(ctx, jobType, payload, o)
}

// DispatchAt enqueues the job to run no earlier than at (sync mode: inline, now).
func (q *Queue) DispatchAt(ctx context.Context, at time.Time, jobType string, payload any, opts ...Option) error {
	o := collect(opts)
	o.processAt = at
	return q.dispatch(ctx, jobType, payload, o)
}

func (q *Queue) dispatch(ctx context.Context, jobType string, payload any, o dispatchOpts) error {
	reg, ok := q.lookup(jobType)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownJob, jobType)
	}
	body, err := encode(payload)
	if err != nil {
		return fmt.Errorf("queue: encode %s: %w", jobType, err)
	}
	if q.opts.Mode == ModeSync {
		if err := q.run(ctx, reg, body, true); err != nil {
			return fmt.Errorf("queue: %s: %w", jobType, err)
		}
		return nil
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(jobType, body), q.taskOptions(reg.job, o)...)
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return fmt.Errorf("%w: %s", ErrDuplicate, jobType)
	}
	if err != nil {
		return fmt.Errorf("queue: enqueue %s: %w", jobType, err)
	}
	return nil
}

// encode marshals payload; nil becomes {} and raw JSON is kept as is.
func encode(payload any) ([]byte, error) {
	switch p := payload.(type) {
	case nil:
		return []byte("{}"), nil
	case json.RawMessage:
		return p, nil
	}
	return json.Marshal(payload)
}

func (q *Queue) taskOptions(job Job, o dispatchOpts) []asynq.Option {
	out := []asynq.Option{
		asynq.Queue(NameFor(q.opts.Name, job.Priority)),
		asynq.MaxRetry(job.MaxTries - 1),
		asynq.Timeout(job.Timeout),
	}
	if o.unique > 0 {
		out = append(out, asynq.Unique(o.unique))
	}
	if o.taskID != "" {
		out = append(out, asynq.TaskID(o.taskID))
	}
	if !o.processAt.IsZero() {
		out = append(out, asynq.ProcessAt(o.processAt))
	}
	return out
}

// run executes one try of reg, observing its duration; last says whether this was the final try (fires OnFailed).
func (q *Queue) run(ctx context.Context, reg registered, payload []byte, last bool) error {
	start := time.Now()
	err := reg.handler(ctx, payload)
	observeTask(reg.job.Type, err, time.Since(start))
	if err != nil {
		q.logger.WarnContext(ctx, "queue: job failed", slog.String("job", reg.job.Type),
			slog.Bool("final", last), slog.String("error", err.Error()))
		if last {
			q.failed(ctx, reg, payload, err)
		}
	}
	return err
}

// RetryDelay is the delay before retry n (asynq passes the number of retries done so far, 0 on the first failure):
// Backoff[min(n, len-1)], or 5s when no backoff is set.
func RetryDelay(backoff []time.Duration, n int) time.Duration {
	if len(backoff) == 0 {
		return 5 * time.Second
	}
	return backoff[min(max(n, 0), len(backoff)-1)]
}

// Start launches the worker (async mode only) and the scheduler. Idempotent.
func (q *Queue) Start() error {
	if q.opts.Mode != ModeSync {
		if err := q.startWorker(); err != nil {
			return err
		}
	}
	return q.scheduler.start()
}

func (q *Queue) startWorker() error {
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
		Queues:      q.Queues(),
		RetryDelayFunc: func(n int, _ error, t *asynq.Task) time.Duration {
			reg, _ := q.lookup(t.Type())
			return RetryDelay(reg.job.Backoff, n)
		},
		Logger:          slogAdapter{q.logger},
		LogLevel:        asynq.WarnLevel,
		ShutdownTimeout: 10 * time.Second,
	})
	if err := q.server.Start(asynq.HandlerFunc(q.process)); err != nil {
		return fmt.Errorf("queue: start worker: %w", err)
	}
	trackQueue(q)
	q.started = true
	return nil
}

// process is the worker's single asynq handler: it looks the job up per task, so jobs registered after Start run too.
func (q *Queue) process(ctx context.Context, t *asynq.Task) error {
	reg, ok := q.lookup(t.Type())
	if !ok {
		return fmt.Errorf("%w: %s: %w", ErrUnknownJob, t.Type(), asynq.SkipRetry)
	}
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	last := retried >= maxRetry
	err := q.run(ctx, reg, t.Payload(), last)
	if err != nil && last {
		incArchived(NameFor(q.opts.Name, reg.job.Priority))
	}
	return err
}

// Shutdown stops the scheduler and the worker, letting running jobs finish (up to 10s).
func (q *Queue) Shutdown() {
	q.scheduler.stop()
	q.mu.Lock()
	srv, started := q.server, q.started
	q.started = false
	q.mu.Unlock()
	if started && srv != nil {
		srv.Shutdown()
		untrackQueue(q)
	}
}

func (q *Queue) failed(ctx context.Context, reg registered, payload []byte, err error) {
	if reg.job.OnFailed != nil {
		reg.job.OnFailed(ctx, payload, err)
	}
	if q.onFailed != nil {
		q.onFailed(ctx, reg.job.Type, payload, err)
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
