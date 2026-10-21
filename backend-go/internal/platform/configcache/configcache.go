// Package configcache holds small admin-edited tables (message contents, recommendations, message rules, challenges,
// plus plans, app modules, …) in process memory, so the request path stops re-reading them (T-M9-13, gap 07 §7.2).
//
// A Cache[T] is an atomic snapshot plus a loader and a content fingerprint. Get is lock-free while the snapshot is
// fresh. A snapshot goes stale, and the next Get reloads it (one load at a time per cache: callers that arrive during
// a reload wait for that same load), when
//
//  1. an admin write in this process called Changed(table) after its commit (local epoch bump),
//  2. another process published the table on the Redis channel ritme-go:config-changed (Start subscribes), or
//  3. the TTL safety net (60 s) ran out — the only path while Redis is down or pub/sub is not started.
//
// A failed reload keeps serving the previous snapshot and retries after retryBackoff; with no snapshot yet the error
// goes to the caller (the same failure a direct read would have had).
//
// The process-wide Hub (Default) is disabled inside `go test` binaries, so integration tests that reset tables between
// tests keep reading the database; tests of the cache itself call SetEnabled(true).
package configcache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// Channel is the pub/sub channel (unprefixed; published as cache.Client.Key(Channel) = "ritme-go:config-changed").
const Channel = "config-changed"

// DefaultTTL is the safety net: a snapshot older than this is reloaded even without an invalidation.
const DefaultTTL = 60 * time.Second

const (
	loadTimeout    = 5 * time.Second
	publishTimeout = 2 * time.Second
	retryBackoff   = 5 * time.Second
)

// Hub owns the per-table epochs, the fingerprints of the loaded snapshots and the pub/sub link. One per process
// (Default); tests build more to play a second process.
type Hub struct {
	id      string
	enabled atomic.Bool
	ttl     time.Duration
	now     func() time.Time

	mu     sync.Mutex
	epochs map[string]*atomic.Uint64
	fps    map[string]map[string]string // table → cache name → fingerprint

	bus    atomic.Pointer[cache.Client]
	logger atomic.Pointer[slog.Logger]
}

// NewHub builds an enabled hub with the default TTL.
func NewHub() *Hub { return newHub(DefaultTTL) }

func newHub(ttl time.Duration) *Hub {
	var b [8]byte
	_, _ = rand.Read(b[:])
	h := &Hub{id: hex.EncodeToString(b[:]), ttl: ttl, now: time.Now, epochs: map[string]*atomic.Uint64{}, fps: map[string]map[string]string{}}
	h.enabled.Store(true)
	h.logger.Store(slog.New(slog.DiscardHandler))
	return h
}

// Default is the process hub every cache uses unless WithHub says otherwise. It starts disabled under `go test`.
var Default = func() *Hub {
	h := NewHub()
	h.enabled.Store(!testing.Testing())
	return h
}()

// SetEnabled turns caching on or off; off = every Get calls the loader (the pre-cache behaviour).
func (h *Hub) SetEnabled(on bool) { h.enabled.Store(on) }

// Enabled reports whether caching is on.
func (h *Hub) Enabled() bool { return h.enabled.Load() }

// SetEnabled turns the Default hub on or off (tests).
func SetEnabled(on bool) { Default.SetEnabled(on) }

func (h *Hub) epoch(table string) *atomic.Uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	e, ok := h.epochs[table]
	if !ok {
		e = new(atomic.Uint64)
		h.epochs[table] = e
	}
	return e
}

// Invalidate marks every cache reading one of tables stale in this process only (the next Get reloads).
func (h *Hub) Invalidate(tables ...string) {
	for _, t := range tables {
		h.epoch(t).Add(1)
	}
}

// Changed is what an admin writer calls after its commit: invalidate here, then tell the other processes. A publish
// failure is logged only (they pick the change up within the TTL).
func (h *Hub) Changed(ctx context.Context, tables ...string) {
	if len(tables) == 0 {
		return
	}
	h.Invalidate(tables...)
	c := h.bus.Load()
	if c == nil {
		return
	}
	payload, _ := json.Marshal(message{Origin: h.id, Tables: tables})
	pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), publishTimeout)
	defer cancel()
	if err := c.Redis().Publish(pctx, c.Key(Channel), payload).Err(); err != nil {
		h.logger.Load().WarnContext(ctx, "configcache: publish failed; other processes see the change within the TTL",
			slog.String("tables", strings.Join(tables, ",")), slog.String("error", err.Error()))
	}
}

