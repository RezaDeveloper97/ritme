package labs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/labs/files"
	"github.com/ritme/backend-go/internal/labs/store"
)

// Job kinds (lab_jobs.kind).
const (
	JobExtract   = "extract"
	JobInterpret = "interpret"
)

// Final job statuses (lab_jobs.status; pending / running are set by the queries).
const (
	jobDone   = "done"
	jobFailed = "failed"
)

// Runner defaults.
const (
	// DefaultLease is how long a claimed job is reserved; a worker that dies leaves the job to be picked up again
	// once it expires (restart-safe). Each attempt runs under a context shorter than the lease.
	DefaultLease = 10 * time.Minute
	// DefaultPoll is how often idle workers look for due jobs (new jobs also wake them at once).
	DefaultPoll = 5 * time.Second
	// MaxAttempts per job; the last one stores a final state (failed extraction, rules summary).
	MaxAttempts = 3
	// DefaultConcurrency is the number of worker goroutines per instance.
	DefaultConcurrency = 2
	// SweepEvery: orphan lab files (an account deleted by a path that did not remove them) and old finished jobs.
	SweepEvery = 6 * time.Hour
	// keepFinishedJobs: finished job rows are deleted after this long.
	keepFinishedJobs = 7 * 24 * time.Hour
)

// backoff is the delay before attempt n+1.
func backoff(attempt int) time.Duration {
	return []time.Duration{15 * time.Second, time.Minute, 5 * time.Minute}[min(max(attempt, 1), 3)-1]
}

// RunnerOptions configures the worker.
type RunnerOptions struct {
	// Sync runs every job inline in the request that created it, once (QUEUE_CONNECTION=sync: the contract stack,
	// tests). Otherwise jobs run on the in-process workers started by Start.
	Sync        bool
	Concurrency int
	Poll        time.Duration
	Lease       time.Duration
	// StoragePath enables the orphan file sweep ("" = off).
	StoragePath string
}

// Runner is the DB-backed job queue of the lab analysis (lab_jobs): pending jobs are claimed atomically by an
// in-process worker, run, and finished / retried; a crashed worker's job is reclaimed after its lease. No Redis.
type Runner struct {
	s      *Service
	o      RunnerOptions
	wake   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// NewRunner attaches a runner to s (the service dispatches its jobs to it).
func NewRunner(s *Service, o RunnerOptions) *Runner {
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.Poll <= 0 {
		o.Poll = DefaultPoll
	}
	if o.Lease <= 0 {
		o.Lease = DefaultLease
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &Runner{s: s, o: o, wake: make(chan struct{}, 1), ctx: ctx, cancel: cancel}
	s.runner = r
	return r
}

// Start launches the workers and the sweep (no-op in sync mode). Safe to call once; errors never.
func (r *Runner) Start() error {
	if r.o.Sync {
		return nil
	}
	r.once.Do(func() {
		for range r.o.Concurrency {
			r.wg.Add(1)
			go r.loop()
		}
		r.wg.Add(1)
		go r.sweepLoop()
	})
	return nil
}

// Shutdown stops the workers and waits for the running jobs (their attempts end with the context; an interrupted
// job is reclaimed after its lease by the next instance).
func (r *Runner) Shutdown() {
	r.cancel()
	r.wg.Wait()
}

// dispatch hands a freshly inserted job to the workers (async) or runs it now (sync).
func (r *Runner) dispatch(ctx context.Context, jobID uint64) {
	if r == nil {
		return
	}
	if r.o.Sync {
		r.runNow(context.WithoutCancel(ctx), jobID)
		return
	}
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Runner) loop() {
	defer r.wg.Done()
	t := time.NewTicker(r.o.Poll)
	defer t.Stop()
	for {
		if r.ctx.Err() != nil {
			return
		}
		if id, ok := r.claim(r.ctx); ok {
			r.execute(r.ctx, id, false)
			continue
		}
		select {
		case <-r.ctx.Done():
			return
		case <-r.wake:
		case <-t.C:
		}
	}
}

// claim takes the oldest due job (or one whose lease expired).
func (r *Runner) claim(ctx context.Context) (uint64, bool) {
	now := tehranSecond(r.s.now(ctx))
	id, err := r.s.q.NextLabJob(ctx, store.NextLabJobParams{Due: now, Expired: sql.NullTime{Time: now, Valid: true}})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			r.s.logger.ErrorContext(ctx, "labs: next job", slog.String("error", err.Error()))
		}
		return 0, false
	}
	return id, r.take(ctx, id, now)
}

