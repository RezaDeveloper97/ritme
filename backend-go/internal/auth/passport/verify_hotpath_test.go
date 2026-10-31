package passport_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// T-M9-09: the auth step of an authed request is one query once the client cache is warm.
func TestVerify_HotPathIsOneQuery(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	hdr := "Bearer " + mint(t, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))

	_, err := v.Verify(context.Background(), hdr) // cold: loads the clients once
	require.NoError(t, err)
	assert.Equal(t, 1, st.count("ListClients"))

	for range 5 {
		st.resetCalls()
		s, err := v.Verify(context.Background(), hdr)
		require.NoError(t, err)
		assert.Equal(t, uint64(1004), s.User.ID)
		assert.Equal(t, "live", s.Token.ID)
		assert.Equal(t, clientID, s.Token.ClientID)
		assert.Equal(t, map[string]int{"GetAccessTokenWithUser": 1}, st.calls)
	}
}

// A revoked token is rejected on the very next request: the token row is never cached.
func TestVerify_RevocationIsReadFresh(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	hdr := "Bearer " + mint(t, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))
	_, err := v.Verify(context.Background(), hdr)
	require.NoError(t, err)

	st.mu.Lock()
	tok := st.tokens["live"]
	tok.Revoked = true
	st.tokens["live"] = tok
	st.mu.Unlock()
	_, err = v.Verify(context.Background(), hdr)
	assert.Equal(t, passport.CodeRevoked, codeOf(t, err))

	// Revoked and the user gone: still token_revoked (the token is checked first, as in Passport).
	st.mu.Lock()
	delete(st.users, 1004)
	st.mu.Unlock()
	_, err = v.Verify(context.Background(), hdr)
	assert.Equal(t, passport.CodeRevoked, codeOf(t, err))
}

// The slow path keeps Passport's order when the joined row is missing.
func TestVerify_FallbackOrder(t *testing.T) {
	year := now.AddDate(1, 0, 0)
	t.Run("token missing, user present → token_revoked", func(t *testing.T) {
		st := newStore()
		v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
		_, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("gone", ts(now), ts(year))))
		assert.Equal(t, passport.CodeRevoked, codeOf(t, err))
	})
	t.Run("user deleted, client unknown → unauthenticated (client checked before user)", func(t *testing.T) {
		st := newStore()
		st.tokens["live"] = store.GetAccessTokenRow{ID: "live"}
		delete(st.users, 1004)
		v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
		_, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year), `"other"`)))
		assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
		assert.Equal(t, 0, st.count("GetUserByID"))
	})
	t.Run("no subject → unauthenticated, no joined read", func(t *testing.T) {
		st := newStore()
		st.tokens["live"] = store.GetAccessTokenRow{ID: "live"}
		v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
		body := `{"aud":"` + clientID + `","jti":"live","iat":` + f6(ts(now)) + `,"exp":` + f6(ts(year)) + `,"scopes":[]}`
		_, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, body))
		assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
		assert.Equal(t, 0, st.count("GetAccessTokenWithUser"))
	})
}

func f6(x float64) string { b, _ := json.Marshal(x); return string(b) }

// A client created after the cache was loaded is found by one reload, not rejected.
func TestVerify_UnknownClientReloadsOnce(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	year := now.AddDate(1, 0, 0)
	_, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year))))
	require.NoError(t, err)

	st.setClient(store.ListClientsRow{ID: "new-client"})
	st.resetCalls()
	_, err = v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year), `"new-client"`)))
	require.NoError(t, err)
	assert.Equal(t, 1, st.count("ListClients"))

	// A truly unknown id costs one reload and is rejected.
	st.resetCalls()
	_, err = v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year), `"nobody"`)))
	assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
	assert.Equal(t, 1, st.count("ListClients"))
}

// clientCacheFixture: a verifier with one live token, a warm client cache and a controllable client-cache wall clock.
func clientCacheFixture(t *testing.T) (*fakeStore, *passport.Verifier, *atomic.Int64, string) {
	t.Helper()
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	wall := &atomic.Int64{}
	wall.Store(now.UnixNano())
	passport.SetClientClock(v, func() time.Time { return time.Unix(0, wall.Load()) })
	hdr := "Bearer " + mint(t, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))
	_, err := v.Verify(context.Background(), hdr)
	require.NoError(t, err)
	return st, v, wall, hdr
}

