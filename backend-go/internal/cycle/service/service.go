// Package service loads the cycle engines' inputs and runs them: the port of
// CycleCalculationController's data path (backend/app/Http/Controllers/Api/V1/
// CycleCalculationController.php) plus the HealthDataEngine loading it relies on.
//
// One request = one Snapshot: the profile, every cycle_histories row (newest first) and the daily
// logs of the requested range, each read with a single query, plus a request-scoped
// recommendation repository. The legacy engine (legacy.Engine → `calculation`, month views) and
// the v1.1 view builder (view.Build → `cycle_view`) both run over it.
//
// Reuse (messages, home):
//
//	sn, err := svc.Load(ctx, userID, from, to, today)  // today = civildate.Today(clock)
//	calc, err := sn.Engine().CalculateForDate(ctx, date, true)
//	calc, cv, err := sn.Day(ctx, date, locale)          // localized calculation + cycle_view
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ritme/backend-go/internal/cycle/cache"
	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/cycle/view"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// Service runs the cycle engines for a user.
type Service struct {
	q     store.Querier
	cache *cache.Cache
}

// New returns the service over db; c may be nil (no cache).
func New(db store.DBTX, c *cache.Cache) *Service {
	return &Service{q: store.New(db), cache: c}
}

// Snapshot is one request's engine inputs.
type Snapshot struct {
	UserID uint64
	// Profile is the user's profile row (nil = none).
	Profile *store.UserProfile
	// HistoryRows / Histories are every cycle_histories row, newest start first.
	HistoryRows []store.CycleHistory
	Histories   []model.History
	// From..To is the loaded daily-log range; LogRows are its rows, Logs the engine view by day.
	From, To civildate.Date
	LogRows  []store.DailyHealthLog
	Logs     map[civildate.Date]*legacy.DailyLog
	// Tips is the request-scoped recommendation repository.
	Tips *recommendation.Repository
	// Today is the Tehran calendar day of the request clock.
	Today civildate.Date
	// LengthsManual is «خودکار از داده‌ها» off (B-N1-09).
	LengthsManual bool
	// LifeMode is the stored life-stage mode ("" = none, B-N2-11b).
	LifeMode enums.LifeMode

	engineOnce sync.Once
	engine     *legacy.Engine
}

// Load reads the profile, the full history and the daily logs of from..to (inclusive).
func (s *Service) Load(ctx context.Context, userID uint64, from, to, today civildate.Date) (*Snapshot, error) {
	sn := &Snapshot{
		UserID: userID, From: from, To: to, Today: today,
		Tips: recommendation.New(RecommendationSource{Q: s.q}),
		Logs: map[civildate.Date]*legacy.DailyLog{},
	}
	p, err := s.q.GetEngineProfileByUserID(ctx, userID) // profile + «خودکار از داده‌ها», one query
	switch {
	case err == nil:
		sn.Profile = &p.UserProfile
		sn.LengthsManual = !p.LengthsAuto
		sn.LifeMode = enums.LifeMode(p.LifeMode.String)
	case !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("cycle: load profile: %w", err)
	}
	if sn.HistoryRows, err = s.q.ListCycleHistoriesNewestFirst(ctx, userID); err != nil {
		return nil, fmt.Errorf("cycle: load histories: %w", err)
	}
	sn.Histories = HistoriesFromRows(sn.HistoryRows)
	sn.LogRows, err = s.q.ListDailyLogsBetween(ctx, store.ListDailyLogsBetweenParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, fmt.Errorf("cycle: load daily logs: %w", err)
	}
	for _, r := range sn.LogRows {
		if _, dup := sn.Logs[r.LogDate]; !dup { // keyBy keeps the last; (user_id, log_date) is unique
			sn.Logs[r.LogDate] = DailyLogFromRow(r)
		}
	}
	return sn, nil
}

// EngineProfile is the profile as the engines read it (nil = no profile).
func (sn *Snapshot) EngineProfile() *model.Profile {
	p := ProfileFromRow(sn.Profile)
	if p != nil {
		p.LengthsManual = sn.LengthsManual
		p.NoFertilityCopy = sn.LifeMode != "" && !sn.LifeMode.AllowsFertilityContent()
	}
	return p
}

// Engine is the legacy HealthDataEngine over the snapshot (built once).
func (sn *Snapshot) Engine() *legacy.Engine {
	sn.engineOnce.Do(func() {
		sn.engine = legacy.New(legacy.Input{
			Profile:   sn.EngineProfile(),
			Histories: sn.Histories,
			Logs:      sn.Logs,
			Today:     sn.Today,
			Tips:      sn.Tips,
		})
	})
	return sn.engine
}

// CalculationStatus is `$profile?->calculation_status ?? 'pending'`.
func (sn *Snapshot) CalculationStatus() string {
	if sn.Profile == nil || sn.Profile.CalculationStatus == "" {
		return string(enums.CalculationStatusPending)
	}
	return sn.Profile.CalculationStatus
}

// IsRecalculating is status === processing.
func (sn *Snapshot) IsRecalculating() bool {
	return sn.CalculationStatus() == string(enums.CalculationStatusProcessing)
}

// Version is `(int) ($profile?->calculation_version ?? 0)`.
func (sn *Snapshot) Version() int64 {
	if sn.Profile == nil {
		return 0
	}
	return int64(sn.Profile.CalculationVersion)
}

