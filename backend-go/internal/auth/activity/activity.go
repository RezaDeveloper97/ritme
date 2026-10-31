// Package activity marks per-user activity days (T-M10-30, SPEC v6 §5.6, DECISIONS.md A-06, D-179):
// user_activity_days.session = 1 on a user's first authenticated request of the user-local day (auth.Guard.RequireUser)
// and logged = 1 on the first engine/analysis-input write of the day (internal/cycle/revision Bump / BumpAnalysis).
//
// The request path never waits: Session / Logged only check an in-process "already marked" set and, on a miss, hand
// the mark to a small buffered writer that drops it (counted) when the buffer is full. The writer, off the request:
// resolves the user's zone (the request's X-Timezone hint → the stored user_profiles.timezone → Asia/Tehran, as usertz),
// claims the day with Redis `SET NX ritme-go:seen:{uid}:{day}` (26 h; `ritme-go:logged:{uid}:{day}` for logs) so the
// API replicas upsert once a day between them, then upserts the row (idempotent). Redis down → upsert anyway.
//
// So a user costs no query and no Redis call on any request after the first one of the day (per process), and the
// auth hot path keeps its single query (T-M9-09).
package activity

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ritme/backend-go/internal/metrics/store"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

// Kind is what a mark records.
type Kind uint8

const (
	// Session is user_activity_days.session (an authenticated request).
	Session Kind = iota
	// Logged is user_activity_days.logged (an engine- or analysis-input write).
	Logged
)

const (
	// Buffer is the writer's queue length; a full queue drops the mark.
	Buffer = 4096
	// SeenTTL is the Redis claim's lifetime (a little over a day, any zone).
	SeenTTL = 26 * time.Hour
	// zoneTTL bounds how long a resolved stored zone is trusted.
	zoneTTL = 15 * time.Minute
	// maxSeen / maxZones bound the in-process sets (they are dropped when full).
	maxSeen  = 200_000
	maxZones = 100_000
	// writeTimeout bounds one mark's Redis + DB work.
	writeTimeout = 5 * time.Second
	// TimezoneHeader is the device zone header (usertz.Header).
	TimezoneHeader = "X-Timezone"
)

var (
	dropped = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "activity_marks_dropped_total",
		Help: "Activity-day marks dropped because the writer's buffer was full or closed.",
	})
	writeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "activity_marks_write_errors_total",
		Help: "Activity-day upserts that failed (logged; retried on the user's next request).",
	})
)

func init() { metrics.Registry().MustRegister(dropped, writeErrors) }

// Store is the data the writer needs (satisfied by *store.Queries).
type Store interface {
	MarkUserSessionDay(ctx context.Context, arg store.MarkUserSessionDayParams) error
	MarkUserLoggedDay(ctx context.Context, arg store.MarkUserLoggedDayParams) error
	GetActivityUserTimezone(ctx context.Context, userID uint64) (sql.NullString, error)
}

type item struct {
	uid  uint64
	kind Kind
	at   time.Time
	zone *time.Location // nil: resolve the stored zone
	ack  chan struct{}  // a Flush marker, not a mark
}

type seenKey struct {
	uid  uint64
	day  civildate.Date
	kind Kind
}

type zoneEntry struct {
	loc *time.Location
	at  time.Time
}

// Marker is the per-process activity writer.
type Marker struct {
	store  Store
	logger *slog.Logger

	mu    sync.RWMutex // guards cache, seen, zones, closed
	cache *cache.Client
	seen  map[seenKey]struct{}
	zones map[uint64]zoneEntry

	closed bool
	ch     chan item
	done   chan struct{}
	once   sync.Once
}

// New starts a marker over st (Redis optional: nil = DB upserts only, deduped per process).
func New(st Store, c *cache.Client, logger *slog.Logger) *Marker {
	if logger == nil {
		logger = slog.Default()
	}
	m := &Marker{store: st, cache: c, logger: logger, seen: map[seenKey]struct{}{}, zones: map[uint64]zoneEntry{},
		ch: make(chan item, Buffer), done: make(chan struct{})}
	go m.run()
	return m
}

// stateKey holds the app's marker in its State.
const stateKey = "ritme.activity"

var installMu sync.Mutex

// Install builds the app's marker over db (once per *fiber.App) and drains it on shutdown; a later call with a cache
// attaches it. Guards read the marker from the app on each request, so the call order of the registrars does not
// matter. On a router that is not the *fiber.App, or without a DB, it does nothing and returns nil.
func Install(r fiber.Router, db *sql.DB, c *cache.Client, logger *slog.Logger) *Marker {
	app, ok := r.(*fiber.App)
	if !ok || db == nil {
		return nil
	}
	installMu.Lock()
	defer installMu.Unlock()
	if m := From(app); m != nil {
		if c != nil {
			m.mu.Lock()
			m.cache = c
			m.mu.Unlock()
		}
		return m
	}
	m := New(store.New(db), c, logger)
	app.State().Set(stateKey, m)
	app.Hooks().OnPreShutdown(func() error {
		m.Close()
		return nil
	})
	return m
}

// From is the app's marker (nil when none is installed).
func From(app *fiber.App) *Marker {
	if app == nil {
		return nil
	}
	v, ok := app.State().Get(stateKey)
	if !ok {
		return nil
	}
	m, _ := v.(*Marker)
	return m
}

