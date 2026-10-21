// Package idempotency honours an optional `Idempotency-Key` request header on write routes
// (D-131, T-M9-04): a client that retries a write (offline outbox replay, double tap, a lost
// response) gets the first response back instead of a second write.
//
//   - Redis key `ritme-go:idem:{uid}:{sha256(key)}` (prefix via cache.Client.Key); the raw key is
//     never stored, logged or echoed.
//   - The first request takes the key with `SET NX` (a short-lived pending marker carrying the
//     request fingerprint and an owner token), runs the handler and replaces the marker with the
//     response (status, content type, body) for 24 h. A handler error or a 5xx releases the key,
//     so the client may retry.
//   - A replay with the same fingerprint (method + URL + body) returns the stored response byte
//     for byte, with `Idempotent-Replayed: true`. The same key with a different request → 422
//     `idempotency_key_reused`. A replay while the first is still running waits briefly for it,
//     then answers 409 `idempotency_key_in_flight`.
//   - No header, no authenticated user, no Redis client or Redis down → the request is processed
//     normally (the per-user DB lock still keeps the write safe).
//
// Mount it after auth RequireUser (it keys by user) and the locale middleware.
package idempotency

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/cache"
	"github.com/ritme/backend-go/internal/platform/httpx"
)

const (
	// Header is the request header carrying the client's key.
	Header = "Idempotency-Key"
	// ReplayedHeader marks a response served from the store.
	ReplayedHeader = "Idempotent-Replayed"

	// TTL is how long a completed response is kept.
	TTL = 24 * time.Hour
	// LockTTL bounds the pending marker, so a crashed request frees its key on its own.
	LockTTL = 60 * time.Second

	// CodeReused / CodeInFlight are the `code` keys of the refusals.
	CodeReused   = "idempotency_key_reused"
	CodeInFlight = "idempotency_key_in_flight"

	statePending = "pending"
	stateDone    = "done"
)

// Identity returns the authenticated user's id (auth.CurrentUserID for routes behind RequireUser).
type Identity func(c fiber.Ctx) (uint64, bool)

// Middleware is the Idempotency-Key handler factory.
type Middleware struct {
	cache    *cache.Client
	identity Identity
	logger   *slog.Logger
	ttl      time.Duration
	lockTTL  time.Duration
	wait     time.Duration
	poll     time.Duration
}

// Option tunes a Middleware (tests).
type Option func(*Middleware)

// WithTimings overrides the stored-response TTL, the pending-marker TTL and how long a concurrent
// replay waits for the first request.
func WithTimings(ttl, lockTTL, wait time.Duration) Option {
	return func(m *Middleware) { m.ttl, m.lockTTL, m.wait = ttl, lockTTL, wait }
}

// New returns the middleware factory. A nil cache makes Handler a pass-through.
func New(c *cache.Client, identity Identity, logger *slog.Logger, opts ...Option) *Middleware {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	m := &Middleware{
		cache: c, identity: identity, logger: logger,
		ttl: TTL, lockTTL: LockTTL, wait: 5 * time.Second, poll: 50 * time.Millisecond,
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Key is the (unprefixed) Redis key of a user's Idempotency-Key: idem:{uid}:{sha256 hex}.
func Key(userID uint64, key string) string {
	sum := sha256.Sum256([]byte(key))
	return "idem:" + strconv.FormatUint(userID, 10) + ":" + hex.EncodeToString(sum[:])
}

// Fingerprint identifies the request a key was first used for: method, URL (path + query) and body.
func Fingerprint(method, url string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte{0})
	h.Write([]byte(url))
	h.Write([]byte{0})
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

type record struct {
	State       string `json:"state"`
	Fingerprint string `json:"fp"`
	Owner       string `json:"owner,omitempty"`
	Status      int    `json:"status,omitempty"`
	ContentType string `json:"ct,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

// finishScript replaces our own pending marker (ARGV[1]) with the response (ARGV[2], TTL ARGV[3] s).
var finishScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('SET', KEYS[1], ARGV[2], 'EX', tonumber(ARGV[3]))
end
return false
`)

// releaseScript deletes our own pending marker (ARGV[1]) only.
var releaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

// Handler is the fiber middleware.
func (m *Middleware) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(Header))
		if raw == "" || m.cache == nil || m.identity == nil {
			return c.Next()
		}
		uid, ok := m.identity(c)
		if !ok {
			return c.Next()
		}
		key := m.cache.Key(Key(uid, raw))
		fp := Fingerprint(c.Method(), c.OriginalURL(), c.Body())
		return m.serve(c, key, fp, time.Now().Add(m.wait))
	}
}

