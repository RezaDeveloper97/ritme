package media

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/media/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Options wires the service.
type Options struct {
	DB      *sql.DB
	Disk    *Disk
	Signer  *Signer // nil = playback disabled (503)
	Scanner Scanner // nil = NoopScanner
	// BaseURL is the API origin the playback URLs start with (APP_URL).
	BaseURL string
	Logger  *slog.Logger
}

// Service is the media data access and rules.
type Service struct {
	q       *store.Queries
	disk    *Disk
	signer  *Signer
	scanner Scanner
	baseURL string
	logger  *slog.Logger
	locks   sync.Map // media id → *sync.Mutex (one chunk at a time per upload)
}

// NewService wires the service.
func NewService(o Options) *Service {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.Scanner == nil {
		o.Scanner = NoopScanner{}
	}
	if o.Disk == nil {
		o.Disk = NewDisk("")
	}
	return &Service{q: store.New(o.DB), disk: o.Disk, signer: o.Signer, scanner: o.Scanner,
		baseURL: strings.TrimRight(o.BaseURL, "/"), logger: o.Logger}
}

// Queries exposes the store (tests).
func (s *Service) Queries() *store.Queries { return s.q }

func nt(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

func nid(id uint64) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0} //nolint:gosec // G115: auto-increment ids
}

func uid(v sql.NullInt64) uint64 {
	if !v.Valid || v.Int64 <= 0 {
		return 0
	}
	return uint64(v.Int64)
}

// CreateInput is a validated POST /media.
type CreateInput struct {
	LessonID uint64
	Size     int64
	Mime     string // declared, optional (checked against the lesson kind; the bytes decide)
}

// Create opens an upload of in.Size bytes for one of instructor's lessons. An unfinished earlier upload of the same
// lesson is cancelled; the lesson's current ready media stays playable until the new one is ready.
func (s *Service) Create(ctx context.Context, instructor uint64, in CreateInput, now time.Time) (store.LearningMedium, error) {
	if !s.disk.Enabled() {
		return store.LearningMedium{}, ErrDisabled
	}
	l, err := s.q.GetOwnedLesson(ctx, store.GetOwnedLessonParams{ID: in.LessonID, InstructorID: instructor})
	if errors.Is(err, sql.ErrNoRows) {
		return store.LearningMedium{}, ErrLessonNotFound
	}
	if err != nil {
		return store.LearningMedium{}, fmt.Errorf("media: lesson: %w", err)
	}
	limit, ok := MaxBytes[l.Kind]
	if !ok {
		return store.LearningMedium{}, ErrType
	}
	if in.Size > limit {
		return store.LearningMedium{}, ErrTooLarge
	}
	if in.Mime != "" && !Accepts(l.Kind, in.Mime) {
		return store.LearningMedium{}, ErrMimeMismatch
	}
	// Earlier unfinished uploads of this lesson make way (before the limits are counted).
	prior, err := s.q.ListLessonMedia(ctx, nid(l.ID))
	if err != nil {
		return store.LearningMedium{}, fmt.Errorf("media: lesson media: %w", err)
	}
	for _, m := range prior {
		if m.Status == StatusUploading || m.Status == StatusFailed {
			if err := s.remove(ctx, m); err != nil {
				return store.LearningMedium{}, err
			}
		}
	}
	active, err := s.q.CountActiveUploads(ctx, instructor)
	if err != nil {
		return store.LearningMedium{}, fmt.Errorf("media: count: %w", err)
	}
	if active >= MaxActiveUploads {
		return store.LearningMedium{}, ErrTooManyUploads
	}
	used, err := s.q.SumInstructorBytes(ctx, instructor)
	if err != nil {
		return store.LearningMedium{}, fmt.Errorf("media: quota: %w", err)
	}
	if used+in.Size > QuotaBytes {
		return store.LearningMedium{}, ErrQuota
	}
	rel, err := s.disk.Create(instructor)
	if err != nil {
		return store.LearningMedium{}, err
	}
	id, err := s.q.InsertMedia(ctx, store.InsertMediaParams{
		InstructorID: instructor, LessonID: nid(l.ID), Kind: l.Kind, SizeBytes: uint64(in.Size), //nolint:gosec // G115: validated ≥ 1
		Path: rel, UploadExpiresAt: nt(now.Add(UploadTTL)), Now: nt(now),
	})
	if err != nil {
		_ = s.disk.Remove(instructor, rel)
		return store.LearningMedium{}, fmt.Errorf("media: insert: %w", err)
	}
	if err := s.q.SetLessonMediaStatus(ctx, store.SetLessonMediaStatusParams{MediaStatus: StatusUploading, Now: nt(now), ID: l.ID}); err != nil {
		return store.LearningMedium{}, fmt.Errorf("media: lesson status: %w", err)
	}
	return s.get(ctx, uint64(id)) //nolint:gosec // G115: auto-increment id
}

