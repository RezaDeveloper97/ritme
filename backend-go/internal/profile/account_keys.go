package profile

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ritme/backend-go/internal/platform/cache"
)

// UserKeyPatterns are the Redis key patterns (without the cache prefix) that carry one user's id, removed on account
// deletion (T-M12-13). Each names the code that writes it; a new per-user key belongs here and in
// docs/quality-2026-11/user-tables.md § Redis.
//
// Not listed on purpose (harmless, documented there): the OTP guard keys (`otp:*:<sha256(mobile)>`, no id or number,
// and keeping them is what stops a delete-and-resignup from resetting the send / failure budget), the rate-limit
// counters (`throttle:*`, seconds-to-minutes TTL, counts only) and queued jobs (their handlers find no user).
func UserKeyPatterns(userID uint64) []string {
	id := strconv.FormatUint(userID, 10)
	return []string{
		"cycle-engine:" + id + ":*", // internal/cycle/cache: computed cycle views (health data), 24 h
		"eng:in:" + id + ":*",       // internal/cycle/state: L2 engine-input snapshot (periods, logs, profile), 7 d
		"ana:" + id + ":*",          // internal/analysis: cached analysis bodies (health data), 24 h
		"preg-alerts:eval:" + id,    // internal/messages/pregnancyalerts: last evaluation digest
		"idem:" + id + ":*",         // internal/platform/idempotency: stored write responses, 24 h
		"msg:shown:" + id + ":*",    // internal/messages/events: daily marks
		"msg:eligible:" + id + ":*",
		"msg:goal:" + id + ":*",
		"seen:" + id + ":*",        // internal/auth/activity: daily activity claims
		"logged:" + id + ":*",      //
		"family:ra:" + id + ":*",   // internal/family/readaudit: she is the owner
		"family:ra:*:" + id + ":*", // … or the reading member
	}
}

// PurgeUserKeys deletes every key matching UserKeyPatterns (SCAN + DEL; nil client = nothing to do). An account
// deletion is rare, so walking the keyspace with SCAN (never KEYS) is acceptable.
func PurgeUserKeys(ctx context.Context, c *cache.Client, userID uint64) error {
	if c == nil {
		return nil
	}
	rdb := c.Redis()
	var errs []error
	for _, p := range UserKeyPatterns(userID) {
		iter := rdb.Scan(ctx, 0, c.Key(p), 1000).Iterator()
		var batch []string
		for iter.Next(ctx) {
			batch = append(batch, iter.Val())
			if len(batch) == 500 {
				errs = append(errs, rdb.Unlink(ctx, batch...).Err())
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			errs = append(errs, rdb.Unlink(ctx, batch...).Err())
		}
		errs = append(errs, iter.Err())
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("profile: purge user keys: %w", err)
	}
	return nil
}
