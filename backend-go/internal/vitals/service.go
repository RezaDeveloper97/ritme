package vitals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	profilestore "github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/vitals/store"
)

// ErrNotFound: no such reading of this user (a foreign id is the same 404).
var ErrNotFound = errors.New("vitals: reading not found")

// MaxListReadings caps one list / report read (a 366-day range of several readings a day fits).
const MaxListReadings = 2000

// LatestLogLookbackDays: the hub's latest card falls back to a log-sheet value of the last 90 days.
const LatestLogLookbackDays = 90

// Service is the vitals data access. Every query is scoped by user id.
type Service struct {
	db    *sql.DB
	q     *store.Queries
	prefs notifications.Getter
}

// NewService wires the service on db.
func NewService(db *sql.DB) *Service {
	return &Service{db: db, q: store.New(db), prefs: profilestore.New(db)}
}

// Input is a validated reading.
type Input struct {
	Type       string
	MeasuredAt time.Time
	Systolic   int
	Diastolic  int
	Pulse      int // BP pulse (0 = none) or heart rate
	Arm        string
	Position   string
	MgDl       float64
	Unit       string
	Context    string
	Method     string
	Note       string
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func nullI16(n int) sql.NullInt16 {
	return sql.NullInt16{Int16: int16(n), Valid: n > 0} //nolint:gosec // validated ≤ 300
}

func dbTime(t time.Time) time.Time { return t.In(civildate.Tehran).Truncate(time.Second) }

// columns maps an input to the type's columns (the other types' columns stay NULL).
func (in Input) columns() (sys, dia, pulse sql.NullInt16, arm, pos, mgdl, unit, ctx, method sql.NullString) {
	switch in.Type {
	case TypeBP:
		sys, dia, pulse = nullI16(in.Systolic), nullI16(in.Diastolic), nullI16(in.Pulse)
		arm, pos = nullStr(in.Arm), nullStr(in.Position)
	case TypeGlucose:
		mgdl = sql.NullString{String: strconv.FormatFloat(in.MgDl, 'f', 1, 64), Valid: true}
		unit, ctx, method = nullStr(in.Unit), nullStr(in.Context), nullStr(in.Method)
	case TypeHR:
		pulse, ctx = nullI16(in.Pulse), nullStr(in.Context)
	}
	return
}

// Create stores a reading.
func (s *Service) Create(ctx context.Context, userID uint64, in Input, now time.Time) (Reading, error) {
	sys, dia, pulse, arm, pos, mgdl, unit, cx, method := in.columns()
	id, err := s.q.InsertReading(ctx, store.InsertReadingParams{
		UserID: userID, Type: in.Type, MeasuredAt: dbTime(in.MeasuredAt), Systolic: sys, Diastolic: dia, Pulse: pulse,
		Arm: arm, Position: pos, GlucoseMgDl: mgdl, GlucoseUnit: unit, Context: cx, Method: method,
		Note: nullStr(in.Note), Now: sql.NullTime{Time: dbTime(now), Valid: true},
	})
	if err != nil {
		return Reading{}, fmt.Errorf("vitals: insert: %w", err)
	}
	return s.Get(ctx, userID, uint64(id)) //nolint:gosec // auto-increment id
}

// Get is one reading of the user.
func (s *Service) Get(ctx context.Context, userID, id uint64) (Reading, error) {
	row, err := s.q.GetReading(ctx, store.GetReadingParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Reading{}, ErrNotFound
	}
	if err != nil {
		return Reading{}, fmt.Errorf("vitals: get: %w", err)
	}
	return FromRow(row), nil
}

// Update replaces a reading of the user.
func (s *Service) Update(ctx context.Context, userID, id uint64, in Input, now time.Time) (Reading, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Reading{}, err
	}
	sys, dia, pulse, arm, pos, mgdl, unit, cx, method := in.columns()
	if _, err := s.q.UpdateReading(ctx, store.UpdateReadingParams{
		ID: id, UserID: userID, Type: in.Type, MeasuredAt: dbTime(in.MeasuredAt), Systolic: sys, Diastolic: dia,
		Pulse: pulse, Arm: arm, Position: pos, GlucoseMgDl: mgdl, GlucoseUnit: unit, Context: cx, Method: method,
		Note: nullStr(in.Note), Now: sql.NullTime{Time: dbTime(now), Valid: true},
	}); err != nil {
		return Reading{}, fmt.Errorf("vitals: update: %w", err)
	}
	return s.Get(ctx, userID, id)
}

