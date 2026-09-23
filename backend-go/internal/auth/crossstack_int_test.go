package auth_test

// Cross-stack token interop against the Laravel contract stack (docker-compose.contract.yml,
// docs/go-migration/contract.md): Laravel-issued tokens are accepted by Go, Go-issued tokens
// by Laravel, and a revocation on either side is seen by the other as token_revoked.
//
// Both stacks sign with contract/fixtures/keys. In production they share one database; here
// the Go side has its own test DB, so the oauth_access_tokens row is copied across after every
// step to model that sharing.
//
// Runs only with TEST_DB_DSN (make test-int) AND a reachable contract stack:
//
//	CONTRACT_PORT=18090 make test-int PKG=./internal/auth/...
//
// It serialises with the contract recorder through contract/.work/record.lock.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ritme/backend-go/internal/auth"
	"github.com/ritme/backend-go/internal/auth/passport"
	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/db/testdb"
)

type laravelStack struct {
	base    string
	compose string
}

func contractStack(t *testing.T) *laravelStack {
	t.Helper()
	if !testdb.Enabled() {
		t.Skip("TEST_DB_DSN unset")
	}
	base := os.Getenv("CONTRACT_BASE_URL")
	if base == "" {
		port := os.Getenv("CONTRACT_PORT")
		if port == "" {
			port = "8090"
		}
		base = "http://127.0.0.1:" + port
	}
	// Probe with the 401 contract itself, so an unrelated service on the port is not mistaken
	// for the Laravel stack.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, base+"/api/v1/auth/user", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("Laravel contract stack not reachable at %s (set CONTRACT_PORT): %v", base, err)
	}
	var probe struct {
		ErrorCode string `json:"error_code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&probe)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || probe.ErrorCode != auth.CodeUnauthenticated {
		t.Skipf("no Laravel contract stack at %s (set CONTRACT_PORT)", base)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	// Serialise with `make contract-record` (it resets the shared Laravel DB).
	_ = os.MkdirAll(filepath.Join(repo, "backend-go", "contract", ".work"), 0o750)
	lock, err := os.OpenFile(filepath.Join(repo, "backend-go", "contract", ".work", "record.lock"), os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // fixed path
	require.NoError(t, err)
	require.NoError(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_EX))                         //nolint:gosec // G115: fd fits int
	t.Cleanup(func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); _ = lock.Close() }) //nolint:gosec // G115

	return &laravelStack{base: base, compose: filepath.Join(repo, "docker-compose.contract.yml")}
}

func (l *laravelStack) sql(t *testing.T, query string) string {
	t.Helper()
	out, err := exec.Command("docker", "compose", "-f", l.compose, "exec", "-T", "contract-mariadb", //nolint:gosec,noctx // test helper
		"mariadb", "-uritme", "-pcontract", "ritme_contract", "-N", "-B", "-e", query).CombinedOutput()
	require.NoError(t, err, string(out))
	return strings.TrimSpace(string(out))
}

func (l *laravelStack) call(t *testing.T, method, path, token, body, now string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, l.base+path, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if now != "" {
		req.Header.Set(clock.Header, now)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m
}

func jtiOf(t *testing.T, token string) string {
	t.Helper()
	payload, err := base64.RawURLEncoding.DecodeString(strings.Split(token, ".")[1])
	require.NoError(t, err)
	var c struct {
		Jti string `json:"jti"`
	}
	require.NoError(t, json.Unmarshal(payload, &c))
	return c.Jti
}

func TestCrossStack_TokensInterop(t *testing.T) {
	lara := contractStack(t)
	keys, err := passport.LoadKeys(filepath.Join("..", "..", "contract", "fixtures", "keys"))
	require.NoError(t, err)
	require.Equal(t, clientID, lara.sql(t, "SELECT id FROM oauth_clients WHERE revoked=0 LIMIT 1"),
		"contract DB not normalised (run make contract-record once)")

	db := testdb.New(t)
	_, err = db.Exec(`INSERT INTO oauth_clients (id, name, provider, redirect_uris, grant_types, revoked, created_at, updated_at)
		VALUES (?, 'Ritme Personal Access Client', 'users', '[]', '["personal_access"]', 0, '2026-09-23 09:00:00', '2026-09-23 09:00:00')`, clientID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (id, name, mobile, created_at, updated_at) VALUES (1004, 'Contract regular', '09900000004', NOW(), NOW())`)
	require.NoError(t, err)
	q := store.New(db)
	guard := auth.NewGuardWith(keys.Public, q, clock.Real{}, quiet)
	issuer := passport.NewIssuer(keys.Private, q, clock.Real{}, 365)
	ctx := context.Background()

	// Carbon's request clock for the Laravel calls (JWT validity always uses the real clock).
	now := time.Now().In(time.FixedZone("", 12600)).Format(time.RFC3339)

	// Copy a token row's current state from Laravel's DB into Go's.
	syncFromLaravel := func(jti string) {
		row := lara.sql(t, "SELECT user_id, client_id, revoked, DATE_FORMAT(expires_at,'%Y-%m-%d %H:%i:%s') FROM oauth_access_tokens WHERE id='"+jti+"'")
		f := strings.Split(row, "\t")
		require.Len(t, f, 4, row)
		_, err := db.Exec(`REPLACE INTO oauth_access_tokens (id, user_id, client_id, name, scopes, revoked, created_at, updated_at, expires_at)
			VALUES (?, ?, ?, 'auth_token', '[]', ?, NOW(), NOW(), ?)`, jti, f[0], f[1], f[2], f[3])
		require.NoError(t, err)
	}
	// Copy a Go-issued row into Laravel's DB.
	syncToLaravel := func(jti string) {
		var userID int64
		var clientRow string
		var revoked int
		var expires time.Time
		require.NoError(t, db.QueryRow("SELECT user_id, client_id, revoked, expires_at FROM oauth_access_tokens WHERE id=?", jti).
			Scan(&userID, &clientRow, &revoked, &expires))
		lara.sql(t, fmt.Sprintf("REPLACE INTO oauth_access_tokens (id,user_id,client_id,name,scopes,revoked,created_at,updated_at,expires_at) "+
			"VALUES ('%s',%d,'%s','auth_token','[]',%d,NOW(),NOW(),'%s')", jti, userID, clientRow, revoked, expires.Format(time.DateTime)))
	}
	verify := func(token string) string {
		_, _, err := guard.Authenticate(ctx, "Bearer "+token)
		if err == nil {
			return "ok"
		}
		var ue *auth.UnauthenticatedError
		require.ErrorAs(t, err, &ue)
		return ue.Code
	}

	t.Run("Laravel-issued token is accepted by Go; Laravel logout → token_revoked in Go", func(t *testing.T) {
		// A previous run may have sent an OTP within the 60 s resend window.
		lara.sql(t, "DELETE FROM otp_verifications WHERE mobile='09900000004'")
		status, _ := lara.call(t, http.MethodPost, "/api/v1/auth/send-otp", "", `{"mobile":"09900000004"}`, now)
		require.Equal(t, 200, status)
		code := lara.sql(t, "SELECT code FROM otp_verifications WHERE mobile='09900000004' ORDER BY id DESC LIMIT 1")
		status, body := lara.call(t, http.MethodPost, "/api/v1/auth/verify-otp", "", `{"mobile":"09900000004","code":"`+code+`"}`, now)
		require.Equal(t, 200, status, body)
		token := body["data"].(map[string]any)["access_token"].(string)
		jti := jtiOf(t, token)
		t.Cleanup(func() { lara.sql(t, "DELETE FROM oauth_access_tokens WHERE id='"+jti+"'") })

		syncFromLaravel(jti)
		assert.Equal(t, "ok", verify(token))

		status, _ = lara.call(t, http.MethodPost, "/api/v1/auth/logout", token, "", now)
		require.Equal(t, 200, status)
		syncFromLaravel(jti)
		assert.Equal(t, auth.CodeRevoked, verify(token))
	})

	t.Run("Go-issued token is accepted by Laravel; Go revoke → token_revoked in Laravel", func(t *testing.T) {
		issued, err := issuer.Issue(ctx, 1004, time.Now())
		require.NoError(t, err)
		t.Cleanup(func() { lara.sql(t, "DELETE FROM oauth_access_tokens WHERE id='"+issued.ID+"'") })
		syncToLaravel(issued.ID)

		status, body := lara.call(t, http.MethodGet, "/api/v1/auth/user", issued.AccessToken, "", now)
		require.Equal(t, 200, status, body)
		assert.InDelta(t, 1004, body["data"].(map[string]any)["user"].(map[string]any)["id"], 0)

		require.NoError(t, auth.RevokeToken(ctx, q, issued.ID, time.Now()))
		syncToLaravel(issued.ID)
		status, body = lara.call(t, http.MethodGet, "/api/v1/auth/user", issued.AccessToken, "", now)
		assert.Equal(t, 401, status)
		assert.Equal(t, auth.CodeRevoked, body["error_code"])
	})

	t.Run("expired and garbage tokens get the same codes on both stacks", func(t *testing.T) {
		// Expired: signed by Go with a clock 400 days back (exp = 35 days ago).
		old := passport.NewIssuer(keys.Private, q, clock.At(time.Now().AddDate(0, 0, -400)), 365)
		issued, err := old.Issue(ctx, 1004, time.Now())
		require.NoError(t, err)
		t.Cleanup(func() { lara.sql(t, "DELETE FROM oauth_access_tokens WHERE id='"+issued.ID+"'") })
		syncToLaravel(issued.ID)
		assert.Equal(t, auth.CodeExpired, verify(issued.AccessToken))
		_, body := lara.call(t, http.MethodGet, "/api/v1/auth/user", issued.AccessToken, "", now)
		assert.Equal(t, auth.CodeExpired, body["error_code"])

		// Garbage / wrong signature.
		for _, tok := range []string{"not-a-jwt", issued.AccessToken[:len(issued.AccessToken)-4] + "AAAA"} {
			assert.Equal(t, auth.CodeUnauthenticated, verify(tok))
			_, body := lara.call(t, http.MethodGet, "/api/v1/auth/user", tok, "", now)
			assert.Equal(t, auth.CodeUnauthenticated, body["error_code"])
		}
	})
}
