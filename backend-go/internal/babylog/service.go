// Package babylog is the baby logs of a child (bloom B-N5-03; artboards nbl_Log_Feed, the «امروز» card of
// nbl_v16_ChildHome, the baby rows of nbl_Log_Sheet_Post and nbl_An_Hub_Post): feeding sessions (breast with a timer
// per side, bottle and pump with ml), baby sleep sessions and diapers, with day summaries.
//
// Built on what exists, nothing duplicated:
//   - the child and who may see it come from children.Service (owner reads and writes; the owner's spouse through a
//     shared family reads, audited, and gets 403 on writes);
//   - the mother's taxonomy v2 slot baby.feeds_count (B-N5-01: postpartum recovery, log sheet, analysis) is derived
//     from the feeding sessions: every feed save / delete rewrites the owner's day count (source baby_log) while she
//     has a postpartum profile; a day without feeds drops only the slot the sessions wrote.
//
// One running feed and one running sleep per child is a database rule (active_lock + unique index). Health data:
// nothing is logged.
package babylog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/babylog/store"
	"github.com/ritme/backend-go/internal/children"
	"github.com/ritme/backend-go/internal/platform/civildate"
)

// Limits.
const (
	MaxNoteLen        = 500
	MaxAmountMl       = 500
	MaxFeedMinutes    = 180 // per side, or the whole bottle / pump
	MaxSleepHours     = 24
	MaxSummaryDays    = 31
	DefaultSummary    = 7
	MaxFeedsCountSlot = 30 // taxonomy baby.feeds_count range
)

// Errors.
var (
	ErrFeedNotFound   = errors.New("babylog: feed not found")
	ErrSleepNotFound  = errors.New("babylog: sleep not found")
	ErrDiaperNotFound = errors.New("babylog: diaper not found")
	ErrFeedActive     = errors.New("babylog: a feed is already running")
	ErrSleepActive    = errors.New("babylog: a sleep is already running")
	ErrEnded          = errors.New("babylog: session already ended")
	ErrRunning        = errors.New("babylog: session still running")
	ErrNotBreast      = errors.New("babylog: sides are for breast feeds")
)

// Children is the part of children.Service the baby logs use.
type Children interface {
	Access(ctx context.Context, userID, childID uint64, now time.Time) (children.Access, error)
	Editable(ctx context.Context, userID, childID uint64, now time.Time) (children.Access, error)
}

// Service is the baby-log logic.
type Service struct {
	db       *sql.DB // nil when built on a transaction (tests)
	q        *store.Queries
	children Children
}

// NewService returns a Service on db (a *sql.DB or a transaction).
func NewService(db store.DBTX, ch Children) *Service {
	s := &Service{q: store.New(db), children: ch}
	if pool, ok := db.(*sql.DB); ok {
		s.db = pool
	}
	return s
}

// Access / Editable resolve a child for reading / writing (children errors pass through).
func (s *Service) Access(ctx context.Context, userID, childID uint64, now time.Time) (children.Access, error) {
	return s.children.Access(ctx, userID, childID, now)
}

// Editable is Access for a write (children.ErrReadOnly for a shared child).
func (s *Service) Editable(ctx context.Context, userID, childID uint64, now time.Time) (children.Access, error) {
	return s.children.Editable(ctx, userID, childID, now)
}

