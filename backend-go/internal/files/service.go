package files

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/files/store"
)

// File is one stored file (a `files` row).
type File struct {
	ID         uint64
	Owner      uint64 // 0 = the Ritme team (public purposes only)
	Purpose    string
	Visibility string
	MIME       string
	Size       int
	SHA256     string
	Path       string // relative to STORAGE_PATH; never shown to clients
	CreatedAt  time.Time
}

// Link is how a client fetches a file: a public URL, or a signed one that expires.
type Link struct {
	URL       string
	ExpiresAt *time.Time
}

// Options configure a Service.
type Options struct {
	DB    *sql.DB
	Vault *Vault
	// BaseURL is the public origin of this API (APP_URL); links are absolute.
	BaseURL string
	// Purposes overrides Registry (tests: small quotas).
	Purposes map[string]Purpose
}

// Service stores files of every non-lab purpose with their metadata.
type Service struct {
	db       *sql.DB
	q        *store.Queries
	vault    *Vault
	signer   *Signer
	base     string
	purposes map[string]Purpose
}

// NewService builds the service.
func NewService(o Options) *Service {
	ps := o.Purposes
	if ps == nil {
		ps = Registry
	}
	return &Service{db: o.DB, q: store.New(o.DB), vault: o.Vault, signer: NewSigner(o.Vault),
		base: strings.TrimRight(o.BaseURL, "/"), purposes: ps}
}

// Disabled reports whether the storage is unavailable (no key outside local / testing, or no STORAGE_PATH).
func (s *Service) Disabled() bool { return s.vault.Disabled() }

// Purpose returns a non-lab purpose by name.
func (s *Service) Purpose(name string) (Purpose, bool) {
	p, ok := s.purposes[name]
	return p, ok && !p.blobOnly
}

func fromRow(r store.File) File {
	f := File{ID: r.ID, Purpose: r.Purpose, Visibility: r.Visibility, MIME: r.Mime, Size: int(r.SizeBytes),
		SHA256: r.Sha256, Path: r.Path}
	if r.UserID.Valid && r.UserID.Int64 > 0 {
		f.Owner = uint64(r.UserID.Int64)
	}
	if r.CreatedAt.Valid {
		f.CreatedAt = r.CreatedAt.Time
	}
	return f
}

func ownerArg(owner uint64) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(owner), Valid: owner > 0} //nolint:gosec // G115: user ids fit int64
}

// Put stores data as a new file of owner (0 = the Ritme team, public purposes only) for purpose: content checked
// and photos re-encoded (prepare), the owner's quota checked under a row lock, the blob encrypted, the row written.
// Errors: ErrPurpose, ErrDisabled, ErrType, ErrTooLarge, ErrBusy, ErrQuota.
func (s *Service) Put(ctx context.Context, owner uint64, purpose string, data []byte, now time.Time) (File, error) {
	p, ok := s.Purpose(purpose)
	if !ok || !ownerOK(p, owner) {
		return File{}, ErrPurpose
	}
	if s.Disabled() {
		return File{}, ErrDisabled
	}
	stored, mime, err := prepare(p, data)
	if err != nil {
		return File{}, err
	}
	defer clear(stored)
	sum := sha256.Sum256(stored)
	f := File{Owner: owner, Purpose: p.Name, Visibility: p.Visibility(), MIME: mime, Size: len(stored),
		SHA256: hex.EncodeToString(sum[:]), CreatedAt: now}
	// The platform's uploads (owner 0) have no row to lock; a deadlock between two of them is retried.
	for attempt := 1; ; attempt++ {
		out, err := s.store(ctx, p, f, stored)
		var me *mysql.MySQLError
		if attempt < 3 && errors.As(err, &me) && me.Number == errDeadlock {
			continue
		}
		return out, err
	}
}

// errDeadlock is MySQL / MariaDB ER_LOCK_DEADLOCK.
const errDeadlock = 1213

// store runs the upload transaction: lock (the user row, then the owner's usage of the purpose), check the quota,
// write the blob, insert the row, commit — or remove the blob again.
func (s *Service) store(ctx context.Context, p Purpose, f File, stored []byte) (File, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return File{}, fmt.Errorf("files: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := s.q.WithTx(tx)
	var count, bytes int64
	if f.Owner > 0 {
		if _, err := q.LockOwner(ctx, f.Owner); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return File{}, ErrPurpose
			}
			return File{}, fmt.Errorf("files: lock: %w", err)
		}
		u, err := q.LockOwnerUsage(ctx, store.LockOwnerUsageParams{UserID: ownerArg(f.Owner), Purpose: p.Name})
		if err != nil {
			return File{}, fmt.Errorf("files: usage: %w", err)
		}
		count, bytes = u.Files, u.Bytes
	} else {
		u, err := q.LockPlatformUsage(ctx, p.Name)
		if err != nil {
			return File{}, fmt.Errorf("files: usage: %w", err)
		}
		count, bytes = u.Files, u.Bytes
	}
	if (p.QuotaFiles > 0 && count+1 > int64(p.QuotaFiles)) || (p.QuotaBytes > 0 && bytes+int64(len(stored)) > p.QuotaBytes) {
		return File{}, ErrQuota
	}
	rel, err := s.vault.Put(p, f.Owner, stored)
	if err != nil {
		return File{}, err
	}
	f.Path = rel
	id, err := q.CreateFile(ctx, store.CreateFileParams{UserID: ownerArg(f.Owner), Purpose: f.Purpose,
		Visibility: f.Visibility, Mime: f.MIME, SizeBytes: uint32(f.Size), Sha256: f.SHA256, Path: rel, //nolint:gosec // G115: ≤ MaxBytes
		Now: sql.NullTime{Time: f.CreatedAt, Valid: true}})
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = s.vault.Remove(p, f.Owner, rel)
		return File{}, fmt.Errorf("files: create: %w", err)
	}
	f.ID = uint64(id) //nolint:gosec // G115: auto-increment id
	return f, nil
}

