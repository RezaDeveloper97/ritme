package analysis

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/sync/singleflight"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/clinicalconfig"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/etag"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/queue"
	"github.com/ritme/backend-go/internal/platform/usertz"
	"github.com/ritme/backend-go/internal/plus"
)

// The analysis cache (T-M10-28, D-177, DECISIONS.md X-02; SPEC §4.1, gap 06 §2.1/§2.8).
//
// Every report is a pure function of the user's analysis inputs, her engine state, her Plus entitlements, her local
// day, the locale, the clinical registry and admin config it reads, and the binary that computed it. The revisions
// come free with the auth query (user_revisions: analysis_rev moves with every engine- or analysis-input write, in the
// write's transaction — internal/cycle/revision), so a body is cached cache-aside in Redis under
//
//	ritme-go:ana:{uid}:{analysis_rev}:{engine_rev}:{endpoint}:{range}:{plus}:{localToday}:{cfgFP}:{locale}:{build}
//
// for CacheTTL, and tagged
//
//	W/"u{uid36}.a{analysis_rev}.e{engine_rev}-{localToday}-{endpoint}-{range}-{plus}-{locale}-{cfgFP}-{build}"
//
// (Cache-Control: private, no-cache; Vary: Authorization, Accept-Language — the etag package). The locale is in the
// key because the reports are localised while they are computed; the build is in the key so a deploy never serves a
// body an older binary rendered. Editing the past moves analysis_rev (a new key), a Plus purchase or expiry flips the
// plus part, a registry or config change moves cfgFP: nothing is ever invalidated by hand, stale keys expire.
//
// A miss is computed once per key (singleflight), stored, and served. Redis errors are logged and the body is
// computed as if there were no cache; an error answer (4xx / 5xx) is never cached.

// CacheTTL is the lifetime of a cached analysis body.
const CacheTTL = 24 * time.Hour

// cacheName labels cache_requests_total{cache}.
const cacheName = "analysis"

// keyPrefix is the analysis namespace under the Go Redis prefix.
const keyPrefix = "ana:"

// KV is the part of Redis the cache uses (*cache.Client: unprefixed keys, Get answers cache.ErrMiss for an absent
// key).
type KV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

// Cache serves analysis bodies through Redis. A nil *Cache, or one without a KV, computes every body (ETags still
// apply when it is non-nil).
type Cache struct {
	kv     KV
	build  string
	ttl    time.Duration
	flight singleflight.Group
	log    *slog.Logger
}

// NewCache builds the cache over kv (nil = no Redis: ETag / 304 only); build identifies the binary (etag.Build()).
func NewCache(kv KV, build string, logger *slog.Logger) *Cache {
	if logger == nil {
		logger = slog.Default()
	}
	if build == "" {
		build = etag.Build()
	}
	return &Cache{kv: kv, build: build, ttl: CacheTTL, log: logger}
}

// CacheKey names one analysis body.
type CacheKey struct {
	UserID      uint64
	AnalysisRev uint64
	EngineRev   uint64
	Endpoint    string // the report, with its own parameters (monthly:2026-09:gregorian, fertility:2026-08-30, …)
	Range       string // the range key ("-" when the report has none)
	Plus        string // the entitlements the report reads (plusPart)
	Today       civildate.Date
	Locale      string
	ConfigFP    string // ConfigFingerprint(Today)
}

