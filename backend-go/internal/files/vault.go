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

// Blob format (every purpose; Lab keeps magic "RLF1" and additional data "lab-file:v1:…" from B-N6-06):
//
//	magic (4) ‖ key id (4 bytes = sha256(key)[:4]) ‖ nonce (12) ‖ AES-256-GCM(ciphertext ‖ tag)
//
// additional data "<adPrefix>:<owner>:<relative path>".

// KeyLen is the AES-256 key length.
const KeyLen = 32

// DevKey is the public development key for label (callers use it only where their config policy allows: never in
// production, see config.Files.Resolve).
func DevKey(label string) []byte {
	k := sha256.Sum256([]byte(label))
	return k[:]
}

type key struct {
	id   [4]byte
	aead cipher.AEAD
}

// Vault encrypts, stores, reads and removes blobs of any purpose.
type Vault struct {
	root     string // STORAGE_PATH
	keys     []key  // keys[0] seals
	current  []byte // keys[0] raw (the URL signing key is derived from it)
	rand     io.Reader
	disabled bool
}

// NewVault builds a vault on storagePath with the current 32-byte key and the previous keys (they only open older
// blobs). disabled (or an empty storagePath / current key) makes every Put / Get answer ErrDisabled; Remove still
// works so deletions never depend on a key.
func NewVault(storagePath string, current []byte, previous [][]byte, disabled bool) (*Vault, error) {
	v := &Vault{root: strings.TrimRight(storagePath, "/"), rand: rand.Reader,
		disabled: disabled || storagePath == "" || len(current) == 0}
	if len(current) == 0 {
		return v, nil
	}
	seen := map[[4]byte]bool{}
	for _, k := range append([][]byte{current}, previous...) {
		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, fmt.Errorf("files: key: %w", pathless(err))
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("files: cipher: %w", pathless(err))
		}
		sum := sha256.Sum256(k)
		var id [4]byte
		copy(id[:], sum[:4])
		if seen[id] {
			continue
		}
		seen[id] = true
		v.keys = append(v.keys, key{id: id, aead: aead})
	}
	v.current = append([]byte(nil), current...)
	return v, nil
}

// Disabled reports whether blobs are unavailable.
func (v *Vault) Disabled() bool { return v == nil || v.disabled }

// WithRand replaces the name / nonce source (tests).
func (v *Vault) WithRand(r io.Reader) *Vault { v.rand = r; return v }

func ownerDir(p Purpose, owner uint64) string { return p.dir + "/" + strconv.FormatUint(owner, 10) }

var reName = regexp.MustCompile(`^[a-f0-9]{40}\.bin$`)

// ownerOK: a real user, or the platform (0) for a public purpose.
func ownerOK(p Purpose, owner uint64) bool { return p.dir != "" && (owner > 0 || p.Public) }

// valid reports whether rel is a blob name this vault writes for (p, owner): no traversal, no other owner's or
// purpose's directory.
func valid(p Purpose, owner uint64, rel string) bool {
	prefix := ownerDir(p, owner) + "/"
	return ownerOK(p, owner) && strings.HasPrefix(rel, prefix) && reName.MatchString(rel[len(prefix):])
}

// Name is the random file name of rel (the public URL's capability part).
func Name(rel string) string {
	base := rel[strings.LastIndexByte(rel, '/')+1:]
	return strings.TrimSuffix(base, ".bin")
}

func ad(p Purpose, owner uint64, rel string) []byte {
	return []byte(p.adPrefix + ":" + strconv.FormatUint(owner, 10) + ":" + rel)
}

