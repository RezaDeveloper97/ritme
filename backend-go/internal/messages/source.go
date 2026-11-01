package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/pregnancyalerts/inboxalerts"
	"github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/pregnancy"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
	pregv2 "github.com/ritme/backend-go/internal/pregnancy/v2"
	"github.com/ritme/backend-go/internal/vitals"
	vstore "github.com/ritme/backend-go/internal/vitals/store"
)

// The feed renders a fanned-out preg.* inbox row (urgent_modal, status_banner) with its stored evidence in the
// pregnancy alert copy's terms (localized digits, dates, symptom labels: inboxalerts.Render), T-M10-08b.
func init() { feed.InstallEvidenceRenderer(inboxalerts.KeyPrefix, inboxalerts.Render) }

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
	now           time.Time // the Plus tier's moment (fact registry)
	factRow       *store.GetMessageFactProfileRow
	factLoaded    bool
	segs          SegmentResolver // T-M10-32: the saved admin segments (nil = the `segment` fact is empty)
	pregAlert     *facts.PregnancyAlertInput
	pregAlertDone bool
}

// SegmentResolver lists the ids of the saved admin segments a user matches at now (admin/segments.Resolver, T-M10-32).
type SegmentResolver interface {
	UserSegments(ctx context.Context, userID uint64, now time.Time) ([]string, error)
}

// WithSegments wires the `segment` fact's resolver (T-M10-32).
func (s *StoreSource) WithSegments(r SegmentResolver) *StoreSource {
	s.segs = r
	return s
}

