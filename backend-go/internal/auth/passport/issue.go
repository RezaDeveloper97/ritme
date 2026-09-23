package passport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ritme/backend-go/internal/auth/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/phpround"
)

// ErrNoPersonalAccessClient mirrors Passport's RuntimeException when no usable personal
// access client exists (the API answers 500).
var ErrNoPersonalAccessClient = errors.New("passport: personal access client not found for 'users' user provider")

// IssuerStore is the data token issuing touches (satisfied by *store.Queries).
type IssuerStore interface {
	ListPersonalAccessClientCandidates(ctx context.Context) ([]store.ListPersonalAccessClientCandidatesRow, error)
	InsertAccessToken(ctx context.Context, arg store.InsertAccessTokenParams) error
}

// Issuer is $user->createToken('auth_token').
type Issuer struct {
	priv         *rsa.PrivateKey
	store        IssuerStore
	clock        clock.Clock
	lifetimeDays int
}

// NewIssuer builds an issuer. lifetimeDays is PASSPORT_TOKEN_LIFETIME_DAYS — a fixed day
// count (P365D), never "one year". clk stamps iat/nbf/exp and expires_at; production passes
// clock.Real{} (league uses new DateTimeImmutable(), not Carbon's test now).
func NewIssuer(priv *rsa.PrivateKey, st IssuerStore, clk clock.Clock, lifetimeDays int) *Issuer {
	return &Issuer{priv: priv, store: st, clock: clk, lifetimeDays: lifetimeDays}
}

// Issued is a freshly created token.
type Issued struct {
	AccessToken string    // the JWT
	ID          string    // jti = oauth_access_tokens.id
	ClientID    string    // aud
	ExpiresAt   time.Time // oauth_access_tokens.expires_at as stored (whole seconds, Asia/Tehran)
}

// jwtHeader is lcobucci's header for RS256 (no kid).
const jwtHeader = `{"typ":"JWT","alg":"RS256"}`

// Issue creates a token for userID and inserts its row. createdAt is the row's
// created_at/updated_at (Eloquent's now(): the request clock).
func (i *Issuer) Issue(ctx context.Context, userID uint64, createdAt time.Time) (*Issued, error) {
	clientID, err := i.personalAccessClient(ctx)
	if err != nil {
		return nil, err
	}
	idBytes := make([]byte, 40)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("passport: token id: %w", err)
	}
	jti := hex.EncodeToString(idBytes) // 80 hex chars, bin2hex(random_bytes(40))

	now := i.clock.Now().In(civildate.Tehran).Truncate(time.Microsecond)
	exp := now.AddDate(0, 0, i.lifetimeDays)

	token, err := i.sign(clientID, jti, strconv.FormatUint(userID, 10), now, exp)
	if err != nil {
		return nil, err
	}
	expiresAt := exp.Truncate(time.Second)
	created := createdAt.In(civildate.Tehran).Truncate(time.Second)
	err = i.store.InsertAccessToken(ctx, store.InsertAccessTokenParams{
		ID:        jti,
		UserID:    sql.NullInt64{Int64: int64(userID), Valid: true}, //nolint:gosec // G115: user ids fit int64
		ClientID:  clientID,
		CreatedAt: sql.NullTime{Time: created, Valid: true},
		UpdatedAt: sql.NullTime{Time: created, Valid: true},
		ExpiresAt: sql.NullTime{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("passport: insert token: %w", err)
	}
	return &Issued{AccessToken: token, ID: jti, ClientID: clientID, ExpiresAt: expiresAt}, nil
}

// personalAccessClient is ClientRepository::personalAccessClient('users'): the latest
// non-revoked client with provider NULL or "users" whose grant_types has personal_access.
func (i *Issuer) personalAccessClient(ctx context.Context) (string, error) {
	rows, err := i.store.ListPersonalAccessClientCandidates(ctx)
	if err != nil {
		return "", fmt.Errorf("passport: load clients: %w", err)
	}
	for _, r := range rows {
		var grants []string
		if json.Unmarshal([]byte(r.GrantTypes), &grants) == nil && slices.Contains(grants, "personal_access") {
			return r.ID, nil
		}
	}
	return "", ErrNoPersonalAccessClient
}

// sign builds the JWT exactly as league's AccessTokenTrait::convertToJWT does: claims
// aud, jti, iat, nbf, exp, sub, scopes in that order, dates as float seconds with
// microseconds (an int when the microseconds are 0), aud as a string, scopes [].
func (i *Issuer) sign(aud, jti, sub string, now, exp time.Time) (string, error) {
	iat, err := unixTimestamp(now)
	if err != nil {
		return "", err
	}
	expTS, err := unixTimestamp(exp)
	if err != nil {
		return "", err
	}
	audJSON, _ := json.Marshal(aud) //nolint:errchkjson // plain strings
	jtiJSON, _ := json.Marshal(jti) //nolint:errchkjson // plain strings
	subJSON, _ := json.Marshal(sub) //nolint:errchkjson // plain strings
	payload := `{"aud":` + string(audJSON) + `,"jti":` + string(jtiJSON) + `,"iat":` + iat + `,"nbf":` + iat +
		`,"exp":` + expTS + `,"sub":` + string(subJSON) + `,"scopes":[]}`

	enc := base64.RawURLEncoding
	signing := enc.EncodeToString([]byte(jwtHeader)) + "." + enc.EncodeToString([]byte(payload))
	sig, err := jwt.SigningMethodRS256.Sign(signing, i.priv)
	if err != nil {
		return "", fmt.Errorf("passport: sign: %w", err)
	}
	return signing + "." + enc.EncodeToString(sig), nil
}

// unixTimestamp is lcobucci's MicrosecondBasedDateConversion + json_encode:
// (int) U when the microseconds are 0, else (float) "U.u" in PHP's shortest float form.
func unixTimestamp(t time.Time) (string, error) {
	us := t.Nanosecond() / 1000
	if us == 0 {
		return strconv.FormatInt(t.Unix(), 10), nil
	}
	f, err := strconv.ParseFloat(fmt.Sprintf("%d.%06d", t.Unix(), us), 64)
	if err != nil {
		return "", fmt.Errorf("passport: timestamp: %w", err)
	}
	s, err := phpround.JSONFloat(f)
	if err != nil {
		return "", fmt.Errorf("passport: timestamp: %w", err)
	}
	return s, nil
}
