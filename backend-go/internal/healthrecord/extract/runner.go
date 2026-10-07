package extract

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/healthrecord"
	"github.com/ritme/backend-go/internal/healthrecord/extract/store"
)

// Runner defaults (the lab runner's numbers, internal/labs/worker.go).
const (
	// DefaultLease is how long a claimed job is reserved; a dead worker's job is taken again once it expires.
	DefaultLease = 10 * time.Minute
	// DefaultPoll is how often idle workers look for due jobs (a new job also wakes them).
	DefaultPoll = 5 * time.Second
	// MaxAttempts per job; the last one stores the final state.
	MaxAttempts = 3
	// DefaultConcurrency is the number of worker goroutines per instance.
	DefaultConcurrency = 2
)

func backoff(attempt int) time.Duration {
	return []time.Duration{15 * time.Second, time.Minute, 5 * time.Minute}[min(max(attempt, 1), 3)-1]
}

// RunnerOptions configures the workers.
type RunnerOptions struct {
	// Sync runs a job inline in the request that queued it, once (QUEUE_CONNECTION=sync: the contract stack, tests).
	Sync        bool
	Concurrency int
	Poll        time.Duration
	Lease       time.Duration
}

// Runner is the in-process, DB-backed queue of the extraction jobs: the queue is record_documents itself (pending
// rows with a job object in `extracted`), claimed with SELECT … FOR UPDATE SKIP LOCKED; a crashed worker's job is
// reclaimed after its lease, a job that used every attempt fails (and refunds). No Redis, no queue table.
type Runner struct {
	s      *Service
	o      RunnerOptions
	wake   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

// NewRunner attaches a runner to s.
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

// Start launches the workers (no-op in sync mode). Errors never.
func (r *Runner) Start() error {
	if r.o.Sync {
		return nil
	}
	r.once.Do(func() {
		for range r.o.Concurrency {
			r.wg.Add(1)
			go r.loop()
		}
	})
	return nil
}

// Shutdown stops the workers and waits for the running jobs (an attempt cut short is released without counting).
func (r *Runner) Shutdown() {
	r.cancel()
	r.wg.Wait()
}

// dispatch hands a freshly queued document to the workers (async) or runs it now (sync).
func (r *Runner) dispatch(ctx context.Context, userID, docID uint64) {
	if r == nil {
		return
	}
	if r.o.Sync {
		wctx := context.WithoutCancel(ctx)
		r.s.safely(wctx, docID, func() {
			if job, ok := r.claimOne(wctx, userID, docID); ok {
				r.execute(wctx, job, true)
			}
		})
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
		worked := false
		r.s.safely(r.ctx, 0, func() {
			if job, ok := r.claim(r.ctx); ok {
				worked = true
				r.execute(r.ctx, job, false)
			}
		})
		if worked {
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

// expired are the jobs found exhausted while claiming: failed (and refunded) after the claim transaction.
type expired struct {
	job      claimed
	reserved time.Time
}

// take decides, on a locked pending row, whether it can be claimed now and claims it (attempts + 1 = the token).
func (r *Runner) take(ctx context.Context, q *store.Queries, doc store.RecordDocument, now time.Time) (claimed, *expired, bool, error) {
	st := parseStored(doc.Extracted)
	job := claimed{docID: doc.ID, userID: doc.UserID, kind: doc.Kind}
	if st == nil || st.Job == nil {
		// A pending row without a job (a concurrent PUT wrote the old state back): end it from what it holds.
		return job, &expired{job: job}, false, nil
	}
	j := st.Job
	switch {
	case j.State == jobRunning && j.LockedUntil >= now.Unix():
		return job, nil, false, nil
	case j.State == jobQueued && j.AvailableAt > now.Unix():
		return job, nil, false, nil
	case j.Attempts >= MaxAttempts:
		job.token = j.Attempts
		at, _ := st.reservedAt()
		return job, &expired{job: job, reserved: at}, false, nil
	}
	j.State, j.Attempts, j.LockedUntil = jobRunning, j.Attempts+1, now.Add(r.o.Lease).Unix()
	raw, err := st.raw()
	if err != nil {
		return job, nil, false, err
	}
	if _, err := q.SetExtractJob(ctx, store.SetExtractJobParams{Extracted: raw, Now: nullTime(now), ID: doc.ID, UserID: doc.UserID}); err != nil {
		return job, nil, false, fmt.Errorf("extract: claim: %w", err)
	}
	job.locale, job.token = j.Locale, j.Attempts
	return job, nil, true, nil
}

// claim takes the first due job in one transaction.
func (r *Runner) claim(ctx context.Context) (claimed, bool) {
	now := r.s.now(ctx)
	var (
		got   claimed
		ok    bool
		stale []expired
	)
	err := r.s.inTx(ctx, func(q *store.Queries) error {
		rows, err := q.ListPendingExtractions(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			doc, err := lockDoc(ctx, q, row.UserID, row.ID)
			if err != nil {
				continue
			}
			job, exp, taken, err := r.take(ctx, q, doc, now)
			if err != nil {
				return err
			}
			if exp != nil {
				stale = append(stale, *exp)
			}
			if taken {
				got, ok = job, true
				return nil
			}
		}
		return nil
	})
	if err != nil {
		if ctx.Err() == nil {
			r.s.logger.ErrorContext(ctx, "extract: claim job", slog.String("error", err.Error()))
		}
		return claimed{}, false
	}
	for _, e := range stale {
		r.s.safely(ctx, e.job.docID, func() { r.s.endStale(ctx, e) })
	}
	return got, ok
}

// claimOne claims the given document's job (sync mode).
func (r *Runner) claimOne(ctx context.Context, userID, docID uint64) (claimed, bool) {
	now := r.s.now(ctx)
	var (
		got claimed
		ok  bool
	)
	err := r.s.inTx(ctx, func(q *store.Queries) error {
		doc, err := lockDoc(ctx, q, userID, docID)
		if err != nil {
			return err
		}
		if doc.ReviewState != healthrecord.ReviewPending {
			return nil
		}
		got, _, ok, err = r.take(ctx, q, doc, now)
		return err
	})
	if err != nil {
		return claimed{}, false
	}
	return got, ok
}

func (r *Runner) shuttingDown() bool { return !r.o.Sync && r.ctx.Err() != nil }

// execute runs a claimed job and records the outcome. A panic fails the document instead of killing the worker.
func (r *Runner) execute(ctx context.Context, job claimed, forceFinal bool) {
	wctx := context.WithoutCancel(ctx)
	final := forceFinal || job.token >= MaxAttempts
	defer func() {
		if p := recover(); p != nil {
			r.s.logger.ErrorContext(wctx, "extract: job panicked", slog.Uint64("document_id", job.docID), slog.String("panic_type", fmt.Sprintf("%T", p)))
			r.s.safely(wctx, job.docID, func() { r.s.finish(wctx, job, outFail, read{}, CodeInternal, time.Time{}) })
		}
	}()
	jctx, cancel := context.WithTimeout(ctx, r.o.Lease-30*time.Second)
	defer cancel()
	res, err := r.s.extract(jctx, job)
	now := r.s.now(wctx)
	je := classify(err)
	switch {
	case err == nil:
		r.s.finish(wctx, job, outDone, res, "", time.Time{})
		return
	case r.shuttingDown() && errors.Is(err, context.Canceled):
		r.s.finish(wctx, job, outRelease, read{}, "", now)
		return
	case je.retry && !final:
		r.s.finish(wctx, job, outRetry, read{}, je.code, now.Add(backoff(job.token)))
	default:
		r.s.finish(wctx, job, outFail, read{}, je.code, time.Time{})
	}
	// Codes and ids only: never a value, a file name or the provider's text.
	r.s.logger.WarnContext(wctx, "extract: job attempt failed", slog.Uint64("document_id", job.docID),
		slog.Int("attempt", job.token), slog.Bool("final", final), slog.String("code", je.code))
}

// safely runs fn and turns a panic into a log line (its Go type and the document id only, never a value) — a
// worker, a claim, a sweep or a refund that panics must neither kill the process nor stop the worker (audit L4).
func (s *Service) safely(ctx context.Context, docID uint64, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			s.logger.ErrorContext(context.WithoutCancel(ctx), "extract: worker panicked", slog.Uint64("document_id", docID),
				slog.String("panic_type", fmt.Sprintf("%T", p)))
		}
	}()
	fn()
}

// endStale finishes a pending document no worker will: a job that used every attempt fails (refund), a row without
// a job gets the state its extraction reached.
func (s *Service) endStale(ctx context.Context, e expired) {
	ctx = context.WithoutCancel(ctx)
	now := s.now(ctx)
	var refundAt time.Time
	failed := false
	err := s.inTx(ctx, func(q *store.Queries) error {
		doc, err := lockDoc(ctx, q, e.job.userID, e.job.docID)
		if err != nil {
			return err
		}
		if doc.ReviewState != healthrecord.ReviewPending {
			return nil
		}
		st := parseStored(doc.Extracted)
		state := healthrecord.ReviewFailed
		switch {
		case st == nil:
			st = &Stored{Schema: doc.Kind, Status: StatusFailed, ErrorCode: strPtr(CodeInternal), RequestedAt: isoTime(now)}
		case st.Job == nil && st.Status == StatusDone:
			state = healthrecord.ReviewNeedsReview
		case st.Job != nil && st.Job.Attempts != e.job.token:
			return nil // claimed again meanwhile
		default:
			refundAt, _ = st.reservedAt()
			st.Status, st.ErrorCode, failed = StatusFailed, strPtr(CodeInternal), true
		}
		st.Job = nil
		raw, err := st.raw()
		if err != nil {
			return err
		}
		_, err = q.SetExtractState(ctx, store.SetExtractStateParams{ReviewState: state, Extracted: raw, Now: nullTime(now),
			ID: doc.ID, UserID: doc.UserID})
		return err
	})
	if err != nil {
		return
	}
	if failed {
		s.refund(ctx, e.job.userID, refundAt, CodeInternal)
	}
}
