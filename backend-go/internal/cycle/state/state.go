// Package state provides prepared engine v6 states (*engine.State) through two cache layers (T-M9-31, DECISIONS.md
// X-02, docs/engine-v6/README.md §2–§3):
//
//	L1  in-process LRU of prepared states, keyed (uid, rev, paramsFP, build)
//	L2  Redis ritme-go:eng:in:{uid}:{rev} = the versioned engine.Input snapshot (inputs.Encode), TTL 7 d
//	DB  inputs.Load (≤ 4 queries)
//
// The revision is DB-authoritative (user_revisions.engine_rev, bumped inside every engine-input write transaction,
// T-M9-30) and read for free in the auth query, so a write can never be served stale and a Redis flush only costs
// reloads. L2 holds the params-independent input, so a clinical-config change (a new params fingerprint) re-prepares
// from L2 without touching the database. A singleflight on uid:rev collapses the concurrent app-open requests of one
// user into one load and one Prepare.
//
// States are immutable: the engine never mutates a prepared State, so one *engine.State is shared read-only by
// every request that gets it. Callers must not modify the slices its accessors return.
package state

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"reflect"
	"strconv"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/singleflight"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/inputs"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/config"
)

// Cache names in cache_requests_total{cache,result}.
const (
	CacheL1 = "engine_l1"
	CacheL2 = "engine_l2"
)

// L2TTL is the lifetime of an input snapshot in Redis. A user who opens the app at least weekly never reloads from
// the database between two of her own writes.
const L2TTL = 7 * 24 * time.Hour

// loadTimeout bounds one shared load: the flight runs detached from the leader's cancellation (a follower must not
// fail because the first caller gave up), so it carries its own deadline.
const loadTimeout = 5 * time.Second

// Params are the engine parameters a state is prepared with, and their fingerprint (engineparams.For /
// engineparams.Fingerprint of the user-local day). FP is part of the L1 key; equal FPs must mean equal params.
type Params struct {
	Engine engine.Params
	FP     string
}

// Options configure a Provider; start from DefaultOptions (or OptionsFrom). A zero L1Entries means the default size.
type Options struct {
	// Enabled false (CYCLE_ENGINE_CACHE=off) bypasses L1 and L2: every call loads from the database and prepares.
	Enabled bool
	// L1Entries is the LRU size (ENGINE_L1_ENTRIES).
	L1Entries int
	// Verify rebuilds the state from the database on a cache hit and counts engine_cache_verify_mismatch_total when
	// it differs (the fresh state is then served and cached). VerifySample is the share of hits verified (0..1).
	Verify       bool
	VerifySample float64
	// Build identifies the binary (etag.Build()); part of the L1 key.
	Build string
	// Logger receives cache errors and verify mismatches (slog.Default when nil).
	Logger *slog.Logger
}

// DefaultOptions are the config defaults: caches on, 20 000 L1 entries, verify off — except under `go test`, where
// verify is on for every hit (README §9: all tests run with ENGINE_CACHE_VERIFY=1).
func DefaultOptions() Options {
	return Options{
		Enabled: true, L1Entries: config.DefaultEngineL1Entries,
		Verify: testing.Testing(), VerifySample: config.DefaultEngineVerifySample,
	}
}

// OptionsFrom maps the loaded configuration (ENGINE_CACHE_VERIFY may only add verification, never remove it from
// a test binary).
func OptionsFrom(c config.EngineCache, build string, logger *slog.Logger) Options {
	o := DefaultOptions()
	o.Enabled = c.Enabled
	o.L1Entries = c.L1Entries
	o.Verify = o.Verify || c.Verify
	o.VerifySample = c.VerifySample
	o.Build = build
	o.Logger = logger
	return o
}

// KV is the part of Redis the provider uses (*cache.Client satisfies it; keys are unprefixed, Get returns
// cache.ErrMiss for an absent key).
type KV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

// Provider hands out prepared states. It is safe for concurrent use.
type Provider struct {
	q      inputs.Querier
	kv     KV // nil = no L2
	clock  clock.Clock
	opts   Options
	l1     *lru.Cache[l1Key, *engine.State]
	flight singleflight.Group
	log    *slog.Logger

	// Seams for tests (export_test.go).
	prepare func(engine.Input, engine.Params) *engine.State
	sample  func() float64
}

type l1Key struct {
	uid, rev uint64
	fp       string
	build    string
}