func (s *Service) get(ctx context.Context, id uint64) (store.LearningMedium, error) {
	m, err := s.q.GetMedia(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, fmt.Errorf("media: read: %w", err)
	}
	return m, nil
}

// Owned reads one of instructor's media (ErrNotFound for a foreign or missing id).
func (s *Service) Owned(ctx context.Context, instructor, id uint64) (store.LearningMedium, error) {
	m, err := s.q.GetOwnedMedia(ctx, store.GetOwnedMediaParams{ID: id, InstructorID: instructor})
	if errors.Is(err, sql.ErrNoRows) {
		return m, ErrNotFound
	}
	if err != nil {
		return m, fmt.Errorf("media: read: %w", err)
	}
	return m, nil
}

func (s *Service) lock(id uint64) (func(), bool) {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	mu, _ := v.(*sync.Mutex)
	if !mu.TryLock() {
		return nil, false
	}
	return mu.Unlock, true
}

// Expired reports whether an unfinished upload is past its resume window.
func Expired(m store.LearningMedium, now time.Time) bool {
	return m.Status == StatusUploading && m.UploadExpiresAt.Valid && !m.UploadExpiresAt.Time.After(now)
}

// WriteChunk appends data at offset to one of instructor's uploads. The last chunk finalizes it.
func (s *Service) WriteChunk(ctx context.Context, instructor, id uint64, offset uint64, data []byte, now time.Time) (store.LearningMedium, error) {
	unlock, ok := s.lock(id)
	if !ok {
		return store.LearningMedium{}, ErrBusy
	}
	defer unlock()
	m, err := s.Owned(ctx, instructor, id)
	if err != nil {
		return m, err
	}
	switch {
	case m.Status != StatusUploading:
		return m, ErrNotUploading
	case Expired(m, now):
		return m, ErrUploadExpired
	case offset != m.OffsetBytes:
		return m, &OffsetMismatchError{Offset: m.OffsetBytes}
	case int64(len(data)) > MaxChunkBytes:
		return m, ErrChunkTooLarge
	case offset+uint64(len(data)) > m.SizeBytes:
		return m, ErrBeyondSize
	case len(data) == 0:
		return m, nil
	}
	var mime sql.NullString
	if offset == 0 {
		t := Sniff(m.Kind, data)
		if t == "" {
			return m, ErrType
		}
		mime = sql.NullString{String: t, Valid: true}
	}
	if err := s.disk.WriteAt(instructor, m.Path, int64(offset), data); err != nil { //nolint:gosec // G115: ≤ 2 GiB
		return m, err
	}
	next := offset + uint64(len(data))
	n, err := s.q.AdvanceOffset(ctx, store.AdvanceOffsetParams{NewOffset: next, Mime: mime, Now: nt(now), ID: m.ID, OldOffset: offset})
	if err != nil {
		return m, fmt.Errorf("media: offset: %w", err)
	}
	if n == 0 {
		cur, err := s.get(ctx, m.ID)
		if err != nil {
			return cur, err
		}
		return cur, &OffsetMismatchError{Offset: cur.OffsetBytes}
	}
	if next == m.SizeBytes {
		return s.finalize(ctx, m.ID, now)
	}
	return s.get(ctx, m.ID)
}

