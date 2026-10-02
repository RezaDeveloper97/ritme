package loss

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// noteVersion versions the stored format: "v1:<kid>:" + base64(nonce ‖ AES-256-GCM ciphertext ‖ tag), where kid
// names the key that sealed it (KeyID), so a rotated key can still open older notes.
const noteVersion = "v1"

// devNoteKey keys the notes when PRIVATE_NOTE_KEY is unset in a local / testing environment. Any other environment
// without a key never reaches it: the note routes answer 503 (NoteBox.Disabled).
var devNoteKey = sha256.Sum256([]byte("ritme-private-note-dev-key"))

// ErrNoteUnreadable is a stored note that does not decrypt (an unknown or rotated-out key, a corrupted row).
var ErrNoteUnreadable = errors.New("loss: private note unreadable")

// KeyID is the public id of a key: the first 8 hex digits of its SHA-256 (not the key, not reversible).
func KeyID(key []byte) string {
	sum := sha256.Sum256(key)
	return hex.EncodeToString(sum[:4])
}

type noteKey struct {
	id   string
	aead cipher.AEAD
}

// NoteBox encrypts the private notes at rest. Each ciphertext is bound to its owner and row (the additional data),
// so a note copied into another row or account does not decrypt.
type NoteBox struct {
	keys     []noteKey // keys[0] seals; every key opens its own notes
	rand     io.Reader
	disabled bool
}

func newNoteKey(key []byte) (noteKey, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return noteKey{}, fmt.Errorf("loss: note key: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return noteKey{}, fmt.Errorf("loss: note cipher: %w", err)
	}
	return noteKey{id: KeyID(key), aead: aead}, nil
}

// NewNoteBox builds the box on the current 32-byte key (nil/empty → the development key) plus previous keys that
// still open older notes. disabled = a non-development environment without a key: the routes answer 503.
func NewNoteBox(key []byte, previous [][]byte, disabled bool) (*NoteBox, error) {
	if len(key) == 0 {
		key = devNoteKey[:]
	}
	b := &NoteBox{rand: rand.Reader, disabled: disabled}
	for _, k := range append([][]byte{key}, previous...) {
		nk, err := newNoteKey(k)
		if err != nil {
			return nil, err
		}
		b.keys = append(b.keys, nk)
	}
	return b, nil
}

// Disabled reports whether notes are unavailable.
func (b *NoteBox) Disabled() bool { return b == nil || b.disabled }

func noteAD(userID, lossID uint64) []byte {
	return []byte("loss-note:" + strconv.FormatUint(userID, 10) + ":" + strconv.FormatUint(lossID, 10))
}

// Seal encrypts plain for (userID, lossID) with the current key.
func (b *NoteBox) Seal(plain string, userID, lossID uint64) (string, error) {
	k := b.keys[0]
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := io.ReadFull(b.rand, nonce); err != nil {
		return "", fmt.Errorf("loss: note nonce: %w", err)
	}
	sealed := k.aead.Seal(nonce, nonce, []byte(plain), noteAD(userID, lossID))
	return noteVersion + ":" + k.id + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a stored note of (userID, lossID) with the key named in it.
func (b *NoteBox) Open(stored string, userID, lossID uint64) (string, error) {
	parts := strings.SplitN(stored, ":", 3)
	if len(parts) != 3 || parts[0] != noteVersion {
		return "", ErrNoteUnreadable
	}
	for _, k := range b.keys {
		if k.id != parts[1] {
			continue
		}
		sealed, err := base64.StdEncoding.DecodeString(parts[2])
		n := k.aead.NonceSize()
		if err != nil || len(sealed) < n {
			return "", ErrNoteUnreadable
		}
		plain, err := k.aead.Open(nil, sealed[:n], sealed[n:], noteAD(userID, lossID))
		if err != nil {
			return "", ErrNoteUnreadable
		}
		return string(plain), nil
	}
	return "", ErrNoteUnreadable
}
