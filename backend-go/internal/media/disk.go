package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Disk is the media root: <root>/<instructor id>/<40 hex>.part while uploading, .bin once ready. Directories 0700,
// files 0600, created with O_EXCL; every relative path is validated, so a stored path can never leave the root.
type Disk struct {
	root string
	rand io.Reader
}

// NewDisk returns the media root at root ("" = disabled).
func NewDisk(root string) *Disk { return &Disk{root: strings.TrimRight(root, "/"), rand: rand.Reader} }

// Enabled reports whether a root is configured.
func (d *Disk) Enabled() bool { return d != nil && d.root != "" }

var reRel = regexp.MustCompile(`^[1-9][0-9]{0,19}/[a-f0-9]{40}\.(part|bin)$`)

// validRel reports whether rel is a path this disk writes for instructor.
func validRel(instructor uint64, rel string) bool {
	return reRel.MatchString(rel) && strings.HasPrefix(rel, strconv.FormatUint(instructor, 10)+"/")
}

func (d *Disk) abs(rel string) string { return filepath.Join(d.root, filepath.FromSlash(rel)) }

// Abs is the absolute path of a validated rel (the scanner's input).
func (d *Disk) Abs(instructor uint64, rel string) (string, error) {
	if !d.Enabled() {
		return "", ErrDisabled
	}
	if !validRel(instructor, rel) {
		return "", ErrNotFound
	}
	return d.abs(rel), nil
}

// Create makes a new empty upload file for instructor and returns its relative path.
func (d *Disk) Create(instructor uint64) (string, error) {
	if !d.Enabled() {
		return "", ErrDisabled
	}
	name := make([]byte, 20)
	if _, err := io.ReadFull(d.rand, name); err != nil {
		return "", fmt.Errorf("media: name: %w", err)
	}
	rel := strconv.FormatUint(instructor, 10) + "/" + hex.EncodeToString(name) + ".part"
	abs := d.abs(rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return "", fmt.Errorf("media: dir: %w", pathless(err))
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // G304: random hex name under the root
	if err != nil {
		return "", fmt.Errorf("media: create: %w", pathless(err))
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("media: create: %w", pathless(err))
	}
	return rel, nil
}

// WriteAt writes one chunk at off and cuts anything after it (bytes of a chunk whose offset update never committed),
// then syncs, so a stored offset always has its bytes on disk.
func (d *Disk) WriteAt(instructor uint64, rel string, off int64, data []byte) error {
	abs, err := d.Abs(instructor, rel)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(abs, os.O_WRONLY, 0) //nolint:gosec // G304: validated path
	if err != nil {
		return fmt.Errorf("media: open: %w", pathless(err))
	}
	if _, err := f.WriteAt(data, off); err != nil {
		_ = f.Close()
		return fmt.Errorf("media: write: %w", pathless(err))
	}
	if err := f.Truncate(off + int64(len(data))); err != nil {
		_ = f.Close()
		return fmt.Errorf("media: truncate: %w", pathless(err))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("media: sync: %w", pathless(err))
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("media: close: %w", pathless(err))
	}
	return nil
}

// Finalize renames a complete .part to .bin and returns the new relative path.
func (d *Disk) Finalize(instructor uint64, rel string, size int64) (string, error) {
	abs, err := d.Abs(instructor, rel)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(rel, ".part") {
		return "", ErrNotUploading
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("media: stat: %w", pathless(err))
	}
	if st.Size() != size {
		return "", fmt.Errorf("media: finalize: %d bytes on disk, %d declared", st.Size(), size)
	}
	next := strings.TrimSuffix(rel, ".part") + ".bin"
	if err := os.Rename(abs, d.abs(next)); err != nil {
		return "", fmt.Errorf("media: rename: %w", pathless(err))
	}
	return next, nil
}

// Open opens a ready file for reading.
func (d *Disk) Open(instructor uint64, rel string) (*os.File, error) {
	abs, err := d.Abs(instructor, rel)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(abs) //nolint:gosec // G304: validated path
	if err != nil {
		return nil, ErrNotFound
	}
	return f, nil
}

// Remove deletes a file (missing is fine; an invalid path is ignored).
func (d *Disk) Remove(instructor uint64, rel string) error {
	if !d.Enabled() || !validRel(instructor, rel) {
		return nil
	}
	if err := os.Remove(d.abs(rel)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("media: remove: %w", pathless(err))
	}
	return nil
}

// instructorDirs lists the instructor ids that have a directory.
func (d *Disk) instructorDirs() ([]uint64, error) {
	if !d.Enabled() {
		return nil, nil
	}
	entries, err := os.ReadDir(d.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("media: list: %w", pathless(err))
	}
	var out []uint64
	for _, e := range entries {
		if id, err := strconv.ParseUint(e.Name(), 10, 64); e.IsDir() && err == nil && id > 0 && strconv.FormatUint(id, 10) == e.Name() {
			out = append(out, id)
		}
	}
	return out, nil
}

// removeInstructor deletes an instructor's whole directory.
func (d *Disk) removeInstructor(id uint64) error {
	if !d.Enabled() || id == 0 {
		return nil
	}
	if err := os.RemoveAll(d.abs(strconv.FormatUint(id, 10))); err != nil {
		return fmt.Errorf("media: remove dir: %w", pathless(err))
	}
	return nil
}

// strayFiles lists the files of an instructor's directory not in keep and older than minAge (a file is created
// just before its row is inserted).
func (d *Disk) strayFiles(_ context.Context, id uint64, keep map[string]bool, minAge time.Duration, now time.Time) ([]string, error) {
	dir := strconv.FormatUint(id, 10)
	entries, err := os.ReadDir(d.abs(dir))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("media: list: %w", pathless(err))
	}
	var out []string
	for _, e := range entries {
		rel := dir + "/" + e.Name()
		if e.IsDir() || keep[rel] || !validRel(id, rel) {
			continue
		}
		if info, err := e.Info(); err == nil && now.Sub(info.ModTime()) >= minAge {
			out = append(out, rel)
		}
	}
	return out, nil
}

// pathless drops the filesystem path from an *os.PathError, so storage paths never reach an error log.
func pathless(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return &os.PathError{Op: pe.Op, Path: "-", Err: pe.Err}
	}
	return err
}
