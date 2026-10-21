package passport_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
	"github.com/ritme/backend-go/internal/platform/metrics"
)

func TestMain(m *testing.M) { testdb.Main(m) }

// T-M9-09 against MariaDB: the auth step costs exactly one query once the client cache is warm, and the session
// rules hold — 365-day lifetime, revoked token rejected on the very next request, blocked (= revoked) and deleted
// users, expired token.
func TestVerifyInt_OneQueryAndSessionRules(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	_, err := pool.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', NULL, 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	res, err := pool.Exec(`INSERT INTO users (mobile, name, created_at, updated_at) VALUES ('09120000909', 'نگار', NOW(), NOW())`)
	require.NoError(t, err)
	uid, err := res.LastInsertId()
	require.NoError(t, err)

	q := store.New(pool)
	issuedAt := time.Now()
	iss := passport.NewIssuer(testKey, q, clock.At(issuedAt), 365)
	tok, err := iss.Issue(ctx, uint64(uid), issuedAt)
	require.NoError(t, err)
	hdr := "Bearer " + tok.AccessToken
	v := passport.NewVerifier(&testKey.PublicKey, q, clock.At(issuedAt))

	verify := func() (*passport.Session, int64, error) {
		cctx, counter := metrics.WithQueryCounter(ctx)
		s, err := v.Verify(cctx, hdr)
		return s, counter.Load(), err
	}

	_, n, err := verify() // cold: + the one client load
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	for range 3 {
		s, n, err := verify()
		require.NoError(t, err)
		assert.Equal(t, int64(1), n, "auth step = one query")
		assert.Equal(t, uint64(uid), s.User.ID)
		assert.Equal(t, "نگار", s.User.Name.String)
		assert.Equal(t, tok.ID, s.Token.ID)
		assert.Equal(t, clientID, s.Token.ClientID)
		assert.Equal(t, uid, s.Token.UserID.Int64)
		require.True(t, s.Token.ExpiresAt.Valid)
		assert.WithinDuration(t, issuedAt.AddDate(0, 0, 365), s.Token.ExpiresAt.Time, 2*time.Second, "365-day session")
	}

	// Still valid one second before 365 days, expired at 365 days.
	_, err = passport.NewVerifier(&testKey.PublicKey, q, clock.At(issuedAt.AddDate(0, 0, 365).Add(-time.Second))).Verify(ctx, hdr)
	require.NoError(t, err)
	_, err = passport.NewVerifier(&testKey.PublicKey, q, clock.At(issuedAt.AddDate(0, 0, 365))).Verify(ctx, hdr)
	assert.Equal(t, passport.CodeExpired, codeOf(t, err))

	// Deleted user (token row still live): unauthenticated.
	_, err = pool.Exec("DELETE FROM users WHERE id = ?", uid)
	require.NoError(t, err)
	_, _, err = verify()
	assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
	_, err = pool.Exec(`INSERT INTO users (id, mobile, created_at, updated_at) VALUES (?, '09120000909', NOW(), NOW())`, uid)
	require.NoError(t, err)
	_, _, err = verify()
	require.NoError(t, err)

	// Admin block revokes every token: rejected on the very next request with token_revoked.
	_, err = pool.Exec("UPDATE users SET blocked_at = NOW() WHERE id = ?", uid)
	require.NoError(t, err)
	_, err = q.RevokeUserAccessTokens(ctx, store.RevokeUserAccessTokensParams{
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true}, UserID: sql.NullInt64{Int64: uid, Valid: true},
	})
	require.NoError(t, err)
	_, n, err = verify()
	assert.Equal(t, passport.CodeRevoked, codeOf(t, err))
	assert.Equal(t, int64(1), n, "a revoked token is answered from the joined row")
}

// An unknown jti is token_revoked (the separate token read tells it from a missing user).
func TestVerifyInt_UnknownToken(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	_, err := pool.Exec(`INSERT INTO oauth_clients (id, name, secret, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'c', NULL, 'users', '[]', '["personal_access"]', 0, NOW(), NOW())`, clientID)
	require.NoError(t, err)
	_, err = pool.Exec(`INSERT INTO users (id, mobile, created_at, updated_at) VALUES (1004, '09120000910', NOW(), NOW())`)
	require.NoError(t, err)
	v := passport.NewVerifier(&testKey.PublicKey, store.New(pool), clock.At(now))
	_, err = v.Verify(ctx, "Bearer "+mint(t, testKey, claims("no-such-jti", ts(now), ts(now.AddDate(1, 0, 0)))))
	assert.Equal(t, passport.CodeRevoked, codeOf(t, err))
}
