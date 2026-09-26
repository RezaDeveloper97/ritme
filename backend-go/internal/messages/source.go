package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// StoreSource is the sqlc-backed manager.Source for one user and request. It memoises every
// read (Laravel's relation cache and HealthDataEngine's per-request caches) and builds the
// legacy engine once. The cycle day is computed with the committed cycle libraries directly
// (profile + cycle_histories → legacy.New), so it does not depend on the cycle service.
type StoreSource struct {
	q      store.Querier
	pq     pstore.Querier
	userID uint64
	today  civildate.Date

	profileRow    *store.GetMessageProfileRow
	profileLoaded bool
	preg          *pstore.PregnancyProfile
	pregLoaded    bool
	engine        *legacy.Engine
}

// NewStoreSource returns the source for userID; today is the request's Tehran day.
func NewStoreSource(q store.Querier, pq pstore.Querier, userID uint64, today civildate.Date) *StoreSource {
	return &StoreSource{q: q, pq: pq, userID: userID, today: today}
}

func (s *StoreSource) profile(ctx context.Context) (*store.GetMessageProfileRow, error) {
	if s.profileLoaded {
		return s.profileRow, nil
	}
	row, err := s.q.GetMessageProfile(ctx, s.userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("messages: load profile: %w", err)
	default:
		s.profileRow = &row
	}
	s.profileLoaded = true
	return s.profileRow, nil
}

// Profile implements manager.Source.
func (s *StoreSource) Profile(ctx context.Context) (*manager.Profile, error) {
	row, err := s.profile(ctx)
	if err != nil || row == nil {
		return nil, err
	}
	return &manager.Profile{
		UserGoal:           row.UserGoal,
		SubscriptionType:   row.SubscriptionType,
		HasLastPeriodStart: row.LastPeriodStart.Valid,
	}, nil
}

// PregnancyProfile implements manager.Source.
func (s *StoreSource) PregnancyProfile(ctx context.Context) (*pstore.PregnancyProfile, error) {
	if !s.pregLoaded {
		p, err := pregnancy.LoadProfile(ctx, s.pq, s.userID)
		if err != nil {
			return nil, err
		}
		s.preg, s.pregLoaded = p, true
	}
	return s.preg, nil
}

// logFromColumns is the loaded columns (manager.LoadedLogColumns) of one row.
func logFromColumns(energy, sleep sql.NullString) manager.Log {
	str := func(v sql.NullString) any {
		if !v.Valid {
			return nil
		}
		return v.String
	}
	return manager.NewLog(map[string]any{"energy_level": str(energy), "sleep_quality": str(sleep)})
}

// DailyLog implements manager.Source.
func (s *StoreSource) DailyLog(ctx context.Context, date civildate.Date) (*manager.Log, error) {
	row, err := s.q.GetMessageDailyLog(ctx, store.GetMessageDailyLogParams{UserID: s.userID, LogDate: date})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("messages: load daily log: %w", err)
	}
	l := logFromColumns(row.EnergyLevel, row.SleepQuality)
	return &l, nil
}

