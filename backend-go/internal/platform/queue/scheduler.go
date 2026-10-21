package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Scheduler fires registered jobs on cron schedules (Asia/Tehran wall clock, standard 5-field specs or @every).
//
// Async mode: an asynq.Scheduler enqueues the entries, but only in the process holding the Redis leader key
// "<base>:scheduler:leader" (SET NX PX, renewed every LeaderTTL/3, released on shutdown). Every process tries to take
// the key on every renewal tick, so when the leader dies another one takes over within LeaderTTL. The job itself runs
// on whichever worker picks it up.
//
// Sync mode: a local cron calls Dispatch inline (no leader key; dev and tests are one process).
type Scheduler struct {
	q       *Queue
	mu      sync.Mutex
	entries []entry
	owner   string

	running bool
	stopCh  chan struct{}
	done    chan struct{}
	leader  bool
	asynqS  *asynq.Scheduler
	local   *cron.Cron
}

type entry struct {
	spec    string
	jobType string
	payload []byte
	opts    dispatchOpts
}

// specParser is the parser asynq's scheduler uses (cron.New default): 5 fields plus descriptors.
var specParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// Next is when spec fires next after t (Asia/Tehran) — for docs and tests.
func Next(spec string, after time.Time) (time.Time, error) {
	s, err := specParser.Parse(spec)
	if err != nil {
		return time.Time{}, fmt.Errorf("queue: cron %q: %w", spec, err)
	}
	return s.Next(after.In(civildate.Tehran)), nil
}

func newScheduler(q *Queue) *Scheduler {
	host, _ := os.Hostname()
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return &Scheduler{q: q, owner: host + ":" + hex.EncodeToString(b)}
}

// Register schedules jobType (already registered on the queue) on spec with payload (nil → {}). Options: Unique(ttl)
// keeps a backlog from stacking up (use a ttl a little under the interval), RunAtStart also fires it when this
// process becomes the leader. Entries added while the scheduler runs take effect immediately.
func (s *Scheduler) Register(spec, jobType string, payload any, opts ...Option) error {
	if _, err := specParser.Parse(spec); err != nil {
		return fmt.Errorf("queue: cron %q for %s: %w", spec, jobType, err)
	}
	if _, ok := s.q.lookup(jobType); !ok {
		return fmt.Errorf("%w: %s (register the job before scheduling it)", ErrUnknownJob, jobType)
	}
	body, err := encode(payload)
	if err != nil {
		return fmt.Errorf("queue: encode %s: %w", jobType, err)
	}
	e := entry{spec: spec, jobType: jobType, payload: body, opts: collect(opts)}
	if e.opts.unique > 0 && e.opts.unique < time.Second {
		return fmt.Errorf("queue: %s: Unique ttl must be at least 1s", jobType)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	if s.asynqS != nil {
		if err := s.addAsynq(s.asynqS, e); err != nil {
			return err
		}
	}
	if s.local != nil {
		if err := s.addLocal(s.local, e); err != nil {
			return err
		}
	}
	return nil
}

// Leader reports whether this process currently schedules (always true in sync mode while running).
func (s *Scheduler) Leader() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.leader
}

func (s *Scheduler) leaderKey() string { return s.q.opts.Name + ":scheduler:leader" }

func (s *Scheduler) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.running = true
	if s.q.opts.Mode == ModeSync {
		c := cron.New(cron.WithLocation(civildate.Tehran), cron.WithParser(specParser))
		for _, e := range s.entries {
			if err := s.addLocal(c, e); err != nil {
				return err
			}
		}
		c.Start()
		s.local, s.leader = c, true
		boot := s.bootEntries()
		go func() {
			for _, e := range boot {
				s.fireLocal(e)
			}
		}()
		return nil
	}
	s.stopCh, s.done = make(chan struct{}), make(chan struct{})
	go s.campaign(s.stopCh, s.done)
	return nil
}

func (s *Scheduler) stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	local, stopCh, done := s.local, s.stopCh, s.done
	if local != nil {
		s.local, s.leader = nil, false
	}
	s.stopCh, s.done = nil, nil
	s.mu.Unlock()
	if local != nil {
		<-local.Stop().Done()
	}
	if stopCh != nil {
		close(stopCh)
		<-done
	}
}

// campaign keeps trying to hold the leader key; it starts the asynq scheduler while it does.
func (s *Scheduler) campaign(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ttl := s.q.opts.LeaderTTL
	t := time.NewTicker(ttl / 3)
	defer t.Stop()
	for {
		s.tick(ttl)
		select {
		case <-stop:
			s.resign()
			return
		case <-t.C:
		}
	}
}

// Lua: extend / delete the key only while we still own it.
var (
	renewScript   = redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("PEXPIRE", KEYS[1], ARGV[2]) end return 0`)
	releaseScript = redis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`)
)

