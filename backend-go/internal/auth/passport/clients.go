package passport

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ritme/backend-go/internal/auth/store"
)

// Client cache (T-M9-09). oauth_clients holds a handful of rows that change only when an operator creates or revokes
// a client, so the verifier keeps them in memory instead of reading one per request:
//
//   - loaded on first use (a load failure is a 500, exactly as a failed GetClient was);
//   - refreshed in the background once the snapshot is older than ClientCacheTTL — the request that notices serves
//     the current snapshot, so the auth step stays one query;
//   - reloaded synchronously once when a token names an id the snapshot does not know (a client created since the
//     last load), before the token is rejected.
//
// The aud is read only after the RS256 signature checked out, so an unknown id cannot be chosen by an attacker.
// Consequence: revoking a client (not a token) takes up to ClientCacheTTL to reach the guard. Token revocation is
// unaffected — the token row is read fresh on every request (DECISIONS.md X-02).

// ClientCacheTTL is how long a client snapshot is served before a background refresh.
const ClientCacheTTL = 5 * time.Minute

// clientRefreshTimeout bounds a background refresh (it runs detached from the request that started it).
const clientRefreshTimeout = 5 * time.Second

// ClientLister loads every oauth_clients row (satisfied by *store.Queries).
type ClientLister interface {
	ListClients(ctx context.Context) ([]store.ListClientsRow, error)
}

type clientSnapshot struct {
	byID     map[string]store.ListClientsRow
	loadedAt time.Time
}

type clientCache struct {
	src        ClientLister
	ttl        time.Duration
	now        func() time.Time // wall clock for the TTL (not the verifier's token clock, which tests pin)
	snap       atomic.Pointer[clientSnapshot]
	loadMu     sync.Mutex // serialises loads
	refreshing atomic.Bool
}

func newClientCache(src ClientLister) *clientCache {
	return &clientCache{src: src, ttl: ClientCacheTTL, now: time.Now}
}

// lookup returns the client with id; ok=false when it does not exist even after a reload.
func (c *clientCache) lookup(ctx context.Context, id string) (store.ListClientsRow, bool, error) {
	s := c.snap.Load()
	if s == nil {
		var err error
		if s, err = c.loadIfOlder(ctx, nil); err != nil {
			return store.ListClientsRow{}, false, err
		}
	} else if c.now().Sub(s.loadedAt) >= c.ttl {
		c.refreshAsync(ctx)
	}
	if row, ok := s.byID[id]; ok {
		return row, true, nil
	}
	// Unknown id: reload once (unless another request already reloaded since our snapshot) and look again.
	s, err := c.loadIfOlder(ctx, s)
	if err != nil {
		return store.ListClientsRow{}, false, err
	}
	row, ok := s.byID[id]
	return row, ok, nil
}

// WarmClients loads the client cache now (T-M9-09c), so the first authed request does not pay for it. It is a no-op
// when a snapshot is already loaded. The snapshot then ages under the usual ClientCacheTTL — warming changes when
// the first load happens, not how long a client revocation takes to reach the guard (≤ ClientCacheTTL, D-140).
// On error the cache stays empty and the first request loads it, exactly as without warming.
func (v *Verifier) WarmClients(ctx context.Context) error {
	_, err := v.clients.loadIfOlder(ctx, nil)
	return err
}

// loadIfOlder loads a fresh snapshot unless the current one is already newer than seen (nil = load unless any).
func (c *clientCache) loadIfOlder(ctx context.Context, seen *clientSnapshot) (*clientSnapshot, error) {
	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	if cur := c.snap.Load(); cur != nil && cur != seen {
		return cur, nil
	}
	return c.load(ctx)
}

// load reads every client and publishes the snapshot. Callers hold loadMu.
func (c *clientCache) load(ctx context.Context) (*clientSnapshot, error) {
	rows, err := c.src.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("passport: load clients: %w", err)
	}
	s := &clientSnapshot{byID: make(map[string]store.ListClientsRow, len(rows)), loadedAt: c.now()}
	for _, r := range rows {
		s.byID[r.ID] = r
	}
	c.snap.Store(s)
	return s, nil
}

// refreshAsync starts one background refresh; on failure the old snapshot stays and the next request retries.
func (c *clientCache) refreshAsync(ctx context.Context) {
	if !c.refreshing.CompareAndSwap(false, true) {
		return
	}
	bg := context.WithoutCancel(ctx)
	go func() {
		defer c.refreshing.Store(false)
		ctx, cancel := context.WithTimeout(bg, clientRefreshTimeout)
		defer cancel()
		c.loadMu.Lock()
		defer c.loadMu.Unlock()
		if cur := c.snap.Load(); cur != nil && c.now().Sub(cur.loadedAt) < c.ttl {
			return // a miss-reload got there first
		}
		_, _ = c.load(ctx)
	}()
}
