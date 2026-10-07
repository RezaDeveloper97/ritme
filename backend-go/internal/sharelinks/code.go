package sharelinks

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// The 24h doctor code (CB-REC-03, nbl_Rec_Share «دسترسی موقت با کد»): CodeLength characters from CodeAlphabet,
// shown as two groups of four ("K7QM-3XRA"). The alphabet leaves out 0/O, 1/I/L and U so a code read aloud or typed
// from paper is unambiguous: 30^8 ≈ 6.6·10^11 codes (~39 bits). A code is not a secret on its own the way a 256-bit
// token is, so:
//
//   - the database keeps only HMAC-SHA256(pepper, code) (`code_hash`), never the code;
//   - the link token is wrapped under a key derived from pepper ‖ code (`code_payload`), so the code opens the
//     report only together with the server pepper — a database dump alone cannot be brute-forced offline;
//   - the public code lookup is throttled per IP and guarded by a global failed-lookup breaker (internal/http), so
//     online guessing is pointless (see docs in routes_recordsharing.go);
//   - a code lives LinkTTL24h and dies with its link (revoke / expiry wipe `code_payload`).
const (
	CodeAlphabet = "23456789ABCDEFGHJKMNPQRSTVWXYZ"
	CodeLength   = 8
	codeInfo     = "ritme-share-code:v1"
)

// devCodePepper is the public development pepper (APP_ENV other than production without SHARE_CODE_PEPPER).
var devCodePepper = []byte("ritme-share-code-dev-pepper-not-secret")

// NewCode returns a fresh code (unformatted, CodeLength characters) read from r without modulo bias.
func NewCode(r io.Reader) (string, error) {
	const n = len(CodeAlphabet)
	limit := byte(256 - 256%n) // reject bytes ≥ limit
	out := make([]byte, 0, CodeLength)
	buf := make([]byte, CodeLength*2)
	for len(out) < CodeLength {
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", fmt.Errorf("sharelinks: code: %w", err)
		}
		for _, b := range buf {
			if b < limit && len(out) < CodeLength {
				out = append(out, CodeAlphabet[int(b)%n])
			}
		}
	}
	return string(out), nil
}

// FormatCode is the code as shown: "XXXX-XXXX".
func FormatCode(code string) string {
	if len(code) != CodeLength {
		return code
	}
	return code[:4] + "-" + code[4:]
}

// NormalizeCode turns what a person typed into a code: Persian / Arabic digits to ASCII, upper case, spaces and
// dashes dropped. ok=false for anything that cannot be a code (no lookup is made for it).
func NormalizeCode(typed string) (string, bool) {
	var b strings.Builder
	for _, r := range typed {
		switch {
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r == ' ' || r == '-' || r == '‌' || r == '\t':
			continue
		}
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		if r > 127 || !strings.ContainsRune(CodeAlphabet, r) {
			return "", false
		}
		b.WriteRune(r)
		if b.Len() > CodeLength {
			return "", false
		}
	}
	if b.Len() != CodeLength {
		return "", false
	}
	return b.String(), true
}

// Coder hashes codes and wraps link tokens under them with the server pepper.
type Coder struct{ pepper []byte }

// NewCoder returns a Coder; an empty pepper takes the public development pepper (the caller decides whether that is
// allowed — config.Sharing.PepperMissing).
func NewCoder(pepper []byte) *Coder {
	if len(pepper) == 0 {
		pepper = devCodePepper
	}
	return &Coder{pepper: pepper}
}

// Hash is the stored lookup key of a normalized code: hex HMAC-SHA256.
func (c *Coder) Hash(code string) string {
	m := hmac.New(sha256.New, c.pepper)
	m.Write([]byte(codeInfo + ":" + code))
	return hex.EncodeToString(m.Sum(nil))
}

func (c *Coder) aead(code string) (cipher.AEAD, error) {
	secret := append(append([]byte{}, c.pepper...), code...)
	key, err := hkdf.Key(sha256.New, secret, nil, codeInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("sharelinks: code key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("sharelinks: code cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

// Wrap seals token under code: "v1:" + base64(nonce ‖ ciphertext ‖ tag), additional data = the code hash.
func (c *Coder) Wrap(code, token string, r io.Reader) (string, error) {
	a, err := c.aead(code)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err := io.ReadFull(r, nonce); err != nil {
		return "", fmt.Errorf("sharelinks: code nonce: %w", err)
	}
	sealed := a.Seal(nonce, nonce, []byte(token), []byte(c.Hash(code)))
	return sealVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// Unwrap opens a token wrapped under code (ErrUnreadable when it does not open).
func (c *Coder) Unwrap(code, stored string) (string, error) {
	v, body, ok := strings.Cut(stored, ":")
	if !ok || v != sealVersion {
		return "", ErrUnreadable
	}
	a, err := c.aead(code)
	if err != nil {
		return "", err
	}
	sealed, err := base64.StdEncoding.DecodeString(body)
	n := a.NonceSize()
	if err != nil || len(sealed) < n {
		return "", ErrUnreadable
	}
	plain, err := a.Open(nil, sealed[:n], sealed[n:], []byte(c.Hash(code)))
	if err != nil {
		return "", ErrUnreadable
	}
	return string(plain), nil
}