func (m *Middleware) serve(c fiber.Ctx, key, fp string, deadline time.Time) error {
	ctx := c.Context()
	owner, err := newOwner()
	if err != nil {
		return fmt.Errorf("idempotency: owner token: %w", err)
	}
	pending, err := json.Marshal(record{State: statePending, Fingerprint: fp, Owner: owner})
	if err != nil {
		return fmt.Errorf("idempotency: encode: %w", err)
	}
	acquired, err := m.cache.Redis().SetNX(ctx, key, pending, m.lockTTL).Result()
	if err != nil {
		m.degraded("acquire", err)
		return c.Next()
	}
	if acquired {
		return m.run(c, key, pending)
	}

	for {
		v, err := m.cache.Redis().Get(ctx, key).Bytes()
		switch {
		case errors.Is(err, redis.Nil):
			// The first request failed (key released) or its marker expired: take it over.
			return m.serve(c, key, fp, deadline)
		case err != nil:
			m.degraded("read", err)
			return c.Next()
		}
		var rec record
		if err := json.Unmarshal(v, &rec); err != nil {
			m.degraded("decode", err)
			return c.Next()
		}
		if rec.Fingerprint != fp {
			return httpx.Fail(fiber.StatusUnprocessableEntity, t("key_reused", i18n.Locale(c)), "code", CodeReused)
		}
		if rec.State == stateDone {
			return replay(c, rec)
		}
		if !time.Now().Before(deadline) {
			return httpx.Fail(fiber.StatusConflict, t("in_flight", i18n.Locale(c)), "code", CodeInFlight)
		}
		if err := sleep(ctx, m.poll); err != nil {
			return fmt.Errorf("idempotency: wait: %w", err)
		}
	}
}

// run executes the handler under the acquired key and stores (or releases) the outcome.
func (m *Middleware) run(c fiber.Ctx, key string, pending []byte) error {
	ctx := context.WithoutCancel(c.Context())
	err := c.Next()
	status := c.Response().StatusCode()
	if err != nil || status >= fiber.StatusInternalServerError {
		if rerr := releaseScript.Run(ctx, m.cache.Redis(), []string{key}, pending).Err(); rerr != nil {
			m.degraded("release", rerr)
		}
		return err
	}
	done, merr := json.Marshal(record{
		State: stateDone, Fingerprint: fingerprintOf(pending), Status: status,
		ContentType: string(c.Response().Header.ContentType()), Body: bytes.Clone(c.Response().Body()),
	})
	if merr != nil {
		m.degraded("encode", merr)
		return nil
	}
	if serr := finishScript.Run(ctx, m.cache.Redis(), []string{key}, pending, done, int64(m.ttl/time.Second)).Err(); serr != nil && !errors.Is(serr, redis.Nil) {
		m.degraded("store", serr)
	}
	return nil
}

func replay(c fiber.Ctx, rec record) error {
	if rec.ContentType != "" {
		c.Set(fiber.HeaderContentType, rec.ContentType)
	}
	c.Set(ReplayedHeader, "true")
	return c.Status(rec.Status).Send(rec.Body)
}

// degraded logs a Redis failure (never the key) and lets the request through.
func (m *Middleware) degraded(op string, err error) {
	m.logger.Warn("idempotency: redis unavailable, processing normally", "op", op, "error", err.Error())
}

func fingerprintOf(pending []byte) string {
	var r record
	_ = json.Unmarshal(pending, &r)
	return r.Fingerprint
}

func newOwner() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sleep(ctx context.Context, d time.Duration) error {
	tm := time.NewTimer(d)
	defer tm.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-tm.C:
		return nil
	}
}

// The refusal copy is data: lang/<code>/idempotency.json (English fallback).
//
//go:embed lang/*/*.json
var langFS embed.FS

var translator = sync.OnceValue(func() *lang.Translator {
	sub, err := fs.Sub(langFS, "lang")
	if err != nil {
		panic(err)
	}
	tr, err := lang.New(sub, lang.FallbackLocale)
	if err != nil {
		panic(err) // embedded files are checked by the package tests
	}
	return tr
})

func t(key, locale string) string { return translator().Trans("idempotency."+key, nil, locale) }