// Day is getCalculationForDate's computation: the request-locale calculation
// (localizeCalculation, read by the §19 display window — D-30) and the §19 cycle_view.
func (sn *Snapshot) Day(ctx context.Context, date civildate.Date, locale string) (legacy.Calculation, view.CycleView, error) {
	calc, err := sn.Engine().CalculateForDisplay(ctx, date)
	if err != nil {
		return legacy.Calculation{}, view.CycleView{}, err
	}
	calc = calc.Localize(locale)
	profile := sn.EngineProfile()
	cv := view.Build(sn.Histories, profile, date, sn.Today, locale, calc)
	if profile != nil && profile.NoFertilityCopy { // CB-TEEN-04b: built from the full calculation, served without its copy
		calc = calc.WithoutFertilityCopy()
	}
	return calc, cv, nil
}

// Month calculates every day of year-month (full or calendar mode) and its summary.
func (sn *Snapshot) Month(ctx context.Context, year, month int, withContent bool) ([]legacy.Calculation, legacy.MonthSummary, error) {
	start, end := MonthRange(year, month)
	calcs := make([]legacy.Calculation, 0, 31)
	profile := sn.EngineProfile()
	noFertility := profile != nil && profile.NoFertilityCopy
	for d := start; !d.After(end); d = d.AddDays(1) {
		c, err := sn.Engine().CalculateForDate(ctx, d, withContent)
		if err != nil {
			return nil, legacy.MonthSummary{}, err
		}
		if noFertility { // CB-TEEN-04b
			c = c.WithoutFertilityCopy()
		}
		calcs = append(calcs, c)
	}
	return calcs, legacy.SummarizeMonth(calcs), nil
}

// MonthRange is Carbon::createFromDate(y, m, 1)->startOfMonth() .. ->endOfMonth().
func MonthRange(year, month int) (civildate.Date, civildate.Date) {
	return civildate.New(year, time.Month(month), 1), civildate.New(year, time.Month(month)+1, 0)
}

// cacheKey is CycleEngineCache::key: the inputs are the profile row, every history row, the logs
// of the range and — when tips are part of the result — the recommendations signature.
func (sn *Snapshot) cacheKey(ctx context.Context, locale, scope string, withContent bool) (cache.Key, error) {
	var sig *string
	if withContent {
		s, err := sn.Tips.Signature(ctx)
		if err != nil {
			return cache.Key{}, err
		}
		sig = &s
	}
	inputs := []any{sn.Profile, sn.HistoryRows, sn.From, sn.To, sn.LogRows, sig}
	if sn.LengthsManual { // only then, so the automatic (default) key is unchanged
		inputs = append(inputs, "lengths_manual")
	}
	if sn.LifeMode != "" && !sn.LifeMode.AllowsFertilityContent() { // B-N2-11b: the day copy differs; others keep their key
		inputs = append(inputs, "no_fertility_copy.v2") // v2: CB-TEEN-04b also drops the calculation's copy
	}
	return cache.Key{
		UserID:  sn.UserID,
		Version: sn.Version(),
		Locale:  locale,
		Today:   sn.Today,
		Scope:   scope,
		Inputs:  inputs,
	}, nil
}

// DayPayload is the cached part of /cycle/today and /cycle/date: {calculation, cycle_view}.
type DayPayload struct {
	Status          string
	IsRecalculating bool
	// JSON is the unescaped `{"calculation":…,"cycle_view":…}` object.
	JSON []byte
}

// DayJSON loads the snapshot for date and returns the (cached) day payload.
func (s *Service) DayJSON(ctx context.Context, userID uint64, date, today civildate.Date, locale string) (*DayPayload, error) {
	sn, err := s.Load(ctx, userID, date, date, today)
	if err != nil {
		return nil, err
	}
	compute := func() ([]byte, error) {
		calc, cv, err := sn.Day(ctx, date, locale)
		if err != nil {
			return nil, err
		}
		return jsonx.Marshal(jsonx.Obj("calculation", calc, "cycle_view", cv), jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	}
	out, err := s.remember(ctx, sn, locale, "day:"+date.String(), true, compute)
	if err != nil {
		return nil, err
	}
	return &DayPayload{Status: sn.CalculationStatus(), IsRecalculating: sn.IsRecalculating(), JSON: out}, nil
}

// MonthJSON loads the snapshot for the month and returns the (cached)
// `{"calculations":[…],"month_summary":{…}}` object.
func (s *Service) MonthJSON(ctx context.Context, userID uint64, year, month int, viewName string, today civildate.Date, locale string) (*DayPayload, error) {
	start, end := MonthRange(year, month)
	sn, err := s.Load(ctx, userID, start, end, today)
	if err != nil {
		return nil, err
	}
	withContent := viewName == "full"
	compute := func() ([]byte, error) {
		calcs, summary, err := sn.Month(ctx, year, month, withContent)
		if err != nil {
			return nil, err
		}
		return jsonx.Marshal(jsonx.Obj("calculations", calcs, "month_summary", summary), jsonx.UnescapedSlashes|jsonx.UnescapedUnicode)
	}
	scope := fmt.Sprintf("month:%04d-%02d:%s", year, month, viewName)
	out, err := s.remember(ctx, sn, locale, scope, withContent, compute)
	if err != nil {
		return nil, err
	}
	return &DayPayload{Status: sn.CalculationStatus(), IsRecalculating: sn.IsRecalculating(), JSON: out}, nil
}

func (s *Service) remember(ctx context.Context, sn *Snapshot, locale, scope string, withContent bool, compute func() ([]byte, error)) ([]byte, error) {
	if !s.cache.Enabled() {
		return compute()
	}
	key, err := sn.cacheKey(ctx, locale, scope, withContent)
	if err != nil {
		return nil, err
	}
	return s.cache.Remember(ctx, key, compute)
}

// Profile returns the user's profile row (nil when none).
func (s *Service) Profile(ctx context.Context, userID uint64) (*store.UserProfile, error) {
	p, err := s.q.GetProfileByUserID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cycle: load profile: %w", err)
	}
	return &p, nil
}
