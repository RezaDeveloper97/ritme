package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

const guardTestClient = "0199c0de-0000-7000-8000-00000c0ffee1"

var guardTestKey = func() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return k
}()

// countingStore is a one-user, one-token VerifierStore that counts ListClients.
type countingStore struct {
	mu        sync.Mutex
	lists     int
	failLists int // the first failLists ListClients calls fail
	revoked   bool
}

func (s *countingStore) ListClients(context.Context) ([]store.ListClientsRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lists++
	if s.lists <= s.failLists {
		return nil, errors.New("db down")
	}
	return []store.ListClientsRow{{ID: guardTestClient, Provider: sql.NullString{String: "users", Valid: true}}}, nil
}

func (s *countingStore) listCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lists
}

func (s *countingStore) GetAccessTokenWithUser(_ context.Context, a store.GetAccessTokenWithUserParams) (store.GetAccessTokenWithUserRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.TokenID != "live" || a.UserID != 1004 {
		return store.GetAccessTokenWithUserRow{}, sql.ErrNoRows
	}
	return store.GetAccessTokenWithUserRow{
		TokenID: "live", TokenClientID: guardTestClient, TokenRevoked: s.revoked, User: store.User{ID: 1004},
	}, nil
}

func (s *countingStore) GetAccessToken(_ context.Context, id string) (store.GetAccessTokenRow, error) {
	return store.GetAccessTokenRow{}, sql.ErrNoRows
}

func (s *countingStore) GetUserByID(_ context.Context, id uint64) (store.User, error) {
	return store.User{}, sql.ErrNoRows
}

func guardTestBearer(t *testing.T) string {
	t.Helper()
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"aud": guardTestClient, "jti": "live", "sub": "1004", "scopes": []string{},
		"iat": now.Add(-time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix(), "exp": now.AddDate(1, 0, 0).Unix(),
	})
	s, err := tok.SignedString(guardTestKey)
	require.NoError(t, err)
	return "Bearer " + s
}

func guardTestBuilder(st passport.VerifierStore, builds *int) func() (*passport.Verifier, error) {
	return func() (*passport.Verifier, error) {
		*builds++
		return passport.NewVerifier(&guardTestKey.PublicKey, st, clock.Real{}), nil
	}
}

var guardTestLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

// T-M9-09c: every guard of one app shares one verifier, warmed once — ListClients runs exactly once.
func TestMustGuard_SharedVerifierWarmsClientsOnce(t *testing.T) {
	st := &countingStore{}
	builds := 0
	app := fiber.New()
	hdr := guardTestBearer(t)

	guards := make([]*Guard, 0, 73)
	for range 73 {
		guards = append(guards, mustGuard(app, true, guardTestLogger, guardTestBuilder(st, &builds)))
	}
	assert.Equal(t, 1, builds, "keys loaded and verifier built once per app")
	assert.Equal(t, 1, st.listCount(), "warmed at first build")

	var wg sync.WaitGroup
	for _, g := range guards {
		wg.Go(func() {
			for range 5 {
				u, tok, err := g.Authenticate(context.Background(), hdr)
				assert.NoError(t, err)
				if err == nil {
					assert.Equal(t, uint64(1004), u.ID)
					assert.Equal(t, "live", tok.ID)
				}
			}
		})
	}
	wg.Wait()
	assert.Equal(t, 1, st.listCount())

	// Another app gets its own verifier (and its own warm-up).
	mustGuard(fiber.New(), true, guardTestLogger, guardTestBuilder(st, &builds))
	assert.Equal(t, 2, builds)
	assert.Equal(t, 2, st.listCount())
}

// A failed warm-up is only logged: the first request loads the clients, as before T-M9-09c.
func TestMustGuard_WarmFailureFallsBackToLazyLoad(t *testing.T) {
	st := &countingStore{failLists: 1}
	builds := 0
	app := fiber.New()
	g := mustGuard(app, true, guardTestLogger, guardTestBuilder(st, &builds))
	require.NoError(t, g.initErr)
	assert.Equal(t, 1, st.listCount())

	hdr := guardTestBearer(t)
	_, _, err := g.Authenticate(context.Background(), hdr)
	require.NoError(t, err)
	_, _, err = mustGuard(app, true, guardTestLogger, guardTestBuilder(st, &builds)).Authenticate(context.Background(), hdr)
	require.NoError(t, err)
	assert.Equal(t, 2, st.listCount())
	assert.Equal(t, 1, builds)
}

// Sharing changes no rule: a revoked token is rejected on the next request with token_revoked, a missing header
// with unauthenticated.
func TestMustGuard_SharedVerifierKeepsRevocationAnd401Codes(t *testing.T) {
	st := &countingStore{}
	builds := 0
	app := fiber.New()
	g := mustGuard(app, true, guardTestLogger, guardTestBuilder(st, &builds))
	hdr := guardTestBearer(t)
	_, _, err := g.Authenticate(context.Background(), hdr)
	require.NoError(t, err)

	st.mu.Lock()
	st.revoked = true
	st.mu.Unlock()
	_, _, err = mustGuard(app, true, guardTestLogger, guardTestBuilder(st, &builds)).Authenticate(context.Background(), hdr)
	var ue *UnauthenticatedError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, CodeRevoked, ue.Code)

	_, _, err = g.Authenticate(context.Background(), "")
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, CodeUnauthenticated, ue.Code)
}

// A build error (missing keys) is never cached: every registrar refuses to start; no DB (share=false) or a group
// keeps one verifier per guard and does not warm.
func TestMustGuard_NoSharingWithoutAppOrDB(t *testing.T) {
	app := fiber.New()
	boom := errors.New("passport: keys missing")
	calls := 0
	failing := func() (*passport.Verifier, error) { calls++; return nil, boom }
	for range 2 {
		g := mustGuard(app, true, guardTestLogger, failing)
		require.ErrorIs(t, g.initErr, boom)
	}
	assert.Equal(t, 2, calls)
	_, ok := app.State().Get(verifierStateKey)
	assert.False(t, ok)

	st := &countingStore{}
	builds := 0
	mustGuard(app, false, guardTestLogger, guardTestBuilder(st, &builds))
	mustGuard(app.Group("/x"), true, guardTestLogger, guardTestBuilder(st, &builds))
	assert.Equal(t, 2, builds)
	assert.Equal(t, 0, st.listCount(), "no warm-up without sharing")
}
