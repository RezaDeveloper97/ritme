package assistant

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/internal/i18n"
)

// Sealer is the PRIVATE_NOTE_KEY AEAD box (*loss.NoteBox, B-N7-04 pattern): message bodies and summaries are sealed
// at rest (B-N7-06b H1), each bound to its table, row and owner through the additional data, so a value copied into
// another row or account does not open.
type Sealer interface {
	SealWith(plain string, ad []byte) (string, error)
	OpenWith(stored string, ad []byte) (string, error)
	Disabled() bool
}

// sealedPrefix starts every sealed value ("v1:<kid>:<base64>"). Rows written before B-N7-06b hold plaintext without
// it; they are read as they are (dev / stage only — production never ran the plaintext version).
const sealedPrefix = "v1:"

// ErrSealUnavailable: no PRIVATE_NOTE_KEY outside local / testing — the assistant fails closed (503).
var ErrSealUnavailable = errors.New("assistant: encryption key unavailable")

func ad(parts ...any) []byte {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(':')
		}
		switch v := p.(type) {
		case string:
			b.WriteString(v)
		case uint64:
			b.WriteString(strconv.FormatUint(v, 10))
		}
	}
	return []byte(b.String())
}

func messageAD(userID, conversationID, id uint64) []byte {
	return ad("assistant_messages.body", userID, conversationID, id)
}

func summaryAD(userID, conversationID uint64) []byte {
	return ad("assistant_conversations.summary", userID, conversationID)
}

func handoffAD(userID, conversationID, bookingID uint64) []byte {
	return ad("assistant_handoffs.summary", userID, conversationID, bookingID)
}

func (s *Service) seal(plain string, data []byte) (string, error) {
	if s.o.Box == nil || s.o.Box.Disabled() {
		return "", ErrSealUnavailable
	}
	if plain == "" {
		return "", nil
	}
	return s.o.Box.SealWith(plain, data)
}

// open reads a stored value: legacy plaintext as is, a sealed value decrypted; one that does not open (a rotated-out
// key, a moved row) reads as "" — never the ciphertext.
func (s *Service) open(stored string, data []byte) string {
	if !strings.HasPrefix(stored, sealedPrefix) {
		return stored
	}
	if s.o.Box == nil {
		return ""
	}
	plain, err := s.o.Box.OpenWith(stored, data)
	if err != nil {
		return ""
	}
	return plain
}

// SealGate answers every assistant route with 503 storage_unavailable while the box is disabled (fail closed), and
// runs next (the auth guard) otherwise.
func SealGate(box Sealer, next fiber.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		if box == nil || box.Disabled() {
			return failf(fiber.StatusServiceUnavailable, "storage_unavailable", CodeStorage, i18n.Locale(c))
		}
		return next(c)
	}
}
