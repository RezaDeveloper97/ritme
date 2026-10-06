// Package files stores the uploaded lab sheets (bloom B-N6-06) encrypted at rest.
//
// Layout: STORAGE_PATH/app/private/labs/<user_id>/<random>.bin — the private part of the storage volume (never
// app/public, which /storage/* serves), directories 0700, files 0600, created with O_EXCL. Each file is
//
//	"RLF1" ‖ key id (4 bytes) ‖ nonce (12) ‖ AES-256-GCM(ciphertext ‖ tag)
//
// with additional data "lab-file:v1:<user_id>:<relative path>", so a file copied into another account or under
// another name does not decrypt. The key comes from LAB_FILE_KEY (config.LabFiles; earlier keys in
// LAB_FILE_KEY_PREVIOUS still open their files); without one a public development key is used outside production
// and the box is disabled in production (callers answer 503).
//
// The package has no dependency on the rest of the API so account deletion (internal/profile, the admin user
// delete) can remove a user's directory without an import cycle.
package files

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Dir is the lab file root relative to STORAGE_PATH.
const Dir = "app/private/labs"

const magic = "RLF1"

// devKey keys the files when LAB_FILE_KEY is unset outside production.
var devKey = sha256.Sum256([]byte("ritme-lab-file-dev-key"))

// Errors.
var (
	// ErrDisabled: production without LAB_FILE_KEY, or no STORAGE_PATH.
	ErrDisabled = errors.New("labs/files: storage unavailable")
	// ErrUnreadable: a missing, foreign, tampered or rotated-out file.
	ErrUnreadable = errors.New("labs/files: file unreadable")
)

type key struct {
	id   [4]byte
	aead cipher.AEAD
}

// Box encrypts, stores, reads and removes lab files.
type Box struct {
	root     string // STORAGE_PATH
	keys     []key  // keys[0] seals
	rand     io.Reader
	disabled bool
}

// New builds the box on storagePath with the current 32-byte key (empty → the development key) and the previous
// keys. disabled = production without a key (or no storage path): every operation answers ErrDisabled.
func New(storagePath string, current []byte, previous [][]byte, disabled bool) (*Box, error) {
	if len(current) == 0 {
		current = devKey[:]
	}
	b := &Box{root: strings.TrimRight(storagePath, "/"), rand: rand.Reader, disabled: disabled || storagePath == ""}
	for _, k := range append([][]byte{current}, previous...) {
		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, fmt.Errorf("labs/files: key: %w", err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("labs/files: cipher: %w", err)
		}
		sum := sha256.Sum256(k)
		var id [4]byte
		copy(id[:], sum[:4])
		b.keys = append(b.keys, key{id: id, aead: aead})
	}
	return b, nil
}

// Disabled reports whether files are unavailable.
func (b *Box) Disabled() bool { return b == nil || b.disabled }

// WithRand replaces the nonce source (tests).
func (b *Box) WithRand(r io.Reader) *Box { b.rand = r; return b }

func userDir(userID uint64) string { return Dir + "/" + strconv.FormatUint(userID, 10) }

var reName = regexp.MustCompile(`^[a-f0-9]{40}\.bin$`)

// valid reports whether rel is a file name this box wrote for userID (no traversal, no other user's directory).
func valid(userID uint64, rel string) bool {
	dir, name, ok := strings.Cut(rel, "/"+strconv.FormatUint(userID, 10)+"/")
	return userID > 0 && ok && dir == Dir && reName.MatchString(name)
}

func ad(userID uint64, rel string) []byte {
	return []byte("lab-file:v1:" + strconv.FormatUint(userID, 10) + ":" + rel)
}

// Put encrypts plain into a new file of userID and returns its path relative to STORAGE_PATH.
func (b *Box) Put(userID uint64, plain []byte) (string, error) {
	if b.Disabled() {
		return "", ErrDisabled
	}
	if userID == 0 {
		return "", errors.New("labs/files: no user")
	}
	name := make([]byte, 20)
	if _, err := io.ReadFull(b.rand, name); err != nil {
		return "", fmt.Errorf("labs/files: name: %w", err)
	}
	rel := userDir(userID) + "/" + hex.EncodeToString(name) + ".bin"
	k := b.keys[0]
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := io.ReadFull(b.rand, nonce); err != nil {
		return "", fmt.Errorf("labs/files: nonce: %w", err)
	}
	out := make([]byte, 0, len(magic)+len(k.id)+len(nonce)+len(plain)+k.aead.Overhead())
	out = append(out, magic...)
	out = append(out, k.id[:]...)
	out = append(out, nonce...)
	out = k.aead.Seal(out, nonce, plain, ad(userID, rel))
	abs := filepath.Join(b.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return "", fmt.Errorf("labs/files: dir: %w", err)
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: path built from a random hex name
	if err != nil {
		return "", fmt.Errorf("labs/files: create: %w", err)
	}
	if _, err := f.Write(out); err != nil {
		_ = f.Close()
		_ = os.Remove(abs)
		return "", fmt.Errorf("labs/files: write: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(abs)
		return "", fmt.Errorf("labs/files: close: %w", err)
	}
	return rel, nil
}

// Get reads and decrypts a file of userID (ErrUnreadable for anything this box did not write for that user).
func (b *Box) Get(userID uint64, rel string) ([]byte, error) {
	if b.Disabled() {
		return nil, ErrDisabled
	}
	if !valid(userID, rel) {
		return nil, ErrUnreadable
	}
	raw, err := os.ReadFile(filepath.Join(b.root, filepath.FromSlash(rel))) //nolint:gosec // G304: validated path
	if err != nil {
		return nil, ErrUnreadable
	}
	head := len(magic) + 4
	if len(raw) < head || string(raw[:len(magic)]) != magic {
		return nil, ErrUnreadable
	}
	for _, k := range b.keys {
		if string(k.id[:]) != string(raw[len(magic):head]) {
			continue
		}
		n := k.aead.NonceSize()
		if len(raw) < head+n {
			return nil, ErrUnreadable
		}
		plain, err := k.aead.Open(nil, raw[head:head+n], raw[head+n:], ad(userID, rel))
		if err != nil {
			return nil, ErrUnreadable
		}
		return plain, nil
	}
	return nil, ErrUnreadable
}

// Remove deletes a file of userID; a missing file is fine, a path this box did not write is ignored.
func (b *Box) Remove(userID uint64, rel string) error {
	if b == nil || b.root == "" || !valid(userID, rel) {
		return nil
	}
	if err := os.Remove(filepath.Join(b.root, filepath.FromSlash(rel))); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("labs/files: remove: %w", err)
	}
	return nil
}

// RemoveUser deletes every lab file of userID (account deletion). Works without a key.
func RemoveUser(storagePath string, userID uint64) error {
	if storagePath == "" || userID == 0 {
		return nil
	}
	if err := os.RemoveAll(filepath.Join(strings.TrimRight(storagePath, "/"), filepath.FromSlash(userDir(userID)))); err != nil {
		return fmt.Errorf("labs/files: remove user: %w", err)
	}
	return nil
}

// Users lists the user ids that have a lab file directory (the orphan sweep).
func Users(storagePath string) ([]uint64, error) {
	if storagePath == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(strings.TrimRight(storagePath, "/"), filepath.FromSlash(Dir)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("labs/files: list: %w", err)
	}
	var out []uint64
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if id, err := strconv.ParseUint(e.Name(), 10, 64); err == nil && id > 0 && strconv.FormatUint(id, 10) == e.Name() {
			out = append(out, id)
		}
	}
	return out, nil
}
