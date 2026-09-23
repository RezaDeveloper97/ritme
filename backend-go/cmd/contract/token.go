package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Session is an access token for one persona plus its oauth_access_tokens row, so a
// DB reset can put the token back.
type Session struct {
	Persona string
	Token   string // the bearer value ("" = no Authorization header)
	Row     *TokenRow
}

// TokenRow mirrors oauth_access_tokens.
type TokenRow struct {
	ID        string
	UserID    int
	ClientID  string
	Name      string
	Scopes    string
	Revoked   bool
	CreatedAt string // Y-m-d H:i:s Tehran wall-clock
	ExpiresAt string
}

// InsertSQL renders the row as an INSERT.
func (r *TokenRow) InsertSQL() string {
	rev := 0
	if r.Revoked {
		rev = 1
	}
	return fmt.Sprintf("INSERT INTO oauth_access_tokens (id,user_id,client_id,name,scopes,revoked,created_at,updated_at,expires_at) "+
		"VALUES (%s,%d,%s,%s,%s,%d,%s,%s,%s);",
		sqlQuote(r.ID), r.UserID, sqlQuote(r.ClientID), sqlQuote(r.Name), sqlQuote(r.Scopes), rev,
		sqlQuote(r.CreatedAt), sqlQuote(r.CreatedAt), sqlQuote(r.ExpiresAt))
}

func sqlQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(s) + "'"
}

// Minter signs Passport-compatible personal access tokens (league/oauth2-server:
// RS256, claims aud/jti/iat/nbf/exp/sub/scopes, float timestamps with microseconds)
// with the contract key pair. Used for the synthetic 401 personas and for the Go side.
type Minter struct {
	key      *rsa.PrivateKey
	clientID string
	tehran   *time.Location
}

// NewMinter loads a PEM private key (PKCS#1 or PKCS#8).
func NewMinter(keyPath, clientID string) (*Minter, error) {
	raw, err := os.ReadFile(keyPath) //nolint:gosec // G304: contract fixture key
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("%s: no PEM block", keyPath)
	}
	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = k
	} else {
		k8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", keyPath, err)
		}
		rk, ok := k8.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("%s: not an RSA key", keyPath)
		}
		key = rk
	}
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		return nil, err
	}
	return &Minter{key: key, clientID: clientID, tehran: loc}, nil
}

// Mint signs a token for userID that expires at exp, and returns the matching row.
// createdAt is the (test-clock) creation time stored in the row.
func (m *Minter) Mint(userID int, now, exp time.Time, createdAt time.Time, revoked bool) (string, *TokenRow, error) {
	idBytes := make([]byte, 40)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, err
	}
	jti := hex.EncodeToString(idBytes) // 80 hex chars, like Passport
	ts := func(t time.Time) string { return strconv.FormatFloat(float64(t.UnixMicro())/1e6, 'f', 6, 64) }
	header := `{"typ":"JWT","alg":"RS256"}`
	claims := fmt.Sprintf(`{"aud":%q,"jti":%q,"iat":%s,"nbf":%s,"exp":%s,"sub":%q,"scopes":[]}`,
		m.clientID, jti, ts(now), ts(now), ts(exp), strconv.Itoa(userID))
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString([]byte(header)) + "." + enc.EncodeToString([]byte(claims))
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, m.key, crypto.SHA256, sum[:])
	if err != nil {
		return "", nil, err
	}
	row := &TokenRow{
		ID: jti, UserID: userID, ClientID: m.clientID, Name: "auth_token", Scopes: "[]", Revoked: revoked,
		CreatedAt: createdAt.In(m.tehran).Format(time.DateTime),
		ExpiresAt: exp.In(m.tehran).Format(time.DateTime),
	}
	return signing + "." + enc.EncodeToString(sig), row, nil
}

// Synthetic personas that exercise the 401 error codes and refresh-session.
// They all belong to the `regular` persona (user 1004).
var syntheticPersonas = []string{
	"token_revoked", "token_expired", "token_near_expiry", "token_unknown", "token_garbage",
}

// MintSynthetic creates the synthetic sessions.
func (m *Minter) MintSynthetic(now time.Time) ([]Session, error) {
	const user = 1004
	created, err := time.ParseInLocation(time.RFC3339, DefaultNow, m.tehran)
	if err != nil {
		return nil, err
	}
	var out []Session
	add := func(persona string, exp time.Time, revoked, withRow bool) error {
		tok, row, err := m.Mint(user, now, exp, created, revoked)
		if err != nil {
			return err
		}
		if !withRow {
			row = nil
		}
		out = append(out, Session{Persona: persona, Token: tok, Row: row})
		return nil
	}
	year := now.AddDate(1, 0, 0)
	errs := errors.Join(
		add("token_revoked", year, true, true),
		add("token_expired", now.Add(-48*time.Hour), false, true),
		add("token_near_expiry", now.AddDate(0, 0, 10), false, true),
		add("token_unknown", year, false, false),
	)
	if errs != nil {
		return nil, errs
	}
	out = append(out, Session{Persona: "token_garbage", Token: "not-a-jwt"})
	return out, nil
}
