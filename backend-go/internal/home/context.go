// Package home is the home page (backend/app/Services/HomePage/**, DailyChallengeService and
// HomeController): the 17 sections, the task / challenge toggles and the in-app notifications.
//
// One request = one Context: the cycle service snapshot (profile, cycle_histories and the daily
// logs of the log window, each read once), the pregnancy profile, and per-request memos of the
// legacy engine's calculations, the history digest and the smart-message result. The message
// system is fed from the same snapshot (msgSource), so the cycle is computed once per request.
package home

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/cycle/metrics"
	cycleservice "github.com/ritme/backend-go/internal/cycle/service"
	cyclestore "github.com/ritme/backend-go/internal/cycle/store"
	"github.com/ritme/backend-go/internal/enums"
	hlmodel "github.com/ritme/backend-go/internal/healthlog/model"
	hlstore "github.com/ritme/backend-go/internal/healthlog/store"
	"github.com/ritme/backend-go/internal/home/store"
	"github.com/ritme/backend-go/internal/messages"
	msgcontent "github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/messages/manager"
	msgstore "github.com/ritme/backend-go/internal/messages/store"
	"github.com/ritme/backend-go/internal/platform/civildate"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// logWindowPastDays is HomeContext::LOG_WINDOW_PAST_DAYS.
const logWindowPastDays = 29

// Deps are the data sources a Context reads.
type Deps struct {
	Queries store.Querier
	// Messages / Pregnancy are the message system's and the pregnancy domain's queries.
	Messages  msgstore.Querier
	Pregnancy pstore.Querier
	// AppURL is APP_URL (article image URLs).
	AppURL string
	Logger *slog.Logger
	// Plus renders the Ritme Plus trial banner of /home/cycle-overview (nil = always null).
	Plus TrialBanner
}

// TrialBanner is the Plus port of the home (internal/plus.Service): the trial-offer banner of the user at now,
// null when no offer runs (B-N2-06).
type TrialBanner interface {
	TrialBannerJSON(ctx context.Context, userID uint64, now time.Time, locale, defaultLocale string) (any, error)
}

// Context is HomeContext: the request's inputs plus memoised derived data.
type Context struct {
	Ctx      context.Context
	UserID   uint64
	UserName sql.NullString
	// Date is the context date (?date or today); Today is the request's Tehran day.
	Date, Today civildate.Date
	Locale      string
	// DefaultLocale is the default language code (Translatable::pick fallback).
	DefaultLocale string
	Mode          enums.MessageMode
	Pregnancy     *pstore.PregnancyProfile
	// Snapshot holds the profile, the history and the daily logs of LogWindow.
	Snapshot *cycleservice.Snapshot

	deps *Deps

	cycleByDate    map[civildate.Date]legacy.Calculation
	calendarByDate map[civildate.Date]legacy.Calculation
	logsByRange    map[[2]civildate.Date][]cyclestore.DailyHealthLog
	digest         *Digest
	messagesLoaded bool
	messages       *manager.Result
}

// LogWindow is HomeContext::logWindow(): 29 days before date through the end of its
// Saturday-to-Friday week.
func LogWindow(date civildate.Date) (civildate.Date, civildate.Date) {
	from := date.AddDays(-logWindowPastDays)
	to := date.EndOfWeek()
	if to.Before(date) {
		to = date
	}
	return from, to
}

func newContext(ctx context.Context, d *Deps) *Context {
	return &Context{
		Ctx:            ctx,
		deps:           d,
		cycleByDate:    map[civildate.Date]legacy.Calculation{},
		calendarByDate: map[civildate.Date]legacy.Calculation{},
		logsByRange:    map[[2]civildate.Date][]cyclestore.DailyHealthLog{},
	}
}

// q is the home store.
func (hc *Context) q() store.Querier { return hc.deps.Queries }

// Profile is $user->profile (nil when none).
func (hc *Context) Profile() *cyclestore.UserProfile {
	if hc.Snapshot == nil {
		return nil
	}
	return hc.Snapshot.Profile
}

// IsFa is `$locale === 'fa'`.
func (hc *Context) IsFa() bool { return hc.Locale == "fa" }

// T is HomeContext::t(): the Persian text for fa, English for every other locale.
func (hc *Context) T(fa, en string) string {
	if hc.IsFa() {
		return fa
	}
	return en
}

var persianDigits = []rune("۰۱۲۳۴۵۶۷۸۹")

