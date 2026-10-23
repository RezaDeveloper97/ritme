// Package periods is the port of PeriodLogController (backend/app/Http/Controllers/Api/V1/
// PeriodLogController.php): period status/history and the start / end / store / update /
// destroy writes with their cycle_histories reconciliation (reuse or promote the same-day row,
// drop onboarding estimates, recompute cycle lengths, re-anchor the profile LMP,
// markRecalculated).
//
// Every write runs in one transaction. Eloquent's `update()` writes only dirty attributes and
// touches updated_at only then; saveDirty reproduces that, so the rows (timestamps included)
// end up exactly as Laravel leaves them.
package periods

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/cycle/revision"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	filestore "github.com/ritme/backend-go/internal/files/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/profile"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
)

// Service runs the period queries and writes.
type Service struct {
	db *sql.DB
	q  *store.Queries
}

// NewService returns the service over db.
func NewService(conn *sql.DB) *Service { return &Service{db: conn, q: store.New(conn)} }

// Refusal is a 4xx outcome of a write (the handler renders it).
type Refusal struct {
	Status int
	// Code is the `code` key (previous_period_open, period_overlap) or "".
	Code string
	// OpenPeriodStart is data.open_period_start for previous_period_open.
	OpenPeriodStart civildate.Date
	Kind            refusalKind
}

type refusalKind int

const (
	refusePreviousOpen refusalKind = iota + 1
	refuseStartOverlap
	refuseRangeOverlap
	refuseNoOngoing
	refuseEndBeforeStart
	refuseNotFound
)

func (r *Refusal) Error() string { return fmt.Sprintf("periods: refused (%d %s)", r.Status, r.Code) }

// Result is a successful write: the affected row as Laravel holds it after the write.
type Result struct {
	Row store.CycleHistory
}

// Status is GET /cycle/period/status: whether a period is open and the latest period's dates.
func (s *Service) Status(ctx context.Context, userID uint64) (active bool, latest *store.CycleHistory, err error) {
	if _, err := s.q.GetOngoingPeriod(ctx, userID); err == nil {
		active = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return false, nil, fmt.Errorf("periods: ongoing: %w", err)
	}
	row, err := s.q.GetLatestPeriod(ctx, userID)
	switch {
	case err == nil:
		return active, &row, nil
	case errors.Is(err, sql.ErrNoRows):
		return active, nil, nil
	default:
		return false, nil, fmt.Errorf("periods: latest: %w", err)
	}
}

// History is GET /cycle/period/history: the non-estimated periods, newest first.
func (s *Service) History(ctx context.Context, userID uint64) ([]store.CycleHistory, error) {
	rows, err := s.q.ListLoggedPeriods(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("periods: history: %w", err)
	}
	return rows, nil
}

// Start is POST /cycle/period/start with the parsed start day.
func (s *Service) Start(ctx context.Context, userID uint64, start civildate.Date, now time.Time) (*Result, error) {
	var res *Result
	err := s.tx(ctx, userID, func(q *store.Queries, pq *profilestore.Queries) error {
		period, err := optional(q.GetPeriodByStart(ctx, store.GetPeriodByStartParams{UserID: userID, PeriodStartDate: start}))
		if err != nil {
			return err
		}
		today := civildate.InTehran(now)
		blocker, err := optional(q.GetBlockingOpenPeriod(ctx, store.GetBlockingOpenPeriodParams{
			UserID: userID, BeforeDate: start, SinceDate: today.AddDays(-hardCapDays),
		}))
		if err != nil {
			return err
		}
		if blocker != nil {
			return &Refusal{Status: 422, Code: "previous_period_open", OpenPeriodStart: blocker.PeriodStartDate, Kind: refusePreviousOpen}
		}
		overlap, err := overlapsExisting(ctx, q, userID, start, start, idOf(period))
		if err != nil {
			return err
		}
		if overlap {
			return &Refusal{Status: 422, Code: "period_overlap", Kind: refuseStartOverlap}
		}

		if period != nil {
			next := *period
			next.PeriodEndDate = civildate.NullDate{}
			next.BleedingLength = sql.NullInt32{}
			next.IsConfirmed, next.IsEstimated = true, false
			next.Source = string(enums.DataSourceUserLogged)
			if next, err = saveDirty(ctx, q, *period, next, now); err != nil {
				return err
			}
			period = &next
		} else {
			previous, err := optional(q.GetPreviousPeriod(ctx, store.GetPreviousPeriodParams{UserID: userID, PeriodStartDate: start}))
			if err != nil {
				return err
			}
			var cycleLength *int
			if previous != nil {
				n := previous.PeriodStartDate.DiffDays(start)
				cycleLength = &n
			}
			row := store.CycleHistory{
				UserID: userID, PeriodStartDate: start, CycleLength: nullInt(cycleLength),
				IsConfirmed: true, Source: string(enums.DataSourceUserLogged),
				DataQualityFlags: flagsJSON(QualityFlags(cycleLength, nil)),
				CreatedAt:        dbTime(now), UpdatedAt: dbTime(now),
			}
			if row.ID, err = insert(ctx, q, row); err != nil {
				return err
			}
			period = &row
		}

		if err := q.DeleteEstimatesExcept(ctx, store.DeleteEstimatesExceptParams{UserID: userID, ID: period.ID}); err != nil {
			return fmt.Errorf("periods: delete estimates: %w", err)
		}
		prof, err := optional(q.GetProfileByUserID(ctx, userID))
		if err != nil {
			return err
		}
		if prof != nil {
			if err := setLastPeriodStart(ctx, q, pq, prof, civildate.NullDate{Date: start, Valid: true}, now); err != nil {
				return err
			}
		}
		res = &Result{Row: *period}
		return nil
	})
	if errors.Is(err, errDuplicateStart) {
		return s.existingStart(ctx, userID, start, refuseStartOverlap)
	}
	return res, err
}