// RecentLogs implements manager.Source.
func (s *StoreSource) RecentLogs(ctx context.Context, from, to civildate.Date) ([]manager.Log, error) {
	rows, err := s.q.ListMessageRecentLogs(ctx, store.ListMessageRecentLogsParams{UserID: s.userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, fmt.Errorf("messages: load recent logs: %w", err)
	}
	out := make([]manager.Log, len(rows))
	for i, r := range rows {
		out[i] = logFromColumns(r.EnergyLevel, r.SleepQuality)
	}
	return out, nil
}

// Cycle implements manager.Source: HealthDataEngine::calculateForDate in calendar mode (the
// fields the manager reads do not depend on the content part).
func (s *StoreSource) Cycle(ctx context.Context, date civildate.Date) (legacy.Calculation, error) {
	if s.engine == nil {
		row, err := s.profile(ctx)
		if err != nil {
			return legacy.Calculation{}, err
		}
		rows, err := s.q.ListMessageCycleHistories(ctx, s.userID)
		if err != nil {
			return legacy.Calculation{}, fmt.Errorf("messages: load cycle histories: %w", err)
		}
		histories := make([]model.History, len(rows))
		for i, r := range rows {
			histories[i] = historyFromRow(r)
		}
		s.engine = legacy.New(legacy.Input{Profile: engineProfile(row), Histories: histories, Today: s.today})
	}
	return s.engine.CalculateForDate(ctx, date, false)
}

// engineProfile maps the profile row onto model.Profile (nil = no profile).
func engineProfile(r *store.GetMessageProfileRow) *model.Profile {
	if r == nil {
		return nil
	}
	p := &model.Profile{Goal: r.UserGoal}
	if r.LastPeriodStart.Valid {
		p.LastPeriodStart = r.LastPeriodStart.Date
	}
	if r.Birthday.Valid {
		p.Birthday = r.Birthday.Date
	}
	if r.CycleDuration.Valid {
		p.CycleDuration = model.Int(int(r.CycleDuration.Int16))
	}
	if r.PeriodDuration.Valid {
		p.PeriodDuration = model.Int(int(r.PeriodDuration.Int16))
	}
	return p
}

// historyFromRow maps a cycle_histories row onto model.History (CycleHistory casts).
func historyFromRow(r store.ListMessageCycleHistoriesRow) model.History {
	h := model.History{
		ID:          int64(r.ID), //nolint:gosec // G115: auto-increment ids fit int64
		PeriodStart: r.PeriodStartDate,
		IsConfirmed: r.IsConfirmed,
		IsEstimated: r.IsEstimated,
		Source:      r.Source,
	}
	if r.PeriodEndDate.Valid {
		h.PeriodEnd = r.PeriodEndDate.Date
	}
	if r.CycleLength.Valid {
		h.CycleLength = model.Int(int(r.CycleLength.Int32))
	}
	if r.BleedingLength.Valid {
		h.BleedingLength = model.Int(int(r.BleedingLength.Int32))
	}
	if r.DataQualityFlags.Valid {
		var flags []string
		if json.Unmarshal(r.DataQualityFlags.V, &flags) == nil {
			h.DataQualityFlags = flags
		}
	}
	return h
}

// PregnancySymptoms implements manager.PregnancySymptomSource: the day's pregnancy symptom log
// and extras as message-system symptom names (T-M7-04). Mood 1–2 of 5 reads as mood_sad.
func (s *StoreSource) PregnancySymptoms(ctx context.Context, date civildate.Date) ([]string, error) {
	var out []string
	add := func(on bool, name string) {
		if on {
			out = append(out, name)
		}
	}
	has := func(b sql.NullBool) bool { return b.Valid && b.Bool }
	row, err := s.pq.GetSymptomLog(ctx, pstore.GetSymptomLogParams{UserID: s.userID, LogDate: date})
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("messages: pregnancy symptoms: %w", err)
	default:
		add(has(row.HasNausea), "nausea")
		add(has(row.HasVomiting), "vomiting")
		add(has(row.HasFatigue), "fatigue")
		add(has(row.HasBackPain), "backache")
		add(has(row.HasHeadache), "headache")
		add(has(row.HasBreastPain), "breast_tenderness")
	}
	ex, err := s.pq.GetDailyExtras(ctx, pstore.GetDailyExtrasParams{UserID: s.userID, LogDate: date})
	var me *mysql.MySQLError
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case errors.As(err, &me) && me.Number == 1146:
		// A Laravel-schema database without the v2 tables (the contract fixture dump predates
		// 2026_09_26_000001): no extras, same as no row.
	case err != nil:
		return nil, fmt.Errorf("messages: pregnancy extras: %w", err)
	default:
		add(ex.Mood.Valid && ex.Mood.Int16 <= 2, "mood_sad")
	}
	return out, nil
}