func revokeClient(st *fakeStore) {
	st.mu.Lock()
	c := st.clients[clientID]
	st.mu.Unlock()
	c.Revoked = true
	st.setClient(c)
}

// Well within the TTL the snapshot is served as is: no client query at all, so a revoked client is still accepted.
func TestVerify_ClientCacheWithinTTLNoExtraQuery(t *testing.T) {
	st, v, wall, hdr := clientCacheFixture(t)
	revokeClient(st)
	wall.Add(int64(passport.ClientCacheTTL - time.Minute))
	st.resetCalls()
	_, err := v.Verify(context.Background(), hdr)
	require.NoError(t, err)
	passport.WaitClientRefresh(v)
	assert.Equal(t, 0, st.count("ListClients"))
}

// Shortly before the TTL the request is served from the snapshot and refreshes it in the background (one query,
// not on the request's path); the next request sees the revocation.
func TestVerify_ClientCacheRefreshesAheadOfTTL(t *testing.T) {
	st, v, wall, hdr := clientCacheFixture(t)
	revokeClient(st)
	wall.Add(int64(passport.ClientCacheTTL - time.Second))
	st.resetCalls()
	_, err := v.Verify(context.Background(), hdr)
	require.NoError(t, err)
	passport.WaitClientRefresh(v)
	assert.Equal(t, 1, st.count("ListClients"))

	_, err = v.Verify(context.Background(), hdr)
	assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
	assert.Equal(t, 1, st.count("ListClients"), "fresh snapshot: no further load")
}

// T-M9-09d: a snapshot past the TTL (long idle, e.g. the boot-time warm one) is never served — the first request
// reloads synchronously and a client revoked meanwhile is rejected right there.
func TestVerify_StaleSnapshotRevokedClientRejectedOnFirstRequest(t *testing.T) {
	st, v, wall, hdr := clientCacheFixture(t)
	revokeClient(st)
	wall.Add(int64(time.Hour))
	st.resetCalls()
	_, err := v.Verify(context.Background(), hdr)
	assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
	assert.Equal(t, 1, st.count("ListClients"), "one synchronous reload, no second reload for the miss path")
}

// T-M9-09d: when the reload past the TTL fails, the request fails closed with an infrastructure error (the guard's
// 500), never an *AuthError (401) — and never with the stale snapshot. Once the DB is back the next request recovers.
func TestVerify_StaleSnapshotReloadErrorIsServerError(t *testing.T) {
	st, v, wall, hdr := clientCacheFixture(t)
	st.setListErr(errors.New("db down"))

	// In the refresh-ahead window a failing background refresh leaves the snapshot in use.
	wall.Add(int64(passport.ClientCacheTTL - time.Second))
	_, err := v.Verify(context.Background(), hdr)
	require.NoError(t, err)
	passport.WaitClientRefresh(v)

	// Past the TTL: fail closed.
	wall.Add(int64(2 * time.Second))
	_, err = v.Verify(context.Background(), hdr)
	require.Error(t, err)
	var ae *passport.AuthError
	assert.False(t, errors.As(err, &ae), "a reload failure must not be a 401: %v", err)

	st.setListErr(nil)
	_, err = v.Verify(context.Background(), hdr)
	require.NoError(t, err)
}

// Concurrent requests that all find the snapshot past the TTL share one synchronous reload.
func TestVerify_StaleSnapshotConcurrentReloadOnce(t *testing.T) {
	st, v, wall, hdr := clientCacheFixture(t)
	wall.Add(int64(time.Hour))
	st.resetCalls()
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			_, err := v.Verify(context.Background(), hdr)
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, 1, st.count("ListClients"))
}

func TestVerify_ConcurrentColdStartLoadsOnce(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	hdr := "Bearer " + mint(t, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			_, err := v.Verify(context.Background(), hdr)
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, 1, st.count("ListClients"))
}

// Task budget: Verify ≤ 150 µs (RS256 verify dominates; the store is in memory).
func BenchmarkVerify(b *testing.B) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	hdr := "Bearer " + mint(&testing.T{}, testKey, claims("live", ts(now), ts(now.AddDate(1, 0, 0))))
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := v.Verify(ctx, hdr); err != nil {
			b.Fatal(err)
		}
	}
}
