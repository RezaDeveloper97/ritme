package passport_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

const clientID = "0199c0de-0000-7000-8000-00000c0ffee1"

// fakeStore is an in-memory oauth_* / users store.
type fakeStore struct {
	tokens  map[string]store.GetAccessTokenRow
	clients map[string]store.GetClientRow
	users   map[uint64]store.User
	cands   []store.ListPersonalAccessClientCandidatesRow
}

func newStore() *fakeStore {
	return &fakeStore{
		tokens:  map[string]store.GetAccessTokenRow{},
		clients: map[string]store.GetClientRow{clientID: {ID: clientID, Provider: sql.NullString{String: "users", Valid: true}}},
		users:   map[uint64]store.User{1004: {ID: 1004}},
		cands:   []store.ListPersonalAccessClientCandidatesRow{{ID: clientID, GrantTypes: `["personal_access"]`}},
	}
}

func (s *fakeStore) GetAccessToken(_ context.Context, id string) (store.GetAccessTokenRow, error) {
	r, ok := s.tokens[id]
	if !ok {
		return r, sql.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) GetClient(_ context.Context, id string) (store.GetClientRow, error) {
	r, ok := s.clients[id]
	if !ok {
		return r, sql.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) GetUserByID(_ context.Context, id uint64) (store.User, error) {
	r, ok := s.users[id]
	if !ok {
		return r, sql.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) ListPersonalAccessClientCandidates(context.Context) ([]store.ListPersonalAccessClientCandidatesRow, error) {
	return s.cands, nil
}

func (s *fakeStore) InsertAccessToken(_ context.Context, a store.InsertAccessTokenParams) error {
	s.tokens[a.ID] = store.GetAccessTokenRow{ID: a.ID, UserID: a.UserID, ClientID: a.ClientID, ExpiresAt: a.ExpiresAt}
	return nil
}

var testKey = func() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return k
}()

var now = time.Date(2026, 9, 23, 6, 30, 0, 123456000, time.UTC)

// mint signs a token the way lcobucci does (raw claims JSON, RS256 unless alg given).
func mint(t *testing.T, key *rsa.PrivateKey, claims string) string {
	t.Helper()
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString([]byte(`{"typ":"JWT","alg":"RS256"}`)) + "." + enc.EncodeToString([]byte(claims))
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	require.NoError(t, err)
	return signing + "." + enc.EncodeToString(sig)
}

func claims(jti string, iat, exp float64, extra ...string) string {
	f := func(x float64) string { b, _ := json.Marshal(x); return string(b) }
	aud := `"` + clientID + `"`
	if len(extra) > 0 {
		aud = extra[0]
	}
	return `{"aud":` + aud + `,"jti":"` + jti + `","iat":` + f(iat) + `,"nbf":` + f(iat) + `,"exp":` + f(exp) + `,"sub":"1004","scopes":[]}`
}

func ts(t time.Time) float64 { return float64(t.UnixMicro()) / 1e6 }

func codeOf(t *testing.T, err error) string {
	t.Helper()
	var ae *passport.AuthError
	require.ErrorAs(t, err, &ae)
	return ae.Code
}

func TestVerify_ErrorCodes(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live", ClientID: clientID}
	st.tokens["revoked"] = store.GetAccessTokenRow{ID: "revoked", Revoked: true}
	st.tokens["expired"] = store.GetAccessTokenRow{ID: "expired"}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	year := now.AddDate(1, 0, 0)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	cases := []struct {
		name, header, code string
	}{
		{"no header", "", passport.CodeUnauthenticated},
		{"basic scheme", "Basic dXNlcjpwYXNz", passport.CodeUnauthenticated},
		{"garbage", "Bearer not-a-jwt", passport.CodeUnauthenticated},
		{"wrong signature", "Bearer " + mint(t, other, claims("live", ts(now), ts(year))), passport.CodeUnauthenticated},
		{"revoked row", "Bearer " + mint(t, testKey, claims("revoked", ts(now), ts(year))), passport.CodeRevoked},
		{"unknown row", "Bearer " + mint(t, testKey, claims("nope", ts(now), ts(year))), passport.CodeRevoked},
		{"expired only", "Bearer " + mint(t, testKey, claims("expired", ts(now.Add(-72*time.Hour)), ts(now.Add(-time.Hour)))), passport.CodeExpired},
		{"expired at exactly now", "Bearer " + mint(t, testKey, claims("expired", ts(now.Add(-time.Hour)), ts(now))), passport.CodeExpired},
		{"expired + wrong signature", "Bearer " + mint(t, other, claims("expired", ts(now.Add(-72*time.Hour)), ts(now.Add(-time.Hour)))), passport.CodeUnauthenticated},
		{"expired + issued in future", "Bearer " + mint(t, testKey, claims("expired", ts(now.Add(time.Hour)), ts(now.Add(-time.Hour)))), passport.CodeUnauthenticated},
		{"issued in future", "Bearer " + mint(t, testKey, claims("live", ts(now.Add(time.Microsecond)), ts(year))), passport.CodeUnauthenticated},
		{"expired + revoked", "Bearer " + mint(t, testKey, claims("revoked", ts(now.Add(-72*time.Hour)), ts(now.Add(-time.Hour)))), passport.CodeExpired},
		{"unknown client", "Bearer " + mint(t, testKey, claims("live", ts(now), ts(year), `"other"`)), passport.CodeUnauthenticated},
		{"two audiences", "Bearer " + mint(t, testKey, claims("live", ts(now), ts(year), `["`+clientID+`","x"]`)), passport.CodeUnauthenticated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := v.Verify(context.Background(), tc.header)
			assert.Equal(t, tc.code, codeOf(t, err))
		})
	}

	t.Run("valid: string aud, single-element array aud", func(t *testing.T) {
		for _, aud := range []string{`"` + clientID + `"`, `["` + clientID + `"]`} {
			s, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year), aud)))
			require.NoError(t, err)
			assert.Equal(t, uint64(1004), s.User.ID)
			assert.Equal(t, "live", s.Token.ID)
		}
	})
}

