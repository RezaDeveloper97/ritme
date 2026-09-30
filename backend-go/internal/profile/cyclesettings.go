package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/ritme/backend-go/db"
	"github.com/ritme/backend-go/internal/auth"
	cyclemetrics "github.com/ritme/backend-go/internal/cycle/metrics"
	cyclemodel "github.com/ritme/backend-go/internal/cycle/model"
	"github.com/ritme/backend-go/internal/i18n"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/notifications"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/clock"
	"github.com/ritme/backend-go/internal/platform/httpx"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/profile/store"
	"github.com/ritme/backend-go/internal/reminders"
)

// Cycle settings (B-N1-09, `nbl_Cycle_Settings`): «سیکل تو» (cycle / period length, «خودکار از داده‌ها» = the
// engine's medians vs the manual profile values) and «یادآورها» (before period, PMS, fertile window, daily log,
// pill — switches in notification_preferences.categories, times in .schedule, both read by every sender through
// internal/notifications and planned by internal/reminders). Go only: GET/PUT /api/v1/profile/cycle-settings.

// Manual length bounds — the POST /profile rules for cycle_duration / period_duration.
const (
	minManualCycle  = 15
	maxManualCycle  = 60
	minManualPeriod = 1
	maxManualPeriod = 15
)

// CS is the cycle-settings line for key ("messages.saved") in locale (lang/<code>/cycle_settings.json).
func CS(key, locale string) string {
	return privacyTranslator().Trans("cycle_settings."+key, nil, locale)
}

func cycleSettingsAttributes(locale string) []string {
	line, ok := privacyTranslator().Get("cycle_settings.attributes", locale)
	m, isMap := line.(phpval.Map)
	if !ok || !isMap {
		return nil
	}
	var kv []string
	for _, k := range m.Keys() {
		if v, ok := m.Get(k); ok {
			if s, ok := v.(string); ok {
				kv = append(kv, k, s)
			}
		}
	}
	return kv
}

// CycleSettingsStore is the persistence the endpoints read and write (profile store, user_id in every query).
type CycleSettingsStore interface {
	notifications.Getter
	GetProfileByUserID(ctx context.Context, userID uint64) (store.UserProfile, error)
	ListCycleHistories(ctx context.Context, userID uint64) ([]store.CycleHistory, error)
	GetCyclePreferences(ctx context.Context, userID uint64) (store.CyclePreference, error)
	UpsertCyclePreferences(ctx context.Context, arg store.UpsertCyclePreferencesParams) error
	UpsertReminderPreferences(ctx context.Context, arg store.UpsertReminderPreferencesParams) error
}

// LengthSaver writes manual cycle_duration / period_duration the way POST /profile does (dirty check,
// calculation_version bump so the engine cache is dropped).
type LengthSaver interface {
	Save(ctx context.Context, u *auth.User, input phpval.Map, now time.Time) (*SaveResult, error)
}

// CycleSettingsHandlers are GET/PUT /profile/cycle-settings. Mount behind locale + auth RequireUser.
type CycleSettingsHandlers struct {
	q     CycleSettingsStore
	save  LengthSaver
	clock clock.Clock
}

// NewCycleSettingsHandlers wires the handlers on d (base = fallback clock; tests pin it per request).
func NewCycleSettingsHandlers(d *sql.DB, base clock.Clock) *CycleSettingsHandlers {
	q := store.New(d)
	if base == nil {
		base = clock.Real{}
	}
	return &CycleSettingsHandlers{q: q, save: &Service{DB: d, Q: q}, clock: base}
}

func (h *CycleSettingsHandlers) now(c fiber.Ctx) time.Time {
	return clock.FromContext(c, h.clock).Now().In(civildate.Tehran).Truncate(time.Second)
}

// cycleState is everything the response is built from.
type cycleState struct {
	auto    bool
	metrics cyclemetrics.Metrics
	prefs   notifications.Preferences
}

func (h *CycleSettingsHandlers) load(ctx context.Context, userID uint64) (cycleState, error) {
	st := cycleState{auto: true}
	cp, err := h.q.GetCyclePreferences(ctx, userID)
	switch {
	case err == nil:
		st.auto = cp.LengthsAuto
	case !errors.Is(err, sql.ErrNoRows):
		return st, fmt.Errorf("profile: cycle preferences: %w", err)
	}
	var prof *cyclemodel.Profile
	row, err := h.q.GetProfileByUserID(ctx, userID)
	switch {
	case err == nil:
		prof = cycleProfile(row)
	case !errors.Is(err, sql.ErrNoRows):
		return st, fmt.Errorf("profile: load: %w", err)
	}
	rows, err := h.q.ListCycleHistories(ctx, userID)
	if err != nil {
		return st, fmt.Errorf("profile: cycle histories: %w", err)
	}
	if prof != nil {
		prof.LengthsManual = !st.auto // the same resolution the engine runs (cycle/service Snapshot)
	}
	st.metrics = cyclemetrics.Calculate(cycleHistories(rows), prof)
	if st.prefs, err = notifications.Load(ctx, h.q, userID); err != nil {
		return st, err
	}
	return st, nil
}

