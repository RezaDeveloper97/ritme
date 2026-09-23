package httpadmin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// Session lifetimes (backend config/session.php lifetime 120 min; remember-me 30 days).
// Both slide: every authenticated request pushes the expiry forward.
const (
	IdleTTL     = 120 * time.Minute
	RememberTTL = 30 * 24 * time.Hour
)

// tokenBytes is the entropy of session ids and CSRF tokens (256 bit).
const tokenBytes = 32

// Session is one admin login. The cookie carries only the opaque id; Redis stores the
// record under admin-session:<sha256(id)> so a Redis dump does not reveal live cookies.
type Session struct {
	AdminID   uint64 `json:"admin_id"`
	CSRF      string `json:"csrf"`
	Remember  bool   `json:"remember"`
	CreatedAt int64  `json:"created_at"`

	id string // raw cookie value (never stored)
}

// ID returns the raw session id (the cookie value).
func (s *Session) ID() string { return s.id }

// TTL is the sliding lifetime of the session.
func (s *Session) TTL() time.Duration {
	if s.Remember {
		return RememberTTL
	}
	return IdleTTL
}

// Sessions is the Redis session store (keys under the Go prefix, see cache.Client.Key).
type Sessions struct {
	c *cache.Client
}

// NewSessions returns the store.
func NewSessions(c *cache.Client) *Sessions { return &Sessions{c: c} }

func hashID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

func (s *Sessions) key(id string) string { return s.c.Key("admin-session:" + hashID(id)) }

func (s *Sessions) indexKey(adminID uint64) string {
	return s.c.Key("admin-sessions:" + strconv.FormatUint(adminID, 10))
}

// RandomToken returns a URL-safe random token (session ids, CSRF tokens).
func RandomToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("httpadmin: random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// validID rejects cookie values that cannot be a session id before touching Redis.
func validID(id string) bool {
	if len(id) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(id)
	return err == nil
}

// Create starts a new session (always a fresh id: no session fixation).
func (s *Sessions) Create(ctx context.Context, adminID uint64, remember bool, now time.Time) (*Session, error) {
	id, err := RandomToken()
	if err != nil {
		return nil, err
	}
	csrf, err := RandomToken()
	if err != nil {
		return nil, err
	}
	sess := &Session{AdminID: adminID, CSRF: csrf, Remember: remember, CreatedAt: now.Unix(), id: id}
	b, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("httpadmin: encode session: %w", err)
	}
	idx := s.indexKey(adminID)
	pipe := s.c.Redis().TxPipeline()
	pipe.Set(ctx, s.key(id), b, sess.TTL())
	pipe.SAdd(ctx, idx, hashID(id))
	pipe.Expire(ctx, idx, RememberTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("httpadmin: store session: %w", err)
	}
	return sess, nil
}

// Load returns the session for a cookie value, or (nil, nil) when it does not exist
// (expired, destroyed, malformed).
func (s *Sessions) Load(ctx context.Context, id string) (*Session, error) {
	if !validID(id) {
		return nil, nil
	}
	b, err := s.c.Redis().Get(ctx, s.key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("httpadmin: load session: %w", err)
	}
	var sess Session
	if err := json.Unmarshal(b, &sess); err != nil || sess.AdminID == 0 || sess.CSRF == "" {
		return nil, nil //nolint:nilerr // a corrupt record is treated as no session
	}
	sess.id = id
	return &sess, nil
}

// Touch slides the expiry of an active session.
func (s *Sessions) Touch(ctx context.Context, sess *Session) error {
	pipe := s.c.Redis().TxPipeline()
	pipe.Expire(ctx, s.key(sess.id), sess.TTL())
	pipe.Expire(ctx, s.indexKey(sess.AdminID), RememberTTL) // the index outlives every session in it
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("httpadmin: touch session: %w", err)
	}
	return nil
}

// Destroy deletes one session.
func (s *Sessions) Destroy(ctx context.Context, sess *Session) error {
	pipe := s.c.Redis().TxPipeline()
	pipe.Del(ctx, s.key(sess.id))
	pipe.SRem(ctx, s.indexKey(sess.AdminID), hashID(sess.id))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("httpadmin: destroy session: %w", err)
	}
	return nil
}

// DestroyAll deletes every session of an admin except keep (nil = all). Used when an
// admin's password changes or the account is deactivated or deleted.
func (s *Sessions) DestroyAll(ctx context.Context, adminID uint64, keep *Session) error {
	idx := s.indexKey(adminID)
	hashes, err := s.c.Redis().SMembers(ctx, idx).Result()
	if err != nil {
		return fmt.Errorf("httpadmin: list sessions: %w", err)
	}
	keepHash := ""
	if keep != nil {
		keepHash = hashID(keep.id)
	}
	pipe := s.c.Redis().TxPipeline()
	for _, h := range hashes {
		if h == keepHash {
			continue
		}
		pipe.Del(ctx, s.c.Key("admin-session:"+h))
		pipe.SRem(ctx, idx, h)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("httpadmin: destroy sessions: %w", err)
	}
	return nil
}
