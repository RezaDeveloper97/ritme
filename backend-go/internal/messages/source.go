package messages

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// StoreSource is the sqlc-backed manager.Source for one user and request. It memoises every
// read (Laravel's relation cache and HealthDataEngine's per-request caches) and builds the
// legacy engine once (CYCLE_ENGINE=v1). With an engine source (WithEngine, shadow|v6) the cycle facts come from the
// shared v6 engine state instead (EngineDay, T-M9-38).
type StoreSource struct {
	q      store.Querier
	pq     pstore.Querier
	userID uint64
	today  civildate.Date

	profileRow    *store.GetMessageProfileRow
	profileLoaded bool
	preg          *pstore.PregnancyProfile
	pregLoaded    bool
	lifeMode      string
	lifeLoaded    bool
	engine        *legacy.Engine
	states        state.UserStates // nil = CYCLE_ENGINE=v1
	state         *engine.State
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

// LifeMode implements manager.LifeModeSource (B-N2-01): the stored life-stage mode, "" when none.
func (s *StoreSource) LifeMode(ctx context.Context) (string, error) {
	if !s.lifeLoaded {
		v, err := s.q.GetMessageLifeMode(ctx, s.userID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("messages: life mode: %w", err)
		}
		s.lifeMode, s.lifeLoaded = v.String, true
	}
	return s.lifeMode, nil
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
// fields the manager reads do not depend on the content part). CYCLE_ENGINE=v1 only: with an engine the manager reads
// EngineDay and never calls Cycle. The rows map through the cycle service's adapters (one mapping for every reader).
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
			histories[i] = cycleservice.HistoryFromRow(cyclestore.CycleHistory{
				ID: r.ID, PeriodStartDate: r.PeriodStartDate, PeriodEndDate: r.PeriodEndDate, CycleLength: r.CycleLength,
				BleedingLength: r.BleedingLength, IsConfirmed: r.IsConfirmed, IsEstimated: r.IsEstimated, Source: r.Source,
				DataQualityFlags: r.DataQualityFlags,
			})
		}
		var profile *model.Profile
		if row != nil {
			profile = cycleservice.ProfileFromRow(&cyclestore.UserProfile{
				UserGoal: row.UserGoal, LastPeriodStart: row.LastPeriodStart, Birthday: row.Birthday,
				CycleDuration: row.CycleDuration, PeriodDuration: row.PeriodDuration,
			})
		}
		s.engine = legacy.New(legacy.Input{Profile: profile, Histories: histories, Today: s.today})
	}
	return s.engine.CalculateForDate(ctx, date, false)
}

// WithEngine attaches the v6 engine state source (T-M9-38, D-130; nil = CYCLE_ENGINE=v1, the legacy Cycle path).
func (s *StoreSource) WithEngine(states state.UserStates) *StoreSource {
	s.states = states
	return s
}

// EngineDay implements manager.EngineSource: the v6 status of date resolved for the source's today (the user-local
// day). ok is false without an engine.
func (s *StoreSource) EngineDay(ctx context.Context, date civildate.Date) (manager.EngineDay, bool, error) {
	if s.states == nil {
		return manager.EngineDay{}, false, nil
	}
	if s.state == nil {
		st, err := s.states.ForUser(ctx, s.userID, s.today)
		if err != nil {
			return manager.EngineDay{}, false, fmt.Errorf("messages: engine state: %w", err)
		}
		s.state = st
	}
	return manager.EngineDay{
		Status:    s.state.Resolve(engine.DayOf(date), engine.DayOf(s.today)),
		Mode:      s.state.Mode(),
		HasAnchor: len(s.state.Periods()) > 0 || s.state.Declared().LastPeriodStart.OK(),
	}, true, nil
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
