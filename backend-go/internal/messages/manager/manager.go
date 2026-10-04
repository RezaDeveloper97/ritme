// Package manager is the smart-message system (backend/app/Services/MessageSystem): the
// MessageManager, the cycle and pregnancy engines, the correlation and pattern layers and the
// nutrition / sleep / exercise modules. It is pure apart from its two inputs: a Source (the
// user's profile, pregnancy profile, logs and the legacy cycle calculation) and the
// request-scoped content repository.
//
// Typical use (GET /messages/daily, the home page's smart tip):
//
//	m := manager.New(src, content.NewRepository(q), locale, today)
//	res, err := m.Generate(ctx, date, "")   // "" = detect the mode
//	res.JSON()                               // MessageResult::toArray()
//
// Laravel quirks kept (D-05, D-11): the symptom and pattern detectors read columns that do not
// exist, so only low_energy / insufficient_data / ttc_tracking / chronic_fatigue can fire, and a
// low-energy day in cycle mode fails with ErrUndefinedOverrideCase (HTTP 500).
package manager

import (
	"context"
	"fmt"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/legacy"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/content"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	pstore "github.com/ritme/backend-go/internal/pregnancy/store"
)

// Content resolves one entry of message_contents with its code fallback
// (*content.Repository).
type Content interface {
	Resolve(ctx context.Context, group, itemKey, locale string) (content.Payload, error)
}

// Profile is what the message system reads off $user->profile.
type Profile struct {
	UserGoal         string
	SubscriptionType string
	// HasLastPeriodStart is `(bool) $profile->last_period_start`.
	HasLastPeriodStart bool
}

// Source is the user's data. Implementations may memoise (the manager asks at most once per
// method and date).
type Source interface {
	// Profile is $user->profile (nil when the user has none).
	Profile(ctx context.Context) (*Profile, error)
	// PregnancyProfile is $user->pregnancyProfile (nil when none).
	PregnancyProfile(ctx context.Context) (*pstore.PregnancyProfile, error)
	// DailyLog is HealthDataEngine::dailyLogFor($date) (nil when none). At least the
	// LoadedLogColumns must be loaded.
	DailyLog(ctx context.Context, date civildate.Date) (*Log, error)
	// RecentLogs are the logs with from ≤ log_date ≤ to, newest first.
	RecentLogs(ctx context.Context, from, to civildate.Date) ([]Log, error)
	// Cycle is HealthDataEngine::calculateForDate($date); only the phase, subphase, cycle day,
	// cycle length, ovulation day and the fertile / PMS flags are read, so calendar mode is
	// enough.
	Cycle(ctx context.Context, date civildate.Date) (legacy.Calculation, error)
}

// PregnancySymptomSource is an optional Source extension (T-M7-04): in pregnancy mode the
// override layer reads the day's pregnancy_symptom_logs (+ pregnancy_daily_extras) as symptom
// names ExtractSymptoms understands (nausea, vomiting, fatigue, backache, mood_sad, …).
type PregnancySymptomSource interface {
	PregnancySymptoms(ctx context.Context, date civildate.Date) ([]string, error)
}

// LifeModeSource is an optional Source extension (B-N2-01): the stored life-stage mode
// (user_life_profiles.life_mode; "" = none, i.e. every user created before it). A stored postpartum mode
// picks the engine-less postpartum mode in DetectMode; menopause and teen run on the cycle engine with the
// safe defaults of enums.LifeMode.AllowsFertilityContent (never TTC content, whatever user_goal says).
type LifeModeSource interface {
	LifeMode(ctx context.Context) (string, error)
}

// storedLifeMode is the source's stored life mode ("" when the source has none or does not know it).
func (m *Manager) storedLifeMode(ctx context.Context) (enums.LifeMode, error) {
	ls, ok := m.src.(LifeModeSource)
	if !ok {
		return "", nil
	}
	v, err := ls.LifeMode(ctx)
	return enums.LifeMode(v), err
}

// Manager is one MessageManager (user + locale). Today is the request's Tehran day: the
// pregnancy context is computed for today whatever date is asked (PregnancyCalculationService
// uses Carbon::today()).
type Manager struct {
	src     Source
	content Content
	locale  string
	today   civildate.Date
}

