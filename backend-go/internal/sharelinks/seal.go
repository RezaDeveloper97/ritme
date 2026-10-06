package sharelinks

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Token format: 32 random bytes, base64url without padding (43 characters).
const (
	tokenBytes  = 32
	sealVersion = "v1"
	hkdfInfo    = "ritme-share-link:v1"
)

var reToken = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// ErrUnreadable: a payload that does not open with the token (tampered, moved, or a wrong token).
var ErrUnreadable = errors.New("sharelinks: snapshot unreadable")

// NewToken returns a fresh link token read from r (crypto/rand.Reader in production).
func NewToken(r io.Reader) (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("sharelinks: token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidToken reports whether s has the token format (anything else is a 404 without a lookup).
func ValidToken(s string) bool { return reToken.MatchString(s) }

// HashToken is the stored lookup key of a token: hex SHA-256.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func aead(token string) (cipher.AEAD, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenBytes {
		return nil, ErrUnreadable
	}
	key, err := hkdf.Key(sha256.New, raw, nil, hkdfInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("sharelinks: key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sharelinks: cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// Seal encrypts plain under token: "v1:" + base64(nonce ‖ ciphertext ‖ tag), additional data = the token hash.
func Seal(token string, plain []byte, r io.Reader) (string, error) {
	a, err := aead(token)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err := io.ReadFull(r, nonce); err != nil {
		return "", fmt.Errorf("sharelinks: nonce: %w", err)
	}
	sealed := a.Seal(nonce, nonce, plain, []byte(HashToken(token)))
	return sealVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a payload sealed under token.
func Open(token, stored string) ([]byte, error) {
	v, body, ok := strings.Cut(stored, ":")
	if !ok || v != sealVersion {
		return nil, ErrUnreadable
	}
	a, err := aead(token)
	if err != nil {
		return nil, err
	}
	sealed, err := base64.StdEncoding.DecodeString(body)
	n := a.NonceSize()
	if err != nil || len(sealed) < n {
		return nil, ErrUnreadable
	}
	plain, err := a.Open(nil, sealed[:n], sealed[n:], []byte(HashToken(token)))
	if err != nil {
		return nil, ErrUnreadable
	}
	return plain, nil
}