func (s *Scheduler) tick(ttl time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), ttl/3)
	defer cancel()
	key := s.leaderKey()
	if s.Leader() {
		n, err := renewScript.Run(ctx, s.q.rdb, []string{key}, s.owner, ttl.Milliseconds()).Int()
		if err == nil && n == 1 {
			return
		}
		if err != nil {
			s.q.logger.Warn("queue: scheduler leader renewal failed", slog.String("error", err.Error()))
		}
		s.demote()
	}
	ok, err := s.q.rdb.SetNX(ctx, key, s.owner, ttl).Result()
	if err != nil {
		s.q.logger.Warn("queue: scheduler leader election failed", slog.String("error", err.Error()))
		return
	}
	if ok {
		s.promote(ctx)
	}
}

func (s *Scheduler) promote(ctx context.Context) {
	sch := asynq.NewSchedulerFromRedisClient(s.q.rdb, &asynq.SchedulerOpts{
		Location: civildate.Tehran,
		Logger:   slogAdapter{s.q.logger},
		LogLevel: asynq.FatalLevel, // enqueue errors are logged by PostEnqueueFunc; shared-client close noise is not
		PostEnqueueFunc: func(info *asynq.TaskInfo, err error) {
			if err != nil && !errors.Is(err, asynq.ErrDuplicateTask) && !errors.Is(err, asynq.ErrTaskIDConflict) {
				s.q.logger.Warn("queue: scheduled enqueue failed", slog.String("error", err.Error()))
			}
		},
	})
	s.mu.Lock()
	for _, e := range s.entries {
		if err := s.addAsynq(sch, e); err != nil {
			s.q.logger.Error("queue: scheduler entry rejected", slog.String("job", e.jobType), slog.String("error", err.Error()))
		}
	}
	boot := s.bootEntries()
	s.mu.Unlock()
	if err := sch.Start(); err != nil {
		s.q.logger.Error("queue: scheduler start failed", slog.String("error", err.Error()))
		_, _ = releaseScript.Run(ctx, s.q.rdb, []string{s.leaderKey()}, s.owner).Result()
		return
	}
	s.mu.Lock()
	s.asynqS, s.leader = sch, true
	s.mu.Unlock()
	s.q.logger.Info("queue: scheduler leader", slog.String("owner", s.owner))
	for _, e := range boot {
		err := s.q.dispatch(ctx, e.jobType, json.RawMessage(e.payload), dispatchOpts{unique: bootUnique(e)})
		if err != nil && !errors.Is(err, ErrDuplicate) {
			s.q.logger.Warn("queue: start-up run enqueue failed", slog.String("job", e.jobType), slog.String("error", err.Error()))
		}
	}
}

// bootUnique dedupes start-up runs of processes that become leader back to back.
func bootUnique(e entry) time.Duration {
	if e.opts.unique > 0 {
		return e.opts.unique
	}
	return time.Minute
}

func (s *Scheduler) demote() {
	s.mu.Lock()
	sch := s.asynqS
	s.asynqS, s.leader = nil, false
	s.mu.Unlock()
	if sch != nil {
		sch.Shutdown()
	}
}

func (s *Scheduler) resign() {
	wasLeader := s.Leader()
	s.demote()
	if wasLeader {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = releaseScript.Run(ctx, s.q.rdb, []string{s.leaderKey()}, s.owner).Result()
	}
}

func (s *Scheduler) bootEntries() []entry {
	var out []entry
	for _, e := range s.entries {
		if e.opts.runAtStart {
			out = append(out, e)
		}
	}
	return out
}

func (s *Scheduler) addAsynq(sch *asynq.Scheduler, e entry) error {
	reg, ok := s.q.lookup(e.jobType)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownJob, e.jobType)
	}
	o := e.opts
	o.taskID = "" // a fixed id would block every later tick
	if _, err := sch.Register(e.spec, asynq.NewTask(e.jobType, e.payload), s.q.taskOptions(reg.job, o)...); err != nil {
		return fmt.Errorf("queue: schedule %s: %w", e.jobType, err)
	}
	return nil
}

func (s *Scheduler) addLocal(c *cron.Cron, e entry) error {
	if _, err := c.AddFunc(e.spec, func() { s.fireLocal(e) }); err != nil {
		return fmt.Errorf("queue: schedule %s: %w", e.jobType, err)
	}
	return nil
}

func (s *Scheduler) fireLocal(e entry) {
	// Dispatch runs the job inline in sync mode and already logs a failure.
	_ = s.q.dispatch(context.Background(), e.jobType, json.RawMessage(e.payload), dispatchOpts{})
}

// RegisterPeriodic registers job with h and schedules it on spec with an empty payload ({}): the usual shape of a
// housekeeping job (docs/engine-v6/jobs.md).
func (q *Queue) RegisterPeriodic(job Job, h Handler, spec string, opts ...Option) error {
	q.Register(job, h)
	return q.scheduler.Register(spec, job.Type, nil, opts...)
}
