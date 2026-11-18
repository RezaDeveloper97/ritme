package care

import "time"

// Retry after a passing failure (T-M12-22). A dose or an inbox row is claimed in the database before it is sent, so
// only the claiming process may announce it: when no browser accepted it because of a timeout, a 429 or a 5xx, that
// process keeps the claim and tries again on its next ticks — at most PushAttempts tries in all, each still inside
// the lookback window (PushLookback / NoticeLookback). Another process never takes it over, so it is never announced
// twice; a restart loses the pending retries (the reminder is then dropped, as a dose missed in a restart is).

// pending is a claimed item and how many times this process has tried it.
type pending[T any] struct {
	item  T
	tries int
}

// retry queues item for the next tick after its tries-th failed try, while tries < PushAttempts.
func retry[T any](p *Pusher, list *[]pending[T], item T, tries int) {
	if tries >= PushAttempts {
		return // given up: stays claimed and silent
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	*list = append(*list, pending[T]{item: item, tries: tries})
}

// takeRetries empties list and returns what was queued in it.
func takeRetries[T any](p *Pusher, list *[]pending[T]) []pending[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := *list
	*list = nil
	return out
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
