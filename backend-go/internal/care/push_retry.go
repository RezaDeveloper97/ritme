package care

import (
	"time"

	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Durable retry after a passing failure (T-M12-22, T-M12-22b). A dose (reminder_pushes) or an inbox row
// (user_notifications.pushed_at) is claimed in the database before it is sent, so only one process announces it.
// When no browser accepted it because of a timeout, a 429 or a 5xx, the claiming process releases the claim in the
// database: the claim column becomes a marker, releaseEpoch + <failed tries> seconds, a year-2000 value no real claim
// ever has. Any process — the same one, another one, or one started after a restart or a deploy — takes a released
// item again on a later tick with a compare-and-set on that exact marker (so only one of them retries it) and knows
// how many tries were spent. At most PushAttempts tries in all, each still inside the lookback window (PushLookback /
// NoticeLookback, since only due doses and fresh rows are read); after the last one the item stays claimed and
// silent. A refusal for good (404 / 410, any other 4xx, no browser) is never released, so never retried.
//
// No schema change: the marker lives in the existing NOT NULL sent_at / nullable pushed_at datetime columns. The
// DSN's loc (Asia/Tehran) maps the marker to the same wall-clock value both ways.

// releaseEpoch is the base of the released-claim marker (before any real claim).
var releaseEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, civildate.Tehran)

// releasedAt is the claim value of an item released after its tries-th failed try (1 ≤ tries < PushAttempts).
func releasedAt(tries int) time.Time { return releaseEpoch.Add(time.Duration(tries) * time.Second) }

// releasedBefore bounds every marker: a claim value before it is a released item.
func releasedBefore() time.Time { return releasedAt(PushAttempts) }

// releasedTries is how many tries the released marker at records; false = at is not a retryable marker.
func releasedTries(at time.Time) (int, bool) {
	d := at.Sub(releaseEpoch)
	if d < time.Second || d >= time.Duration(PushAttempts)*time.Second || d%time.Second != 0 {
		return 0, false
	}
	return int(d / time.Second), true
}

// prunePushes reports whether this tick prunes reminder_pushes (at most once an hour).
func (p *Pusher) prunePushes(now time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if now.Sub(p.pruned) <= time.Hour {
		return false
	}
	p.pruned = now
	return true
}
