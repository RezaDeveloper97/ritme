// Package healthlog is DailyHealthLogController (GET/POST /health-logs, GET/DELETE
// /health-logs/{date}, GET /health-logs/enums) and the live half of CycleHistoryService
// (the spotting warning and the period-start reconciliation run by POST /health-logs).
package healthlog

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/healthlog/model"
	"github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// PerPage is the index paginator size ($query->paginate(30)).
const PerPage = 30

// Service holds the DB access of the domain.
type Service struct {
	q  *store.Queries
	db *sql.DB // nil when the Service already runs inside a transaction
}

// NewService returns a Service on db (a *sql.DB or a transaction).
func NewService(db store.DBTX) *Service {
	s := &Service{q: store.New(db)}
	if pool, ok := db.(*sql.DB); ok {
		s.db = pool
	}
	return s
}

// inTx runs fn on a Service bound to one READ COMMITTED transaction (so createOrFirst's re-read sees a
// concurrent insert, as it does without a transaction), or on s itself when s is already transactional.
func (s *Service) inTx(ctx context.Context, fn func(t *Service) error) error {
	if s.db == nil {
		return fn(s)
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("healthlog: begin: %w", err)
	}
	if err := fn(&Service{q: store.New(tx)}); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("healthlog: commit: %w", err)
	}
	return nil
}

// StoreResult is the outcome of DailyHealthLog::updateOrCreate plus its side effects.
type StoreResult struct {
	Log     *model.DailyHealthLog
	Created bool     // wasRecentlyCreated → 201
	Keys    []string // attribute keys a freshly created model serialises (nil when updated)
	Warning *jsonx.OrderedMap
}

// Data is the response `data`: the created model's own attributes, or the full row.
func (r *StoreResult) Data() *jsonx.OrderedMap {
	if r.Created {
		return r.Log.Attributes(r.Keys)
	}
	return r.Log.ToArray()
}

// Store is DailyHealthLogController::store after validation: updateOrCreate on
// (user_id, log_date), then getSpottingWarning, checkAndUpdatePeriodStart and
// markRecalculatedIfNeeded — in that order, as Laravel runs them. The columns written are then
// projected into the log taxonomy v2 entries (B-N3-01) in the same transaction, so the v2 day keeps
// matching the legacy row until the client switches (B-N3-03).
func (s *Service) Store(ctx context.Context, userID uint64, attrs phpval.Map, locale string, now time.Time) (*StoreResult, error) {
	var res *StoreResult
	err := s.inTx(ctx, func(t *Service) error {
		r, err := t.store(ctx, userID, attrs, locale, now)
		if err != nil {
			return err
		}
		columns := []string{}
		for _, k := range attrs.Keys() {
			if k != "log_date" {
				columns = append(columns, k)
			}
		}
		if err := t.syncFromLegacy(ctx, r.Log, columns, dbNow(now)); err != nil {
			return err
		}
		res = r
		return nil
	})
	return res, err
}

func (s *Service) store(ctx context.Context, userID uint64, attrs phpval.Map, locale string, now time.Time) (*StoreResult, error) {
	now = dbNow(now)
	rawDate, _ := attrs.Get("log_date")
	t, err := civildate.ParseLenient(phpval.ToString(rawDate), now, civildate.Tehran)
	if err != nil {
		return nil, fmt.Errorf("healthlog: log_date: %w", err)
	}
	logDate := civildate.FromTime(t)

	res, err := s.upsert(ctx, userID, logDate, attrs, now)
	if err != nil {
		return nil, err
	}

	profile, err := s.profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	res.Warning = SpottingWarning(profile, res.Log, locale)
	if err := s.checkAndUpdatePeriodStart(ctx, userID, res.Log, profile, now); err != nil {
		return nil, err
	}
	// markRecalculatedIfNeeded: the log is an engine input.
	if profile != nil && profile.LastPeriodStart.Valid {
		if err := s.q.MarkProfileRecalculated(ctx, store.MarkProfileRecalculatedParams{
			Now: sql.NullTime{Time: now, Valid: true}, ID: profile.ID,
		}); err != nil {
			return nil, fmt.Errorf("healthlog: mark recalculated: %w", err)
		}
	}
	return res, nil
}

func (s *Service) upsert(ctx context.Context, userID uint64, logDate civildate.Date, attrs phpval.Map, now time.Time) (*StoreResult, error) {
	existing, err := s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: logDate})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, err := s.create(ctx, userID, logDate, attrs, now)
		if !isDuplicate(err) {
			return res, err
		}
		// createOrFirst: a concurrent insert won → update that row instead.
		existing, err = s.q.GetDailyHealthLogOn(ctx, store.GetDailyHealthLogOnParams{UserID: userID, LogDate: logDate})
		if err != nil {
			return nil, fmt.Errorf("healthlog: reload after duplicate: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("healthlog: find log: %w", err)
	}
	return s.update(ctx, existing, attrs, now)
}

