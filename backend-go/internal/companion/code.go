package companion

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// CodeLength is the length of an invite code (Hamdam_Invite: R T 7 K 2 M).
const CodeLength = 6

// CodeAlphabet has 32 symbols without the look-alikes I, O, 0 and 1 (32^6 ≈ 1.07e9 codes).
const CodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var mobileRe = regexp.MustCompile(`^09[0-9]{9}$`)

// newCode draws a uniform code from r (crypto/rand in production): 32 symbols, so the low 5 bits of a byte are
// unbiased.
func newCode(r io.Reader) (string, error) {
	buf := make([]byte, CodeLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("companion: code: %w", err)
	}
	out := make([]byte, CodeLength)
	for i, b := range buf {
		out[i] = CodeAlphabet[int(b)&31]
	}
	return string(out), nil
}

// NormalizeCode upper-cases a typed code, drops spaces and dashes and maps Persian / Arabic-Indic digits to ASCII.
// ok is false when the result is not a well-formed code.
func NormalizeCode(in string) (string, bool) {
	var b strings.Builder
	for _, r := range in {
		switch {
		case r == ' ' || r == '-' || r == '‌' || r == '\t':
			continue
		case r >= '۰' && r <= '۹':
			r = '0' + (r - '۰')
		case r >= '٠' && r <= '٩':
			r = '0' + (r - '٠')
		case r >= 'a' && r <= 'z':
			r -= 'a' - 'A'
		}
		b.WriteRune(r)
	}
	code := b.String()
	if len(code) != CodeLength {
		return "", false
	}
	for i := range len(code) {
		if !strings.ContainsRune(CodeAlphabet, rune(code[i])) {
			return "", false
		}
	}
	return code, true
}

// hashCode is the at-rest form of a code: hex HMAC-SHA256 under the service pepper (domain-separated).
func (s *Service) hashCode(code string) string {
	m := hmac.New(sha256.New, s.pepper)
	_, _ = m.Write([]byte("companion-invite:" + code))
	return hex.EncodeToString(m.Sum(nil))
}

// NormalizeMobile returns the 09xxxxxxxxx form of a mobile number (Persian / Arabic-Indic digits and a +98 / 0098 /
// 98 prefix accepted), or ok=false.
func NormalizeMobile(in string) (string, bool) {
	var b strings.Builder
	for _, r := range in {
		switch {
		case r >= '۰' && r <= '۹':
			b.WriteRune('0' + (r - '۰'))
		case r >= '٠' && r <= '٩':
			b.WriteRune('0' + (r - '٠'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '+' || r == '(' || r == ')':
		default:
			return "", false
		}
	}
	m := b.String()
	switch {
	case strings.HasPrefix(m, "0098"):
		m = "0" + m[4:]
	case strings.HasPrefix(m, "98") && len(m) == 12:
		m = "0" + m[2:]
	case strings.HasPrefix(m, "9") && len(m) == 10:
		m = "0" + m
	}
	if !mobileRe.MatchString(m) {
		return "", false
	}
	return m, true
}
