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
//   - refreshed in the background once the snapshot is older than ClientCacheTTL-clientRefreshAhead — the request
//     that notices serves the current snapshot, so on a busy process the auth step stays one query;
//   - never served once it is ClientCacheTTL old (T-M9-09d): the request reloads synchronously first, and if that
//     reload fails the request fails closed with a 500 (an infrastructure error, never a 401 — session rules);
//   - reloaded synchronously once when a token names an id the snapshot does not know (a client created since the
//     last load), before the token is rejected.
//
// The aud is read only after the RS256 signature checked out, so an unknown id cannot be chosen by an attacker.
// Consequence: revoking a client (not a token) takes at most ClientCacheTTL to reach the guard — a hard bound even
// when ListClients keeps failing or after a long idle (D-140). Token revocation is unaffected — the token row is
// read fresh on every request (DECISIONS.md X-02).

// ClientCacheTTL is the hard maximum age of a served client snapshot.
const ClientCacheTTL = 5 * time.Minute

// clientRefreshAhead is how long before ClientCacheTTL the background refresh starts, so steady traffic renews the
// snapshot before any request has to wait for a synchronous reload.
const clientRefreshAhead = 30 * time.Second

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
	switch {
	case s == nil || c.now().Sub(s.loadedAt) >= c.ttl:
		// No snapshot, or one past the hard TTL: reload before answering (fail closed on error). loadIfOlder lets
		// concurrent requests share one reload — and the age check below covers a stale snapshot published by a
		// background refresh that raced us.
		fresh, err := c.loadIfOlder(ctx, s)
		if err == nil && c.now().Sub(fresh.loadedAt) >= c.ttl {
			fresh, err = c.loadIfOlder(ctx, fresh)
		}
		if err != nil {
			return store.ListClientsRow{}, false, err
		}
		row, ok := fresh.byID[id]
		return row, ok, nil // just loaded: an unknown id is not reloaded again
	case c.now().Sub(s.loadedAt) >= c.ttl-clientRefreshAhead:
		c.refreshAsync(ctx, s)
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

// refreshAsync starts one background refresh of seen; on failure the old snapshot stays and the next request
// retries (synchronously once it reaches ClientCacheTTL).
func (c *clientCache) refreshAsync(ctx context.Context, seen *clientSnapshot) {
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
		if c.snap.Load() != seen {
			return // a miss-reload or a synchronous reload got there first
		}
		_, _ = c.load(ctx)
	}()
}
