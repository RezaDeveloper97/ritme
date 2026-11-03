package compiled

import (
	"context"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/rules"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/configcache"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

var (
	compileErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_rules_compile_errors_total",
		Help: "Message rule rows skipped because they did not compile (counted on every compile of the snapshot).",
	})
	snapshotErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "messages_rules_compile_snapshot_errors_total",
		Help: "Compiles of the message rule snapshot that failed as a whole (the previous snapshot kept serving).",
	})
	revGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "messages_rules_rev",
		Help: "Revision of the compiled message rules snapshot this process serves (bumps on every recompile).",
	})
)

func init() { metrics.Registry().MustRegister(compileErrors, snapshotErrors, revGauge) }

// RowsLoader reads the served rule rows (rules.ActiveRows in production).
type RowsLoader func(ctx context.Context) ([]store.MessageRule, error)

// Cache is the compiled snapshot of one process. It rides on configcache: it reads the message_rules table, so a
// local admin write (configcache.Changed), a pub/sub message from another process (ritme-go:config-changed) or the
// 60 s TTL makes the next Get recompile — one compile at a time (configcache's singleflight), served lock-free from
// an atomic snapshot. A rule that fails to compile is skipped and counted; a failed load or a snapshot-level compile
// failure keeps serving the previous snapshot.
type Cache struct {
	cc  *configcache.Cache[*Compiled]
	rev atomic.Uint64
	cur atomic.Pointer[Compiled]
}

// New builds a cache (configcache options: WithHub, WithTTL).
func New(opts ...configcache.Option) *Cache {
	return &Cache{cc: configcache.New[*Compiled]("message_rules:compiled", []string{rules.Table}, nil, opts...)}
}

// Get returns the current snapshot, recompiling it from load when the rule rows changed.
func (c *Cache) Get(ctx context.Context, load RowsLoader) (*Compiled, error) {
	return c.cc.GetWith(ctx, func(ctx context.Context) (*Compiled, error) {
		rows, err := load(ctx)
		if err != nil {
			return nil, err
		}
		comp, err := Compile(rows)
		if err != nil {
			snapshotErrors.Inc()
			return nil, err
		}
		compileErrors.Add(float64(len(comp.Errors)))
		comp.Rows = rows
		comp.Rev = c.rev.Add(1)
		revGauge.Set(float64(comp.Rev))
		c.cur.Store(comp)
		return comp, nil
	})
}

// Current is the last compiled snapshot without any freshness check (nil before the first Get).
func (c *Cache) Current() *Compiled { return c.cur.Load() }

// Default is the process cache.
var Default = New()

// Load is the process snapshot compiled from rules.ActiveRows (the rows snapshot, approved versions applied).
func Load(ctx context.Context, q store.Querier) (*Compiled, error) {
	return Default.Get(ctx, func(ctx context.Context) ([]store.MessageRule, error) {
		return rules.ActiveRows(ctx, q, "")
	})
}

// Prefetch computes, in one Need pass, every fact selection over slots reads: the slots' NeedFacts and SelectionFacts
// (T-M10-03: each producer group runs once).
func (c *Compiled) Prefetch(ctx context.Context, f *facts.Facts, slots ...string) error {
	return f.Need(ctx, append(c.Need(slots...), SelectionFacts...)...)
}
