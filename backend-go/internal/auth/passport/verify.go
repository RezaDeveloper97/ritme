package passport

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/clock"
)

// The 401 error codes of the API contract (backend/bootstrap/app.php). Clients drop their
// session only on these, so the mapping below must stay exactly Laravel's.
const (
	CodeRevoked         = "token_revoked"
	CodeExpired         = "token_expired"
	CodeUnauthenticated = "unauthenticated"
)

// UserProvider is the Passport guard's user provider name (config/auth.php guards.api.provider).
const UserProvider = "users"

// AuthError is a rejected bearer token. Code is one of the Code* constants; Reason is for
// logs only and never reaches the client.
type AuthError struct {
	Code   string
	Reason string
}

func (e *AuthError) Error() string { return "passport: " + e.Code + ": " + e.Reason }

func reject(code, format string, args ...any) *AuthError {
	return &AuthError{Code: code, Reason: fmt.Sprintf(format, args...)}
}

// VerifierStore is the data the guard reads (satisfied by *store.Queries).
type VerifierStore interface {
	GetAccessToken(ctx context.Context, id string) (store.GetAccessTokenRow, error)
	GetClient(ctx context.Context, id string) (store.GetClientRow, error)
	GetUserByID(ctx context.Context, id uint64) (store.User, error)
}

// Session is an authenticated request: the user and the access token it presented.
type Session struct {
	User  store.User
	Token store.GetAccessTokenRow // the oauth_access_tokens row (id = jti), incl. expires_at
}

// Verifier is the `auth:api` guard (Passport TokenGuard + league BearerTokenValidator).
type Verifier struct {
	pub    *rsa.PublicKey
	store  VerifierStore
	clock  clock.Clock
	parser *jwt.Parser
}

// NewVerifier builds a verifier. clk decides token time validity; production passes
// clock.Real{} — like league's SystemClock, the request's X-Test-Now is not consulted.
func NewVerifier(pub *rsa.PublicKey, st VerifierStore, clk clock.Clock) *Verifier {
	return &Verifier{
		pub:   pub,
		store: st,
		clock: clk,
		// RS256 only; time claims are checked below with µs precision and no leeway
		// (golang-jwt's own validation truncates NumericDate to seconds).
		parser: jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithoutClaimsValidation()),
	}
}

// bearerPrefix is the pattern league's BearerTokenValidator strips from the header: /^\s*Bearer\s/.
var bearerPrefix = regexp.MustCompile(`^[ \t\n\r\v\f]*Bearer[ \t\n\r\v\f]`)

// BearerToken is Illuminate\Http\Request::bearerToken(): the text after the last
// "Bearer " (case-insensitive), cut at the first comma; "" when there is none.
func BearerToken(header string) string {
	pos := strings.LastIndex(strings.ToLower(header), "bearer ")
	if pos < 0 {
		return ""
	}
	tok := header[pos+len("bearer "):]
	if i := strings.IndexByte(tok, ','); i >= 0 {
		tok = tok[:i]
	}
	return tok
}

// jwtFromHeader is what league's BearerTokenValidator parses: the header with a leading
// "Bearer " removed, trimmed.
func jwtFromHeader(header string) string {
	return strings.Trim(bearerPrefix.ReplaceAllString(header, ""), " \t\n\r\x00\x0B")
}

// Verify authenticates an Authorization header value. It returns *AuthError for every
// rejection (401) and a plain error for infrastructure failures (500).
//
// Order (as in Passport): Bearer present → signature (RS256) and LooseValidAt without
// leeway → token row id=jti AND revoked=0 (DB expires_at is NOT checked) → client by aud
// (exists, not revoked, provider NULL or "users") → user by sub.
func (v *Verifier) Verify(ctx context.Context, authorization string) (*Session, error) {
	if BearerToken(authorization) == "" {
		return nil, reject(CodeUnauthenticated, "no bearer token")
	}
	c, err := v.parseAndValidate(jwtFromHeader(authorization))
	if err != nil {
		return nil, err
	}

	row, err := v.store.GetAccessToken(ctx, c.jti)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, reject(CodeRevoked, "token row missing")
	case err != nil:
		return nil, fmt.Errorf("passport: load token: %w", err)
	case row.Revoked:
		return nil, reject(CodeRevoked, "token revoked")
	}

	client, err := v.store.GetClient(ctx, c.aud)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, reject(CodeUnauthenticated, "client not found")
	case err != nil:
		return nil, fmt.Errorf("passport: load client: %w", err)
	case client.Revoked:
		return nil, reject(CodeUnauthenticated, "client revoked")
	case phpTruthy(client.Provider) && client.Provider.String != UserProvider:
		return nil, reject(CodeUnauthenticated, "client provider %q", client.Provider.String)
	}

	if c.sub == 0 {
		return nil, reject(CodeUnauthenticated, "no subject")
	}
	user, err := v.store.GetUserByID(ctx, c.sub)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, reject(CodeUnauthenticated, "user not found")
	case err != nil:
		return nil, fmt.Errorf("passport: load user: %w", err)
	}
	return &Session{User: user, Token: row}, nil
}