// finalize runs a complete upload through the scanner and makes it the lesson's media.
func (s *Service) finalize(ctx context.Context, id uint64, now time.Time) (store.LearningMedium, error) {
	m, err := s.get(ctx, id)
	if err != nil {
		return m, err
	}
	if n, err := s.q.SetMediaStatus(ctx, store.SetMediaStatusParams{Status: StatusProcessing, Now: nt(now), ID: id, FromStatus: StatusUploading}); err != nil || n == 0 {
		if err == nil {
			err = ErrNotUploading
		}
		return m, err
	}
	lesson := uid(m.LessonID)
	if lesson != 0 {
		if err := s.q.SetLessonMediaStatus(ctx, store.SetLessonMediaStatusParams{MediaStatus: StatusProcessing, Now: nt(now), ID: lesson}); err != nil {
			return m, fmt.Errorf("media: lesson status: %w", err)
		}
	}
	mime := m.Mime.String
	abs, err := s.disk.Abs(m.InstructorID, m.Path)
	if err == nil {
		err = s.scanner.Scan(ctx, abs, m.Kind, mime)
	}
	if err != nil {
		if !errors.Is(err, ErrInfected) {
			s.logger.ErrorContext(ctx, "media: scan failed", slog.Uint64("media_id", id), slog.String("error", pathless(err).Error()))
		}
		return m, s.fail(ctx, m, now)
	}
	rel, err := s.disk.Finalize(m.InstructorID, m.Path, int64(m.SizeBytes)) //nolint:gosec // G115: ≤ 2 GiB
	if err != nil {
		_ = s.fail(ctx, m, now)
		return m, err
	}
	if _, err := s.q.MarkMediaReady(ctx, store.MarkMediaReadyParams{Path: rel, Now: nt(now), ID: id}); err != nil {
		return m, fmt.Errorf("media: ready: %w", err)
	}
	if lesson != 0 {
		ref, err := s.q.GetLessonMediaRef(ctx, lesson)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return m, fmt.Errorf("media: lesson: %w", err)
		}
		if err := s.q.SetLessonMedia(ctx, store.SetLessonMediaParams{
			MediaID: nid(id), MediaStatus: StatusReady, SizeBytes: sql.NullInt64{Int64: int64(m.SizeBytes), Valid: true}, //nolint:gosec // G115: ≤ 2 GiB
			Now: nt(now), ID: lesson,
		}); err != nil {
			return m, fmt.Errorf("media: lesson media: %w", err)
		}
		if old := uid(ref.MediaID); old != 0 && old != id {
			if prev, err := s.get(ctx, old); err == nil {
				if err := s.remove(ctx, prev); err != nil {
					s.logger.ErrorContext(ctx, "media: replaced file not removed", slog.Uint64("media_id", old), slog.String("error", err.Error()))
				}
			}
		}
	}
	return s.get(ctx, id)
}

// fail marks a rejected upload failed, removes its bytes and puts the lesson back on its current media.
func (s *Service) fail(ctx context.Context, m store.LearningMedium, now time.Time) error {
	if _, err := s.q.SetMediaStatus(ctx, store.SetMediaStatusParams{Status: StatusFailed, Now: nt(now), ID: m.ID, FromStatus: StatusProcessing}); err != nil {
		return fmt.Errorf("media: fail: %w", err)
	}
	if err := s.disk.Remove(m.InstructorID, m.Path); err != nil {
		s.logger.ErrorContext(ctx, "media: rejected file not removed", slog.Uint64("media_id", m.ID), slog.String("error", err.Error()))
	}
	if lesson := uid(m.LessonID); lesson != 0 {
		if err := s.q.SetLessonMediaStatus(ctx, store.SetLessonMediaStatusParams{MediaStatus: StatusFailed, Now: nt(now), ID: lesson}); err != nil {
			return fmt.Errorf("media: lesson status: %w", err)
		}
	}
	return ErrRejected
}

