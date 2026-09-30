package fertility

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ritme/backend-go/internal/cycle/metrics"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/resolver"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/fertility/store"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// DB is the handle the service reads through and opens its write transaction on (*sql.DB).
type DB interface {
	store.DBTX
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Service reads and writes fertility days.
type Service struct {
	db DB
}

// NewService returns a Service on db.
func NewService(db DB) *Service { return &Service{db: db} }

// Load is the merged day of date plus its cycle day and chance, in three queries: the profile,
// the cycle history (the cycle engine's own inputs, read-only reuse of internal/cycle) and the
// merged day. today separates past and predicted cycles, as in the cycle view.
func (s *Service) Load(ctx context.Context, userID uint64, date, today civildate.Date) (Day, Context, error) {
	in, err := s.cycleInputs(ctx, userID)
	if err != nil {
		return Day{}, Context{}, err
	}
	row, err := store.New(s.db).GetMergedDay(ctx, store.GetMergedDayParams{UserID: userID, LogDate: date})
	if err != nil {
		return Day{}, Context{}, fmt.Errorf("fertility: load day: %w", err)
	}
	return DayFromRow(date, row), ContextOf(in.resolve(date, today)), nil
}

// cycleInputs are the cycle engine's inputs for a user (read-only reuse of internal/cycle).
type cycleInputs struct {
	histories []model.History
	profile   *model.Profile
	metrics   metrics.Metrics
}

// resolve is the cycle engine's status of date.
func (in cycleInputs) resolve(date, today civildate.Date) resolver.Status {
	return resolver.Resolve(in.histories, in.profile, date, today, in.metrics)
}

// cycleInputs reads the profile and the cycle history (two queries).
func (s *Service) cycleInputs(ctx context.Context, userID uint64) (cycleInputs, error) {
	cq := cyclestore.New(s.db)
	var profile *cyclestore.UserProfile
	manual := false
	p, err := cq.GetEngineProfileByUserID(ctx, userID) // B-N1-09: + «خودکار از داده‌ها»
	switch {
	case err == nil:
		profile = &p.UserProfile
		manual = !p.LengthsAuto
	case !errors.Is(err, sql.ErrNoRows):
		return cycleInputs{}, fmt.Errorf("fertility: load profile: %w", err)
	}
	rows, err := cq.ListCycleHistoriesNewestFirst(ctx, userID)
	if err != nil {
		return cycleInputs{}, fmt.Errorf("fertility: load histories: %w", err)
	}
	histories := cycleservice.HistoriesFromRows(rows)
	engineProfile := cycleservice.ProfileFromRow(profile)
	if engineProfile != nil { // «خودکار از داده‌ها» off → the profile lengths win
		engineProfile.LengthsManual = manual
	}
	return cycleInputs{histories: histories, profile: engineProfile, metrics: metrics.Calculate(histories, engineProfile)}, nil
}

// Save writes a validated PUT in one transaction: lh / mucus / bbt_time into fertility_logs,
// bbt / intercourse / symptoms / note into daily_health_logs through the healthlog service
// (POST /health-logs' validation, upsert and side effects: period-start check, spotting
// warning, recalculation mark). Returns the health log's luteal-spotting warning, if any.
func (s *Service) Save(ctx context.Context, userID uint64, in DayInput, locale string, now time.Time) (*jsonx.OrderedMap, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fertility: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if in.sent("lh", "mucus", "bbt_time") {
		if err := saveFertilityLog(ctx, store.New(tx), userID, in, now); err != nil {
			return nil, err
		}
	}
	var warning *jsonx.OrderedMap
	if in.sent("bbt", "intercourse", "symptoms", "note") {
		if warning, err = saveHealthLog(ctx, healthlog.NewService(tx), userID, in, locale, now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("fertility: commit: %w", err)
	}
	return warning, nil
}

func nullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

// saveFertilityLog merges the sent keys over the day's row; a row left empty is deleted.
func saveFertilityLog(ctx context.Context, q *store.Queries, userID uint64, in DayInput, now time.Time) error {
	key := store.GetFertilityLogParams{UserID: userID, LogDate: in.Date}
	row, err := q.GetFertilityLog(ctx, key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("fertility: find log: %w", err)
	}
	if in.Set["lh"] {
		row.LhTest = nullString(in.LH)
	}
	if in.Set["mucus"] {
		row.CervicalMucus = nullString(in.Mucus)
	}
	if in.Set["bbt_time"] {
		row.BbtTime = nullString(in.BBTTime)
	}
	if !row.LhTest.Valid && !row.CervicalMucus.Valid && !row.BbtTime.Valid {
		if err := q.DeleteFertilityLog(ctx, store.DeleteFertilityLogParams(key)); err != nil {
			return fmt.Errorf("fertility: delete log: %w", err)
		}
		return nil
	}
	if err := q.UpsertFertilityLog(ctx, store.UpsertFertilityLogParams{
		UserID: userID, LogDate: in.Date, LhTest: row.LhTest, CervicalMucus: row.CervicalMucus,
		BbtTime: row.BbtTime, Now: sql.NullTime{Time: now.In(civildate.Tehran).Truncate(time.Second), Valid: true},
	}); err != nil {
		return fmt.Errorf("fertility: upsert log: %w", err)
	}
	return nil
}

// saveHealthLog maps the sent keys onto daily_health_logs columns and stores them exactly as
// POST /health-logs would (same validation, same service call).
func saveHealthLog(ctx context.Context, svc *healthlog.Service, userID uint64, in DayInput, locale string, now time.Time) (*jsonx.OrderedMap, error) {
	body := phpval.NewMap()
	body.Set("log_date", in.Date.String())
	if in.Set["bbt"] {
		body.Set("basal_body_temperature", in.BBT)
	}
	if in.Set["intercourse"] {
		body.Set("intercourse_type", ptrValue(in.Intercourse))
	}
	if in.Set["symptoms"] {
		existing, err := svc.Find(ctx, userID, in.Date.String())
		if err != nil {
			return nil, err
		}
		for _, sym := range []string{SymptomOvarianPain, SymptomBloating, SymptomBreastSensitivity} {
			col := symptomColumns[sym]
			var v any
			if slices.Contains(in.Symptoms, sym) {
				v = selectedIntensity
				if existing != nil {
					if cur := existing.Str(col); cur != nil && *cur != "" {
						v = *cur // keep a medium/high logged through the full health log
					}
				}
			}
			body.Set(col, v)
		}
		var spotting any
		if slices.Contains(in.Symptoms, SymptomSpotting) {
			spotting = true
		}
		body.Set("spotting", spotting)
	}
	if in.Set["note"] {
		body.Set("notes", ptrValue(in.Note))
	}

	attrs, err := healthlog.ValidateStore(body, locale, now)
	if err != nil {
		return nil, fmt.Errorf("fertility: health log rejected the mapped day: %w", err)
	}
	res, err := svc.Store(ctx, userID, attrs, locale, now)
	if err != nil {
		return nil, err
	}
	return res.Warning, nil
}

func ptrValue(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}