// Put encrypts plain into a new blob of (p, owner) and returns its path relative to STORAGE_PATH.
func (v *Vault) Put(p Purpose, owner uint64, plain []byte) (string, error) {
	if v.Disabled() {
		return "", ErrDisabled
	}
	if !ownerOK(p, owner) {
		return "", errors.New("files: no owner")
	}
	name := make([]byte, 20)
	if _, err := io.ReadFull(v.rand, name); err != nil {
		return "", fmt.Errorf("files: name: %w", pathless(err))
	}
	rel := ownerDir(p, owner) + "/" + hex.EncodeToString(name) + ".bin"
	k := v.keys[0]
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := io.ReadFull(v.rand, nonce); err != nil {
		return "", fmt.Errorf("files: nonce: %w", pathless(err))
	}
	out := make([]byte, 0, len(p.magic)+len(k.id)+len(nonce)+len(plain)+k.aead.Overhead())
	out = append(out, p.magic...)
	out = append(out, k.id[:]...)
	out = append(out, nonce...)
	out = k.aead.Seal(out, nonce, plain, ad(p, owner, rel))
	abs := filepath.Join(v.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return "", fmt.Errorf("files: dir: %w", pathless(err))
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: path built from a random hex name
	if err != nil {
		return "", fmt.Errorf("files: create: %w", pathless(err))
	}
	if _, err := f.Write(out); err != nil {
		_ = f.Close()
		_ = os.Remove(abs)
		return "", fmt.Errorf("files: write: %w", pathless(err))
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(abs)
		return "", fmt.Errorf("files: close: %w", pathless(err))
	}
	return rel, nil
}

// Get reads and decrypts a blob of (p, owner) (ErrUnreadable for anything this vault did not write for them).
func (v *Vault) Get(p Purpose, owner uint64, rel string) ([]byte, error) {
	if v.Disabled() {
		return nil, ErrDisabled
	}
	if !valid(p, owner, rel) {
		return nil, ErrUnreadable
	}
	raw, err := os.ReadFile(filepath.Join(v.root, filepath.FromSlash(rel))) //nolint:gosec // G304: validated path
	if err != nil {
		return nil, ErrUnreadable
	}
	head := len(p.magic) + 4
	if len(raw) < head || string(raw[:len(p.magic)]) != p.magic {
		return nil, ErrUnreadable
	}
	for _, k := range v.keys {
		if string(k.id[:]) != string(raw[len(p.magic):head]) {
			continue
		}
		n := k.aead.NonceSize()
		if len(raw) < head+n {
			return nil, ErrUnreadable
		}
		plain, err := k.aead.Open(nil, raw[head:head+n], raw[head+n:], ad(p, owner, rel))
		if err != nil {
			return nil, ErrUnreadable
		}
		return plain, nil
	}
	return nil, ErrUnreadable
}

// Remove deletes a blob of (p, owner); a missing blob is fine, a path this vault does not write is ignored.
func (v *Vault) Remove(p Purpose, owner uint64, rel string) error {
	if v == nil || v.root == "" || !valid(p, owner, rel) {
		return nil
	}
	if err := os.Remove(filepath.Join(v.root, filepath.FromSlash(rel))); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("files: remove: %w", pathless(err))
	}
	return nil
}

// RemoveOwner deletes every blob of owner for purpose p. Works without a key. Owner 0 (platform) is never removed.
func RemoveOwner(storagePath string, p Purpose, owner uint64) error {
	if storagePath == "" || owner == 0 || p.dir == "" {
		return nil
	}
	if err := os.RemoveAll(filepath.Join(strings.TrimRight(storagePath, "/"), filepath.FromSlash(ownerDir(p, owner)))); err != nil {
		return fmt.Errorf("files: remove owner: %w", pathless(err))
	}
	return nil
}

// RemoveUser deletes every blob of userID, all purposes (account deletion). Works without a key.
func RemoveUser(storagePath string, userID uint64) error {
	var errs []error
	for _, p := range Registry {
		errs = append(errs, RemoveOwner(storagePath, p, userID))
	}
	return errors.Join(errs...)
}

// pathless drops the filesystem path from an *os.PathError (op + cause only), so storage paths — which carry the
// owner id — never reach an error log. errors.Is(err, os.ErrNotExist) and friends still work.
func pathless(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return &os.PathError{Op: pe.Op, Path: "-", Err: pe.Err}
	}
	return err
}

// Owners lists the owner ids that have a blob directory for p (orphan sweeps).
func Owners(storagePath string, p Purpose) ([]uint64, error) {
	if storagePath == "" || p.dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(filepath.Join(strings.TrimRight(storagePath, "/"), filepath.FromSlash(p.dir)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("files: list: %w", pathless(err))
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