// cycleProfile / cycleHistories map the profile store rows onto the engine's pure inputs (the same casts as
// cycle/service's adapters, which this package cannot import).
func cycleProfile(r store.UserProfile) *cyclemodel.Profile {
	p := &cyclemodel.Profile{Goal: r.UserGoal}
	if r.LastPeriodStart.Valid {
		p.LastPeriodStart = r.LastPeriodStart.Date
	}
	if r.CycleDuration.Valid {
		p.CycleDuration = cyclemodel.Int(int(r.CycleDuration.Int16))
	}
	if r.PeriodDuration.Valid {
		p.PeriodDuration = cyclemodel.Int(int(r.PeriodDuration.Int16))
	}
	return p
}

func cycleHistories(rows []store.CycleHistory) []cyclemodel.History {
	out := make([]cyclemodel.History, len(rows))
	for i, r := range rows {
		h := cyclemodel.History{
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
			h.CycleLength = cyclemodel.Int(int(r.CycleLength.Int32))
		}
		if r.BleedingLength.Valid {
			h.BleedingLength = cyclemodel.Int(int(r.BleedingLength.Int32))
		}
		if r.DataQualityFlags.Valid {
			var flags []string
			if json.Unmarshal(r.DataQualityFlags.V, &flags) == nil {
				h.DataQualityFlags = flags
			}
		}
		out[i] = h
	}
	return out
}

func intOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// CycleSettingsJSON is the response body.
func CycleSettingsJSON(st cycleState) *jsonx.OrderedMap {
	m := st.metrics
	var basedOn any
	if m.ValidCyclesCount > 0 {
		basedOn = m.ValidCyclesCount
	}
	cycleLen := m.EffectiveCycleLength // manual → profile first (metrics.resolveManual)
	items := make([]any, 0, len(notifications.Timed))
	for _, cat := range notifications.Timed {
		minute, _ := st.prefs.TimeOf(cat)
		kv := []any{"code", string(cat), "enabled", st.prefs.Enabled(cat), "time", notifications.FormatClock(minute)}
		switch cat {
		case notifications.BeforePeriod:
			kv = append(kv, "days_before", st.prefs.DaysBeforePeriod())
		case notifications.PMS:
			kv = append(kv, "cycle_day", reminders.PMSCycleDay(cycleLen))
		}
		items = append(items, jsonx.Obj(kv...))
	}
	return jsonx.Obj(
		"lengths", jsonx.Obj(
			"auto", st.auto,
			"cycle_length", cycleLen,
			"period_length", m.EffectivePeriodDuration,
			"calculated", jsonx.Obj(
				"cycle_length", intOrNil(m.CalculatedCycleLength),
				"period_length", intOrNil(m.CalculatedPeriodDuration),
				"based_on_cycles", basedOn,
			),
			"manual", jsonx.Obj(
				"cycle_length", intOrNil(m.ProfileCycleLength),
				"period_length", intOrNil(m.ProfilePeriodDuration),
			),
		),
		"reminders", items,
	)
}

// Show is GET /profile/cycle-settings (automatic lengths and the default reminders when never saved).
func (h *CycleSettingsHandlers) Show(c fiber.Ctx) error {
	id, err := privacyUserID(c)
	if err != nil {
		return err
	}
	st, err := h.load(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, CycleSettingsJSON(st))
}

func cycleSettingsRules() validation.Rules {
	F := validation.F
	rules := validation.Rules{
		F("lengths_auto", "sometimes", "boolean"),
		F("cycle_length", "sometimes", "integer", fmt.Sprintf("min:%d", minManualCycle), fmt.Sprintf("max:%d", maxManualCycle)),
		F("period_length", "sometimes", "integer", fmt.Sprintf("min:%d", minManualPeriod), fmt.Sprintf("max:%d", maxManualPeriod)),
		F("reminders", "sometimes", "array"),
		F("reminders.before_period.days_before", "sometimes", "integer",
			fmt.Sprintf("min:%d", notifications.MinDaysBefore), fmt.Sprintf("max:%d", notifications.MaxDaysBefore)),
	}
	for _, cat := range notifications.Timed {
		k := "reminders." + string(cat)
		rules = append(rules,
			F(k, "sometimes", "array"),
			F(k+".enabled", "sometimes", "boolean"),
			F(k+".time", "sometimes", "date_format:H:i"),
		)
	}
	return rules
}