// Get returns owner's file id (ErrNotFound for another owner's, a lab or an unknown purpose).
func (s *Service) Get(ctx context.Context, owner, id uint64) (File, error) {
	var (
		row store.File
		err error
	)
	if owner > 0 {
		row, err = s.q.GetOwnedFile(ctx, store.GetOwnedFileParams{ID: id, UserID: ownerArg(owner)})
	} else {
		row, err = s.q.GetPlatformFile(ctx, id)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, fmt.Errorf("files: get: %w", err)
	}
	if _, ok := s.Purpose(row.Purpose); !ok {
		return File{}, ErrNotFound
	}
	return fromRow(row), nil
}

// Read decrypts f's bytes (ErrDisabled / ErrUnreadable).
func (s *Service) Read(f File) ([]byte, error) {
	p, ok := s.Purpose(f.Purpose)
	if !ok {
		return nil, ErrNotFound
	}
	return s.vault.Get(p, f.Owner, f.Path)
}

// Open is Get + Read for owner.
func (s *Service) Open(ctx context.Context, owner, id uint64) (File, []byte, error) {
	f, err := s.Get(ctx, owner, id)
	if err != nil {
		return File{}, nil, err
	}
	data, err := s.Read(f)
	return f, data, err
}

// Delete removes owner's file id (row first, then the blob; a failed blob removal leaves an unreferenced encrypted
// blob that RemoveUser still deletes with the account).
func (s *Service) Delete(ctx context.Context, owner, id uint64) error {
	f, err := s.Get(ctx, owner, id)
	if err != nil {
		return err
	}
	var n int64
	if owner > 0 {
		n, err = s.q.DeleteOwnedFile(ctx, store.DeleteOwnedFileParams{ID: id, UserID: ownerArg(owner)})
	} else {
		n, err = s.q.DeletePlatformFile(ctx, id)
	}
	if err != nil {
		return fmt.Errorf("files: delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	p, _ := s.Purpose(f.Purpose)
	return s.vault.Remove(p, f.Owner, f.Path)
}

// Link returns how a client fetches f at now: the public URL for a public file, else a signed URL valid for ttl
// (DefaultLinkTTL when ≤ 0, at most MaxLinkTTL).
func (s *Service) Link(f File, now time.Time, ttl time.Duration) (Link, error) {
	if f.Visibility == Public {
		return Link{URL: s.base + "/api/v1/files/public/" + strconv.FormatUint(f.ID, 10) + "/" + Name(f.Path)}, nil
	}
	if s.signer == nil {
		return Link{}, ErrDisabled
	}
	if ttl <= 0 {
		ttl = DefaultLinkTTL
	}
	exp := now.Add(min(ttl, MaxLinkTTL)).Truncate(time.Second)
	v := url.Values{"expires": {strconv.FormatInt(exp.Unix(), 10)}, "signature": {s.signer.Sign(f, exp)}}
	return Link{URL: s.base + "/api/v1/files/" + strconv.FormatUint(f.ID, 10) + "/download?" + v.Encode(), ExpiresAt: &exp}, nil
}

// OpenSigned serves a signed link: the row by id, the signature recomputed over its owner / purpose / path.
// ErrLinkInvalid also for an unknown id (no existence oracle); ErrLinkExpired once past expiry.
func (s *Service) OpenSigned(ctx context.Context, id uint64, expires, signature string, now time.Time) (File, []byte, error) {
	if s.signer == nil {
		return File{}, nil, ErrDisabled
	}
	row, err := s.q.GetFile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, nil, ErrLinkInvalid
	}
	if err != nil {
		return File{}, nil, fmt.Errorf("files: get: %w", err)
	}
	f := fromRow(row)
	if _, ok := s.Purpose(f.Purpose); !ok {
		return File{}, nil, ErrLinkInvalid
	}
	if err := s.signer.Verify(f, expires, signature, now); err != nil {
		return File{}, nil, err
	}
	data, err := s.Read(f)
	return f, data, err
}

// OpenPublic serves a public file: its purpose must be public in the registry AND the row public AND name its
// random blob name. Anything else — a private file under any id — is ErrNotFound.
func (s *Service) OpenPublic(ctx context.Context, id uint64, name string) (File, []byte, error) {
	row, err := s.q.GetFile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, nil, ErrNotFound
	}
	if err != nil {
		return File{}, nil, fmt.Errorf("files: get: %w", err)
	}
	f := fromRow(row)
	p, ok := s.Purpose(f.Purpose)
	if !ok || !p.Public || f.Visibility != Public || !strings.HasPrefix(f.MIME, "image/") ||
		subtle.ConstantTimeCompare([]byte(Name(f.Path)), []byte(name)) != 1 {
		return File{}, nil, ErrNotFound
	}
	data, err := s.vault.Get(p, f.Owner, f.Path)
	return f, data, err
}
