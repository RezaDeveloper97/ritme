package messages

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ritme/backend-go/internal/checkups"
	chkengine "github.com/ritme/backend-go/internal/checkups/engine"
	cstore "github.com/ritme/backend-go/internal/conditions/store"
	"github.com/ritme/backend-go/internal/contraception"
	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	"github.com/ritme/backend-go/internal/cycle/state"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/healthlog"
	"github.com/ritme/backend-go/internal/menopause"
	"github.com/ritme/backend-go/internal/messages/conditionnudges"
	"github.com/ritme/backend-go/internal/messages/facts"
	"github.com/ritme/backend-go/internal/messages/feed"
	"github.com/ritme/backend-go/internal/messages/manager"
	"github.com/ritme/backend-go/internal/messages/menomessages"
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
//
// Its open rows are fanned out only while she is in pregnancy (feed.InstallModeScope): after a switch to another life
// mode an unacknowledged pregnancy alert stays in the alerts list but no longer pops up as the urgent modal.
func init() {
	feed.InstallEvidenceRenderer(inboxalerts.KeyPrefix, inboxalerts.Render)
	feed.InstallModeScope(inboxalerts.KeyPrefix, string(enums.LifeModePregnancy))
}

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
	mods          *ModuleReads // T-M10-09b: the meno.* / cond.* reads (nil = the installed ones)
	acct          *store.GetMessageAccountRow
	acctLoaded    bool
	recent        []store.ListMessageRecentLogsRow // the last recent-logs window read (recentFrom..recentTo)
	recentFrom    civildate.Date
	recentTo      civildate.Date
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

// account is the account row (T-M9-49b): the gender, the stored life mode, the profile and whether a pregnancy
// profile exists in one read. nil = no user row.
func (s *StoreSource) account(ctx context.Context) (*store.GetMessageAccountRow, error) {
	if s.acctLoaded {
		return s.acct, nil
	}
	row, err := s.q.GetMessageAccount(ctx, s.userID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
	case err != nil:
		return nil, fmt.Errorf("messages: load account: %w", err)
	default:
		s.acct = &row
	}
	s.acctLoaded = true
	return s.acct, nil
}

// IsCompanion is companion.Accounts.IsCompanion off the account row: gender male. No row or no gender = a woman's
// account.
func (s *StoreSource) IsCompanion(ctx context.Context) (bool, error) {
	a, err := s.account(ctx)
	if err != nil || a == nil {
		return false, err
	}
	return a.Gender.Valid && a.Gender.String == string(enums.GenderMale), nil
}

func (s *StoreSource) profile(ctx context.Context) (*store.GetMessageProfileRow, error) {
	if s.profileLoaded {
		return s.profileRow, nil
	}
	a, err := s.account(ctx)
	if err != nil {
		return nil, fmt.Errorf("messages: load profile: %w", err)
	}
	if a != nil && a.ProfileID.Valid {
		s.profileRow = &store.GetMessageProfileRow{
			ID: uint64(a.ProfileID.Int64), Birthday: a.Birthday, PeriodDuration: a.PeriodDuration, //nolint:gosec // auto-increment id
			CycleDuration: a.CycleDuration, LastPeriodStart: a.LastPeriodStart, UserGoal: a.UserGoal.String,
			SubscriptionType: a.SubscriptionType.String,
		}
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
		a, err := s.account(ctx)
		if err != nil {
			return "", fmt.Errorf("messages: life mode: %w", err)
		}
		if a != nil {
			s.lifeMode = a.LifeMode.String
		}
		s.lifeLoaded = true
	}
	return s.lifeMode, nil
}

