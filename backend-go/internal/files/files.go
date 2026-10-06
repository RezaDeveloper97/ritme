// Package files is the encrypted file storage of every document kind (canvas-build CB-CORE-05, generalised from the
// lab-sheet storage of bloom B-N6-06; docs/canvas-build/files.md).
//
// Three layers, each usable on its own:
//
//   - Vault (vault.go): AES-256-GCM blobs on the private part of the storage volume,
//     STORAGE_PATH/<purpose dir>/<owner>/<random>.bin, directories 0700, files 0600, created with O_EXCL. The
//     additional data binds purpose, owner and path, so a blob copied to another account or name does not open.
//     internal/labs/files is a thin adapter over it (purpose Lab, the B-N6-06 on-disk format unchanged).
//   - Signer (signer.go): short-lived HMAC download URLs bound to the file id, its owner, purpose and path.
//   - Service (service.go): the `files` table (owner, purpose, visibility, mime, size, sha256), per-purpose type /
//     size / quota rules, photo re-encoding, and the owner-scoped / signed / public reads. Handlers (handlers.go)
//     expose it under /api/v1/files (Go only, deviations.md D-59).
//
// Health data: blobs are encrypted at rest and user-scoped; nothing here logs a file's content, name, hash or owner.
// A public purpose (place_photo, product_image) only ever holds re-encoded images; a private file is never served
// by the public route, whatever its id.
//
// The vault part has no dependency on the rest of the API besides the generated store, so account deletion
// (internal/profile, the admin user delete) can call RemoveUser without an import cycle.
package files

import (
	"errors"
	"slices"
)

// Content kinds a purpose accepts (sniffed from the bytes, never from the client's name or Content-Type).
const (
	KindImage = "image" // JPEG / PNG / WebP, re-encoded to WebP (EXIF / GPS dropped)
	KindPDF   = "pdf"   // stored as sent (encrypted)
)

// Visibility of a stored file.
const (
	Private = "private"
	Public  = "public"
)

// Purpose names (the `files.purpose` column).
const (
	PurposeLab            = "lab"
	PurposeRecordDocument = "record_document"
	PurposeClaimDocument  = "claim_document"
	PurposePlacePhoto     = "place_photo"
	PurposePlaceLicence   = "place_licence"
	PurposeProductImage   = "product_image"
)

// Purpose is one kind of stored file: where its blobs live, how they are sealed, what they may contain and how much
// one owner may keep.
type Purpose struct {
	Name string
	// Public files are served without auth by GET /files/public/{id}/{name}; only images are accepted for them.
	Public bool
	// UserUpload purposes are accepted by POST /api/v1/files; the others are written by Go code only (admin uploads).
	UserUpload bool
	// Kinds are the accepted content kinds (KindImage, KindPDF).
	Kinds []string
	// MaxBytes bounds one upload as received (before re-encoding).
	MaxBytes int
	// QuotaFiles / QuotaBytes bound what one owner keeps for this purpose (0 = no bound). Platform-owned files
	// (owner 0) share one quota per purpose.
	QuotaFiles int
	QuotaBytes int64
	// Blob layout: directory under STORAGE_PATH, file magic and the additional-data prefix.
	dir, magic, adPrefix string
	// blobOnly purposes keep their metadata in their own table (labs → lab_files) and are refused by Service.
	blobOnly bool
}

// Accepts reports whether kind is allowed.
func (p Purpose) Accepts(kind string) bool { return slices.Contains(p.Kinds, kind) }

// Visibility is the stored visibility of the purpose's files.
func (p Purpose) Visibility() string {
	if p.Public {
		return Public
	}
	return Private
}

// Dir is the purpose's blob root relative to STORAGE_PATH.
func (p Purpose) Dir() string { return p.dir }

const (
	mb        = 1 << 20
	genMagic  = "RFF1"
	genPrefix = "file:v1:"
	genRoot   = "app/private/files/"
)

func generic(name string) (dir, magic, ad string) {
	return genRoot + name, genMagic, genPrefix + name
}

func purpose(p Purpose) Purpose {
	p.dir, p.magic, p.adPrefix = generic(p.Name)
	return p
}

// Lab is the lab-sheet purpose of bloom B-N6-06 (internal/labs/files): its own directory, magic and additional data
// so files written before CB-CORE-05 still open; metadata in lab_files.
var Lab = Purpose{Name: PurposeLab, Kinds: []string{KindImage, KindPDF}, MaxBytes: 10 * mb,
	dir: "app/private/labs", magic: "RLF1", adPrefix: "lab-file:v1", blobOnly: true}

// Registry is every purpose by name. Limits are product decisions documented in docs/canvas-build/files.md.
// place_photo / place_licence are not user uploads yet: the directory tasks (CB-DIR) open them together with the
// place-ownership checks; until then only Go code writes them.
var Registry = map[string]Purpose{
	PurposeLab: Lab,
	PurposeRecordDocument: purpose(Purpose{Name: PurposeRecordDocument, UserUpload: true,
		Kinds: []string{KindImage, KindPDF}, MaxBytes: 10 * mb, QuotaFiles: 300, QuotaBytes: 500 * mb}),
	PurposeClaimDocument: purpose(Purpose{Name: PurposeClaimDocument, UserUpload: true,
		Kinds: []string{KindImage, KindPDF}, MaxBytes: 10 * mb, QuotaFiles: 200, QuotaBytes: 300 * mb}),
	PurposePlaceLicence: purpose(Purpose{Name: PurposePlaceLicence,
		Kinds: []string{KindImage, KindPDF}, MaxBytes: 10 * mb, QuotaFiles: 20, QuotaBytes: 100 * mb}),
	PurposePlacePhoto: purpose(Purpose{Name: PurposePlacePhoto, Public: true,
		Kinds: []string{KindImage}, MaxBytes: 5 * mb, QuotaFiles: 30, QuotaBytes: 60 * mb}),
	PurposeProductImage: purpose(Purpose{Name: PurposeProductImage, Public: true,
		Kinds: []string{KindImage}, MaxBytes: 5 * mb, QuotaFiles: 20000, QuotaBytes: 10 << 30}),
}

// Lookup returns the purpose named name.
func Lookup(name string) (Purpose, bool) {
	p, ok := Registry[name]
	return p, ok
}

// Errors.
var (
	// ErrDisabled: no usable key (fail closed) or no STORAGE_PATH.
	ErrDisabled = errors.New("files: storage unavailable")
	// ErrUnreadable: a missing, foreign, tampered or rotated-out blob.
	ErrUnreadable = errors.New("files: file unreadable")
	// ErrNotFound: no such file for this owner (or link).
	ErrNotFound = errors.New("files: file not found")
	// ErrPurpose: an unknown purpose, or one this caller may not write.
	ErrPurpose = errors.New("files: purpose not allowed")
	// ErrType: content the purpose does not accept (or unreadable image).
	ErrType = errors.New("files: content type not accepted")
	// ErrTooLarge: bigger than the purpose's MaxBytes.
	ErrTooLarge = errors.New("files: file too large")
	// ErrQuota: the owner's quota for the purpose is used up.
	ErrQuota = errors.New("files: quota exceeded")
	// ErrBusy: too many concurrent photo re-encodes.
	ErrBusy = errors.New("files: busy")
	// ErrLinkInvalid / ErrLinkExpired: a signed URL that does not verify / is past its expiry.
	ErrLinkInvalid = errors.New("files: invalid link")
	ErrLinkExpired = errors.New("files: link expired")
)