// End is POST /cycle/period/end: closes the ongoing period on endDay (nil = today).
func (s *Service) End(ctx context.Context, userID uint64, endDay *civildate.Date, now time.Time) (*Result, error) {
	var res *Result
	err := s.tx(ctx, userID, func(q *store.Queries, _ *profilestore.Queries) error {
		period, err := optional(q.GetOngoingPeriod(ctx, userID))
		if err != nil {
			return err
		}
		if period == nil {
			return &Refusal{Status: 422, Kind: refuseNoOngoing}
		}
		end := civildate.InTehran(now)
		if endDay != nil {
			end = *endDay
		}
		if end.Before(period.PeriodStartDate) {
			return &Refusal{Status: 422, Kind: refuseEndBeforeStart}
		}
		bleeding := period.PeriodStartDate.DiffDays(end) + 1
		next := *period
		next.PeriodEndDate = civildate.NullDate{Date: end, Valid: true}
		next.BleedingLength = nullInt(&bleeding)
		next.DataQualityFlags = flagsJSON(QualityFlags(intPtr(period.CycleLength), &bleeding))
		if next, err = saveDirty(ctx, q, *period, next, now); err != nil {
			return err
		}
		res = &Result{Row: next}
		return nil
	})
	return res, err
}

// Store is POST /cycle/period: a whole range (end nil = open), reusing the same-day row.
func (s *Service) Store(ctx context.Context, userID uint64, start civildate.Date, end *civildate.Date, now time.Time) (*Result, error) {
	var res *Result
	err := s.tx(ctx, userID, func(q *store.Queries, pq *profilestore.Queries) error {
		period, err := optional(q.GetPeriodByStart(ctx, store.GetPeriodByStartParams{UserID: userID, PeriodStartDate: start}))
		if err != nil {
			return err
		}
		overlap, err := overlapsExisting(ctx, q, userID, start, orStart(end, start), idOf(period))
		if err != nil {
			return err
		}
		if overlap {
			return &Refusal{Status: 422, Code: "period_overlap", Kind: refuseRangeOverlap}
		}

		apply := func(r store.CycleHistory) store.CycleHistory {
			r.PeriodStartDate = start
			r.PeriodEndDate, r.BleedingLength = rangeEnd(start, end)
			r.IsConfirmed, r.IsEstimated = true, false
			r.Source = string(enums.DataSourceUserLogged)
			return r
		}
		var id uint64
		if period != nil {
			if _, err := saveDirty(ctx, q, *period, apply(*period), now); err != nil {
				return err
			}
			id = period.ID
		} else {
			row := apply(store.CycleHistory{UserID: userID, CreatedAt: dbTime(now), UpdatedAt: dbTime(now)})
			if id, err = insert(ctx, q, row); err != nil {
				return err
			}
		}
		if err := q.DeleteEstimatesExcept(ctx, store.DeleteEstimatesExceptParams{UserID: userID, ID: id}); err != nil {
			return fmt.Errorf("periods: delete estimates: %w", err)
		}
		row, err := s.reconcile(ctx, q, pq, userID, id, now)
		if err != nil {
			return err
		}
		res = &Result{Row: row}
		return nil
	})
	if errors.Is(err, errDuplicateStart) {
		return s.existingStart(ctx, userID, start, refuseRangeOverlap)
	}
	return res, err
}