// PregnancyProfile implements manager.Source.
func (s *StoreSource) PregnancyProfile(ctx context.Context) (*pstore.PregnancyProfile, error) {
	if !s.pregLoaded {
		if a, err := s.account(ctx); err != nil {
			return nil, err
		} else if a == nil || a.HasPregnancyProfile == 0 { // no pregnancy profile row: nothing to read
			s.pregLoaded = true
			return nil, nil
		}
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

// DailyLog implements manager.Source: the date's row of the 90-day recent logs window ending on date (T-M9-49b: the
// manager reads both, so one query answers them; (user_id, log_date) is unique).
func (s *StoreSource) DailyLog(ctx context.Context, date civildate.Date) (*manager.Log, error) {
	rows, err := s.recentRows(ctx, date.AddDays(-90), date)
	if err != nil {
		return nil, fmt.Errorf("messages: load daily log: %w", err)
	}
	for _, r := range rows {
		if r.LogDate.Compare(date) == 0 {
			l := logFromColumns(r.EnergyLevel, r.SleepQuality)
			return &l, nil
		}
	}
	return nil, nil
}

// RecentLogs implements manager.Source.
func (s *StoreSource) RecentLogs(ctx context.Context, from, to civildate.Date) ([]manager.Log, error) {
	rows, err := s.recentRows(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("messages: load recent logs: %w", err)
	}
	out := make([]manager.Log, len(rows))
	for i, r := range rows {
		out[i] = logFromColumns(r.EnergyLevel, r.SleepQuality)
	}
	return out, nil
}

// recentRows is ListMessageRecentLogs of [from, to], memoised for the last window read.
func (s *StoreSource) recentRows(ctx context.Context, from, to civildate.Date) ([]store.ListMessageRecentLogsRow, error) {
	if s.recent != nil && s.recentFrom == from && s.recentTo == to {
		return s.recent, nil
	}
	rows, err := s.q.ListMessageRecentLogs(ctx, store.ListMessageRecentLogsParams{UserID: s.userID, DateFrom: from, DateTo: to})
	if err != nil {
		return nil, err
	}
	s.recent, s.recentFrom, s.recentTo = rows, from, to
	return rows, nil
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
		UserID: s.userID, Now: now, Today: s.today, Since: s.logsSince(), CondSince: s.condSince(),
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

// condSince is the first day of the condition entries the fact profile row carries (T-M9-49b).
func (s *StoreSource) condSince() civildate.Date { return s.today.AddDays(-condLookbackDays) }

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

// DayEntries implements facts.LogSource.
func (s *StoreSource) DayEntries(ctx context.Context, day civildate.Date) ([]facts.SymptomEntry, error) {
	rows, err := s.q.ListMessageFactDayEntries(ctx, store.ListMessageFactDayEntriesParams{UserID: s.userID, LogDate: day})
	if err != nil {
		return nil, fmt.Errorf("messages: day entries: %w", err)
	}
	out := make([]facts.SymptomEntry, len(rows))
	for i, r := range rows {
		out[i] = facts.SymptomEntry{Category: r.Category, Param: r.Param, Item: r.Item,
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

// ---- Menopause and condition nudge facts (T-M10-09b, D-166) ----
//
// The meno.* and cond.* facts read other modules (the menopause signals, the condition enrolments and the day logs)
// through ModuleReads, which the route mount installs once (InstallModuleReads), as the feed's inbox and challenges
// are installed. Without them both groups stay absent (every meno.* / cond.* rule silent), as before T-M10-09b.

// ModuleReads are the module reads behind the meno.* and cond.* facts.
type ModuleReads struct {
	meno menomessages.SignalSource // *menopause.Service
	cq   cstore.Querier
	logs *healthlog.Service
}

// NewModuleReads wires the reads on db; meno is the menopause signal source (*menopause.Service on the same db).
func NewModuleReads(db store.DBTX, meno menomessages.SignalSource) *ModuleReads {
	return &ModuleReads{meno: meno, cq: cstore.New(db), logs: healthlog.NewService(db)}
}

type moduleReadsBox struct{ m *ModuleReads }

var installedModuleReads atomic.Pointer[moduleReadsBox]

// InstallModuleReads makes m the module reads of every StoreSource without its own (WithModuleReads); nil uninstalls.
func InstallModuleReads(m *ModuleReads) {
	if m == nil {
		installedModuleReads.Store(nil)
		return
	}
	installedModuleReads.Store(&moduleReadsBox{m: m})
}

// WithModuleReads sets this source's module reads (instead of the installed ones).
func (s *StoreSource) WithModuleReads(m *ModuleReads) *StoreSource {
	s.mods = m
	return s
}

func (s *StoreSource) moduleReads() *ModuleReads {
	if s.mods != nil {
		return s.mods
	}
	if b := installedModuleReads.Load(); b != nil {
		return b.m
	}
	return nil
}

// menoDefaultLocale is the language of the checkup titles of a request in no known language (the jobs): the default
// language when no row says otherwise (i18n.Languages.DefaultCode).
const menoDefaultLocale = "fa"

// MenopauseInput implements facts.MenopauseSource: the menopause signals (menopause.Service.Signals, the GET's own
// detection) as the meno.* input (menomessages.Input), checkup titles in locale. A user whose stored life mode is not
// menopause can never be in menopause mode (enums.ResolveLifeMode): she gets the «nothing» input without a read, so
// her open meno.* rows resolve and the feed pays nothing for her.
func (s *StoreSource) MenopauseInput(ctx context.Context, day civildate.Date, locale string) (*facts.MenopauseInput, error) {
	m := s.moduleReads()
	if m == nil || m.meno == nil {
		return nil, nil
	}
	row, err := s.factProfile(ctx)
	if err != nil {
		return nil, err
	}
	if row == nil || row.LifeMode.String != string(enums.LifeModeMenopause) {
		mode := enums.LifeModeCycle
		if row != nil {
			mode = enums.ResolveLifeMode(row.LifeMode.String, row.PregnancyActive != 0, row.UserGoal.String)
		}
		return menomessages.Input(menopause.Signals{Mode: mode, Date: day}, nil), nil
	}
	now := s.alertNow()
	if civildate.InTehran(now) != day {
		now = day.TehranMidnight().Add(12 * time.Hour)
	}
	sig, err := m.meno.Signals(ctx, s.userID, now)
	if err != nil {
		return nil, fmt.Errorf("messages: menopause signals: %w", err)
	}
	if locale == "" {
		locale = menoDefaultLocale
	}
	lang := checkups.Lang{Locale: locale, Default: menoDefaultLocale}
	title := func(it *chkengine.Item) string {
		if sig.Plan == nil {
			return ""
		}
		t, _ := sig.Plan.ItemJSON(*it, lang).Get("title")
		str, _ := t.(string)
		return str
	}
	return menomessages.Input(sig, title), nil
}

// condNudgedModes are the life modes the condition nudges run in (conditionnudges' nudged modes): outside them the
// cond.* input is «nothing» without a read.
var condNudgedModes = []string{string(enums.LifeModeCycle), string(enums.LifeModeTTC), string(enums.LifeModeTeen)}

// ConditionInput implements facts.ConditionSource: the condition nudge input of the source's today, read as the
// nudges GET reads it (conditionnudges.Engine): under an engine source the engine's mode and current cycle (the
// shared state, no extra read), else the v1 mode and window; then the enrolments and the day logs of the window.
func (s *StoreSource) ConditionInput(ctx context.Context) (*facts.ConditionInput, error) {
	m := s.moduleReads()
	if m == nil {
		return nil, nil
	}
	f := conditionnudges.Facts{Enrolled: map[string]bool{}}
	if s.states != nil {
		st, err := s.EngineState(ctx)
		if err != nil {
			return nil, err
		}
		f.Mode = st.Mode().String()
		if slices.Contains(condNudgedModes, f.Mode) {
			if from, to, ok := conditionnudges.EngineCycleWindow(st, s.today); ok {
				f.From, f.To = from, to
			}
		}
	} else {
		mode, err := m.logs.LifeMode(ctx, s.userID)
		if err != nil {
			return nil, fmt.Errorf("messages: condition facts: %w", err)
		}
		f.Mode = mode
		if slices.Contains(condNudgedModes, f.Mode) {
			if f.From, f.To, err = s.v1CycleWindow(ctx); err != nil {
				return nil, err
			}
		}
	}
	if f.From.IsZero() || !s.loggedSince(ctx, f.From, f.To) {
		// Nothing to nudge: no current cycle (the GET skips the reads too), or no log day in it — the fact profile row
		// (read anyway) says so, which keeps a warm feed at its query budget. cond.programs is then empty, as outside a
		// current cycle.
		return conditionnudges.Input(f), nil
	}
	// T-M9-49b: the enrolments and the window's day entries off the fact profile row when it carries them.
	row, _ := s.factProfile(ctx)
	programs, ok := []string(nil), false
	if row != nil {
		programs, ok = parsePrograms(row.ConditionPrograms)
	}
	if !ok {
		rows, err := m.cq.ListEnrolments(ctx, s.userID)
		if err != nil {
			return nil, fmt.Errorf("messages: condition enrolments: %w", err)
		}
		for _, r := range rows {
			programs = append(programs, r.Program)
		}
	}
	for _, p := range programs {
		f.Enrolled[p] = true
	}
	ok = false
	if row != nil && !f.From.Before(s.condSince()) && !f.To.After(s.today) {
		f.Days, ok = parseCondEntries(row.CondEntries)
	}
	if !ok {
		var err error
		if f.Days, err = m.logs.Range(ctx, s.userID, f.From, f.To); err != nil {
			return nil, fmt.Errorf("messages: condition day logs: %w", err)
		}
	}
	return conditionnudges.Input(f), nil
}

// v1CycleWindow is the nudges' v1 current cycle (conditionnudges.CycleWindow): the latest period start on or before
// today of the profile and the cycle histories, else the profile's cycle length.
func (s *StoreSource) v1CycleWindow(ctx context.Context) (civildate.Date, civildate.Date, error) {
	row, err := s.profile(ctx)
	if err != nil {
		return civildate.Date{}, civildate.Date{}, err
	}
	var latest *civildate.Date
	cycleLen := 0
	if row != nil {
		if row.CycleDuration.Valid {
			cycleLen = int(row.CycleDuration.Int16)
		}
		if d := row.LastPeriodStart; d.Valid && !d.Date.After(s.today) {
			latest = &d.Date
		}
	}
	histories, err := s.q.ListMessageCycleHistories(ctx, s.userID)
	if err != nil {
		return civildate.Date{}, civildate.Date{}, fmt.Errorf("messages: cycle histories: %w", err)
	}
	for _, h := range histories {
		if d := h.PeriodStartDate; !d.After(s.today) && (latest == nil || d.After(*latest)) {
			latest = &d
		}
	}
	from, to := conditionnudges.CycleWindow(s.today, latest, cycleLen)
	return from, to, nil
}

// loggedSince reports whether the user may have a log day in [from, to]: false only when the fact profile row (its
// newest log day up to the source's today) proves there is none; true when it cannot tell.
func (s *StoreSource) loggedSince(ctx context.Context, from, to civildate.Date) bool {
	if to != s.today {
		return true
	}
	row, err := s.factProfile(ctx)
	if err != nil || row == nil {
		return true
	}
	last, err := scanDay(row.LastLogDate)
	if err != nil {
		return true
	}
	return !last.IsZero() && !last.Before(from)
}
