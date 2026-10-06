package files

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// Signed download URLs: GET /api/v1/files/{id}/download?expires=<unix>&signature=<base64url>, where
//
//	signature = HMAC-SHA256(K, "ritme-file-url:v1\n<id>\n<owner>\n<purpose>\n<path>\n<expires>")
//	K         = HMAC-SHA256(file key, "ritme-file-url-key:v1")   (never the AES key itself)
//
// The verifier recomputes it from the stored row, so a link opens exactly one blob of one owner: another id, another
// owner's file, a re-uploaded file under the same id (another path) or a changed expiry all fail. Links live at most
// MaxLinkTTL; rotating the file key invalidates outstanding links (they are minutes long anyway).

// DefaultLinkTTL / MaxLinkTTL bound a signed link's lifetime.
const (
	DefaultLinkTTL = 5 * time.Minute
	MaxLinkTTL     = 15 * time.Minute
)

// Signer mints and verifies signed download links.
type Signer struct{ key []byte }

// NewSigner derives the signing key from the vault's current key (nil for a disabled vault).
func NewSigner(v *Vault) *Signer {
	if v.Disabled() {
		return nil
	}
	m := hmac.New(sha256.New, v.current)
	m.Write([]byte("ritme-file-url-key:v1"))
	return &Signer{key: m.Sum(nil)}
}

func (s *Signer) mac(f File, expires int64) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte("ritme-file-url:v1\n" + strconv.FormatUint(f.ID, 10) + "\n" + strconv.FormatUint(f.Owner, 10) + "\n" +
		f.Purpose + "\n" + f.Path + "\n" + strconv.FormatInt(expires, 10)))
	return m.Sum(nil)
}

// Sign returns the signature of f valid until expires.
func (s *Signer) Sign(f File, expires time.Time) string {
	return base64.RawURLEncoding.EncodeToString(s.mac(f, expires.Unix()))
}

// Verify checks a link for f at now: ErrLinkInvalid for a wrong / malformed signature or an expiry further away
// than MaxLinkTTL, ErrLinkExpired once expires has passed (checked after the signature, so it says nothing about
// a forged link).
func (s *Signer) Verify(f File, expiresRaw, signature string, now time.Time) error {
	if s == nil {
		return ErrDisabled
	}
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil || expires <= 0 || strconv.FormatInt(expires, 10) != expiresRaw {
		return ErrLinkInvalid
	}
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(got, s.mac(f, expires)) {
		return ErrLinkInvalid
	}
	switch {
	case expires > now.Add(MaxLinkTTL).Unix():
		return ErrLinkInvalid
	case now.Unix() > expires:
		return ErrLinkExpired
	}
	return nil
}