// Update is PUT /cycle/period/{id}.
func (s *Service) Update(ctx context.Context, userID, id uint64, start civildate.Date, end *civildate.Date, now time.Time) (*Result, error) {
	var res *Result
	err := s.tx(ctx, userID, func(q *store.Queries, pq *profilestore.Queries) error {
		record, err := optional(q.GetUserPeriod(ctx, store.GetUserPeriodParams{UserID: userID, ID: id}))
		if err != nil {
			return err
		}
		if record == nil {
			return &Refusal{Status: 404, Kind: refuseNotFound}
		}
		overlap, err := overlapsExisting(ctx, q, userID, start, orStart(end, start), &record.ID)
		if err != nil {
			return err
		}
		if overlap {
			return &Refusal{Status: 422, Code: "period_overlap", Kind: refuseRangeOverlap}
		}
		next := *record
		next.PeriodStartDate = start
		next.PeriodEndDate, next.BleedingLength = rangeEnd(start, end)
		// Engine v6 (gap 02 S5, DECISIONS.md I-02, D-124): an edited row is the user's own data — a bleeding-log
		// period (or a leftover estimate / seed) becomes a confirmed user_logged row the bridge never rebuilds.
		next.IsConfirmed, next.IsEstimated = true, false
		next.Source = string(enums.DataSourceUserLogged)
		if _, err := saveDirty(ctx, q, *record, next, now); err != nil {
			return err
		}
		row, err := s.reconcile(ctx, q, pq, userID, record.ID, now)
		if err != nil {
			return err
		}
		res = &Result{Row: row}
		return nil
	})
	return res, err
}

// Destroy is DELETE /cycle/period/{id}.
func (s *Service) Destroy(ctx context.Context, userID, id uint64, now time.Time) error {
	return s.tx(ctx, userID, func(q *store.Queries, pq *profilestore.Queries) error {
		record, err := optional(q.GetUserPeriod(ctx, store.GetUserPeriodParams{UserID: userID, ID: id}))
		if err != nil {
			return err
		}
		if record == nil {
			return &Refusal{Status: 404, Kind: refuseNotFound}
		}
		if err := q.DeletePeriod(ctx, record.ID); err != nil {
			return fmt.Errorf("periods: delete: %w", err)
		}
		if err := RecomputeCycleLengths(ctx, q, userID, now); err != nil {
			return err
		}
		return Reanchor(ctx, q, pq, userID, now)
	})
}

// reconcile is the tail of store() / update(): RecomputeCycleLengths, Reanchor, then
// `$period->refresh()` and the quality flags of the re-anchored lengths.
func (s *Service) reconcile(ctx context.Context, q *store.Queries, pq *profilestore.Queries, userID, id uint64, now time.Time) (store.CycleHistory, error) {
	if err := RecomputeCycleLengths(ctx, q, userID, now); err != nil {
		return store.CycleHistory{}, err
	}
	if err := Reanchor(ctx, q, pq, userID, now); err != nil {
		return store.CycleHistory{}, err
	}
	row, err := q.GetUserPeriod(ctx, store.GetUserPeriodParams{UserID: userID, ID: id})
	if err != nil {
		return store.CycleHistory{}, fmt.Errorf("periods: refresh: %w", err)
	}
	next := row
	next.DataQualityFlags = flagsJSON(QualityFlags(intPtr(row.CycleLength), intPtr(row.BleedingLength)))
	return saveDirty(ctx, q, row, next, now)
}

// RecomputeCycleLengths: every row's cycle_length = days since the previous start (null first). Shared with the
// bleeding-log bridge (internal/healthlog, gap 02 B5): every period write of either writer runs it on q (the
// caller's transaction, after LockUser).
func RecomputeCycleLengths(ctx context.Context, q *store.Queries, userID uint64, now time.Time) error {
	rows, err := q.ListPeriodsOldestFirst(ctx, userID)
	if err != nil {
		return fmt.Errorf("periods: recompute: %w", err)
	}
	var previous *civildate.Date
	for _, r := range rows {
		var length *int
		if previous != nil {
			n := previous.DiffDays(r.PeriodStartDate)
			length = &n
		}
		next := r
		next.CycleLength = nullInt(length)
		if _, err := saveDirty(ctx, q, r, next, now); err != nil {
			return err
		}
		start := r.PeriodStartDate
		previous = &start
	}
	return nil
}