// New builds a provider over the cycle store q and the Redis client kv (nil → no L2). clk is the fallback clock when
// the request context carries none (clock.Real{} in production).
func New(q inputs.Querier, kv KV, clk clock.Clock, opts Options) (*Provider, error) {
	if opts.L1Entries <= 0 {
		opts.L1Entries = config.DefaultEngineL1Entries
	}
	if opts.VerifySample < 0 || opts.VerifySample > 1 {
		return nil, fmt.Errorf("state: verify sample %v not in 0..1", opts.VerifySample)
	}
	if clk == nil {
		clk = clock.Real{}
	}
	l1, err := lru.New[l1Key, *engine.State](opts.L1Entries)
	if err != nil {
		return nil, fmt.Errorf("state: l1: %w", err)
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &Provider{
		q: q, kv: kv, clock: clk, opts: opts, l1: l1, log: log,
		prepare: engine.Prepare, sample: rand.Float64, //nolint:gosec // G404: sampling, not security
	}, nil
}

// For returns uid's engine state at engine revision rev, prepared with p. rev comes from the auth query
// (auth.CurrentRevisions); the user-local day of ctx (civildate.LocalToday) bounds the marker window of a database
// load. The returned state is shared: never modify it.
func (pr *Provider) For(ctx context.Context, uid, rev uint64, p Params) (*engine.State, error) {
	if !pr.opts.Enabled {
		in, err := pr.loadDB(ctx, uid)
		if err != nil {
			return nil, err
		}
		return pr.build(in, p, sourceDB), nil
	}

	key := l1Key{uid: uid, rev: rev, fp: p.FP, build: pr.opts.Build}
	if st, ok := pr.l1.Get(key); ok {
		cache.Hit(CacheL1)
		return pr.maybeVerify(ctx, uid, rev, p, key, st), nil
	}
	cache.Miss(CacheL1)

	r, err, _ := pr.flight.Do(flightKey(uid, rev), func() (any, error) {
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), loadTimeout)
		defer cancel()
		return pr.fill(fctx, uid, rev, p, key)
	})
	if err != nil {
		return nil, err
	}
	res := r.(*flightResult)
	st := res.state
	if res.fp != p.FP {
		// A concurrent caller with other params led the flight (a config change in between): same input, our params.
		if !res.hasInput {
			// The leader found its own state in L1, so no input to re-prepare; load for ourselves (rare).
			if res, err = pr.fill(ctx, uid, rev, p, key); err != nil {
				return nil, err
			}
			st = res.state
		} else {
			st = pr.build(res.input, p, sourceShared)
			pr.l1.Add(key, st)
		}
	}
	if res.source == sourceDB {
		return st, nil
	}
	return pr.maybeVerify(ctx, uid, rev, p, key, st), nil
}

// Source of a freshly built state (engine_build_seconds{source}).
const (
	sourceDB     = "db"
	sourceL2     = "l2"
	sourceShared = "shared"
	sourceL1     = "l1" // not a build: the flight found the state in L1
)

// flightResult is what one singleflight load hands to every waiter: the normalized input (to re-prepare with other
// params), the leader's params fingerprint and state, and where the input came from.
type flightResult struct {
	input    engine.Input
	hasInput bool // false when the leader found its state in L1
	state    *engine.State
	fp       string
	source   string // sourceDB | sourceL2 | sourceL1 (another flight filled L1 just before)
}

func flightKey(uid, rev uint64) string {
	return strconv.FormatUint(uid, 10) + ":" + strconv.FormatUint(rev, 10)
}

// l2Key is the unprefixed Redis key of uid's input snapshot at rev (cache.Client adds ritme-go:).
func l2Key(uid, rev uint64) string {
	return "eng:in:" + strconv.FormatUint(uid, 10) + ":" + strconv.FormatUint(rev, 10)
}

// fill runs once per uid:rev at a time: L1 again (a flight that just finished), then L2, then the database.
func (pr *Provider) fill(ctx context.Context, uid, rev uint64, p Params, key l1Key) (*flightResult, error) {
	if st, ok := pr.l1.Peek(key); ok {
		return &flightResult{state: st, fp: p.FP, source: sourceL1}, nil
	}
	if in, ok := pr.readL2(ctx, uid, rev); ok {
		st := pr.build(in, p, sourceL2)
		pr.l1.Add(key, st)
		return &flightResult{input: in, hasInput: true, state: st, fp: p.FP, source: sourceL2}, nil
	}
	raw, err := pr.loadDB(ctx, uid)
	if err != nil {
		return nil, err
	}
	in, snap, err := normalize(raw)
	if err != nil {
		return nil, err
	}
	pr.writeL2(ctx, uid, rev, snap)
	st := pr.build(in, p, sourceDB)
	pr.l1.Add(key, st)
	return &flightResult{input: in, hasInput: true, state: st, fp: p.FP, source: sourceDB}, nil
}

