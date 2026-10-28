package family

import (
	"context"
	"fmt"
	"time"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// ReadAuditWindow coalesces member reads (F-16, D-203): one activity row per (owner, member, domain) per 15 minutes.
// Writes, proposals, nudges and lifecycle events are always logged.
const ReadAuditWindow = 15 * time.Minute

// Claim is ReadAudit's answer for one read.
type Claim uint8

// Claims.
const (
	// ClaimRecord: first read in the window — write the row.
	ClaimRecord Claim = iota + 1
	// ClaimSkip: already recorded in the window.
	ClaimSkip
	// ClaimCheckDB: no Redis answer (not configured or an error): the caller checks the DB (fail towards logging).
	ClaimCheckDB
)

// ReadAudit coalesces read-audit rows in Redis: `SET ritme-go:family:ra:{owner}:{member}:{domain} NX EX 900`.
// A nil ReadAudit (or one without Redis) always answers ClaimCheckDB.
type ReadAudit struct {
	cache *cache.Client
}

// NewReadAudit returns the Redis coalescer (nil cache → DB check only).
func NewReadAudit(c *cache.Client) *ReadAudit { return &ReadAudit{cache: c} }

func (r *ReadAudit) key(ownerID, memberID uint64, domain string) string {
	return r.cache.Key(fmt.Sprintf("family:ra:%d:%d:%s", ownerID, memberID, domain))
}

// Claim reserves the window for this read at now (the request clock). The key holds the claim's Unix time, so a
// window is measured on the request clock even when Redis' own TTL (wall clock) has not run out yet.
func (r *ReadAudit) Claim(ctx context.Context, ownerID, memberID uint64, domain string, now time.Time) Claim {
	if r == nil || r.cache == nil {
		return ClaimCheckDB
	}
	rdb, key, stamp := r.cache.Redis(), r.key(ownerID, memberID, domain), now.Unix()
	ok, err := rdb.SetNX(ctx, key, stamp, ReadAuditWindow).Result()
	switch {
	case err != nil:
		return ClaimCheckDB
	case ok:
		return ClaimRecord
	}
	prev, err := rdb.Get(ctx, key).Int64()
	if err != nil {
		return ClaimCheckDB
	}
	if stamp-prev < int64(ReadAuditWindow/time.Second) && stamp >= prev {
		return ClaimSkip
	}
	if err := rdb.Set(ctx, key, stamp, ReadAuditWindow).Err(); err != nil {
		return ClaimCheckDB
	}
	return ClaimRecord
}

// Release drops a claim whose row could not be written, so the next read records it.
func (r *ReadAudit) Release(ctx context.Context, ownerID, memberID uint64, domain string) {
	if r == nil || r.cache == nil {
		return
	}
	_ = r.cache.Redis().Del(ctx, r.key(ownerID, memberID, domain)).Err()
}