// Reanchor: profile LMP = the latest period start (null without periods), then markRecalculated. Shared with the
// bleeding-log bridge (internal/healthlog).
func Reanchor(ctx context.Context, q *store.Queries, pq *profilestore.Queries, userID uint64, now time.Time) error {
	prof, err := optional(q.GetProfileByUserID(ctx, userID))
	if err != nil || prof == nil {
		return err
	}
	latest, err := optional(q.GetLatestPeriod(ctx, userID))
	if err != nil {
		return err
	}
	var lmp civildate.NullDate
	if latest != nil {
		lmp = civildate.NullDate{Date: latest.PeriodStartDate, Valid: true}
	}
	return setLastPeriodStart(ctx, q, pq, prof, lmp, now)
}

// setLastPeriodStart is `$profile->update(['last_period_start' => …])` (only when dirty) followed
// by `$profile->markRecalculated()`.
func setLastPeriodStart(ctx context.Context, q *store.Queries, pq *profilestore.Queries, prof *store.UserProfile, lmp civildate.NullDate, now time.Time) error {
	if prof.LastPeriodStart != lmp {
		if err := q.UpdateProfileLastPeriodStart(ctx, store.UpdateProfileLastPeriodStartParams{
			LastPeriodStart: lmp, UpdatedAt: dbTime(now), ID: prof.ID,
		}); err != nil {
			return fmt.Errorf("periods: update lmp: %w", err)
		}
	}
	return profile.MarkRecalculated(ctx, pq, prof.ID, now)
}

// overlapsExistingPeriod: [start, end] intersects another non-estimated period (an open one
// counts as its start day only).
func overlapsExisting(ctx context.Context, q *store.Queries, userID uint64, start, end civildate.Date, exclude *uint64) (bool, error) {
	rows, err := q.ListLoggedPeriods(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("periods: overlap: %w", err)
	}
	for _, o := range rows {
		if exclude != nil && o.ID == *exclude {
			continue
		}
		oEnd := o.PeriodStartDate
		if o.PeriodEndDate.Valid {
			oEnd = o.PeriodEndDate.Date
		}
		if !start.After(oEnd) && !o.PeriodStartDate.After(end) {
			return true, nil
		}
	}
	return false, nil
}

// saveDirty is Eloquent's `$model->update($attributes)`: nothing is written unless an attribute
// changed by the model's comparison rules (dates by day, numbers by value, the JSON flags by
// decoded value); then the row and updated_at = now are written.
func saveDirty(ctx context.Context, q *store.Queries, orig, next store.CycleHistory, now time.Time) (store.CycleHistory, error) {
	if !dirty(orig, next) {
		return orig, nil
	}
	next.UpdatedAt = dbTime(now)
	if err := q.UpdatePeriod(ctx, store.UpdatePeriodParams{
		PeriodStartDate: next.PeriodStartDate, PeriodEndDate: next.PeriodEndDate,
		CycleLength: next.CycleLength, BleedingLength: next.BleedingLength,
		IsConfirmed: next.IsConfirmed, IsEstimated: next.IsEstimated, Source: next.Source,
		DataQualityFlags: next.DataQualityFlags, UpdatedAt: next.UpdatedAt, ID: next.ID,
	}); err != nil {
		return store.CycleHistory{}, fmt.Errorf("periods: update: %w", err)
	}
	return next, nil
}

func dirty(a, b store.CycleHistory) bool {
	return a.PeriodStartDate != b.PeriodStartDate || a.PeriodEndDate != b.PeriodEndDate ||
		a.CycleLength != b.CycleLength || a.BleedingLength != b.BleedingLength ||
		a.IsConfirmed != b.IsConfirmed || a.IsEstimated != b.IsEstimated || a.Source != b.Source ||
		!sameJSON(a.DataQualityFlags, b.DataQualityFlags)
}

func sameJSON(a, b db.NullRawJSON) bool {
	if a.Valid != b.Valid {
		return false
	}
	if !a.Valid {
		return true
	}
	var x, y any
	if json.Unmarshal(a.V, &x) != nil || json.Unmarshal(b.V, &y) != nil {
		return string(a.V) == string(b.V)
	}
	return reflect.DeepEqual(x, y)
}

