package media

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

// URL lifetimes. A player keeps issuing range requests while it plays, so the link has to outlive a typical
// viewing session's start; when it does expire mid-lesson the stream answers 410 and the client mints a new one.
const (
	DefaultURLTTL = 30 * time.Minute
	MaxURLTTL     = time.Hour
)

// DevKey is the public development URL key (only where config.Media.ResolveKey allows it: never in production or
// on stage).
func DevKey() []byte {
	k := sha256.Sum256([]byte("ritme-media-dev-url-key"))
	return k[:]
}

// Signer mints and verifies playback URLs bound to one media id, one viewer and an expiry.
type Signer struct{ key []byte }

// NewSigner returns a signer for key (nil for an empty key: playback disabled).
func NewSigner(key []byte) *Signer {
	if len(key) == 0 {
		return nil
	}
	return &Signer{key: append([]byte(nil), key...)}
}

func (s *Signer) mac(id, viewer uint64, expires int64) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte("ritme-media-url:v1\n" + strconv.FormatUint(id, 10) + "\n" + strconv.FormatUint(viewer, 10) + "\n" +
		strconv.FormatInt(expires, 10)))
	return m.Sum(nil)
}

// Sign returns the signature of (id, viewer) valid until expires.
func (s *Signer) Sign(id, viewer uint64, expires time.Time) string {
	return base64.RawURLEncoding.EncodeToString(s.mac(id, viewer, expires.Unix()))
}

// Verify checks a link at now: ErrLinkInvalid for a malformed / wrong signature or an expiry further away than
// MaxURLTTL, ErrLinkExpired once it has passed (checked after the signature, so it says nothing about a forged link).
func (s *Signer) Verify(id uint64, viewerRaw, expiresRaw, signature string, now time.Time) (uint64, error) {
	if s == nil {
		return 0, ErrDisabled
	}
	viewer, err := strconv.ParseUint(viewerRaw, 10, 64)
	if err != nil || viewer == 0 || strconv.FormatUint(viewer, 10) != viewerRaw {
		return 0, ErrLinkInvalid
	}
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil || expires <= 0 || strconv.FormatInt(expires, 10) != expiresRaw {
		return 0, ErrLinkInvalid
	}
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(got, s.mac(id, viewer, expires)) {
		return 0, ErrLinkInvalid
	}
	switch {
	case expires > now.Add(MaxURLTTL).Unix():
		return 0, ErrLinkInvalid
	case now.Unix() > expires:
		return 0, ErrLinkExpired
	}
	return viewer, nil
}
