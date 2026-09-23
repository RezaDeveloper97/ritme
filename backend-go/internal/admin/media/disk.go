// Package media is the admin upload pipeline: Laravel's "public" disk on the
// backend-storage volume (STORAGE_PATH/app/public, served by /storage/*), upload
// inspection (the file's own bytes decide its type — never the client's name or
// Content-Type) and App\Services\Media\ImageOptimizer (fit 1080×1080, WebP q82, random
// 40-character names).
//
// WebP encoder: github.com/gen2brain/webp — libwebp compiled to WebAssembly and
// transpiled to Go (wasm2go), so it needs no cgo. The API image is built with
// CGO_ENABLED=0 on alpine, which rules out chai2010/webp and govips (both cgo + a C
// library in the runtime image).
package media

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ErrBadPath is returned for a path that could escape the disk root.
var ErrBadPath = errors.New("media: invalid path")

// Disk is the Laravel "public" disk: STORAGE_PATH/app/public. Paths are relative
// ("banners/x.jpg"), exactly what the DB stores in image_path.
type Disk struct {
	root string
}

// NewDisk returns the public disk under storagePath ("" = uploads disabled: every
// write fails).
func NewDisk(storagePath string) *Disk {
	if storagePath == "" {
		return &Disk{}
	}
	return &Disk{root: filepath.Join(storagePath, "app", "public")}
}

// Root is the absolute directory of the disk ("" when disabled).
func (d *Disk) Root() string { return d.root }

// resolve maps a relative disk path to an absolute one. Only plain relative paths made
// of safe segments are accepted: no absolute paths, "..", ".", empty or dot-file
// segments, backslashes or NUL bytes.
func (d *Disk) resolve(rel string) (string, error) {
	if d.root == "" {
		return "", fmt.Errorf("media: public disk not configured: %w", ErrBadPath)
	}
	if rel == "" || strings.ContainsAny(rel, "\\\x00") || strings.HasPrefix(rel, "/") || path.Clean(rel) != rel {
		return "", ErrBadPath
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") {
			return "", ErrBadPath
		}
	}
	abs := filepath.Join(d.root, filepath.FromSlash(rel))
	if !strings.HasPrefix(abs, d.root+string(filepath.Separator)) {
		return "", ErrBadPath
	}
	return abs, nil
}

// Put writes data at rel (creating the directory), atomically: a temp file in the same
// directory renamed into place. Files are 0644 and directories 0755, like Laravel's
// public visibility.
func (d *Disk) Put(rel string, data []byte) error {
	abs, err := d.resolve(rel)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: public disk, served by /storage
		return fmt.Errorf("media: mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("media: create: %w", err)
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("media: write: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("media: chmod: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("media: close: %w", err)
	}
	if err := os.Rename(name, abs); err != nil {
		return fmt.Errorf("media: rename: %w", err)
	}
	ok = true
	return nil
}

// Exists reports whether rel is a regular file on the disk.
func (d *Disk) Exists(rel string) bool {
	abs, err := d.resolve(rel)
	if err != nil {
		return false
	}
	fi, err := os.Lstat(abs)
	return err == nil && fi.Mode().IsRegular()
}

// Delete removes rel if it exists (Storage::disk('public')->delete after exists()).
// An empty path, an invalid path or a missing file is a no-op; a symlink is removed
// itself, never its target.
func (d *Disk) Delete(rel string) error {
	if rel == "" {
		return nil
	}
	abs, err := d.resolve(rel)
	if err != nil {
		return nil //nolint:nilerr // a path we could never have written is not ours to delete
	}
	if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("media: delete: %w", err)
	}
	return nil
}

const alnum = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// RandomName is Str::random(n): n characters from [A-Za-z0-9], from crypto/rand.
func RandomName(n int) string {
	out := make([]byte, n)
	buf := make([]byte, n)
	for i := 0; i < n; {
		if _, err := rand.Read(buf); err != nil {
			panic(fmt.Sprintf("media: crypto/rand: %v", err)) // never fails on supported platforms
		}
		for _, b := range buf {
			if i == n {
				break
			}
			if int(b) < 256-256%len(alnum) { // reject to keep the distribution uniform
				out[i] = alnum[int(b)%len(alnum)]
				i++
			}
		}
	}
	return string(out)
}

// NewPath is "<dir>/<40 random chars>.<ext>" (UploadedFile::hashName / ImageOptimizer).
func NewPath(dir, ext string) string {
	return strings.Trim(dir, "/") + "/" + RandomName(40) + "." + ext
}