// phpTruthy is `$client->provider` in a boolean context ("" and "0" are false).
func phpTruthy(s sql.NullString) bool { return s.Valid && s.String != "" && s.String != "0" }

// rawClaims keeps the claims undecoded; they are interpreted like lcobucci's parser does.
type rawClaims struct {
	Aud json.RawMessage `json:"aud"`
	Jti json.RawMessage `json:"jti"`
	Iat json.RawMessage `json:"iat"`
	Nbf json.RawMessage `json:"nbf"`
	Exp json.RawMessage `json:"exp"`
	Sub json.RawMessage `json:"sub"`
}

// jwt.Claims — never used for validation (WithoutClaimsValidation).
func (rawClaims) GetExpirationTime() (*jwt.NumericDate, error) { return nil, nil } //nolint:nilnil // interface stub
func (rawClaims) GetIssuedAt() (*jwt.NumericDate, error)       { return nil, nil } //nolint:nilnil // interface stub
func (rawClaims) GetNotBefore() (*jwt.NumericDate, error)      { return nil, nil } //nolint:nilnil // interface stub
func (rawClaims) GetIssuer() (string, error)                   { return "", nil }
func (rawClaims) GetSubject() (string, error)                  { return "", nil }
func (rawClaims) GetAudience() (jwt.ClaimStrings, error)       { return nil, nil }

type claims struct {
	aud string
	jti string
	sub uint64 // 0 = missing / not a user id
}

func (v *Verifier) parseAndValidate(raw string) (*claims, error) {
	var rc rawClaims
	if _, err := v.parser.ParseWithClaims(raw, &rc, func(*jwt.Token) (any, error) { return v.pub, nil }); err != nil {
		return nil, reject(CodeUnauthenticated, "parse/signature: %v", err)
	}

	// LooseValidAt: every violated constraint is collected; "expired" alone → token_expired.
	now := v.clock.Now()
	var violations []string
	for _, tc := range []struct {
		name string
		raw  json.RawMessage
		bad  func(t time.Time) bool
	}{
		{"iat", rc.Iat, now.Before},                                       // issued in the future
		{"nbf", rc.Nbf, now.Before},                                       // cannot be used yet
		{"exp", rc.Exp, func(t time.Time) bool { return !now.Before(t) }}, // now >= exp: expired
	} {
		if isAbsent(tc.raw) {
			continue
		}
		t, err := parseDate(tc.raw)
		if err != nil {
			return nil, reject(CodeUnauthenticated, "%s: %v", tc.name, err)
		}
		if tc.bad(t) {
			violations = append(violations, tc.name)
		}
	}
	if len(violations) == 1 && violations[0] == "exp" {
		return nil, reject(CodeExpired, "expired")
	}
	if len(violations) > 0 {
		return nil, reject(CodeUnauthenticated, "time constraints %v", violations)
	}

	c := &claims{}
	var err error
	if c.aud, err = parseAudience(rc.Aud); err != nil {
		return nil, reject(CodeUnauthenticated, "aud: %v", err)
	}
	if err := json.Unmarshal(rc.Jti, &c.jti); err != nil || c.jti == "" {
		// isAccessTokenRevoked() cannot find a row for a missing jti.
		return nil, reject(CodeRevoked, "no jti")
	}
	c.sub = parseSubject(rc.Sub)
	return c, nil
}

func isAbsent(raw json.RawMessage) bool { return len(raw) == 0 || string(raw) == "null" }

// parseDate is lcobucci Parser::convertDate: a number or numeric string of unix seconds,
// normalised with number_format($ts, 6) and read with microsecond precision.
func parseDate(raw json.RawMessage) (time.Time, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return time.Time{}, errors.New("not a timestamp")
		}
		s = n.String()
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return time.Time{}, errors.New("not numeric")
	}
	norm := strconv.FormatFloat(f, 'f', 6, 64)
	secPart, usPart, _ := strings.Cut(norm, ".")
	sec, err := strconv.ParseInt(secPart, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	us, err := strconv.ParseInt(usPart, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if strings.HasPrefix(secPart, "-") {
		us = -us
	}
	return time.Unix(sec, us*1000), nil
}

// parseAudience: lcobucci turns aud into a list; league's convertSingleRecordAudToString
// makes a one-element list a string. Anything else cannot name a client.
func parseAudience(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil || len(list) != 1 {
		return "", errors.New("not a single audience")
	}
	return list[0], nil
}

// parseSubject returns the user id in sub ("1004" or 1004); 0 when absent or not an id
// (`oauth_user_id ?: null` → no user).
func parseSubject(raw json.RawMessage) uint64 {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return 0
		}
		s = n.String()
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