// redisKey is the unprefixed Redis key of k under build.
func (k CacheKey) redisKey(build string) string {
	var b strings.Builder
	b.Grow(96)
	b.WriteString(keyPrefix)
	b.WriteString(strconv.FormatUint(k.UserID, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatUint(k.AnalysisRev, 10))
	b.WriteByte(':')
	b.WriteString(strconv.FormatUint(k.EngineRev, 10))
	for _, p := range []string{k.Endpoint, k.Range, k.Plus, k.Today.String(), k.ConfigFP, k.Locale, build} {
		b.WriteByte(':')
		b.WriteString(p)
	}
	return b.String()
}

// Tag is the weak entity tag of k under build.
func (k CacheKey) Tag(build string) string {
	var b strings.Builder
	b.Grow(96)
	b.WriteString(`W/"u`)
	b.WriteString(strconv.FormatUint(k.UserID, 36))
	b.WriteString(".a")
	b.WriteString(strconv.FormatUint(k.AnalysisRev, 10))
	b.WriteString(".e")
	b.WriteString(strconv.FormatUint(k.EngineRev, 10))
	for _, p := range []string{k.Today.String(), k.Endpoint, k.Range, k.Plus, k.Locale, k.ConfigFP, build} {
		b.WriteByte('-')
		for i := 0; i < len(p); i++ { // RFC 9110 etagc only: a client-chosen locale never breaks the header
			c := p[i]
			if c < 0x21 || c == '"' || c > 0x7e {
				c = '_'
			}
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// Compute renders the data of a report (the envelope's `data`); an error is answered as is and never cached.
type Compute func(ctx context.Context) (any, error)

// Serve answers a report: 304 when If-None-Match matches k's tag, else the cached body or compute's.
func (c *Cache) Serve(fc fiber.Ctx, k CacheKey, compute Compute) error {
	if c == nil {
		data, err := compute(fc.Context())
		if err != nil {
			return err
		}
		return httpx.OK(fc, data)
	}
	if etag.NotModified(fc, k.Tag(c.build)) {
		return nil
	}
	body, err := c.Body(fc.Context(), k, compute)
	if err != nil {
		return err
	}
	fc.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return fc.Status(fiber.StatusOK).Send(body)
}

// Body is the encoded response ({"success":true,"data":…}) of k: Redis first, else computed once per key and stored.
func (c *Cache) Body(ctx context.Context, k CacheKey, compute Compute) ([]byte, error) {
	rk := k.redisKey(c.build)
	if b, ok := c.get(ctx, rk); ok {
		return b, nil
	}
	v, err, _ := c.flight.Do(rk, func() (any, error) {
		data, err := compute(ctx)
		if err != nil {
			return nil, err
		}
		b, err := jsonx.Marshal(httpx.Envelope(data), 0)
		if err != nil {
			return nil, err
		}
		c.set(ctx, rk, b)
		return b, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

// Cached reports whether k has a stored body (a Redis error counts as not cached).
func (c *Cache) Cached(ctx context.Context, k CacheKey) bool {
	_, ok := c.get(ctx, k.redisKey(c.build))
	return ok
}

func (c *Cache) get(ctx context.Context, rk string) ([]byte, bool) {
	if c.kv == nil {
		return nil, false
	}
	s, err := c.kv.Get(ctx, rk)
	cache.Record(cacheName, err)
	if err != nil {
		if !errors.Is(err, cache.ErrMiss) {
			c.log.WarnContext(ctx, "analysis cache: get failed", slog.String("error", err.Error()))
		}
		return nil, false
	}
	return []byte(s), true
}

func (c *Cache) set(ctx context.Context, rk string, b []byte) {
	if c.kv == nil {
		return
	}
	if err := c.kv.Set(context.WithoutCancel(ctx), rk, b, c.ttl); err != nil {
		c.log.WarnContext(ctx, "analysis cache: set failed", slog.String("error", err.Error()))
	}
}

// ConfigFingerprint is the cfgFP part of a key on today: the version of every clinical-registry key valid on today
// (the FIGO ranges, the vitals classifier, the pregnancy thresholds, the engine params, …). It moves when an admin
// publishes a new registry version (or one takes effect). It reads only the registry snapshot every process loads at
// start, so the API replicas and the worker that warms agree on it. Admin Plus allowances reach the key through the
// {plus} part (the resolved entitlements), the module switches through the 403 gate and the postpartum {plus} part.
func ConfigFingerprint(today civildate.Date) string {
	keys := make([]string, 0, len(clinicalconfig.Defs))
	for _, d := range clinicalconfig.Defs {
		keys = append(keys, d.Key)
	}
	sum := sha256.Sum256([]byte(clinicalconfig.Fingerprint(today, keys...)))
	return hex.EncodeToString(sum[:6])
}

// entitlements are the Plus answers of one request (one plus.Usage read).
type entitlements map[plus.Key]plus.Entitlement

func entitlementsOf(u plus.Usage) entitlements {
	out := make(entitlements, len(u.Entitlements))
	for _, e := range u.Entitlements {
		out[e.Key] = e
	}
	return out
}

func (e entitlements) allowed(k plus.Key) bool { return e[k].Allowed }

// plusPart is the {plus} part of a key: the entitlements a report reads, so a purchase, a trial or its end changes
// the key and two plans never share a body.
func (e entitlements) plusPart() string {
	b := []byte("d0l0p0")
	if e.allowed(plus.DeepAnalysis) {
		b[1] = '1'
	}
	if e.allowed(plus.AnalysisLongRange) {
		b[3] = '1'
	}
	if e.allowed(plus.PDFShare) {
		b[5] = '1'
	}
	return string(b)
}

// requestKey is the key of a report served outside Handlers (pregnancy, postpartum): the revisions come from the
// auth query; ok is false without them (the report is then computed, uncached).
func requestKey(c fiber.Ctx, uid uint64, endpoint, plusPart string, today civildate.Date) (CacheKey, bool) {
	revs, ok := auth.CurrentRevisions(c)
	return CacheKey{UserID: uid, AnalysisRev: revs.Analysis, EngineRev: revs.Engine, Endpoint: endpoint, Range: "-",
		Plus: plusPart, Today: today, Locale: i18n.Locale(c), ConfigFP: ConfigFingerprint(today)}, ok
}

// deepPart is the {plus} part of a report that reads plus.deep_analysis only.
func deepPart(allowed bool) string {
	if allowed {
		return "d1"
	}
	return "d0"
}

// serveKey is Serve when ok, else a plain computed answer.
func (c *Cache) serveKey(fc fiber.Ctx, k CacheKey, ok bool, compute Compute) error {
	if !ok {
		c = nil
	}
	return c.Serve(fc, k, compute)
}

// The nightly warm job (T-M10-28 Scope 4, docs/engine-v6/jobs.md).
const (
	// JobWarm is the warm job's type.
	JobWarm = "analysis.warm"
	// WarmCron runs it after the 03:30–04:35 nightly jobs (Tehran), before the morning opens.
	WarmCron = "0 5 * * *"
	// WarmActiveDays: users with a session, log or event in the last 7 days are warmed.
	WarmActiveDays = 7
	// WarmBatch is one page of users.
	WarmBatch = 200
)

// WarmStats is what one run did.
type WarmStats struct{ Warmed, Skipped, Failed int }

// Warmer fills the hub (summary, DefaultRange, the default language) of the recently active users, so the first
// open of the day is a cache hit.
type Warmer struct {
	h   *Handlers
	q   *cyclestore.Queries
	log *slog.Logger
}

// NewWarmer warms h's summary (h must carry a cache) reading users and revisions from db.
func NewWarmer(h *Handlers, db *sql.DB, logger *slog.Logger) *Warmer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Warmer{h: h, q: cyclestore.New(db), log: logger}
}

// RegisterJobs puts analysis.warm on q (queue low, nightly, unique per day); now is the Tehran clock.
func (w *Warmer) RegisterJobs(q *queue.Queue, now func() time.Time) error {
	return q.RegisterPeriodic(queue.Job{
		Type: JobWarm, Priority: queue.Low, MaxTries: 2, Backoff: []time.Duration{30 * time.Minute}, Timeout: 30 * time.Minute,
	}, func(ctx context.Context, _ []byte) error {
		_, err := w.Run(ctx, now())
		return err
	}, WarmCron, queue.Unique(23*time.Hour))
}

// Run warms every active user once. Idempotent: a user whose body for today's key is already cached is skipped, and a
// rerun the same day only fills what is missing. A failing user is counted and skipped (no user data in the log).
func (w *Warmer) Run(ctx context.Context, now time.Time) (WarmStats, error) {
	var st WarmStats
	if w.h.cache == nil || w.h.cache.kv == nil {
		return st, nil
	}
	tehran := now.In(civildate.Tehran)
	since := tehran.AddDate(0, 0, -WarmActiveDays)
	def := w.h.languages.DefaultCode(ctx)
	var after uint64
	for {
		page, err := w.q.ListActiveSnapshotUsers(ctx, cyclestore.ListActiveSnapshotUsersParams{
			AfterID: after, Since: sql.NullTime{Time: since, Valid: true}, SinceDay: civildate.FromTime(since), Limit: WarmBatch,
		})
		if err != nil {
			return st, fmt.Errorf("analysis warm: active users: %w", err)
		}
		for _, u := range page {
			switch warmed, err := w.user(ctx, u, now, def); {
			case err != nil:
				st.Failed++
			case warmed:
				st.Warmed++
			default:
				st.Skipped++
			}
		}
		if len(page) < WarmBatch {
			break
		}
		after = page[len(page)-1].UserID
	}
	if st.Failed > 0 {
		w.log.WarnContext(ctx, "analysis warm: users failed", slog.Int("failed", st.Failed), slog.Int("warmed", st.Warmed))
	}
	return st, nil
}

// user warms one user's hub; false when it was already cached.
func (w *Warmer) user(ctx context.Context, u cyclestore.ListActiveSnapshotUsersRow, now time.Time, def string) (bool, error) {
	revs, err := w.q.GetUserRevisions(ctx, u.UserID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	today := civildate.InTehran(now)
	if w.h.states != nil && u.Timezone.Valid { // the request's LocalToday under v6 (D-126)
		if loc, ok := usertz.Parse(u.Timezone.String); ok {
			today = civildate.FromTime(now.In(loc))
		}
	}
	ri := &reqInfo{ctx: ctx, uid: u.UserID, now: now, today: today, locale: def, def: def,
		revs: auth.Revisions{Engine: revs.EngineRev, Analysis: revs.AnalysisRev}, hasRevs: true}
	if err := w.h.loadEntitlements(ri); err != nil {
		return false, err
	}
	k := ri.key(EndpointSummary, DefaultRange)
	if w.h.cache.Cached(ctx, k) {
		return false, nil
	}
	_, err = w.h.cache.Body(ctx, k, func(context.Context) (any, error) { return w.h.summary(ri, DefaultRange) })
	return err == nil, err
}