// Segments implements facts.SegmentSource; it costs the resolver's queries only when a rule names `segment`.
func (s *StoreSource) Segments(ctx context.Context) ([]string, error) {
	if s.segs == nil {
		return []string{}, nil
	}
	now := s.now
	if now.IsZero() {
		now = time.Now()
	}
	return s.segs.UserSegments(ctx, s.userID, now)
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
	if _, err := s.EngineState(ctx); err != nil {
		return manager.EngineDay{}, false, err
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

// ---- Message fact registry v6 sources (T-M10-03, D-161) ----
//
// StoreSource implements every facts.Sources interface for its user and today. The account, contraception and
// postpartum facts and the vitals diabetic flag share one memoised profile row (GetMessageFactProfile); the logs,
// symptom history and vitals read one small query each, and only when a rule needs their facts.

// FactSources are the registry sources of this user and request.
func (s *StoreSource) FactSources() facts.Sources {
	src := facts.Sources{Account: s, Logs: s, Vitals: s, Pregnancy: s, Postpartum: s, Contraception: s}
	if s.states != nil {
		src.Engine = s
	}
	return src
}

// WithNow sets the request time the Plus tier is read at (zero = the start of today in Tehran).
func (s *StoreSource) WithNow(now time.Time) *StoreSource {
	s.now = now
	return s
}

// EngineState implements facts.EngineSource: the shared v6 state for the source's today (memoised with EngineDay).
func (s *StoreSource) EngineState(ctx context.Context) (*engine.State, error) {
	if s.states == nil {
		return nil, nil
	}
	if s.state == nil {
		st, err := s.states.ForUser(ctx, s.userID, s.today)
		if err != nil {
			return nil, fmt.Errorf("messages: engine state: %w", err)
		}
		s.state = st
	}
	return s.state, nil
}

func (s *StoreSource) factProfile(ctx context.Context) (*store.GetMessageFactProfileRow, error) {
	if s.factLoaded {
		return s.factRow, nil
	}
	now := s.now
	if now.IsZero() {
		now = s.today.TehranMidnight()
	}
	row, err := s.q.GetMessageFactProfile(ctx, store.GetMessageFactProfileParams{
		UserID: s.userID, Now: now, Today: s.today, Since: s.logsSince(),
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("messages: fact profile: %w", err)
	default:
		s.factRow = &row
	}
	s.factLoaded = true
	return s.factRow, nil
}

// Account implements facts.AccountSource.
func (s *StoreSource) Account(ctx context.Context) (*facts.Account, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return nil, err
	}
	a := &facts.Account{Name: row.Name.String, UserGoal: row.UserGoal.String, SubscriptionType: row.SubscriptionType.String,
		LifeMode: row.LifeMode.String, Plan: "free"}
	if row.CreatedAt.Valid {
		a.SignupDay = civildate.InTehran(row.CreatedAt.Time)
	}
	if row.Birthday.Valid {
		a.Birthday = row.Birthday.Date
	}
	switch {
	case row.PlusActive != 0:
		a.Plan = "plus"
	case row.TrialRunning != 0:
		a.Plan = "trial"
	}
	return a, nil
}

// HormonalContraception implements facts.ContraceptionSource (I-03, contraception.HormonalActive).
func (s *StoreSource) HormonalContraception(ctx context.Context) (bool, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return false, err
	}
	var meds []string
	if row.Medications.Valid && len(row.Medications.V) > 0 {
		_ = json.Unmarshal(row.Medications.V, &meds) // not a list = no onboarding answer
	}
	return contraception.HormonalActive(row.ContraceptionMethod.String, row.ContraceptionMethod.Valid, meds), nil
}

// PostpartumBirth implements facts.PostpartumSource: the birth date while the stored life mode is postpartum.
func (s *StoreSource) PostpartumBirth(ctx context.Context) (civildate.Date, bool, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil || !row.PostpartumBirthDate.Valid || row.LifeMode.String != string(enums.LifeModePostpartum) {
		return civildate.Date{}, false, err
	}
	return row.PostpartumBirthDate.Date, true, nil
}

// PregnancyDating implements facts.PregnancySource: the active pregnancy's week and trimester on day (pregnancy v2
// Resolve — the dating the pregnancy screens use).
func (s *StoreSource) PregnancyDating(ctx context.Context, day civildate.Date) (*facts.PregnancyDating, error) {
	p, err := s.activePregnancy(ctx)
	if err != nil || p == nil {
		return nil, err
	}
	d, ok := pregv2.Resolve(p, day)
	if !ok {
		return nil, nil
	}
	return &facts.PregnancyDating{Week: d.CurrentWeek(), Trimester: d.Trimester()}, nil
}

// PregnancyAlertInput implements facts.PregnancyAlertSource (T-M10-08b): the pregnancy alert detectors' input of an
// active pregnancy, read by inboxalerts.Input — the same input the pregnancy alert paths evaluate the preg.* rules on,
// so the feed's alerts inbox sees the preg.* facts (nil without an active pregnancy; one read per request).
func (s *StoreSource) PregnancyAlertInput(ctx context.Context, _ civildate.Date) (*facts.PregnancyAlertInput, error) {
	if s.pregAlertDone {
		return s.pregAlert, nil
	}
	p, err := s.activePregnancy(ctx)
	if err != nil {
		return nil, err
	}
	if p != nil {
		if s.pregAlert, err = inboxalerts.Input(ctx, s.pq, s.q, s.userID, s.alertNow()); err != nil {
			return nil, fmt.Errorf("messages: pregnancy alert input: %w", err)
		}
	}
	s.pregAlertDone = true
	return s.pregAlert, nil
}

// alertNow is the moment the pregnancy alert input is read at: the request time, else now when it falls on the
// source's today, else the start of today in Tehran.
func (s *StoreSource) alertNow() time.Time {
	if !s.now.IsZero() {
		return s.now
	}
	if now := time.Now(); civildate.InTehran(now) == s.today {
		return now
	}
	return s.today.TehranMidnight()
}

// logsSince is the first day of the logs_30d window of the source's today.
func (s *StoreSource) logsSince() civildate.Date { return s.today.AddDays(-29) }

// scanDay reads a MAX(date) column (nil = no row).
func scanDay(v any) (civildate.Date, error) {
	var d civildate.Date
	if v == nil {
		return d, nil
	}
	if err := d.Scan(v); err != nil {
		return d, fmt.Errorf("messages: last log date: %w", err)
	}
	return d, nil
}

// LogStats implements facts.LogSource: the profile row's numbers for the source's own window, else one query.
func (s *StoreSource) LogStats(ctx context.Context, since, today civildate.Date) (facts.LogStats, error) {
	if since == s.logsSince() && today == s.today {
		row, err := s.factProfile(ctx)
		if err != nil || row == nil {
			return facts.LogStats{}, err
		}
		last, err := scanDay(row.LastLogDate)
		return facts.LogStats{Last: last, Days30d: int(row.LogDaysSince)}, err
	}
	row, err := s.q.GetMessageFactLogStats(ctx, store.GetMessageFactLogStatsParams{UserID: s.userID, Since: since, Today: today})
	if err != nil {
		return facts.LogStats{}, fmt.Errorf("messages: log stats: %w", err)
	}
	last, err := scanDay(row.LastLogDate)
	return facts.LogStats{Last: last, Days30d: int(row.DaysSince)}, err
}

// activePregnancy is the pregnancy profile when a pregnancy is active (read only then; nil otherwise).
func (s *StoreSource) activePregnancy(ctx context.Context) (*pstore.PregnancyProfile, error) {
	row, err := s.factProfile(ctx)
	if err != nil || row == nil || row.PregnancyActive == 0 {
		return nil, err
	}
	p, err := s.PregnancyProfile(ctx)
	if err != nil || p == nil || !p.PregnancyMode {
		return nil, err
	}
	return p, nil
}

// SymptomEntries implements facts.LogSource.
func (s *StoreSource) SymptomEntries(ctx context.Context, from, to civildate.Date) ([]facts.SymptomEntry, error) {
	rows, err := s.q.ListMessageFactSymptomEntries(ctx, store.ListMessageFactSymptomEntriesParams{
		UserID: s.userID, DateFrom: from, DateTo: to,
	})
	if err != nil {
		return nil, fmt.Errorf("messages: symptom entries: %w", err)
	}
	out := make([]facts.SymptomEntry, len(rows))
	for i, r := range rows {
		out[i] = facts.SymptomEntry{Date: r.LogDate, Category: r.Category, Param: r.Param, Item: r.Item,
			Level: r.ValueCode.String, Score: r.ValueNum.String}
	}
	return out, nil
}

// LastVitals implements facts.VitalsSource: the newest blood pressure and glucose reading of [from, to] (timed and
// log-sheet values merged by vitals.Merge), classified by vitals.Reading.Class with the user's context — the pregnancy
// table for a reading inside the active pregnancy, the diabetic targets for a diabetic user — as the vitals screens.
func (s *StoreSource) LastVitals(ctx context.Context, from, to civildate.Date) (facts.LastVitals, error) {
	rows, err := s.q.ListMessageFactVitalReadings(ctx, store.ListMessageFactVitalReadingsParams{
		UserID: s.userID, DateFrom: from.TehranMidnight(), DateTo: to.AddDays(1).TehranMidnight(),
	})
	if err != nil {
		return facts.LastVitals{}, fmt.Errorf("messages: vitals: %w", err)
	}
	logRows, err := s.q.ListMessageFactLogMeasurements(ctx, store.ListMessageFactLogMeasurementsParams{
		UserID: s.userID, DateFrom: from, DateTo: to,
	})
	if err != nil {
		return facts.LastVitals{}, fmt.Errorf("messages: vitals log values: %w", err)
	}
	timed := make([]vitals.Reading, len(rows))
	for i, r := range rows {
		timed[i] = vitals.FromRow(vstore.VitalReading(r))
	}
	logs := make([]vstore.ListLogMeasurementsRow, len(logRows))
	for i, r := range logRows {
		logs[i] = vstore.ListLogMeasurementsRow(r)
	}
	var bp, glucose *vitals.Reading
	merged := vitals.Merge(timed, logs)
	for i := range merged {
		switch r := &merged[i]; {
		case r.Type == vitals.TypeBP && bp == nil:
			bp = r
		case r.Type == vitals.TypeGlucose && glucose == nil:
			glucose = r
		}
	}
	if bp == nil && glucose == nil {
		return facts.LastVitals{}, nil
	}
	if err := s.markVitals(ctx, bp, glucose); err != nil {
		return facts.LastVitals{}, err
	}
	var out facts.LastVitals
	if bp != nil {
		out.BP = bp.Class()
	}
	if glucose != nil {
		out.Glucose = glucose.Class()
	}
	return out, nil
}

// markVitals sets the user's context on the readings (vitals.Service.mark): Pregnant for a day inside the active
// pregnancy (dated start .. today), Diabetic + Targets on glucose for a diabetic user.
func (s *StoreSource) markVitals(ctx context.Context, bp, glucose *vitals.Reading) error {
	p, err := s.activePregnancy(ctx)
	if err != nil {
		return err
	}
	if p != nil {
		if d, ok := pregv2.Resolve(p, s.today); ok {
			for _, r := range []*vitals.Reading{bp, glucose} {
				if r != nil {
					r.Pregnant = !r.Date.Before(d.Start()) && !r.Date.After(s.today)
				}
			}
		}
	}
	if glucose == nil {
		return nil
	}
	row, err := s.factProfile(ctx)
	if err != nil || row == nil || !row.ChronicIllnesses.Valid || !vitals.HasDiabetes(row.ChronicIllnesses.V) {
		return err
	}
	targets, err := s.q.ListMessageFactVitalTargets(ctx, s.userID)
	if err != nil {
		return fmt.Errorf("messages: vital targets: %w", err)
	}
	glucose.Diabetic, glucose.Targets = true, vitals.Targets{}
	for _, t := range targets {
		glucose.Targets[t.Context] = vitals.TargetFromRow(vstore.VitalTarget(t))
	}
	return nil
}
