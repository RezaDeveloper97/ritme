// Package files stores the uploaded lab sheets (bloom B-N6-06) encrypted at rest — a thin adapter over the generic
// internal/files vault (CB-CORE-05) with the lab purpose, so the on-disk format and every behaviour below are
// unchanged.
//
// Layout: STORAGE_PATH/app/private/labs/<user_id>/<random>.bin — the private part of the storage volume (never
// app/public, which /storage/* serves), directories 0700, files 0600, created with O_EXCL. Each file is
//
//	"RLF1" ‖ key id (4 bytes) ‖ nonce (12) ‖ AES-256-GCM(ciphertext ‖ tag)
//
// with additional data "lab-file:v1:<user_id>:<relative path>", so a file copied into another account or under
// another name does not decrypt. The key comes from LAB_FILE_KEY (config.LabFiles; earlier keys in
// LAB_FILE_KEY_PREVIOUS still open their files); without one a public development key is used where the config
// allows it and the box is disabled elsewhere (callers answer 503).
package files

import (
	"io"

	gfiles "github.com/ritme/backend-go/internal/files"
)

// Dir is the lab file root relative to STORAGE_PATH.
var Dir = gfiles.Lab.Dir()

const magic = "RLF1"

// devKey keys the files when LAB_FILE_KEY is unset where the config allows the development key.
var devKey = gfiles.DevKey("ritme-lab-file-dev-key")

// Errors (the generic vault's, so errors.Is works across both packages).
var (
	// ErrDisabled: no usable LAB_FILE_KEY, or no STORAGE_PATH.
	ErrDisabled = gfiles.ErrDisabled
	// ErrUnreadable: a missing, foreign, tampered or rotated-out file.
	ErrUnreadable = gfiles.ErrUnreadable
)

// Box encrypts, stores, reads and removes lab files.
type Box struct{ v *gfiles.Vault }

// New builds the box on storagePath with the current 32-byte key (empty → the development key) and the previous
// keys. disabled = no usable key (or no storage path): every operation answers ErrDisabled.
func New(storagePath string, current []byte, previous [][]byte, disabled bool) (*Box, error) {
	if len(current) == 0 {
		current = devKey
	}
	v, err := gfiles.NewVault(storagePath, current, previous, disabled)
	if err != nil {
		return nil, err
	}
	return &Box{v: v}, nil
}

// Disabled reports whether files are unavailable.
func (b *Box) Disabled() bool { return b == nil || b.v.Disabled() }

// WithRand replaces the nonce source (tests).
func (b *Box) WithRand(r io.Reader) *Box { b.v.WithRand(r); return b }

// Put encrypts plain into a new file of userID and returns its path relative to STORAGE_PATH.
func (b *Box) Put(userID uint64, plain []byte) (string, error) {
	return b.v.Put(gfiles.Lab, userID, plain)
}

// Get reads and decrypts a file of userID (ErrUnreadable for anything this box did not write for that user).
func (b *Box) Get(userID uint64, rel string) ([]byte, error) { return b.v.Get(gfiles.Lab, userID, rel) }

// Remove deletes a file of userID; a missing file is fine, a path this box did not write is ignored.
func (b *Box) Remove(userID uint64, rel string) error {
	if b == nil {
		return nil
	}
	return b.v.Remove(gfiles.Lab, userID, rel)
}

// RemoveUser deletes every lab file of userID. Works without a key.
func RemoveUser(storagePath string, userID uint64) error {
	return gfiles.RemoveOwner(storagePath, gfiles.Lab, userID)
}

// Users lists the user ids that have a lab file directory (the orphan sweep).
func Users(storagePath string) ([]uint64, error) { return gfiles.Owners(storagePath, gfiles.Lab) }
