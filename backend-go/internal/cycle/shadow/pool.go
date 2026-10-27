package shadow

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// PoolSize is the number of comparisons that may run at once; a job that finds them all busy is dropped.
const PoolSize = 2

// MaxSamples is the length of the Redis sample list.
const MaxSamples = 1000

// SampleTTL is the lifetime of the sample list and the count hash (refreshed on every write).
const SampleTTL = 7 * 24 * time.Hour

// jobTimeout bounds one comparison (the state load may hit the database on a cold cache).
const jobTimeout = 5 * time.Second

// Redis keys (unprefixed; cache.Client adds ritme-go:).
const (
	SamplesKey = "eng:shadow:samples"
	CountsKey  = "eng:shadow:counts"
)

var (
	diffTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "engine_shadow_diff_total",
		Help: "CYCLE_ENGINE=shadow comparisons of the served v1.1 cycle_view with v6, by difference class.",
	}, []string{"class"})
	jobsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "engine_shadow_jobs_total",
		Help: "CYCLE_ENGINE=shadow jobs by result (compared, dropped = pool full, failed).",
	}, []string{"result"})
)

func init() {
	metrics.Registry().MustRegister(diffTotal, jobsTotal)
}

// Compute returns the v6 status of uid's target day (today = the user-local day of the request). The service
// provides it (the same state cache and resolve as CYCLE_ENGINE=v6).
type Compute func(ctx context.Context, uid, rev uint64, target, today engine.Day) (engine.Status, error)

// Job is one served v1 day to compare.
type Job struct {
	// Ctx carries what the state load needs from the request (zone, clock) but no cancellation or request state: the
	// job runs after the response. Nil = context.Background().
	Ctx    context.Context
	UserID uint64
	Rev    uint64
	Target engine.Day
	Today  engine.Day
	// V1 is the served `data` object JSON ({calculation, cycle_view}; owned by the job).
	V1 []byte
}

// Pool runs shadow comparisons. It is safe for concurrent use.
type Pool struct {
	compute Compute
	rdb     *cache.Client // nil = metrics only
	log     *slog.Logger
	key     []byte // HMAC key of the sample user hash, random per process
	slots   chan struct{}
	wg      sync.WaitGroup
}

// New returns a pool over compute; rdb may be nil (no samples).
func New(compute Compute, rdb *cache.Client, logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return &Pool{compute: compute, rdb: rdb, log: logger, key: key, slots: make(chan struct{}, PoolSize)}
}

// Submit starts j when a slot is free and reports whether it did; a full pool drops the job (never blocks).
func (p *Pool) Submit(j Job) bool {
	select {
	case p.slots <- struct{}{}:
	default:
		jobsTotal.WithLabelValues("dropped").Inc()
		return false
	}
	p.wg.Add(1)
	go func() {
		defer func() {
			<-p.slots
			p.wg.Done()
		}()
		defer func() {
			if r := recover(); r != nil {
				jobsTotal.WithLabelValues("failed").Inc()
				p.log.Error("engine shadow: job panicked", slog.Any("panic", r))
			}
		}()
		p.run(j)
	}()
	return true
}

// Wait blocks until every started job has finished (tests, shutdown).
func (p *Pool) Wait() { p.wg.Wait() }

func (p *Pool) run(j Job) {
	ctx := j.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()
	st, err := p.compute(ctx, j.UserID, j.Rev, j.Target, j.Today)
	if err != nil {
		jobsTotal.WithLabelValues("failed").Inc()
		p.log.WarnContext(ctx, "engine shadow: v6 compute failed", slog.Any("error", err))
		return
	}
	d, err := Classify(j.V1, &st)
	if err != nil {
		jobsTotal.WithLabelValues("failed").Inc()
		p.log.WarnContext(ctx, "engine shadow: classify failed", slog.Any("error", err))
		return
	}
	jobsTotal.WithLabelValues("compared").Inc()
	for _, c := range d.Classes {
		diffTotal.WithLabelValues(string(c)).Inc()
	}
	if err := p.record(ctx, j, d); err != nil {
		p.log.WarnContext(ctx, "engine shadow: sample not stored", slog.Any("error", err))
	}
}

// Sample is one stored comparison.
type Sample struct {
	At string `json:"at"` // UTC, minute precision
	// User is an HMAC-SHA256 of the user id under a per-process random key (16 hex): equal within one process run,
	// never reversible to the id.
	User string `json:"user"`
	// DateOffset is target − today in days (0 = /cycle/today); no absolute dates are stored.
	DateOffset int `json:"date_offset"`
	Diff
}

func (p *Pool) userHash(uid uint64) string {
	m := hmac.New(sha256.New, p.key)
	_, _ = m.Write([]byte(strconv.FormatUint(uid, 10)))
	return hex.EncodeToString(m.Sum(nil)[:8])
}

func (p *Pool) record(ctx context.Context, j Job, d Diff) error {
	if p.rdb == nil {
		return nil
	}
	s := Sample{
		At:         time.Now().UTC().Truncate(time.Minute).Format(time.RFC3339),
		User:       p.userHash(j.UserID),
		DateOffset: int(j.Target - j.Today),
		Diff:       d,
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("encode sample: %w", err)
	}
	samples, counts := p.rdb.Key(SamplesKey), p.rdb.Key(CountsKey)
	pipe := p.rdb.Redis().TxPipeline()
	pipe.LPush(ctx, samples, raw)
	pipe.LTrim(ctx, samples, 0, MaxSamples-1)
	pipe.Expire(ctx, samples, SampleTTL)
	pipe.HIncrBy(ctx, counts, "compared", 1)
	for _, c := range d.Classes {
		pipe.HIncrBy(ctx, counts, string(c), 1)
	}
	pipe.Expire(ctx, counts, SampleTTL)
	_, err = pipe.Exec(ctx)
	return err
}
