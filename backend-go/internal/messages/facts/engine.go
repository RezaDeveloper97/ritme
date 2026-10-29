package facts

import (
	"context"
	"slices"

	"github.com/ritme/backend-go/internal/cycle/engine"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/messages/manager"
)

// Engine facts (SPEC §2.35, §5.3; T-M9-32 state provider): one engine state load serves all of them, resolved for the
// target day (LocalToday). The cycle numbers, content phase and fertility strip are manager.EngineFacts (T-M9-38): under
// limits.fertility_suppressed (hormonal contraception, pregnancy, postpartum not returned) or a mode without fertility
// copy (teen, menopause) is_fertile_window is absent, the fertile main phase reads follicular, fertility-worded
// subphases are dropped and fertility_suppressed is true.

// EngineSource is the v6 engine state of the user, built for the request's today (state.UserStates.ForUser). A nil
// state means no engine (CYCLE_ENGINE=v1): every engine fact is then absent.
type EngineSource interface {
	EngineState(ctx context.Context) (*engine.State, error)
}

// GroupEngine is the engine producer group.
const GroupEngine = "engine"

// engineMemo is the resolved state of the request (read by the symptom history too).
type engineMemo struct {
	state  *engine.State
	status engine.Status
	strip  bool
}

func enumStrings[T interface{ String() string }](vals ...T) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, v.String())
	}
	return out
}

func allWarnings() []string {
	var out []string // Warning.String is "" past the last value
	for w := engine.Warning(0); w.String() != ""; w++ {
		out = append(out, w.String())
	}
	return out
}

var (
	engineModes = enumStrings(engine.ModeCycle, engine.ModeTTC, engine.ModePregnancy, engine.ModePostpartum,
		engine.ModeMenopause, engine.ModeTeen)
	mainPhases = enumStrings(engine.MainPhaseUnknown, engine.MainPhaseMenstrual, engine.MainPhaseFollicular,
		engine.MainPhaseFertile, engine.MainPhaseLuteal, engine.MainPhasePeriodExpected)
	// subphases are the v6 engine's and the legacy engine's (CYCLE_ENGINE=v1) values.
	subphases = func() []string {
		out := enumStrings(engine.SubphaseMenstrual, engine.SubphaseMenstrualPossible, engine.SubphaseEarlyFollicular,
			engine.SubphaseMidFollicular, engine.SubphaseLateFollicularTransition, engine.SubphaseFertileRising,
			engine.SubphaseHighFertility, engine.SubphaseOvulationLikely, engine.SubphasePostOvulation,
			engine.SubphaseEarlyLuteal, engine.SubphaseMidLuteal, engine.SubphaseLateLuteal, engine.SubphasePMSPossible,
			engine.SubphasePeriodExpected)
		for _, s := range enums.CycleSubphaseValues() {
			if !slices.Contains(out, s) && s != string(enums.CycleSubphaseUnknown) {
				out = append(out, s)
			}
		}
		return out
	}()
	contentPhases = []string{string(enums.CyclePhaseMenstruation), string(enums.CyclePhaseFollicular),
		string(enums.CyclePhaseOvulation), string(enums.CyclePhaseLuteal)}
	regularities = enumStrings(engine.RegularityInsufficientData, engine.RegularityRegular, engine.RegularityIrregular)
	qualities    = enumStrings(engine.DataQualityInsufficient, engine.DataQualityGood, engine.DataQualityPartial,
		engine.DataQualityPoor)
	confidences = enumStrings(engine.ConfidenceUnknown, engine.ConfidenceLow, engine.ConfidenceMedium,
		engine.ConfidenceHigh)
	// BannerNone reads "none" (a known «no banner», not an absent fact).
	banners = []string{BannerNone, engine.BannerEndMissing.String(), engine.BannerOverdue.String(),
		engine.BannerAskLastPeriod.String()}
)

// BannerNone is today_ui.banner when the engine shows no banner.
const BannerNone = "none"