// readL2 returns the snapshot at uid:rev. A Redis error, a corrupt value or another snapshot version is a miss
// (counted as an error) and the caller reloads from the database.
func (pr *Provider) readL2(ctx context.Context, uid, rev uint64) (engine.Input, bool) {
	if pr.kv == nil {
		return engine.Input{}, false
	}
	v, err := pr.kv.Get(ctx, l2Key(uid, rev))
	switch {
	case errors.Is(err, cache.ErrMiss):
		cache.Miss(CacheL2)
		return engine.Input{}, false
	case err != nil:
		cache.Error(CacheL2)
		pr.log.WarnContext(ctx, "engine state: l2 read failed, loading from the database", slog.Any("error", err))
		return engine.Input{}, false
	}
	in, err := inputs.Decode([]byte(v))
	if err != nil {
		cache.Error(CacheL2)
		pr.log.WarnContext(ctx, "engine state: l2 snapshot unreadable, reloading", slog.Any("error", err))
		return engine.Input{}, false
	}
	cache.Hit(CacheL2)
	return in, true
}

func (pr *Provider) writeL2(ctx context.Context, uid, rev uint64, snap []byte) {
	if pr.kv == nil {
		return
	}
	if err := pr.kv.Set(ctx, l2Key(uid, rev), snap, L2TTL); err != nil {
		pr.log.WarnContext(ctx, "engine state: l2 write failed", slog.Any("error", err))
	}
}

func (pr *Provider) loadDB(ctx context.Context, uid uint64) (engine.Input, error) {
	today := civildate.LocalToday(ctx, clock.FromContext(ctx, pr.clock))
	in, err := inputs.Load(ctx, pr.q, uid, today)
	if err != nil {
		return engine.Input{}, fmt.Errorf("state: load inputs: %w", err)
	}
	return in, nil
}

// normalize round-trips a loaded input through the L2 snapshot format, so a state prepared from the database is
// identical to one prepared from L2 (empty lists are nil in both) and verify can compare states exactly.
func normalize(in engine.Input) (engine.Input, []byte, error) {
	snap, err := inputs.Encode(in)
	if err != nil {
		return engine.Input{}, nil, fmt.Errorf("state: encode snapshot: %w", err)
	}
	out, err := inputs.Decode(snap)
	if err != nil {
		return engine.Input{}, nil, fmt.Errorf("state: decode snapshot: %w", err)
	}
	return out, snap, nil
}

func (pr *Provider) build(in engine.Input, p Params, source string) *engine.State {
	start := time.Now()
	st := pr.prepare(in, p.Engine)
	buildSeconds.WithLabelValues(source).Observe(time.Since(start).Seconds())
	return st
}

// maybeVerify is verify mode on a cache hit: rebuild from the database and compare. A mismatch is counted and
// logged, and the fresh state replaces L1 and L2 and is served. A failed rebuild keeps the cached state.
func (pr *Provider) maybeVerify(ctx context.Context, uid, rev uint64, p Params, key l1Key, cached *engine.State) *engine.State {
	if !pr.opts.Verify || pr.opts.VerifySample <= 0 || (pr.opts.VerifySample < 1 && pr.sample() >= pr.opts.VerifySample) {
		return cached
	}
	raw, err := pr.loadDB(ctx, uid)
	if err != nil {
		pr.log.WarnContext(ctx, "engine state: verify load failed", slog.Any("error", err))
		return cached
	}
	in, snap, err := normalize(raw)
	if err != nil {
		pr.log.WarnContext(ctx, "engine state: verify normalize failed", slog.Any("error", err))
		return cached
	}
	fresh := pr.prepare(in, p.Engine)
	if reflect.DeepEqual(fresh, cached) {
		return cached
	}
	verifyMismatch.Inc()
	pr.log.ErrorContext(ctx, "engine state: cached state differs from the database",
		slog.Uint64("user_id", uid), slog.Uint64("engine_rev", rev), slog.String("params_fp", p.FP))
	pr.l1.Add(key, fresh)
	pr.writeL2(ctx, uid, rev, snap)
	return fresh
}