// New returns a manager.
func New(src Source, c Content, locale string, today civildate.Date) *Manager {
	return &Manager{src: src, content: c, locale: locale, today: today}
}

// DetectMode is MessageManager::detectMode(): pregnancy when the pregnancy profile has
// pregnancy_mode, else cycle.
func (m *Manager) DetectMode(ctx context.Context) (enums.MessageMode, error) {
	p, err := m.src.PregnancyProfile(ctx)
	if err != nil {
		return "", err
	}
	if p != nil && p.PregnancyMode {
		return enums.MessageModePregnancy, nil
	}
	// B-N2-01: a stored postpartum mode has the PHP enum's (engine-less) postpartum mode; every other stored
	// mode runs on the cycle engine. No stored mode = Laravel's detection.
	lm, err := m.storedLifeMode(ctx)
	if err != nil {
		return "", err
	}
	if lm == enums.LifeModePostpartum {
		return enums.MessageModePostpartum, nil
	}
	return enums.MessageModeCycle, nil
}

// BuildContext is MessageManager::buildContext(). force "" detects the mode.
func (m *Manager) BuildContext(ctx context.Context, date civildate.Date, force enums.MessageMode) (*Context, error) {
	mode := force
	if mode == "" {
		var err error
		if mode, err = m.DetectMode(ctx); err != nil {
			return nil, err
		}
	}
	profile, err := m.src.Profile(ctx)
	if err != nil {
		return nil, err
	}
	mc := &Context{Locale: m.locale, Date: date, Mode: mode, UserGoal: "non_ttc", SubscriptionType: "free"}
	if profile != nil {
		mc.UserGoal, mc.SubscriptionType = profile.UserGoal, profile.SubscriptionType
	}
	lm, err := m.storedLifeMode(ctx)
	if err != nil {
		return nil, err
	}
	if lm != "" && !lm.AllowsFertilityContent() { // B-N2-01: menopause / teen never get TTC content
		mc.UserGoal = string(enums.UserGoalNonTtc)
	}
	if mc.DailyLog, err = m.src.DailyLog(ctx, date); err != nil {
		return nil, err
	}
	if mc.RecentLogs, err = m.src.RecentLogs(ctx, date.AddDays(-90), date); err != nil {
		return nil, err
	}
	mc.Symptoms = ExtractSymptoms(mc.DailyLog)

	if mode == enums.MessageModePregnancy {
		if ps, ok := m.src.(PregnancySymptomSource); ok {
			extra, err := ps.PregnancySymptoms(ctx, date)
			if err != nil {
				return nil, err
			}
			for _, s := range extra {
				if !mc.HasSymptom(s) {
					mc.Symptoms = append(mc.Symptoms, s)
				}
			}
		}
		return mc, m.pregnancyContext(ctx, mc)
	}
	if err := m.cycleContext(ctx, mc); err != nil {
		return mc, err
	}
	if lm != "" && !lm.AllowsFertilityContent() { // CB-TEEN-04b: no ovulation / fertile-window message either
		mc.withoutFertility()
	}
	return mc, nil
}

// fertileSubphases are the sub-phases whose labels name fertility or ovulation.
var fertileSubphases = []enums.CycleSubphase{
	enums.CycleSubphaseFertileRising, enums.CycleSubphaseHighFertility,
	enums.CycleSubphaseOvulationLikely, enums.CycleSubphasePostOvulation,
}

// withoutFertility (CB-TEEN-04b, extends B-N2-11b's NoFertilityCopy) is the day as a mode without fertility
// content (teen, menopause) reads it: the ovulation phase is the neutral follicular / luteal phase (so the
// base message and the nutrition / sleep / exercise modules never pick their ovulation copy), a
// fertility-worded sub-phase is null, and there is no fertile window or ovulation day.
func (mc *Context) withoutFertility() {
	if mc.CycleDay != nil && mc.EstimatedOvulationDay != nil {
		mc.CyclePhase = string(legacy.NeutralPhase(enums.CyclePhase(mc.CyclePhase), *mc.CycleDay, *mc.EstimatedOvulationDay))
	}
	if slices.Contains(fertileSubphases, enums.CycleSubphase(mc.CycleSubphase)) {
		mc.CycleSubphase = ""
	}
	mc.IsFertileWindow = false
	mc.EstimatedOvulationDay = nil
}