func (r *Runner) take(ctx context.Context, id uint64, now time.Time) bool {
	n, err := r.s.q.ClaimLabJob(ctx, store.ClaimLabJobParams{LockedUntil: sql.NullTime{Time: now.Add(r.o.Lease), Valid: true},
		Now: sql.NullTime{Time: now, Valid: true}, ID: id, Due: now, Expired: sql.NullTime{Time: now, Valid: true}})
	return err == nil && n == 1
}

// runNow claims and runs one job inline (sync mode): one attempt, final.
func (r *Runner) runNow(ctx context.Context, id uint64) {
	if r.take(ctx, id, tehranSecond(r.s.now(ctx))) {
		r.execute(ctx, id, true)
	}
}

// execute runs a claimed job and records the outcome.
func (r *Runner) execute(ctx context.Context, id uint64, forceFinal bool) {
	job, err := r.s.q.GetLabJob(ctx, id)
	if err != nil {
		return
	}
	final := forceFinal || int(job.Attempts) >= MaxAttempts
	jctx, cancel := context.WithTimeout(ctx, r.o.Lease-30*time.Second)
	defer cancel()
	switch job.Kind {
	case JobExtract:
		err = r.s.runExtract(jctx, job, final)
	case JobInterpret:
		err = r.s.runInterpret(jctx, job, final)
	default:
		err = &jobError{code: CodeInternal}
	}
	now := r.s.now(ctx)
	var je *jobError
	switch {
	case err == nil:
		r.finish(ctx, job, jobDone, "")
	case errors.As(err, &je) && je.retry && !final:
		if rerr := r.s.q.RetryLabJob(context.WithoutCancel(ctx), store.RetryLabJobParams{
			AvailableAt: tehranSecond(now.Add(backoff(int(job.Attempts)))), LastError: nullString(je.code),
			Now: nullTime(now), ID: job.ID,
		}); rerr != nil {
			r.s.logger.ErrorContext(ctx, "labs: retry job", slog.String("error", rerr.Error()))
		}
	default:
		code := CodeInternal
		if je != nil {
			code = je.code
		}
		r.finish(ctx, job, jobFailed, code)
	}
	if err != nil {
		// Codes and ids only: never a value, a file name or the provider's text.
		r.s.logger.WarnContext(ctx, "labs: job attempt failed", slog.String("kind", job.Kind), slog.Uint64("job_id", job.ID),
			slog.Int("attempt", int(job.Attempts)), slog.Bool("final", final), slog.String("code", codeOf(err)))
	}
}

func codeOf(err error) string {
	var je *jobError
	if errors.As(err, &je) {
		return je.code
	}
	return CodeInternal
}

func (r *Runner) finish(ctx context.Context, job store.LabJob, status, code string) {
	if err := r.s.q.FinishLabJob(context.WithoutCancel(ctx), store.FinishLabJobParams{Status: status, LastError: nullString(code),
		Now: nullTime(r.s.now(ctx)), ID: job.ID}); err != nil {
		r.s.logger.ErrorContext(ctx, "labs: finish job", slog.String("error", err.Error()))
	}
}

func (r *Runner) sweepLoop() {
	defer r.wg.Done()
	t := time.NewTicker(SweepEvery)
	defer t.Stop()
	for {
		r.Sweep(r.ctx)
		select {
		case <-r.ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Sweep removes lab file directories of users that no longer exist (an account deleted by a path that could not
// remove them, e.g. the Laravel stack before the cutover) and finished job rows older than a week.
func (r *Runner) Sweep(ctx context.Context) {
	now := r.s.now(ctx)
	if err := r.s.q.DeleteFinishedLabJobs(ctx, nullTime(now.Add(-keepFinishedJobs))); err != nil && ctx.Err() == nil {
		r.s.logger.ErrorContext(ctx, "labs: sweep jobs", slog.String("error", err.Error()))
	}
	if r.o.StoragePath == "" {
		return
	}
	ids, err := files.Users(r.o.StoragePath)
	if err != nil {
		r.s.logger.ErrorContext(ctx, "labs: sweep files", slog.String("error", err.Error()))
		return
	}
	for _, id := range ids {
		n, err := r.s.q.LabUserExists(ctx, id)
		if err != nil || n > 0 {
			continue
		}
		if err := files.RemoveUser(r.o.StoragePath, id); err != nil {
			r.s.logger.ErrorContext(ctx, "labs: sweep user files", slog.String("error", err.Error()))
		}
	}
}
