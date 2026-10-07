package learning

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/learning/store"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Errors mapped to 403 / 404 / 422 by the handlers. A foreign id is the same 404 as a missing one.
var (
	ErrNotInstructor     = errors.New("learning: not an approved instructor")
	ErrInstructorPending = errors.New("learning: instructor application pending")
	ErrCourseNotFound    = errors.New("learning: course not found")
	ErrChapterNotFound   = errors.New("learning: chapter not found")
	ErrLessonNotFound    = errors.New("learning: lesson not found")
	ErrGroupNotFound     = errors.New("learning: group not found")
	ErrGrantNotFound     = errors.New("learning: grant not found")
	ErrAccessExpired     = errors.New("learning: access expired")
	ErrTooMany           = errors.New("learning: limit reached")
	ErrStandaloneSingle  = errors.New("learning: a standalone item has one lesson")
	ErrStandaloneChapter = errors.New("learning: a standalone item has no chapters")
	ErrChapterInvalid    = errors.New("learning: chapter of another course")
	ErrTargetInvalid     = errors.New("learning: group or course of another instructor")
	ErrCourseIDsInvalid  = errors.New("learning: course of another instructor")
)

// ChapterLockedError is a lesson of a chapter that opens later («فصل ۳ از ۱۵ مهر توسط مدرس باز می‌شود»).
type ChapterLockedError struct{ UnlockAt time.Time }

func (e *ChapterLockedError) Error() string { return "learning: chapter locked" }

// Options wires the service.
type Options struct {
	DB *sql.DB
	// SMS is the «دوره برایت باز شد» sender (nil = none).
	SMS Sender
	// Languages lists the active language codes (the `languages` table) for the inbox notice JSON.
	Languages func(ctx context.Context) []string
	Logger    *slog.Logger
}

// Service is the learning data access and rules.
type Service struct {
	db     *sql.DB
	q      *store.Queries
	prefs  notifications.Getter
	sms    Sender
	langs  func(ctx context.Context) []string
	logger *slog.Logger
}

// NewService wires the service.
func NewService(o Options) *Service {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.SMS == nil {
		o.SMS = NoSMS{}
	}
	if o.Languages == nil {
		o.Languages = func(context.Context) []string { return nil }
	}
	return &Service{db: o.DB, q: store.New(o.DB), prefs: profilestore.New(o.DB), sms: o.SMS, langs: o.Languages, logger: o.Logger}
}

// Queries exposes the store (tests, admin moderation B-N8-08).
func (s *Service) Queries() *store.Queries { return s.q }

// tx runs fn in a transaction.
func (s *Service) tx(ctx context.Context, fn func(q *store.Queries) error) error {
	t, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("learning: begin: %w", err)
	}
	if err := fn(s.q.WithTx(t)); err != nil {
		_ = t.Rollback()
		return err
	}
	if err := t.Commit(); err != nil {
		return fmt.Errorf("learning: commit: %w", err)
	}
	return nil
}

func nt(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}

func ns(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nid(id uint64) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(id), Valid: id != 0} //nolint:gosec // G115: auto-increment ids
}

func uid(v sql.NullInt64) uint64 {
	if !v.Valid || v.Int64 <= 0 {
		return 0
	}
	return uint64(v.Int64)
}

func timeOf(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func lastID(id int64, err error) (uint64, error) {
	if err != nil {
		return 0, err
	}
	return uint64(id), nil //nolint:gosec // G115: auto-increment ids are positive
}

// ───────────── instructors ─────────────

// InstructorByUser reads userID's instructor row (ok=false: never applied).
func (s *Service) InstructorByUser(ctx context.Context, userID uint64) (store.LearningInstructor, bool, error) {
	r, err := s.q.GetInstructorByUser(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return r, false, nil
	}
	if err != nil {
		return r, false, fmt.Errorf("learning: instructor: %w", err)
	}
	return r, true, nil
}

// ApplyInput is a validated instructor application / profile.
type ApplyInput struct {
	DisplayName, Title, Bio string
}

// Apply records userID's instructor application (pending until an admin approves it). A pending or revoked
// applicant updates the profile (revoked → pending again); an approved instructor updates the profile only.
func (s *Service) Apply(ctx context.Context, userID uint64, in ApplyInput, now time.Time) (store.LearningInstructor, bool, error) {
	cur, ok, err := s.InstructorByUser(ctx, userID)
	if err != nil {
		return cur, false, err
	}
	if !ok {
		id, err := lastID(s.q.InsertInstructor(ctx, store.InsertInstructorParams{
			UserID: userID, DisplayName: in.DisplayName, Title: ns(in.Title), Bio: ns(in.Bio), Now: nt(now),
		}))
		if err != nil {
			return cur, false, fmt.Errorf("learning: apply: %w", err)
		}
		r, err := s.q.GetInstructor(ctx, id)
		if err != nil {
			return r, false, fmt.Errorf("learning: apply: %w", err)
		}
		return r, true, nil
	}
	if err := s.q.UpdateInstructorProfile(ctx, store.UpdateInstructorProfileParams{
		DisplayName: in.DisplayName, Title: ns(in.Title), Bio: ns(in.Bio), Now: nt(now), ID: cur.ID,
	}); err != nil {
		return cur, false, fmt.Errorf("learning: profile: %w", err)
	}
	if cur.Status == InstructorRevoked {
		if _, err := s.q.SetInstructorStatus(ctx, store.SetInstructorStatusParams{
			Status: InstructorPending, Now: nt(now), ID: cur.ID,
		}); err != nil {
			return cur, false, fmt.Errorf("learning: re-apply: %w", err)
		}
	}
	r, err := s.q.GetInstructor(ctx, cur.ID)
	if err != nil {
		return r, false, fmt.Errorf("learning: apply: %w", err)
	}
	return r, false, nil
}

// SetInstructorStatus approves (adminID = the admin) or revokes an instructor (admin moderation, B-N8-08).
func (s *Service) SetInstructorStatus(ctx context.Context, instructorID uint64, status string, adminID uint64, now time.Time) error {
	p := store.SetInstructorStatusParams{Status: status, Now: nt(now), ID: instructorID}
	switch status {
	case InstructorApproved:
		p.ApprovedAt, p.ApprovedBy = nt(now), nid(adminID)
	case InstructorRevoked:
		p.RevokedAt = nt(now)
	}
	n, err := s.q.SetInstructorStatus(ctx, p)
	if err != nil {
		return fmt.Errorf("learning: instructor status: %w", err)
	}
	if n == 0 {
		return ErrNotInstructor
	}
	return nil
}