func dbTime(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

func nullTime(t time.Time) sql.NullTime { return sql.NullTime{Time: dbTime(t), Valid: true} }

var active = sql.NullInt16{Int16: 1, Valid: true}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

func (s *Service) inTx(ctx context.Context, fn func(q *store.Queries) error) error {
	if s.db == nil {
		return fn(s.q)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("babylog: begin: %w", err)
	}
	if err := fn(s.q.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("babylog: commit: %w", err)
	}
	return nil
}

func dayRange(d civildate.Date, days int) (time.Time, time.Time) {
	return d.TehranMidnight(), d.AddDays(days).TehranMidnight()
}

// ---------------------------------------------------------------- feeds

// Feeds are the feeds started on day, newest first.
func (s *Service) Feeds(ctx context.Context, childID uint64, day civildate.Date) ([]store.BabyFeed, error) {
	from, to := dayRange(day, 1)
	rows, err := s.q.ListFeedsBetween(ctx, store.ListFeedsBetweenParams{ChildID: childID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("babylog: feeds: %w", err)
	}
	return rows, nil
}

// ActiveFeed is the running feed (nil when none).
func (s *Service) ActiveFeed(ctx context.Context, childID uint64) (*store.BabyFeed, error) {
	f, err := s.q.GetActiveFeed(ctx, childID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("babylog: active feed: %w", err)
	}
	return &f, nil
}

// LastFeed is the last ended feed (nil when none).
func (s *Service) LastFeed(ctx context.Context, childID uint64) (*store.BabyFeed, error) {
	f, err := s.q.LastEndedFeed(ctx, childID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("babylog: last feed: %w", err)
	}
	return &f, nil
}

// NextSide is the side to offer first («دفعه قبل · راست» → چپ): the other one than the last breast feed ended on.
func (s *Service) NextSide(ctx context.Context, childID uint64) (string, error) {
	side, err := s.q.LastBreastSide(ctx, childID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("babylog: last side: %w", err)
	}
	return NextSide(side.String), nil
}

// Feed loads one feed of the child.
func (s *Service) Feed(ctx context.Context, childID, id uint64) (store.BabyFeed, error) {
	f, err := s.q.GetFeed(ctx, store.GetFeedParams{ID: id, ChildID: childID})
	if errors.Is(err, sql.ErrNoRows) {
		return f, ErrFeedNotFound
	}
	if err != nil {
		return f, fmt.Errorf("babylog: feed: %w", err)
	}
	return f, nil
}

// StartFeed opens a running feed; a breast feed may start on a side.
func (s *Service) StartFeed(ctx context.Context, a children.Access, typ, side string, now time.Time) (store.BabyFeed, error) {
	p := store.InsertFeedParams{ChildID: a.Child.ID, Type: typ, StartedAt: dbTime(now), ActiveLock: active, Now: nullTime(now)}
	if typ == typeBreast && side != "" {
		p.ActiveSide = sql.NullString{String: side, Valid: true}
		p.SideStartedAt = nullTime(now)
	}
	id, err := s.q.InsertFeed(ctx, p)
	if isDuplicate(err) {
		return store.BabyFeed{}, ErrFeedActive
	}
	if err != nil {
		return store.BabyFeed{}, fmt.Errorf("babylog: start feed: %w", err)
	}
	if err := s.writeFeedsCount(ctx, s.q, a.Child.OwnerID, civildate.InTehran(now), now); err != nil {
		return store.BabyFeed{}, err
	}
	return s.Feed(ctx, a.Child.ID, uint64(id)) //nolint:gosec // auto-increment id
}

// closeSegment folds the running side segment into its side total.
func closeSegment(f *store.BabyFeed, now time.Time) {
	if f.ActiveSide.Valid && f.SideStartedAt.Valid {
		run := uint32(span(f.SideStartedAt.Time, now)) //nolint:gosec // a feed segment, far below 2^32 s
		if f.ActiveSide.String == "left" {
			f.LeftSeconds += run
		} else {
			f.RightSeconds += run
		}
		f.LastSide = f.ActiveSide
	}
	f.ActiveSide, f.SideStartedAt = sql.NullString{}, sql.NullTime{}
}

func updateParams(f store.BabyFeed, now time.Time) store.UpdateFeedParams {
	return store.UpdateFeedParams{
		Type: f.Type, StartedAt: f.StartedAt, EndedAt: f.EndedAt, ActiveSide: f.ActiveSide, SideStartedAt: f.SideStartedAt,
		LastSide: f.LastSide, LeftSeconds: f.LeftSeconds, RightSeconds: f.RightSeconds, DurationSeconds: f.DurationSeconds,
		AmountMl: f.AmountMl, Note: f.Note, ActiveLock: f.ActiveLock, Now: nullTime(now), ID: f.ID, ChildID: f.ChildID,
	}
}

// lockRunning loads a running feed for update.
func lockRunning(ctx context.Context, q *store.Queries, childID, id uint64) (store.BabyFeed, error) {
	f, err := q.LockFeed(ctx, store.LockFeedParams{ID: id, ChildID: childID})
	if errors.Is(err, sql.ErrNoRows) {
		return f, ErrFeedNotFound
	}
	if err != nil {
		return f, fmt.Errorf("babylog: lock feed: %w", err)
	}
	if !f.ActiveLock.Valid {
		return f, ErrEnded
	}
	return f, nil
}

// SetSide switches the running breast feed to side ("" pauses both sides).
func (s *Service) SetSide(ctx context.Context, childID, id uint64, side string, now time.Time) (store.BabyFeed, error) {
	var out store.BabyFeed
	err := s.inTx(ctx, func(q *store.Queries) error {
		f, err := lockRunning(ctx, q, childID, id)
		if err != nil {
			return err
		}
		if f.Type != typeBreast {
			return ErrNotBreast
		}
		if f.ActiveSide.Valid && f.ActiveSide.String == side {
			out = f
			return nil // already on that side
		}
		closeSegment(&f, now)
		if side != "" {
			f.ActiveSide, f.SideStartedAt = sql.NullString{String: side, Valid: true}, nullTime(now)
		}
		if err := q.UpdateFeed(ctx, updateParams(f, now)); err != nil {
			return fmt.Errorf("babylog: side: %w", err)
		}
		out = f
		return nil
	})
	return out, err
}

// StopInput is the optional data given when a feed ends.
type StopInput struct {
	AmountMl sql.NullInt16
	Note     sql.NullString
}

// StopFeed ends the running feed.
func (s *Service) StopFeed(ctx context.Context, a children.Access, id uint64, in StopInput, now time.Time) (store.BabyFeed, error) {
	var out store.BabyFeed
	err := s.inTx(ctx, func(q *store.Queries) error {
		f, err := lockRunning(ctx, q, a.Child.ID, id)
		if err != nil {
			return err
		}
		closeSegment(&f, now)
		f.EndedAt, f.ActiveLock = nullTime(now), sql.NullInt16{}
		if f.Type == typeBreast {
			f.DurationSeconds = sql.NullInt32{Int32: int32(f.LeftSeconds + f.RightSeconds), Valid: true} //nolint:gosec // seconds
		} else {
			f.DurationSeconds = sql.NullInt32{Int32: int32(span(f.StartedAt, now)), Valid: true} //nolint:gosec // seconds
			if in.AmountMl.Valid {
				f.AmountMl = in.AmountMl
			}
		}
		if in.Note.Valid {
			f.Note = in.Note
		}
		if err := q.UpdateFeed(ctx, updateParams(f, now)); err != nil {
			return fmt.Errorf("babylog: stop feed: %w", err)
		}
		out = f
		return nil
	})
	return out, err
}

// FeedInput is a validated manual (ended) feed.
type FeedInput struct {
	Type         string
	StartedAt    time.Time
	LeftSeconds  uint32
	RightSeconds uint32
	Duration     int64 // bottle / pump seconds
	LastSide     sql.NullString
	AmountMl     sql.NullInt16
	Note         sql.NullString
}

func (in FeedInput) row(childID uint64) store.BabyFeed {
	f := store.BabyFeed{
		ChildID: childID, Type: in.Type, StartedAt: dbTime(in.StartedAt), LastSide: in.LastSide, Note: in.Note,
	}
	dur := in.Duration
	if in.Type == typeBreast {
		f.LeftSeconds, f.RightSeconds = in.LeftSeconds, in.RightSeconds
		dur = int64(in.LeftSeconds) + int64(in.RightSeconds)
	} else {
		f.AmountMl = in.AmountMl
	}
	f.DurationSeconds = sql.NullInt32{Int32: int32(dur), Valid: true} //nolint:gosec // ≤ 2 × MaxFeedMinutes
	f.EndedAt = nullTime(in.StartedAt.Add(time.Duration(dur) * time.Second))
	return f
}

// CreateFeed adds an ended feed («ثبت دستی»).
func (s *Service) CreateFeed(ctx context.Context, a children.Access, in FeedInput, now time.Time) (store.BabyFeed, error) {
	f := in.row(a.Child.ID)
	id, err := s.q.InsertFeed(ctx, store.InsertFeedParams{
		ChildID: f.ChildID, Type: f.Type, StartedAt: f.StartedAt, EndedAt: f.EndedAt, LastSide: f.LastSide,
		LeftSeconds: f.LeftSeconds, RightSeconds: f.RightSeconds, DurationSeconds: f.DurationSeconds,
		AmountMl: f.AmountMl, Note: f.Note, Now: nullTime(now),
	})
	if err != nil {
		return store.BabyFeed{}, fmt.Errorf("babylog: create feed: %w", err)
	}
	if err := s.writeFeedsCount(ctx, s.q, a.Child.OwnerID, civildate.InTehran(f.StartedAt), now); err != nil {
		return store.BabyFeed{}, err
	}
	return s.Feed(ctx, a.Child.ID, uint64(id)) //nolint:gosec // auto-increment id
}

// UpdateFeed replaces an ended feed (ErrRunning for the running one).
func (s *Service) UpdateFeed(ctx context.Context, a children.Access, id uint64, in FeedInput, now time.Time) (store.BabyFeed, error) {
	var out store.BabyFeed
	var oldDay civildate.Date
	err := s.inTx(ctx, func(q *store.Queries) error {
		cur, err := q.LockFeed(ctx, store.LockFeedParams{ID: id, ChildID: a.Child.ID})
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFeedNotFound
		}
		if err != nil {
			return fmt.Errorf("babylog: lock feed: %w", err)
		}
		if cur.ActiveLock.Valid {
			return ErrRunning
		}
		oldDay = civildate.InTehran(cur.StartedAt)
		f := in.row(a.Child.ID)
		f.ID = cur.ID
		if err := q.UpdateFeed(ctx, updateParams(f, now)); err != nil {
			return fmt.Errorf("babylog: update feed: %w", err)
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	for _, d := range uniqueDays(oldDay, civildate.InTehran(in.StartedAt)) {
		if err := s.writeFeedsCount(ctx, s.q, a.Child.OwnerID, d, now); err != nil {
			return out, err
		}
	}
	return s.Feed(ctx, a.Child.ID, id)
}

// DeleteFeed removes a feed (running or ended).
func (s *Service) DeleteFeed(ctx context.Context, a children.Access, id uint64, now time.Time) error {
	f, err := s.Feed(ctx, a.Child.ID, id)
	if err != nil {
		return err
	}
	if _, err := s.q.DeleteFeed(ctx, store.DeleteFeedParams{ID: id, ChildID: a.Child.ID}); err != nil {
		return fmt.Errorf("babylog: delete feed: %w", err)
	}
	return s.writeFeedsCount(ctx, s.q, a.Child.OwnerID, civildate.InTehran(f.StartedAt), now)
}

func uniqueDays(a, b civildate.Date) []civildate.Date {
	if a == b {
		return []civildate.Date{a}
	}
	return []civildate.Date{a, b}
}

// writeFeedsCount rewrites the owner's baby.feeds_count of day from the feeds of all her children (only while she
// has a postpartum profile; capped at the taxonomy's range).
func (s *Service) writeFeedsCount(ctx context.Context, q *store.Queries, ownerID uint64, day civildate.Date, now time.Time) error {
	on, err := q.HasPostpartumProfile(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("babylog: postpartum: %w", err)
	}
	if !on {
		return nil
	}
	from, to := dayRange(day, 1)
	n, err := q.CountOwnerFeedsBetween(ctx, store.CountOwnerFeedsBetweenParams{OwnerID: ownerID, DateFrom: from, DateTo: to})
	if err != nil {
		return fmt.Errorf("babylog: count feeds: %w", err)
	}
	if n == 0 {
		if err := q.DeleteFeedsCount(ctx, store.DeleteFeedsCountParams{UserID: ownerID, LogDate: day}); err != nil {
			return fmt.Errorf("babylog: clear feeds count: %w", err)
		}
		return nil
	}
	if err := q.UpsertFeedsCount(ctx, store.UpsertFeedsCountParams{
		UserID: ownerID, LogDate: day, Now: nullTime(now),
		ValueNum: sql.NullString{String: strconv.FormatInt(min(n, MaxFeedsCountSlot), 10), Valid: true},
	}); err != nil {
		return fmt.Errorf("babylog: feeds count: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------- sleep

// Sleeps are the sleeps overlapping day, newest first.
func (s *Service) Sleeps(ctx context.Context, childID uint64, day civildate.Date) ([]store.BabySleep, error) {
	from, to := dayRange(day, 1)
	rows, err := s.q.ListSleepsOverlapping(ctx, store.ListSleepsOverlappingParams{ChildID: childID, DateFrom: sql.NullTime{Time: from, Valid: true}, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("babylog: sleeps: %w", err)
	}
	return rows, nil
}

// ActiveSleep is the running sleep (nil when none).
func (s *Service) ActiveSleep(ctx context.Context, childID uint64) (*store.BabySleep, error) {
	sl, err := s.q.GetActiveSleep(ctx, childID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("babylog: active sleep: %w", err)
	}
	return &sl, nil
}

// Sleep loads one sleep of the child.
func (s *Service) Sleep(ctx context.Context, childID, id uint64) (store.BabySleep, error) {
	sl, err := s.q.GetSleep(ctx, store.GetSleepParams{ID: id, ChildID: childID})
	if errors.Is(err, sql.ErrNoRows) {
		return sl, ErrSleepNotFound
	}
	if err != nil {
		return sl, fmt.Errorf("babylog: sleep: %w", err)
	}
	return sl, nil
}

// StartSleep opens a running sleep.
func (s *Service) StartSleep(ctx context.Context, childID uint64, now time.Time) (store.BabySleep, error) {
	id, err := s.q.InsertSleep(ctx, store.InsertSleepParams{ChildID: childID, StartedAt: dbTime(now), ActiveLock: active, Now: nullTime(now)})
	if isDuplicate(err) {
		return store.BabySleep{}, ErrSleepActive
	}
	if err != nil {
		return store.BabySleep{}, fmt.Errorf("babylog: start sleep: %w", err)
	}
	return s.Sleep(ctx, childID, uint64(id)) //nolint:gosec // auto-increment id
}

// StopSleep ends the running sleep.
func (s *Service) StopSleep(ctx context.Context, childID, id uint64, note sql.NullString, now time.Time) (store.BabySleep, error) {
	n, err := s.q.StopSleep(ctx, store.StopSleepParams{EndedAt: nullTime(now), Note: note, Now: nullTime(now), ID: id, ChildID: childID})
	if err != nil {
		return store.BabySleep{}, fmt.Errorf("babylog: stop sleep: %w", err)
	}
	sl, err := s.Sleep(ctx, childID, id)
	if err != nil {
		return sl, err
	}
	if n == 0 {
		return sl, ErrEnded
	}
	return sl, nil
}

// SleepInput is a validated manual (ended) sleep.
type SleepInput struct {
	StartedAt, EndedAt time.Time
	Note               sql.NullString
}

// CreateSleep adds an ended sleep.
func (s *Service) CreateSleep(ctx context.Context, childID uint64, in SleepInput, now time.Time) (store.BabySleep, error) {
	id, err := s.q.InsertSleep(ctx, store.InsertSleepParams{
		ChildID: childID, StartedAt: dbTime(in.StartedAt), EndedAt: nullTime(in.EndedAt), Note: in.Note, Now: nullTime(now),
	})
	if err != nil {
		return store.BabySleep{}, fmt.Errorf("babylog: create sleep: %w", err)
	}
	return s.Sleep(ctx, childID, uint64(id)) //nolint:gosec // auto-increment id
}

// UpdateSleep replaces an ended sleep (ErrRunning for the running one).
func (s *Service) UpdateSleep(ctx context.Context, childID, id uint64, in SleepInput, now time.Time) (store.BabySleep, error) {
	cur, err := s.Sleep(ctx, childID, id)
	if err != nil {
		return cur, err
	}
	if cur.ActiveLock.Valid {
		return cur, ErrRunning
	}
	if err := s.q.UpdateSleep(ctx, store.UpdateSleepParams{
		StartedAt: dbTime(in.StartedAt), EndedAt: nullTime(in.EndedAt), Note: in.Note, Now: nullTime(now), ID: id, ChildID: childID,
	}); err != nil {
		return cur, fmt.Errorf("babylog: update sleep: %w", err)
	}
	return s.Sleep(ctx, childID, id)
}

// DeleteSleep removes a sleep (running or ended).
func (s *Service) DeleteSleep(ctx context.Context, childID, id uint64) error {
	n, err := s.q.DeleteSleep(ctx, store.DeleteSleepParams{ID: id, ChildID: childID})
	if err != nil {
		return fmt.Errorf("babylog: delete sleep: %w", err)
	}
	if n == 0 {
		return ErrSleepNotFound
	}
	return nil
}

// ---------------------------------------------------------------- diapers

// Diapers are the diapers changed on day, newest first.
func (s *Service) Diapers(ctx context.Context, childID uint64, day civildate.Date) ([]store.BabyDiaper, error) {
	from, to := dayRange(day, 1)
	rows, err := s.q.ListDiapersBetween(ctx, store.ListDiapersBetweenParams{ChildID: childID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("babylog: diapers: %w", err)
	}
	return rows, nil
}

// Diaper loads one diaper of the child.
func (s *Service) Diaper(ctx context.Context, childID, id uint64) (store.BabyDiaper, error) {
	d, err := s.q.GetDiaper(ctx, store.GetDiaperParams{ID: id, ChildID: childID})
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrDiaperNotFound
	}
	if err != nil {
		return d, fmt.Errorf("babylog: diaper: %w", err)
	}
	return d, nil
}

// DiaperInput is a validated diaper change.
type DiaperInput struct {
	ChangedAt time.Time
	Kind      string
	Note      sql.NullString
}

// CreateDiaper adds a diaper change.
func (s *Service) CreateDiaper(ctx context.Context, childID uint64, in DiaperInput, now time.Time) (store.BabyDiaper, error) {
	id, err := s.q.InsertDiaper(ctx, store.InsertDiaperParams{
		ChildID: childID, ChangedAt: dbTime(in.ChangedAt), Kind: in.Kind, Note: in.Note, Now: nullTime(now),
	})
	if err != nil {
		return store.BabyDiaper{}, fmt.Errorf("babylog: create diaper: %w", err)
	}
	return s.Diaper(ctx, childID, uint64(id)) //nolint:gosec // auto-increment id
}

// UpdateDiaper replaces a diaper change.
func (s *Service) UpdateDiaper(ctx context.Context, childID, id uint64, in DiaperInput, now time.Time) (store.BabyDiaper, error) {
	if _, err := s.Diaper(ctx, childID, id); err != nil {
		return store.BabyDiaper{}, err
	}
	if err := s.q.UpdateDiaper(ctx, store.UpdateDiaperParams{
		ChangedAt: dbTime(in.ChangedAt), Kind: in.Kind, Note: in.Note, Now: nullTime(now), ID: id, ChildID: childID,
	}); err != nil {
		return store.BabyDiaper{}, fmt.Errorf("babylog: update diaper: %w", err)
	}
	return s.Diaper(ctx, childID, id)
}

// DeleteDiaper removes a diaper change.
func (s *Service) DeleteDiaper(ctx context.Context, childID, id uint64) error {
	n, err := s.q.DeleteDiaper(ctx, store.DeleteDiaperParams{ID: id, ChildID: childID})
	if err != nil {
		return fmt.Errorf("babylog: delete diaper: %w", err)
	}
	if n == 0 {
		return ErrDiaperNotFound
	}
	return nil
}

// ---------------------------------------------------------------- summaries

// Days are the day summaries of [from, from+days) at now.
func (s *Service) Days(ctx context.Context, childID uint64, from civildate.Date, days int, now time.Time) ([]Day, error) {
	lo, hi := dayRange(from, days)
	feeds, err := s.q.ListFeedsBetween(ctx, store.ListFeedsBetweenParams{ChildID: childID, DateFrom: lo, DateTo: hi})
	if err != nil {
		return nil, fmt.Errorf("babylog: feeds: %w", err)
	}
	sleeps, err := s.q.ListSleepsOverlapping(ctx, store.ListSleepsOverlappingParams{ChildID: childID, DateFrom: sql.NullTime{Time: lo, Valid: true}, DateTo: hi})
	if err != nil {
		return nil, fmt.Errorf("babylog: sleeps: %w", err)
	}
	diapers, err := s.q.ListDiapersBetween(ctx, store.ListDiapersBetweenParams{ChildID: childID, DateFrom: lo, DateTo: hi})
	if err != nil {
		return nil, fmt.Errorf("babylog: diapers: %w", err)
	}
	return Summarize(from, days, feeds, sleeps, diapers, now), nil
}