// Delete removes a reading of the user.
func (s *Service) Delete(ctx context.Context, userID, id uint64) error {
	n, err := s.q.DeleteReading(ctx, store.DeleteReadingParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("vitals: delete: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Timed is the user's timed readings of [from, to] (days), newest first; typ "" = all types.
func (s *Service) Timed(ctx context.Context, userID uint64, typ string, from, to civildate.Date) ([]Reading, error) {
	rows, err := s.q.ListReadings(ctx, store.ListReadingsParams{
		UserID: userID, Type: typ, DateFrom: from.TehranMidnight(), DateTo: to.AddDays(1).TehranMidnight(),
		Limit: MaxListReadings,
	})
	if err != nil {
		return nil, fmt.Errorf("vitals: list: %w", err)
	}
	out := make([]Reading, 0, len(rows))
	for _, r := range rows {
		out = append(out, FromRow(r))
	}
	return out, nil
}

// Merged is Timed plus the log-sheet day values of [from, to] (merge.go), newest first.
func (s *Service) Merged(ctx context.Context, userID uint64, typ string, from, to civildate.Date) ([]Reading, error) {
	timed, err := s.Timed(ctx, userID, "", from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListLogMeasurements(ctx, store.ListLogMeasurementsParams{UserID: userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("vitals: log measurements: %w", err)
	}
	return OfType(Merge(timed, rows), typ), nil
}

// Latest is the newest reading of each type (a timed one, or a log-sheet value of the last 90 days when newer).
func (s *Service) Latest(ctx context.Context, userID uint64, today civildate.Date) (map[string]*Reading, error) {
	recent, err := s.Merged(ctx, userID, "", today.AddDays(-LatestLogLookbackDays), today)
	if err != nil {
		return nil, err
	}
	out := map[string]*Reading{}
	for i := range recent {
		if out[recent[i].Type] == nil {
			out[recent[i].Type] = &recent[i]
		}
	}
	for _, typ := range Types {
		if out[typ] != nil {
			continue
		}
		row, err := s.q.LatestReading(ctx, store.LatestReadingParams{UserID: userID, Type: typ, Before: today.AddDays(1).TehranMidnight()})
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("vitals: latest: %w", err)
		}
		r := FromRow(row)
		out[typ] = &r
	}
	return out, nil
}

// Plan is the user's plan.
func (s *Service) Plan(ctx context.Context, userID uint64) ([]PlanItem, error) {
	rows, err := s.q.ListPlanItems(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("vitals: plan: %w", err)
	}
	return PlanFromRows(rows), nil
}

// SavePlan replaces the user's plan.
func (s *Service) SavePlan(ctx context.Context, userID uint64, items []PlanItem, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vitals: begin: %w", err)
	}
	q := s.q.WithTx(tx)
	err = q.DeletePlanItems(ctx, userID)
	for i, it := range items {
		if err != nil {
			break
		}
		err = q.InsertPlanItem(ctx, store.InsertPlanItemParams{
			UserID: userID, Type: it.Type, Slot: it.Slot, Days: it.Days, RemindAt: nullStr(it.RemindAt),
			SortOrder: uint8(i), //nolint:gosec // ≤ MaxPlanItems
			Now:       sql.NullTime{Time: dbTime(now), Valid: true},
		})
	}
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("vitals: save plan: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vitals: commit: %w", err)
	}
	return nil
}

// AlertCopy is the live urgent copy.
func (s *Service) AlertCopy(ctx context.Context) (AlertCopy, error) { return LoadAlertCopy(ctx, s.q) }

// Notifications is the user's notification settings (the plan's reminder switch).
func (s *Service) Notifications(ctx context.Context, userID uint64) (notifications.Preferences, error) {
	p, err := notifications.Load(ctx, s.prefs, userID)
	if err != nil {
		return p, fmt.Errorf("vitals: notification settings: %w", err)
	}
	return p, nil
}