// Changed is Default.Changed.
func Changed(ctx context.Context, tables ...string) { Default.Changed(ctx, tables...) }

// message is the pub/sub payload.
type message struct {
	Origin string   `json:"origin"`
	Tables []string `json:"tables"`
}

func (h *Hub) receive(payload string) {
	var m message
	if json.Unmarshal([]byte(payload), &m) != nil || m.Origin == h.id {
		return
	}
	h.Invalidate(m.Tables...)
}

// Start links the hub to Redis: Changed publishes on ritme-go:config-changed, and a goroutine applies what the other
// processes publish until ctx ends. go-redis re-subscribes by itself after a disconnect; while Redis is down only the
// TTL refreshes. c nil = local invalidation + TTL only.
func (h *Hub) Start(ctx context.Context, c *cache.Client, logger *slog.Logger) {
	if logger != nil {
		h.logger.Store(logger)
	}
	if c == nil {
		return
	}
	h.bus.Store(c)
	ps := c.Redis().Subscribe(ctx, c.Key(Channel))
	ch := ps.Channel()
	go func() {
		defer func() { _ = ps.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				h.receive(m.Payload)
			}
		}
	}()
}

// Start is Default.Start: call it once at start-up, after the Redis client exists.
func Start(ctx context.Context, c *cache.Client, logger *slog.Logger) { Default.Start(ctx, c, logger) }

func (h *Hub) recordFingerprint(tables []string, name, fp string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, t := range tables {
		m, ok := h.fps[t]
		if !ok {
			m = map[string]string{}
			h.fps[t] = m
		}
		m[name] = fp
	}
}

// Fingerprint digests the content of every loaded snapshot reading one of tables (for ETags, T-M9-31). It changes
// only when the loaded content changes; a table nothing has loaded yet contributes nothing.
func (h *Hub) Fingerprint(tables ...string) string {
	ts := append([]string(nil), tables...)
	sort.Strings(ts)
	sum := sha256.New()
	h.mu.Lock()
	for _, t := range ts {
		names := make([]string, 0, len(h.fps[t]))
		for n := range h.fps[t] {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			_, _ = sum.Write([]byte(t + "|" + n + "|" + h.fps[t][n] + "\n"))
		}
	}
	h.mu.Unlock()
	return hex.EncodeToString(sum.Sum(nil)[:12])
}

// Fingerprint is Default.Fingerprint.
func Fingerprint(tables ...string) string { return Default.Fingerprint(tables...) }

// Option tunes one cache.
type Option func(*settings)

type settings struct {
	hub *Hub
	ttl time.Duration
	fp  func(any) string
}

// WithHub binds the cache to h instead of Default.
func WithHub(h *Hub) Option { return func(s *settings) { s.hub = h } }

// WithTTL overrides the hub's TTL safety net.
func WithTTL(d time.Duration) Option { return func(s *settings) { s.ttl = d } }

// jsonFingerprint is the default fingerprint: SHA-256 of the JSON encoding (maps encode with sorted keys).
func jsonFingerprint(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:12])
}

// Loader reads the current content.
type Loader[T any] func(ctx context.Context) (T, error)

type snapshot[T any] struct {
	val    T
	fp     string
	at     time.Time
	epochs []uint64
}

type flight[T any] struct {
	done chan struct{}
	val  T
	err  error
}

// Cache is one cached value of type T, read from tables.
type Cache[T any] struct {
	name   string
	tables []string
	load   Loader[T]
	hub    *Hub
	ttl    time.Duration
	fp     func(any) string
	epochs []*atomic.Uint64

	snap    atomic.Pointer[snapshot[T]]
	retryAt atomic.Int64 // unix nanos: no reload attempt before this after a failure

	mu       sync.Mutex
	inflight *flight[T]
}

// New builds a cache. name labels its fingerprint (unique per content); load may be nil when every call uses GetWith.
func New[T any](name string, tables []string, load Loader[T], opts ...Option) *Cache[T] {
	s := settings{hub: Default, fp: jsonFingerprint}
	for _, o := range opts {
		o(&s)
	}
	if s.ttl <= 0 {
		s.ttl = s.hub.ttl
	}
	c := &Cache[T]{name: name, tables: append([]string(nil), tables...), load: load, hub: s.hub, ttl: s.ttl, fp: s.fp}
	for _, t := range c.tables {
		c.epochs = append(c.epochs, s.hub.epoch(t))
	}
	return c
}