// cycleContext is buildCycleContext: the legacy engine's day.
func (m *Manager) cycleContext(ctx context.Context, mc *Context) error {
	c, err := m.src.Cycle(ctx, mc.Date)
	if err != nil {
		return err
	}
	if !c.Complete { // getEmptyCalculation: nulls and false flags
		return nil
	}
	day, length, ovulation := c.Day, c.CycleLength, c.OvulationDay
	// D-28: the §19 display window, shared with /cycle/today|date (D-30).
	phase, fertile := legacy.DisplayWindow(c.Phase, c.IsFertileWindow, day, ovulation)
	mc.CyclePhase, mc.CycleSubphase = string(phase), string(c.CurrentSubphase)
	mc.CycleDay, mc.CycleLength, mc.EstimatedOvulationDay = &day, &length, &ovulation
	mc.IsFertileWindow, mc.IsPmsWindow = fertile, c.IsPmsWindow
	return nil
}

// pregnancyContext is buildPregnancyContext: getPregnancyStatus() for today; the week is the
// 0-based gestational_age.weeks, not current_week.
func (m *Manager) pregnancyContext(ctx context.Context, mc *Context) error {
	p, err := m.src.PregnancyProfile(ctx)
	if err != nil {
		return err
	}
	cl := calc.New(p, m.locale, m.today)
	if ga := cl.GestationalAge(); ga.Valid {
		weeks, days, trimester := ga.Weeks, ga.Days, ga.Trimester
		mc.PregnancyWeek, mc.PregnancyDay, mc.Trimester = &weeks, &days, &trimester
	}
	if due := cl.EDD(); due != nil {
		d := due.Date
		mc.DueDate = &d
	}
	return nil
}

// Result is MessageResult (backend/app/Services/MessageSystem/Core/MessageResult.php).
type Result struct {
	Mode             enums.MessageMode
	Date             civildate.Date
	UserGoal         string
	SubscriptionType string
	ContextInfo      *jsonx.OrderedMap
	PrimaryMessage   *jsonx.OrderedMap
	Correlations     []*jsonx.OrderedMap
	Patterns         []*jsonx.OrderedMap
	Nutrition        *jsonx.OrderedMap
	Sleep            *jsonx.OrderedMap
	Exercise         *jsonx.OrderedMap
}

// JSON is MessageResult::toArray() (`tips` is always []).
func (r *Result) JSON() *jsonx.OrderedMap {
	orEmpty := func(m *jsonx.OrderedMap) *jsonx.OrderedMap {
		if m == nil {
			return jsonx.NewArray()
		}
		return m
	}
	return jsonx.Obj(
		"mode", string(r.Mode),
		"date", r.Date.String(),
		"user_goal", r.UserGoal,
		"subscription_type", r.SubscriptionType,
		"context_info", orEmpty(r.ContextInfo),
		"primary_message", orEmpty(r.PrimaryMessage),
		"correlations", nonNil(r.Correlations),
		"patterns", nonNil(r.Patterns),
		"supplements", jsonx.Obj(
			"nutrition", orEmpty(r.Nutrition),
			"sleep", orEmpty(r.Sleep),
			"exercise", orEmpty(r.Exercise),
		),
		"tips", emptyList(),
	)
}

func nonNil(s []*jsonx.OrderedMap) []*jsonx.OrderedMap {
	if s == nil {
		return []*jsonx.OrderedMap{}
	}
	return s
}

