package labs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	// JobSweepEvery: exhausted jobs (lease expired after the last attempt) and busy labs without a live job are
	// finished (B-N6-06b).
	JobSweepEvery = time.Minute
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

// Shutdown stops the workers and waits for the running jobs. An attempt cut short by the shutdown is released back
// to pending without counting (ReleaseLabJob); one cut short by a crash is reclaimed after its lease.
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
		if job, ok := r.claim(r.ctx); ok {
			r.execute(r.ctx, job, false)
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

func (r *Runner) claimParams(id uint64, now time.Time) store.ClaimLabJobParams {
	return store.ClaimLabJobParams{LockedUntil: sql.NullTime{Time: now.Add(r.o.Lease), Valid: true},
		Now: sql.NullTime{Time: now, Valid: true}, ID: id, Due: now, Expired: sql.NullTime{Time: now, Valid: true},
		MaxAttempts: MaxAttempts}
}

// claim takes the oldest due job (or one whose lease expired) in one transaction: SELECT … FOR UPDATE SKIP LOCKED,
// then the claiming UPDATE. Returns the job as claimed (Attempts = its claim token).
func (r *Runner) claim(ctx context.Context) (store.LabJob, bool) {
	now := tehranSecond(r.s.now(ctx))
	var job store.LabJob
	err := r.s.inTx(ctx, func(q *store.Queries) error {
		id, err := q.NextLabJob(ctx, store.NextLabJobParams{Due: now, Expired: sql.NullTime{Time: now, Valid: true}, MaxAttempts: MaxAttempts})
		if err != nil {
			return err
		}
		n, err := q.ClaimLabJob(ctx, r.claimParams(id, now))
		if err != nil {
			return err
		}
		if n != 1 {
			return sql.ErrNoRows
		}
		job, err = q.GetLabJob(ctx, id)
		return err
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) && ctx.Err() == nil {
			r.s.logger.ErrorContext(ctx, "labs: claim job", slog.String("error", err.Error()))
		}
		return store.LabJob{}, false
	}
	return job, true
}

// runNow claims and runs one job inline (sync mode): one attempt, final.
func (r *Runner) runNow(ctx context.Context, id uint64) {
	now := tehranSecond(r.s.now(ctx))
	n, err := r.s.q.ClaimLabJob(ctx, r.claimParams(id, now))
	if err != nil || n != 1 {
		return
	}
	job, err := r.s.q.GetLabJob(ctx, id)
	if err != nil {
		return
	}
	r.execute(ctx, job, true)
}

// shuttingDown reports whether the runner is stopping (an attempt cut short then is not counted).
func (r *Runner) shuttingDown() bool { return !r.o.Sync && r.ctx.Err() != nil }

// execute runs a claimed job (job.Attempts is the claim token) and records the outcome. A panic finishes the job
// and its lab for good (failed extraction / rules summary) instead of killing the worker.
func (r *Runner) execute(ctx context.Context, job store.LabJob, forceFinal bool) {
	wctx := context.WithoutCancel(ctx)
	final := forceFinal || int(job.Attempts) >= MaxAttempts
	defer func() {
		if p := recover(); p != nil {
			r.s.logger.ErrorContext(wctx, "labs: job panicked", slog.String("kind", job.Kind), slog.Uint64("job_id", job.ID),
				slog.String("panic", fmt.Sprint(p)))
			r.finish(wctx, job, jobFailed, CodeInternal)
			r.s.finalize(wctx, job, CodeInternal)
		}
	}()
	jctx, cancel := context.WithTimeout(ctx, r.o.Lease-30*time.Second)
	defer cancel()
	var err error
	switch job.Kind {
	case JobExtract:
		err = r.s.runExtract(jctx, job, final)
	case JobInterpret:
		err = r.s.runInterpret(jctx, job, final)
	default:
		err = &jobError{code: CodeInternal}
	}
	now := r.s.now(wctx)
	var je *jobError
	switch {
	case err == nil:
		r.finish(wctx, job, jobDone, "")
	case r.shuttingDown() && errors.Is(err, context.Canceled):
		if _, rerr := r.s.q.ReleaseLabJob(wctx, store.ReleaseLabJobParams{Now: nullTime(now), ID: job.ID, Token: job.Attempts}); rerr != nil {
			r.s.logger.ErrorContext(wctx, "labs: release job", slog.String("error", rerr.Error()))
		}
		return
	case errors.As(err, &je) && je.retry && !final:
		if _, rerr := r.s.q.RetryLabJob(wctx, store.RetryLabJobParams{
			AvailableAt: tehranSecond(now.Add(backoff(int(job.Attempts)))), LastError: nullString(je.code),
			Now: nullTime(now), ID: job.ID, Token: job.Attempts,
		}); rerr != nil {
			r.s.logger.ErrorContext(wctx, "labs: retry job", slog.String("error", rerr.Error()))
		}
	default:
		r.finish(wctx, job, jobFailed, codeOf(err))
	}
	if err != nil {
		// Codes and ids only: never a value, a file name or the provider's text.
		r.s.logger.WarnContext(wctx, "labs: job attempt failed", slog.String("kind", job.Kind), slog.Uint64("job_id", job.ID),
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

// finish records the job's end, only while the row still carries this claim (token = attempts).
func (r *Runner) finish(ctx context.Context, job store.LabJob, status, code string) {
	if _, err := r.s.q.FinishLabJob(ctx, store.FinishLabJobParams{Status: status, LastError: nullString(code),
		Now: nullTime(r.s.now(ctx)), ID: job.ID, Token: job.Attempts}); err != nil {
		r.s.logger.ErrorContext(ctx, "labs: finish job", slog.String("error", err.Error()))
	}
}

func (r *Runner) sweepLoop() {
	defer r.wg.Done()
	t := time.NewTicker(JobSweepEvery)
	defer t.Stop()
	var lastFiles time.Time
	for {
		r.SweepJobs(r.ctx)
		if time.Since(lastFiles) >= SweepEvery {
			r.Sweep(r.ctx)
			lastFiles = time.Now()
		}
		select {
		case <-r.ctx.Done():
			return
		case <-t.C:
		}
	}
}

// SweepJobs finishes what no worker will (B-N6-06b): running jobs whose lease expired after their last attempt, and
// labs left queued / extracting / interpreting without a live job. An extraction fails (the Plus use is refunded);
// an interpretation gets the rules summary.
func (r *Runner) SweepJobs(ctx context.Context) {
	wctx := context.WithoutCancel(ctx)
	now := tehranSecond(r.s.now(wctx))
	jobs, err := r.s.q.ListExhaustedLabJobs(wctx, store.ListExhaustedLabJobsParams{Expired: sql.NullTime{Time: now, Valid: true}, MaxAttempts: MaxAttempts})
	if err != nil {
		r.s.logger.ErrorContext(wctx, "labs: sweep exhausted jobs", slog.String("error", err.Error()))
	}
	for _, job := range jobs {
		r.finish(wctx, job, jobFailed, CodeInternal)
		r.s.finalize(wctx, job, CodeInternal)
	}
	labs, err := r.s.q.ListOrphanBusyLabs(wctx, nullTime(now.Add(-2*r.o.Lease)))
	if err != nil {
		r.s.logger.ErrorContext(wctx, "labs: sweep orphan labs", slog.String("error", err.Error()))
	}
	for _, lab := range labs {
		kind := JobExtract
		if lab.Status == StatusInterpreting {
			kind = JobInterpret
		}
		r.s.finalize(wctx, store.LabJob{LabID: lab.ID, UserID: lab.UserID, Kind: kind}, CodeInternal)
	}
}

// Sweep removes lab file directories of users that no longer exist (an account deleted by a path that could not
// remove them, e.g. the Laravel stack before the cutover) and finished job rows older than a week. It never runs
// against an empty users table and never removes a directory that still has lab_files rows (B-N6-06b, L8).
func (r *Runner) Sweep(ctx context.Context) {
	now := r.s.now(ctx)
	if err := r.s.q.DeleteFinishedLabJobs(ctx, nullTime(now.Add(-keepFinishedJobs))); err != nil && ctx.Err() == nil {
		r.s.logger.ErrorContext(ctx, "labs: sweep jobs", slog.String("error", err.Error()))
	}
	if r.o.StoragePath == "" {
		return
	}
	if users, err := r.s.q.CountLabUsers(ctx); err != nil || users == 0 {
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
		if rows, err := r.s.q.CountUserLabFiles(ctx, id); err != nil || rows > 0 {
			continue
		}
		if err := files.RemoveUser(r.o.StoragePath, id); err != nil {
			r.s.logger.ErrorContext(ctx, "labs: sweep user files", slog.String("error", err.Error()))
		}
	}
}