func TestVerify_NoneAndHMACRejected(t *testing.T) {
	st := newStore()
	st.tokens["live"] = store.GetAccessTokenRow{ID: "live"}
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	body := claims("live", ts(now), ts(now.AddDate(1, 0, 0)))
	var mc jwt.MapClaims
	require.NoError(t, json.Unmarshal([]byte(body), &mc))

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, mc).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(&testKey.PublicKey)
	require.NoError(t, err)
	// Classic key-confusion attack: HS256 keyed with the public key.
	hmac, err := jwt.NewWithClaims(jwt.SigningMethodHS256, mc).SignedString(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	require.NoError(t, err)
	for _, tok := range []string{none, hmac} {
		_, err := v.Verify(context.Background(), "Bearer "+tok)
		assert.Equal(t, passport.CodeUnauthenticated, codeOf(t, err))
	}
}

func TestVerify_ClientAndUserChecks(t *testing.T) {
	year := now.AddDate(1, 0, 0)
	for _, tc := range []struct {
		name   string
		mutate func(*fakeStore)
		code   string
	}{
		{"client revoked", func(s *fakeStore) { c := s.clients[clientID]; c.Revoked = true; s.clients[clientID] = c }, passport.CodeUnauthenticated},
		{"client of another provider", func(s *fakeStore) {
			c := s.clients[clientID]
			c.Provider = sql.NullString{String: "admins", Valid: true}
			s.clients[clientID] = c
		}, passport.CodeUnauthenticated},
		{"user deleted", func(s *fakeStore) { delete(s.users, 1004) }, passport.CodeUnauthenticated},
		{"provider NULL is fine", func(s *fakeStore) { c := s.clients[clientID]; c.Provider = sql.NullString{}; s.clients[clientID] = c }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newStore()
			st.tokens["live"] = store.GetAccessTokenRow{ID: "live"}
			tc.mutate(st)
			v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
			_, err := v.Verify(context.Background(), "Bearer "+mint(t, testKey, claims("live", ts(now), ts(year))))
			if tc.code == "" {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, tc.code, codeOf(t, err))
		})
	}
}