func init() {
	g := GroupEngine
	Register(
		Def{Name: "mode", Type: TypeEnum, Enum: engineModes, Group: g, MemberSafe: true,
			Description: desc("حالت موتور (چرخه، اقدام به بارداری، بارداری، بعد از زایمان، یائسگی، نوجوان)", "The engine mode (cycle, ttc, pregnancy, postpartum, menopause, teen).")},
		Def{Name: "main_phase", Type: TypeEnum, Enum: mainPhases, Group: g,
			Description: desc("فاز اصلی امروز؛ زیر سرکوب باروری fertile به\u200cصورت follicular خوانده می\u200cشود", "Today's main phase; under fertility suppression fertile reads as follicular.")},
		Def{Name: "subphase", Type: TypeEnum, Enum: subphases, Group: g,
			Description: desc("زیرفاز امروز (زیرفازهای باروری زیر سرکوب حذف می\u200cشوند)", "Today's subphase (fertility subphases are dropped under suppression).")},
		Def{Name: "phase", Type: TypeEnum, Enum: contentPhases, Group: g,
			Description: desc("فاز محتوایی (I-11)", "The content phase (I-11).")},
		Def{Name: "cycle_day", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("روز چرخه؛ در تأخیر ادامه می\u200cیابد", "The cycle day; it keeps counting while late.")},
		Def{Name: "cycle_length", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("طول مؤثر چرخه", "The effective cycle length.")},
		Def{Name: "days_to_period", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("روز تا پریود بعدی (۰ در تأخیر)", "Days to the next period (0 while late).")},
		Def{Name: "days_late", Type: TypeInt, Unit: "day", Group: g,
			Description: desc("روزهای تأخیر پریود", "Days the period is late.")},
		Def{Name: "confidence", Type: TypeEnum, Enum: confidences, Group: g,
			Description: desc("اطمینان موتور", "The engine's confidence.")},
		Def{Name: "data_quality", Type: TypeEnum, Enum: qualities, Group: g,
			Description: desc("کیفیت داده", "The data quality.")},
		Def{Name: "warnings", Type: TypeSet, Enum: allWarnings(), Group: g,
			Description: desc("هشدارهای موتور (§۲.۳۲)", "The engine warnings (§2.32).")},
		Def{Name: "requires_user_input", Type: TypeBool, Group: g,
			Description: desc("موتور به ورودی کاربر نیاز دارد (§۲.۳۳)", "The engine needs the user's input (§2.33).")},
		Def{Name: "cycle_regularity", Type: TypeEnum, Enum: regularities, Group: g,
			Description: desc("نظم چرخه", "The cycle regularity.")},
		Def{Name: "suggest_pregnancy_test", Type: TypeBool, Group: g,
			Description: desc("پیشنهاد تست بارداری", "Suggest a pregnancy test.")},
		Def{Name: "fertility_suppressed", Type: TypeBool, Group: g,
			Description: desc("باروری سرکوب یا پنهان است (پیشگیری هورمونی، بارداری، نوجوان، یائسگی…)", "Fertility is suppressed or hidden (hormonal contraception, pregnancy, teen, menopause…).")},
		Def{Name: "is_fertile_window", Type: TypeBool, Group: g,
			Description: desc("پنجره باروری نمایشی؛ زیر سرکوب وجود ندارد", "The display fertile window; absent under suppression.")},
		Def{Name: "is_pms_window", Type: TypeBool, Group: g,
			Description: desc("روزهای احتمال PMS", "The PMS-possible days.")},
		Def{Name: "today_ui.banner", Type: TypeEnum, Enum: banners, Group: g,
			Description: desc("بنر وضعیت امروز که موتور انتخاب کرده (M-14)", "The Today status banner the engine chose (M-14).")},
		Def{Name: "current_period_start", Type: TypeDate, Group: g,
			Description: desc("شروع چرخه جاری (برای سقف هر سیکل)", "The current cycle's start (for per-cycle caps).")},
	)
	RegisterGroup(g, engineFacts)
}

// engineFacts is the engine producer.
func engineFacts(ctx context.Context, f *Facts) (Values, error) {
	m, err := f.engineState(ctx)
	if err != nil || m == nil {
		return nil, err
	}
	d := manager.EngineDay{Status: m.status, Mode: m.state.Mode(),
		HasAnchor: len(m.state.Periods()) > 0 || m.state.Declared().LastPeriodStart.OK()}
	v := Values(manager.EngineFacts(&d, false))
	s := &m.status
	v["mode"] = m.state.Mode().String()
	warnings := make([]string, 0, s.Warnings.Len())
	for w := range s.Warnings.All() {
		warnings = append(warnings, w.String())
	}
	v["warnings"] = warnings
	v["requires_user_input"] = s.RequiresUserInput
	v["cycle_regularity"] = s.Metrics.CycleRegularity.String()
	banner := m.state.TodayUI(m.status).Banner
	if banner == engine.BannerNone {
		v["today_ui.banner"] = BannerNone
	} else {
		v["today_ui.banner"] = banner.String()
	}
	if start := s.Anchors.CurrentPeriodStart; start.IsSet() {
		v["current_period_start"] = start.Civil().String()
	}
	return v, nil
}

// engineState loads and resolves the engine state once per request (nil without an engine).
func (f *Facts) engineState(ctx context.Context) (*engineMemo, error) {
	if f.engine != nil {
		return f.engine, nil
	}
	if f.src.Engine == nil {
		return nil, nil
	}
	st, err := f.src.Engine.EngineState(ctx)
	if err != nil || st == nil {
		return nil, err
	}
	today := engine.DayOf(f.today)
	m := &engineMemo{state: st, status: st.Resolve(today, today)}
	d := manager.EngineDay{Status: m.status, Mode: st.Mode()}
	m.strip = d.NoFertility()
	f.engine = m
	return m, nil
}

// mainPhaseOf is the main phase of a status as the main_phase fact reads it (fertile → follicular when stripped).
func mainPhaseOf(s *engine.Status, strip bool) string {
	if strip && s.MainPhase == engine.MainPhaseFertile {
		return engine.MainPhaseFollicular.String()
	}
	return s.MainPhase.String()
}