// Cancel removes one of instructor's media (an upload in flight or a ready file); the lesson follows.
func (s *Service) Cancel(ctx context.Context, instructor, id uint64, now time.Time) error {
	unlock, ok := s.lock(id)
	if !ok {
		return ErrBusy
	}
	defer unlock()
	m, err := s.Owned(ctx, instructor, id)
	if err != nil {
		return err
	}
	if err := s.remove(ctx, m); err != nil {
		return err
	}
	return s.restoreLesson(ctx, m, now)
}

// restoreLesson points the lesson's media columns back at what is still there after m was removed.
func (s *Service) restoreLesson(ctx context.Context, m store.LearningMedium, now time.Time) error {
	lesson := uid(m.LessonID)
	if lesson == 0 {
		return nil
	}
	ref, err := s.q.GetLessonMediaRef(ctx, lesson)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("media: lesson: %w", err)
	}
	if uid(ref.MediaID) == m.ID {
		return s.q.SetLessonMedia(ctx, store.SetLessonMediaParams{MediaStatus: LessonNone, Now: nt(now), ID: lesson})
	}
	status := LessonNone
	if ref.MediaID.Valid {
		status = StatusReady
	}
	if ref.MediaStatus == status {
		return nil
	}
	return s.q.SetLessonMediaStatus(ctx, store.SetLessonMediaStatusParams{MediaStatus: status, Now: nt(now), ID: lesson})
}

// remove deletes a media row and its file.
func (s *Service) remove(ctx context.Context, m store.LearningMedium) error {
	if err := s.disk.Remove(m.InstructorID, m.Path); err != nil {
		return err
	}
	if _, err := s.q.DeleteMedia(ctx, m.ID); err != nil {
		return fmt.Errorf("media: delete: %w", err)
	}
	s.locks.Delete(m.ID)
	return nil
}

// Playback is a minted URL.
type Playback struct {
	URL       string
	ExpiresAt time.Time
	Media     store.LearningMedium
}

// PlaybackFor mints a URL of a ready media for viewer.
func (s *Service) PlaybackFor(m store.LearningMedium, viewer uint64, now time.Time) (Playback, error) {
	if s.signer == nil {
		return Playback{}, ErrDisabled
	}
	if m.Status != StatusReady {
		return Playback{}, ErrNotReady
	}
	exp := now.Add(DefaultURLTTL).Truncate(time.Second)
	q := url.Values{}
	q.Set("u", strconv.FormatUint(viewer, 10))
	q.Set("expires", strconv.FormatInt(exp.Unix(), 10))
	q.Set("signature", s.signer.Sign(m.ID, viewer, exp))
	return Playback{URL: s.baseURL + "/api/v1/media/" + strconv.FormatUint(m.ID, 10) + "/stream?" + q.Encode(),
		ExpiresAt: exp, Media: m}, nil
}

// LessonPlayback mints a URL of a lesson's ready media (the caller checked the viewer's access to the lesson).
func (s *Service) LessonPlayback(ctx context.Context, mediaID sql.NullInt64, viewer uint64, now time.Time) (Playback, error) {
	if !mediaID.Valid {
		return Playback{}, ErrNotReady
	}
	m, err := s.get(ctx, uid(mediaID))
	if errors.Is(err, ErrNotFound) {
		return Playback{}, ErrNotReady
	}
	if err != nil {
		return Playback{}, err
	}
	return s.PlaybackFor(m, viewer, now)
}

// OpenStream verifies a playback link and opens its file: ErrLinkInvalid / ErrLinkExpired / ErrNotFound.
func (s *Service) OpenStream(ctx context.Context, id uint64, viewer, expires, signature string, now time.Time) (store.LearningMedium, *os.File, error) {
	if s.signer == nil || !s.disk.Enabled() {
		return store.LearningMedium{}, nil, ErrDisabled
	}
	if _, err := s.signer.Verify(id, viewer, expires, signature, now); err != nil {
		return store.LearningMedium{}, nil, err
	}
	m, err := s.get(ctx, id)
	if err != nil {
		return m, nil, err
	}
	if m.Status != StatusReady {
		return m, nil, ErrNotFound
	}
	f, err := s.disk.Open(m.InstructorID, m.Path)
	if err != nil {
		return m, nil, err
	}
	return m, f, nil
}