// applyReminders returns p with the validated reminder keys of body applied (omitted keys keep their value).
func applyReminders(p notifications.Preferences, body phpval.Map) (notifications.Preferences, bool) {
	out := p
	out.Categories = make(map[notifications.Category]bool, len(p.Categories)+1)
	for k, v := range p.Categories {
		out.Categories[k] = v
	}
	out.Schedule = notifications.Schedule{Times: make(map[notifications.Category]int, len(p.Schedule.Times)+1), DaysBefore: p.Schedule.DaysBefore}
	for k, v := range p.Schedule.Times {
		out.Schedule.Times[k] = v
	}
	changed := false
	for _, cat := range notifications.Timed {
		k := "reminders." + string(cat)
		if v, ok := phpval.Get(body, k+".enabled"); ok {
			out.Categories[cat] = phpval.Truthy(v)
			changed = true
		}
		if v, ok := phpval.Get(body, k+".time"); ok {
			if m, ok := notifications.ParseClock(phpval.ToString(v)); ok {
				out.Schedule.Times[cat] = m
				changed = true
			}
		}
	}
	if v, ok := phpval.Get(body, "reminders.before_period.days_before"); ok {
		out.Schedule.DaysBefore = int(phpval.ToFloat(v))
		changed = true
	}
	return out, changed
}

// Update is PUT /profile/cycle-settings, a partial update:
//
//	{lengths_auto?, cycle_length?, period_length?, reminders?: {<code>: {enabled?, time?, days_before?}}}
//
// cycle_length / period_length are the manual values (user_profiles, like POST /profile); then the settings.
func (h *CycleSettingsHandlers) Update(c fiber.Ctx) error {
	u := auth.CurrentUser(c)
	if u == nil {
		return &auth.UnauthenticatedError{Code: auth.CodeUnauthenticated}
	}
	locale := i18n.Locale(c)
	now := h.now(c)
	body := validation.Input(c)
	v := validation.Make(lang.Default(), locale, body, cycleSettingsRules(),
		validation.Now(now), validation.Attributes(cycleSettingsAttributes(locale)...))
	if v.Fails() {
		return httpx.Fail(fiber.StatusUnprocessableEntity, CS("messages.validation_failed", locale), "errors", v.ErrorBag())
	}

	at := sql.NullTime{Time: now, Valid: true}
	if val, ok := phpval.Get(body, "lengths_auto"); ok {
		if err := h.q.UpsertCyclePreferences(c, store.UpsertCyclePreferencesParams{
			UserID: u.ID, LengthsAuto: phpval.Truthy(val), Now: at,
		}); err != nil {
			return fmt.Errorf("profile: save cycle preferences: %w", err)
		}
	}
	lengths := phpval.NewMap()
	if val, ok := phpval.Get(body, "cycle_length"); ok {
		lengths.Set("cycle_duration", val)
	}
	if val, ok := phpval.Get(body, "period_length"); ok {
		lengths.Set("period_duration", val)
	}
	if lengths.Len() > 0 {
		if _, err := h.save.Save(c, u, lengths, now); err != nil {
			return err
		}
	}

	cur, err := notifications.Load(c, h.q, u.ID)
	if err != nil {
		return err
	}
	if next, changed := applyReminders(cur, body); changed {
		if err := h.q.UpsertReminderPreferences(c, store.UpsertReminderPreferencesParams{
			UserID:            u.ID,
			Categories:        db.NullRawJSON{V: next.CategoriesJSON(), Valid: true},
			Schedule:          db.NullRawJSON{V: next.ScheduleJSON(), Valid: true},
			QuietHoursEnabled: next.QuietEnabled,
			QuietStart:        notifications.DBClock(next.QuietStart),
			QuietEnd:          notifications.DBClock(next.QuietEnd),
			NeutralCopy:       next.NeutralCopy,
			Now:               at,
		}); err != nil {
			return fmt.Errorf("profile: save reminder preferences: %w", err)
		}
	}

	st, err := h.load(c, u.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, CycleSettingsJSON(st), CS("messages.saved", locale))
}