func (s *Service) create(ctx context.Context, userID uint64, logDate civildate.Date, attrs phpval.Map, now time.Time) (*StoreResult, error) {
	l := model.FromRow(store.DailyHealthLog{UserID: userID, LogDate: logDate})
	keys := []string{"user_id", "log_date"}
	for _, k := range attrs.Keys() {
		if k == "log_date" {
			continue
		}
		v, _ := attrs.Get(k)
		if err := l.Set(k, v); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	ts := sql.NullTime{Time: now, Valid: true}
	l.Row.CreatedAt, l.Row.UpdatedAt = ts, ts
	var p store.InsertDailyHealthLogParams
	copyFields(&p, l.Row)
	id, err := s.q.InsertDailyHealthLog(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("healthlog: insert: %w", err)
	}
	l.Row.ID = uint64(id) //nolint:gosec // G115: AUTO_INCREMENT ids are positive
	return &StoreResult{Log: l, Created: true, Keys: append(keys, "updated_at", "created_at", "id")}, nil
}

func (s *Service) update(ctx context.Context, row store.DailyHealthLog, attrs phpval.Map, now time.Time) (*StoreResult, error) {
	before := model.FromRow(row)
	after := model.FromRow(row)
	dirty := false
	for _, k := range attrs.Keys() {
		if k == "log_date" { // the row was found by this date
			continue
		}
		v, _ := attrs.Get(k)
		if err := after.Set(k, v); err != nil {
			return nil, err
		}
		if !dirty && !sameValue(before, after, k) {
			dirty = true
		}
	}
	if dirty { // save() writes nothing (and keeps updated_at) when no attribute changed
		after.Row.UpdatedAt = sql.NullTime{Time: now, Valid: true}
		var p store.UpdateDailyHealthLogParams
		copyFields(&p, after.Row)
		if err := s.q.UpdateDailyHealthLog(ctx, p); err != nil {
			return nil, fmt.Errorf("healthlog: update: %w", err)
		}
	}
	fresh, err := s.q.GetDailyHealthLog(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("healthlog: reload: %w", err)
	}
	return &StoreResult{Log: model.FromRow(fresh)}, nil
}

// sameValue is Model::originalIsEquivalent on the cast values (strict for strings,
// cast-then-compare for booleans, integers, decimals and arrays).
func sameValue(a, b *model.DailyHealthLog, key string) bool {
	va, _ := a.Value(key)
	vb, _ := b.Value(key)
	ja, errA := jsonx.Marshal(va, 0)
	jb, errB := jsonx.Marshal(vb, 0)
	return errA == nil && errB == nil && bytes.Equal(ja, jb)
}

// profile is $user->profile (nil when the user has none).
func (s *Service) profile(ctx context.Context, userID uint64) (*store.UserProfile, error) {
	p, err := s.q.GetUserProfile(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("healthlog: load profile: %w", err)
	}
	return &p, nil
}

// Find is $user->dailyHealthLogs()->whereDate('log_date', $date)->first() (nil when none).
func (s *Service) Find(ctx context.Context, userID uint64, date string) (*model.DailyHealthLog, error) {
	row, err := s.q.GetDailyHealthLogByDateString(ctx, store.GetDailyHealthLogByDateStringParams{UserID: userID, Date: date})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("healthlog: find by date: %w", err)
	}
	return model.FromRow(row), nil
}

// Delete removes the log and the v2 entries its columns projected (entries only the v2 sheet can
// write — custom items, pain scores… — stay).
func (s *Service) Delete(ctx context.Context, l *model.DailyHealthLog) error {
	return s.inTx(ctx, func(t *Service) error {
		if err := t.q.DeleteDailyHealthLog(ctx, l.Row.ID); err != nil {
			return fmt.Errorf("healthlog: delete: %w", err)
		}
		return t.deleteLegacySlots(ctx, l.Row.UserID, l.Row.LogDate, nil)
	})
}

// Filter is the index's optional whereDate bounds: Has* mirrors $request->has(), the value
// is the raw request string (nil = present but null/not a string → matches nothing).
type Filter struct {
	HasFrom, HasTo bool
	From, To       any
}

// List is one page of the index ($query->paginate(30)): the rows and the total.
func (s *Service) List(ctx context.Context, userID uint64, f Filter, page int) ([]*model.DailyHealthLog, int, error) {
	total, err := s.q.CountDailyHealthLogs(ctx, store.CountDailyHealthLogsParams{
		UserID: userID, HasFrom: f.HasFrom, FromDate: f.From, HasTo: f.HasTo, ToDate: f.To,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("healthlog: count: %w", err)
	}
	out := []*model.DailyHealthLog{}
	offset := (page - 1) * PerPage
	if total == 0 || int64(offset) >= total {
		return out, int(total), nil
	}
	rows, err := s.q.ListDailyHealthLogs(ctx, store.ListDailyHealthLogsParams{
		UserID: userID, HasFrom: f.HasFrom, FromDate: f.From, HasTo: f.HasTo, ToDate: f.To,
		Limit: PerPage, Offset: int32(offset), //nolint:gosec // G115: page offsets fit int32
	})
	if err != nil {
		return nil, 0, fmt.Errorf("healthlog: list: %w", err)
	}
	for _, r := range rows {
		out = append(out, model.FromRow(r))
	}
	return out, int(total), nil
}

// dbNow is freshTimestamp() as it reaches the DB: Tehran wall-clock, whole seconds.
func dbNow(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

// copyFields copies every field of dst (a pointer to a sqlc params struct) from the
// same-named field of src (a row struct); the generated structs share names and types.
func copyFields(dst any, src any) {
	d := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src)
	for i := range d.NumField() {
		name := d.Type().Field(i).Name
		f := sv.FieldByName(name)
		if !f.IsValid() {
			panic("healthlog: params field " + name + " has no row field")
		}
		d.Field(i).Set(f)
	}
}

func isDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