// Num is HomeContext::num(): (string) $value, in Persian digits for fa.
func (hc *Context) Num(v any) string {
	text := phpNumberString(v)
	if !hc.IsFa() {
		return text
	}
	out := make([]rune, 0, len(text))
	for _, r := range text {
		if r >= '0' && r <= '9' {
			r = persianDigits[r-'0']
		}
		out = append(out, r)
	}
	return string(out)
}

// IsCycleMode is `$mode === MessageMode::CYCLE`.
func (hc *Context) IsCycleMode() bool { return hc.Mode == enums.MessageModeCycle }

// HasCycleData is `$profile && $profile->last_period_start`.
func (hc *Context) HasCycleData() bool {
	p := hc.Profile()
	return p != nil && p.LastPeriodStart.Valid
}

// engine is healthEngine(): the legacy engine over the snapshot.
func (hc *Context) engine() *legacy.Engine { return hc.Snapshot.Engine() }

// CycleDataFor is cycleDataFor(): the full calculation for date (memoised).
func (hc *Context) CycleDataFor(date civildate.Date) (legacy.Calculation, error) {
	if c, ok := hc.cycleByDate[date]; ok {
		return c, nil
	}
	c, err := hc.engine().CalculateForDate(hc.Ctx, date, true)
	if err != nil {
		return legacy.Calculation{}, err
	}
	hc.cycleByDate[date] = c
	return c, nil
}

// CalendarDataFor is calendarDataFor(): calendar fields for date, reusing a full calculation.
func (hc *Context) CalendarDataFor(date civildate.Date) (legacy.Calculation, error) {
	if c, ok := hc.cycleByDate[date]; ok {
		return c, nil
	}
	if c, ok := hc.calendarByDate[date]; ok {
		return c, nil
	}
	c, err := hc.engine().CalculateForDate(hc.Ctx, date, false)
	if err != nil {
		return legacy.Calculation{}, err
	}
	hc.calendarByDate[date] = c
	return c, nil
}

