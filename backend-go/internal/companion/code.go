package companion

import (
	"io"

	"github.com/ritme/backend-go/internal/family/invites"
)

// Invite codes live in internal/family/invites, the one invite service (T-M11-08, D-196); these names keep the
// /companions API of the package.

// CodeLength is the length of an invite code (Hamdam_Invite: R T 7 K 2 M).
const CodeLength = invites.CodeLength

// CodeAlphabet has 32 symbols without the look-alikes I, O, 0 and 1 (32^6 ≈ 1.07e9 codes).
const CodeAlphabet = invites.CodeAlphabet

// newCode draws a uniform code from r (crypto/rand in production).
func newCode(r io.Reader) (string, error) { return invites.NewCode(r) }

// NormalizeCode upper-cases a typed code, drops spaces and dashes and maps Persian / Arabic-Indic digits to ASCII.
func NormalizeCode(in string) (string, bool) { return invites.NormalizeCode(in) }

// hashCode is the at-rest form of a code: hex HMAC-SHA256 under the service pepper (domain-separated).
func (s *Service) hashCode(code string) string { return s.inv.Hash(code) }

// NormalizeMobile returns the 09xxxxxxxxx form of a mobile number, or ok=false.
func NormalizeMobile(in string) (string, bool) { return invites.NormalizeMobile(in) }