// Generate is MessageManager::generateMessages(). force "" detects the mode.
func (m *Manager) Generate(ctx context.Context, date civildate.Date, force enums.MessageMode) (*Result, error) {
	mode := force
	if mode == "" {
		var err error
		if mode, err = m.DetectMode(ctx); err != nil {
			return nil, err
		}
	}
	type engine interface {
		base(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error)
		override(ctx context.Context, mc *Context) (*jsonx.OrderedMap, error)
	}
	var eng engine
	switch mode {
	case enums.MessageModeCycle:
		eng = cycleEngine{content: m.content, locale: m.locale}
	case enums.MessageModePregnancy:
		eng = pregnancyEngine{content: m.content, locale: m.locale}
	case enums.MessageModePostpartum: // bloom B-N5-01 (D-54): Go's postpartum engine, see postpartum_engine.go
		return m.postpartum(ctx, date)
	default:
		// No engine (a mode without one): MessageResult::empty(). buildContext has no observable
		// effect for such a mode, so it is skipped.
		msg := "Message engine not found for this mode"
		if m.locale == "fa" {
			msg = "موتور پیام برای این حالت یافت نشد"
		}
		return &Result{
			Mode: mode, Date: date, UserGoal: "non_ttc", SubscriptionType: "free",
			ContextInfo: jsonx.Obj("error", msg),
		}, nil
	}

	mc, err := m.BuildContext(ctx, date, mode)
	if err != nil {
		return nil, err
	}

	primary, err := eng.base(ctx, mc)
	if err != nil {
		return nil, fmt.Errorf("messages: base message: %w", err)
	}
	over, err := eng.override(ctx, mc)
	if err != nil {
		return nil, fmt.Errorf("messages: override message: %w", err)
	}
	if over != nil && over.Len() > 0 { // array_merge($base, $override, ['has_override' => true])
		for _, k := range over.Keys() {
			v, _ := over.Get(k)
			primary.Set(k, v)
		}
		primary.Set("has_override", true)
	} else {
		primary.Set("has_override", false)
	}

	correlations, err := correlationLayer{content: m.content, locale: m.locale}.analyze(ctx, mc)
	if err != nil {
		return nil, err
	}
	if !mc.IsPremium() {
		kept := []*jsonx.OrderedMap{}
		for _, c := range correlations {
			if v, _ := c.Get("is_premium_only"); v != true {
				kept = append(kept, c)
			}
		}
		correlations = kept
	}

	patterns := []*jsonx.OrderedMap{}
	if mc.IsPremium() {
		if patterns, err = (patternLayer{content: m.content, locale: m.locale}).analyze(ctx, mc); err != nil {
			return nil, err
		}
	}

	mods := modules{content: m.content, locale: m.locale}
	res := &Result{
		Mode: mc.Mode, Date: date, UserGoal: mc.UserGoal, SubscriptionType: mc.SubscriptionType,
		ContextInfo:    m.contextInfo(mc),
		PrimaryMessage: primary,
		Correlations:   correlations,
		Patterns:       patterns,
	}
	if res.Nutrition, err = mods.nutrition(ctx, mc); err != nil {
		return nil, err
	}
	if res.Sleep, err = mods.sleep(ctx, mc); err != nil {
		return nil, err
	}
	if res.Exercise, err = mods.exercise(ctx, mc); err != nil {
		return nil, err
	}
	return res, nil
}

// contextInfo is buildContextInfo().
func (m *Manager) contextInfo(mc *Context) *jsonx.OrderedMap {
	if mc.IsCycleMode() {
		var phase, phaseLabel, sub, subLabel any
		if mc.CyclePhase != "" {
			phase, phaseLabel = mc.CyclePhase, enums.CyclePhase(mc.CyclePhase).Label(m.locale)
		}
		if mc.CycleSubphase != "" {
			sub, subLabel = mc.CycleSubphase, enums.CycleSubphase(mc.CycleSubphase).Label(m.locale)
		}
		return jsonx.Obj(
			"phase", phase,
			"phase_label", phaseLabel,
			"subphase", sub,
			"subphase_label", subLabel,
			"cycle_day", intOrNil(mc.CycleDay),
			"cycle_length", intOrNil(mc.CycleLength),
			"is_fertile_window", mc.IsFertileWindow,
			"is_pms_window", mc.IsPmsWindow,
			"estimated_ovulation_day", intOrNil(mc.EstimatedOvulationDay),
		)
	}
	var due any
	if mc.DueDate != nil {
		due = mc.DueDate.String()
	}
	return jsonx.Obj(
		"week", intOrNil(mc.PregnancyWeek),
		"day", intOrNil(mc.PregnancyDay),
		"trimester", intOrNil(mc.Trimester),
		"gestational_age", mc.GestationalAgeString(),
		"due_date", due,
	)
}