func insert(ctx context.Context, q *store.Queries, r store.CycleHistory) (uint64, error) {
	id, err := q.InsertPeriod(ctx, store.InsertPeriodParams{
		UserID: r.UserID, PeriodStartDate: r.PeriodStartDate, PeriodEndDate: r.PeriodEndDate,
		CycleLength: r.CycleLength, BleedingLength: r.BleedingLength, IsConfirmed: r.IsConfirmed,
		IsEstimated: r.IsEstimated, Source: r.Source, DataQualityFlags: r.DataQualityFlags,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	})
	if isDuplicateKey(err) {
		return 0, errDuplicateStart
	}
	if err != nil {
		return 0, fmt.Errorf("periods: insert: %w", err)
	}
	return uint64(id), nil //nolint:gosec // G115: LAST_INSERT_ID is positive
}

// errDuplicateStart is MySQL 1062 on the period insert: a row with the same (user_id,
// period_start_date) appeared after the reads (a writer that did not take LockUser; since T-M9-28 the bleeding-log
// bridge takes it too). The transaction is rolled back and the caller answers from the committed row (D-131).
var errDuplicateStart = errors.New("periods: duplicate period start")

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// existingStart answers a duplicate-start insert: the committed same-day period (idempotent 200),
// or the overlap refusal when it cannot be read back. Never a 500.
func (s *Service) existingStart(ctx context.Context, userID uint64, start civildate.Date, kind refusalKind) (*Result, error) {
	row, err := optional(s.q.GetPeriodByStart(ctx, store.GetPeriodByStartParams{UserID: userID, PeriodStartDate: start}))
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, &Refusal{Status: 422, Code: "period_overlap", Kind: kind}
	}
	return &Result{Row: *row}, nil
}

// tx runs fn in one transaction that first takes the per-user lock (`SELECT id FROM users WHERE
// id = ? FOR UPDATE`, D-131) and bumps the engine revision before the commit (T-M9-30): every overlap / blocker check and write of one user is serialised,
// so two concurrent writes (double tap, two devices, offline replay) can never both pass the
// overlap check. The lock is a locking read, so the REPEATABLE READ snapshot is only taken by the
// first plain read after it and sees whatever the previous holder committed.
func (s *Service) tx(ctx context.Context, userID uint64, fn func(q *store.Queries, pq *profilestore.Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("periods: begin: %w", err)
	}
	if err := LockUser(ctx, tx, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := fn(s.q.WithTx(tx), profilestore.New(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	// T-M9-30: every period write is an engine-input write — the revision moves in the same transaction.
	if err := revision.Bump(ctx, tx, userID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("periods: commit: %w", err)
	}
	return nil
}

// LockUser takes the per-user period lock (D-131) on tx: `SELECT id FROM users WHERE id = ? FOR UPDATE`. Every
// cycle_histories writer — the period button (Service.tx) and the bleeding-log bridge (internal/healthlog) — takes
// it first in its transaction, so their overlap checks and writes of one user never interleave. Re-taking it in the
// same transaction is a no-op.
func LockUser(ctx context.Context, tx filestore.DBTX, userID uint64) error {
	if _, err := filestore.New(tx).LockOwner(ctx, userID); err != nil {
		return fmt.Errorf("periods: lock user: %w", err)
	}
	return nil
}

// optional turns sql.ErrNoRows into a nil row.
func optional[T any](row T, err error) (*T, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("periods: query: %w", err)
	}
	return &row, nil
}

func idOf(r *store.CycleHistory) *uint64 {
	if r == nil {
		return nil
	}
	return &r.ID
}

func orStart(end *civildate.Date, start civildate.Date) civildate.Date {
	if end != nil {
		return *end
	}
	return start
}

// rangeEnd is the period_end_date / bleeding_length pair of a start..end range (end nil = open).
func rangeEnd(start civildate.Date, end *civildate.Date) (civildate.NullDate, sql.NullInt32) {
	if end == nil {
		return civildate.NullDate{}, sql.NullInt32{}
	}
	n := start.DiffDays(*end) + 1
	return civildate.NullDate{Date: *end, Valid: true}, nullInt(&n)
}

func nullInt(v *int) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(*v), Valid: true} //nolint:gosec // G115: day counts
}

func intPtr(v sql.NullInt32) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

// flagsJSON is the `array` cast's json_encode of a list of flag strings.
func flagsJSON(flags []string) db.NullRawJSON {
	b, _ := json.Marshal(flags) //nolint:errchkjson // []string always encodes
	return db.NullRawJSON{V: b, Valid: true}
}

// dbTime is a Carbon value as Eloquent writes it: 'Y-m-d H:i:s' in Asia/Tehran.
func dbTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.In(civildate.Tehran).Truncate(time.Second), Valid: true}
}