// Mark records kind for uid at the request clock of ctx, in zone (nil = the stored zone). It never blocks and never
// fails: a repeat of the day is a set lookup; otherwise the mark is queued, or dropped when the queue is full.
func (m *Marker) Mark(ctx context.Context, uid uint64, kind Kind, zone *time.Location) {
	if m == nil || uid == 0 {
		return
	}
	at := clock.FromContext(ctx, clock.Real{}).Now()
	loc := zone
	if loc == nil {
		loc = m.knownZone(uid, at)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		dropped.Inc()
		return
	}
	if loc != nil {
		if _, ok := m.seen[seenKey{uid, civildate.FromTime(at.In(loc)), kind}]; ok {
			return
		}
	}
	select {
	case m.ch <- item{uid: uid, kind: kind, at: at, zone: zone}:
	default:
		dropped.Inc()
	}
}

// Close stops taking marks and waits until the queued ones are written. Idempotent.
func (m *Marker) Close() {
	if m == nil {
		return
	}
	m.once.Do(func() {
		m.mu.Lock()
		m.closed = true
		close(m.ch)
		m.mu.Unlock()
	})
	<-m.done
}

// Flush waits until every mark queued so far is written (tests, shutdown checks).
func (m *Marker) Flush() {
	ack := make(chan struct{})
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return
	}
	m.ch <- item{ack: ack}
	m.mu.RUnlock()
	<-ack
}

func (m *Marker) run() {
	defer close(m.done)
	for it := range m.ch {
		if it.ack != nil {
			close(it.ack)
			continue
		}
		m.write(it)
	}
}

func (m *Marker) write(it item) {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()
	loc := it.zone
	if loc == nil {
		loc = m.storedZone(ctx, it.uid, it.at)
	}
	day := civildate.FromTime(it.at.In(loc))
	key := seenKey{it.uid, day, it.kind}
	m.mu.RLock()
	_, done := m.seen[key]
	c := m.cache
	m.mu.RUnlock()
	if done {
		return
	}
	claim := ""
	if c != nil {
		claim = c.Key(claimKey(it.uid, day, it.kind))
		ok, err := c.Redis().SetNX(ctx, claim, "1", SeenTTL).Result()
		switch {
		case err != nil:
			claim = "" // Redis down: the idempotent upsert alone
		case !ok:
			m.remember(key) // another process already wrote it
			return
		}
	}
	var err error
	if it.kind == Logged {
		err = m.store.MarkUserLoggedDay(ctx, store.MarkUserLoggedDayParams{UserID: it.uid, Day: day})
	} else {
		err = m.store.MarkUserSessionDay(ctx, store.MarkUserSessionDayParams{UserID: it.uid, Day: day})
	}
	if err != nil {
		writeErrors.Inc()
		m.logger.Warn("activity: mark day", slog.Uint64("user_id", it.uid), slog.String("error", err.Error()))
		if claim != "" {
			_ = c.Redis().Del(ctx, claim).Err() // let the next request try again
		}
		return
	}
	m.remember(key)
}

func claimKey(uid uint64, day civildate.Date, kind Kind) string {
	prefix := "seen:"
	if kind == Logged {
		prefix = "logged:"
	}
	return prefix + strconv.FormatUint(uid, 10) + ":" + day.String()
}

func (m *Marker) remember(k seenKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.seen) >= maxSeen {
		m.seen = map[seenKey]struct{}{}
	}
	m.seen[k] = struct{}{}
}

func (m *Marker) knownZone(uid uint64, now time.Time) *time.Location {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if z, ok := m.zones[uid]; ok && now.Sub(z.at) < zoneTTL && z.at.Sub(now) < zoneTTL {
		return z.loc
	}
	return nil
}

// storedZone is the user's stored zone (remembered, else one SELECT off the request path); errors → Tehran.
func (m *Marker) storedZone(ctx context.Context, uid uint64, now time.Time) *time.Location {
	if loc := m.knownZone(uid, now); loc != nil {
		return loc
	}
	loc := civildate.Tehran
	name, err := m.store.GetActivityUserTimezone(ctx, uid)
	switch {
	case err == nil && name.Valid:
		if z := ParseZone(name.String); z != nil {
			loc = z
		}
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		m.logger.Warn("activity: read stored zone", slog.Uint64("user_id", uid), slog.String("error", err.Error()))
		return loc // not remembered: try again next time
	}
	m.mu.Lock()
	if len(m.zones) >= maxZones {
		m.zones = map[uint64]zoneEntry{}
	}
	m.zones[uid] = zoneEntry{loc: loc, at: now}
	m.mu.Unlock()
	return loc
}

var zones sync.Map // valid IANA name → *time.Location

// ParseZone validates an IANA zone name the way usertz.Parse does (which this package cannot import: usertz imports
// auth) and returns its location, or nil.
func ParseZone(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 || name == "Local" || strings.Contains(name, "..") {
		return nil
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '/', r == '_', r == '-', r == '+':
		default:
			return nil
		}
	}
	if v, ok := zones.Load(name); ok {
		return v.(*time.Location)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil
	}
	zones.Store(name, loc)
	return loc
}

type ctxKey struct{}

type ctxValue struct {
	m    *Marker
	zone *time.Location
}

// WithMarker returns ctx carrying the marker and the request's zone hint (nil = stored zone), for MarkLogged.
func WithMarker(ctx context.Context, m *Marker, zone *time.Location) context.Context {
	return context.WithValue(ctx, ctxKey{}, ctxValue{m: m, zone: zone})
}

// MarkLogged marks a data log for uid when ctx is a user request (RequireUser attached the marker); background jobs
// and admin requests carry none, so their writes never count as the user's log.
func MarkLogged(ctx context.Context, uid uint64) {
	if ctx == nil {
		return
	}
	if v, ok := ctx.Value(ctxKey{}).(ctxValue); ok {
		v.m.Mark(ctx, uid, Logged, v.zone)
	}
}