// Get returns the snapshot, reloading it through the cache's loader when stale.
func (c *Cache[T]) Get(ctx context.Context) (T, error) { return c.GetWith(ctx, c.load) }

// GetWith is Get with the loader of this call (for request-scoped sources such as a per-request sqlc handle). Every
// loader of one cache must read the same content.
func (c *Cache[T]) GetWith(ctx context.Context, load Loader[T]) (T, error) {
	if !c.hub.enabled.Load() {
		return load(ctx)
	}
	s := c.snap.Load()
	if s != nil && c.fresh(s, c.hub.now()) {
		return s.val, nil
	}
	return c.reload(ctx, load)
}

// Fingerprint is the fingerprint of the current snapshot ("" before the first load).
func (c *Cache[T]) Fingerprint() string {
	if s := c.snap.Load(); s != nil {
		return s.fp
	}
	return ""
}

func (c *Cache[T]) fresh(s *snapshot[T], now time.Time) bool {
	stale := now.Sub(s.at) >= c.ttl
	for i, e := range c.epochs {
		if e.Load() != s.epochs[i] {
			stale = true
			break
		}
	}
	if !stale {
		return true
	}
	// After a failed reload, serve the old snapshot until the backoff ends instead of hammering a broken DB.
	return now.UnixNano() < c.retryAt.Load()
}

func (c *Cache[T]) reload(ctx context.Context, load Loader[T]) (T, error) {
	c.mu.Lock()
	if f := c.inflight; f != nil {
		c.mu.Unlock()
		select {
		case <-f.done:
			return f.val, f.err
		case <-ctx.Done():
			if s := c.snap.Load(); s != nil {
				return s.val, nil
			}
			var zero T
			return zero, ctx.Err()
		}
	}
	if s := c.snap.Load(); s != nil && c.fresh(s, c.hub.now()) { // a flight finished while we waited for mu
		c.mu.Unlock()
		return s.val, nil
	}
	f := &flight[T]{done: make(chan struct{})}
	c.inflight = f
	c.mu.Unlock()

	defer func() {
		close(f.done)
		c.mu.Lock()
		c.inflight = nil
		c.mu.Unlock()
	}()

	epochs := make([]uint64, len(c.epochs)) // read before loading: a change during the load stays pending
	for i, e := range c.epochs {
		epochs[i] = e.Load()
	}
	lctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), loadTimeout)
	defer cancel()
	v, err := load(lctx)
	if err != nil {
		if old := c.snap.Load(); old != nil {
			c.retryAt.Store(c.hub.now().Add(retryBackoff).UnixNano())
			c.hub.logger.Load().WarnContext(ctx, "configcache: reload failed, serving the previous snapshot",
				slog.String("cache", c.name), slog.String("error", err.Error()))
			f.val = old.val
			return f.val, nil
		}
		f.err = err
		return f.val, err
	}
	fp := c.fp(v)
	c.snap.Store(&snapshot[T]{val: v, fp: fp, at: c.hub.now(), epochs: epochs})
	c.retryAt.Store(0)
	c.hub.recordFingerprint(c.tables, c.name, fp)
	f.val = v
	return v, nil
}

// Keyed is a family of caches over the same tables, one per key (message contents per locale).
type Keyed[T any] struct {
	name   string
	tables []string
	opts   []Option
	m      sync.Map // key → *Cache[T]
}

// NewKeyed builds an empty family; each key's cache is named name+":"+key.
func NewKeyed[T any](name string, tables []string, opts ...Option) *Keyed[T] {
	return &Keyed[T]{name: name, tables: tables, opts: opts}
}

// Get is the key's cache GetWith(load).
func (k *Keyed[T]) Get(ctx context.Context, key string, load Loader[T]) (T, error) {
	if v, ok := k.m.Load(key); ok {
		return v.(*Cache[T]).GetWith(ctx, load) //nolint:forcetypeassert // only *Cache[T] is stored
	}
	v, _ := k.m.LoadOrStore(key, New[T](k.name+":"+key, k.tables, nil, k.opts...))
	return v.(*Cache[T]).GetWith(ctx, load) //nolint:forcetypeassert // only *Cache[T] is stored
}