// CycleData is cycleData(): the calculation for the context date, nil without cycle data.
func (hc *Context) CycleData() (*legacy.Calculation, error) {
	if !hc.HasCycleData() {
		return nil, nil
	}
	c, err := hc.CycleDataFor(hc.Date)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Phase is phase() ("" = null).
func (hc *Context) Phase() (string, error) {
	c, err := hc.CycleData()
	if err != nil || c == nil || !c.Complete {
		return "", err
	}
	return string(c.Phase), nil
}

// Subphase is subphase() ("" = null).
func (hc *Context) Subphase() (string, error) {
	c, err := hc.CycleData()
	if err != nil || c == nil || !c.Complete {
		return "", err
	}
	return string(c.CurrentSubphase), nil
}

// CycleDay is cycleDay() (nil = unknown).
func (hc *Context) CycleDay() (*int, error) {
	c, err := hc.CycleData()
	if err != nil || c == nil || !c.Complete {
		return nil, err
	}
	d := c.Day
	return &d, nil
}

// CurrentCycleStart is currentCycleStart(): date − (cycle_day − 1), ok=false when unknown.
func (hc *Context) CurrentCycleStart() (civildate.Date, bool, error) {
	day, err := hc.CycleDay()
	if err != nil || day == nil {
		return civildate.Date{}, false, err
	}
	return hc.Date.AddDays(-(*day - 1)), true, nil
}

// LogsBetween is logsBetween(): the window's logs with from <= log_date <= to, ascending. Every
// range the sections ask for lies inside the preloaded window.
func (hc *Context) LogsBetween(from, to civildate.Date) []cyclestore.DailyHealthLog {
	key := [2]civildate.Date{from, to}
	if logs, ok := hc.logsByRange[key]; ok {
		return logs
	}
	out := []cyclestore.DailyHealthLog{}
	for _, r := range hc.Snapshot.LogRows {
		if !r.LogDate.Before(from) && !r.LogDate.After(to) {
			out = append(out, r)
		}
	}
	hc.logsByRange[key] = out
	return out
}

// RecentLogs is recentLogs($days): the last days days up to the context date.
func (hc *Context) RecentLogs(days int) []cyclestore.DailyHealthLog {
	return hc.LogsBetween(hc.Date.AddDays(-(days - 1)), hc.Date)
}

// DailyLog is dailyLog(): the context date's log (nil when none).
func (hc *Context) DailyLog() *cyclestore.DailyHealthLog {
	for i := range hc.Snapshot.LogRows {
		if hc.Snapshot.LogRows[i].LogDate == hc.Date {
			return &hc.Snapshot.LogRows[i]
		}
	}
	return nil
}

// HistoryDigest is cycleHistoryDigest().
func (hc *Context) HistoryDigest() *Digest {
	if hc.digest == nil {
		hc.digest = NewDigest(hc.Snapshot.Histories)
	}
	return hc.digest
}

// Metrics is cycleMetrics(): the engine's three-layer metrics (same history and profile).
func (hc *Context) Metrics() metrics.Metrics { return hc.engine().Metrics() }

// Messages is messages(): the smart-message result, nil when it cannot be produced (failures
// are logged and swallowed, like the PHP try/catch).
func (hc *Context) Messages() *manager.Result {
	if hc.messagesLoaded {
		return hc.messages
	}
	hc.messagesLoaded = true
	if hc.IsCycleMode() && !hc.HasCycleData() {
		return nil
	}
	src := &msgSource{
		StoreSource: messages.NewStoreSource(hc.deps.Messages, hc.deps.Pregnancy, hc.UserID, hc.Today),
		hc:          hc,
	}
	m := manager.New(src, msgcontent.NewRepository(hc.deps.Messages), hc.Locale, hc.Today)
	res, err := m.Generate(hc.Ctx, hc.Date, hc.Mode)
	if err != nil {
		hc.logger().WarnContext(hc.Ctx, "HomeContext: message generation failed",
			slog.Uint64("user_id", hc.UserID), slog.String("error", err.Error()))
		return nil
	}
	hc.messages = res
	return res
}

func (hc *Context) logger() *slog.Logger {
	if hc.deps != nil && hc.deps.Logger != nil {
		return hc.deps.Logger
	}
	return slog.Default()
}

// scored maps log rows onto the scorer's view.
func scored(rows []cyclestore.DailyHealthLog) []scoredLog {
	out := make([]scoredLog, len(rows))
	for i, r := range rows {
		out[i] = scoreView(r)
	}
	return out
}

func scoreView(r cyclestore.DailyHealthLog) scoredLog {
	l := hlmodel.FromRow(hlstore.DailyHealthLog(r))
	return scoredLog{
		Moods:         l.Strings("moods"),
		SleepQuality:  l.Str("sleep_quality"),
		SleepDuration: l.Str("sleep_duration"),
		EnergyLevel:   l.Str("energy_level"),
		Fatigue:       l.Bool("fatigue"),
	}
}

// ---------------------------------------------------------------------------
// msgSource feeds the message manager from the home snapshot.

// msgSource is manager.Source over the home context: the profile, pregnancy profile, the day's
// log and the legacy calculation come from the snapshot (the cycle is computed once); the
// 90-day log history is the message system's own query.
type msgSource struct {
	*messages.StoreSource
	hc *Context
}

var _ manager.Source = (*msgSource)(nil)

// Profile implements manager.Source.
func (s *msgSource) Profile(context.Context) (*manager.Profile, error) {
	p := s.hc.Profile()
	if p == nil {
		return nil, nil
	}
	return &manager.Profile{
		UserGoal:           p.UserGoal,
		SubscriptionType:   p.SubscriptionType,
		HasLastPeriodStart: p.LastPeriodStart.Valid,
	}, nil
}

// PregnancyProfile implements manager.Source.
func (s *msgSource) PregnancyProfile(context.Context) (*pstore.PregnancyProfile, error) {
	return s.hc.Pregnancy, nil
}

// DailyLog implements manager.Source: the engine's preloaded log (dailyLogFor).
func (s *msgSource) DailyLog(ctx context.Context, date civildate.Date) (*manager.Log, error) {
	if date.Before(s.hc.Snapshot.From) || date.After(s.hc.Snapshot.To) {
		return s.StoreSource.DailyLog(ctx, date)
	}
	for _, r := range s.hc.Snapshot.LogRows {
		if r.LogDate == date {
			l := manager.NewLog(map[string]any{
				"energy_level":  nullStr(r.EnergyLevel),
				"sleep_quality": nullStr(r.SleepQuality),
			})
			return &l, nil
		}
	}
	return nil, nil
}

// Cycle implements manager.Source: the home context's memoised calculation.
func (s *msgSource) Cycle(_ context.Context, date civildate.Date) (legacy.Calculation, error) {
	return s.hc.CycleDataFor(date)
}

func nullStr(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}