func TestIssue_ClaimsAndRow(t *testing.T) {
	st := newStore()
	st.cands = []store.ListPersonalAccessClientCandidatesRow{
		{ID: "password-client", GrantTypes: `["password","refresh_token"]`},
		{ID: clientID, GrantTypes: `["personal_access"]`},
	}
	iss := passport.NewIssuer(testKey, st, clock.At(now), 365)
	created := time.Date(2026, 9, 23, 10, 0, 0, 999, time.UTC)
	got, err := iss.Issue(context.Background(), 1004, created)
	require.NoError(t, err)

	assert.Regexp(t, `^[0-9a-f]{80}$`, got.ID)
	assert.Equal(t, clientID, got.ClientID)
	parts := strings.Split(got.AccessToken, ".")
	require.Len(t, parts, 3)
	hdr, _ := base64.RawURLEncoding.DecodeString(parts[0])
	assert.Equal(t, `{"typ":"JWT","alg":"RS256"}`, string(hdr))
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	assert.Equal(t,
		`{"aud":"`+clientID+`","jti":"`+got.ID+`","iat":1790145000.123456,"nbf":1790145000.123456,`+
			`"exp":1821681000.123456,"sub":"1004","scopes":[]}`, string(payload))

	// 365 days (P365D), not "1 year"; the row keeps whole seconds.
	assert.Equal(t, now.AddDate(0, 0, 365).Truncate(time.Second).Unix(), got.ExpiresAt.Unix())
	row := st.tokens[got.ID]
	assert.Equal(t, clientID, row.ClientID)
	assert.Equal(t, int64(1004), row.UserID.Int64)

	// The issued token verifies.
	v := passport.NewVerifier(&testKey.PublicKey, st, clock.At(now))
	s, err := v.Verify(context.Background(), "Bearer "+got.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uint64(1004), s.User.ID)

	// ...and expires after exactly 365 days.
	v = passport.NewVerifier(&testKey.PublicKey, st, clock.At(now.AddDate(0, 0, 365)))
	_, err = v.Verify(context.Background(), "Bearer "+got.AccessToken)
	assert.Equal(t, passport.CodeExpired, codeOf(t, err))
}

func TestIssue_WholeSecondIsInt(t *testing.T) {
	st := newStore()
	iss := passport.NewIssuer(testKey, st, clock.At(time.Unix(1_790_000_000, 0)), 365)
	got, err := iss.Issue(context.Background(), 1004, time.Now())
	require.NoError(t, err)
	payload, _ := base64.RawURLEncoding.DecodeString(strings.Split(got.AccessToken, ".")[1])
	assert.Regexp(t, regexp.MustCompile(`"iat":1790000000,"nbf":1790000000,"exp":1821536000,`), string(payload))
}

func TestIssue_NoPersonalAccessClient(t *testing.T) {
	st := newStore()
	st.cands = []store.ListPersonalAccessClientCandidatesRow{{ID: "x", GrantTypes: `["password"]`}}
	_, err := passport.NewIssuer(testKey, st, clock.At(now), 365).Issue(context.Background(), 1004, now)
	assert.ErrorIs(t, err, passport.ErrNoPersonalAccessClient)
}

func TestBearerToken(t *testing.T) {
	assert.Equal(t, "abc", passport.BearerToken("Bearer abc"))
	assert.Equal(t, "abc", passport.BearerToken("bearer abc"))
	assert.Equal(t, "abc", passport.BearerToken("Bearer abc,def"))
	assert.Empty(t, passport.BearerToken("Basic abc"))
	assert.Empty(t, passport.BearerToken(""))
}

func writeKeys(t *testing.T, dir string, key *rsa.PrivateKey, pub *rsa.PublicKey) {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, passport.PrivateKeyFile), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, passport.PublicKeyFile), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0o600))
}

func TestLoadKeys(t *testing.T) {
	dir := t.TempDir()
	writeKeys(t, dir, testKey, &testKey.PublicKey)
	keys, err := passport.LoadKeys(dir)
	require.NoError(t, err)
	assert.True(t, keys.Public.Equal(&testKey.PublicKey))

	// The contract key pair (Laravel-generated) loads too.
	_, err = passport.LoadKeys(filepath.Join("..", "..", "..", "contract", "fixtures", "keys"))
	require.NoError(t, err)

	// Missing files, empty path and mismatched pairs are start-up errors; nothing is generated.
	_, err = passport.LoadKeys(t.TempDir())
	require.Error(t, err)
	_, err = passport.LoadKeys("")
	require.Error(t, err)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	mismatch := t.TempDir()
	writeKeys(t, mismatch, testKey, &other.PublicKey)
	_, err = passport.LoadKeys(mismatch)
	require.Error(t, err)
	entries, _ := os.ReadDir(t.TempDir())
	assert.Empty(t, entries)
}
