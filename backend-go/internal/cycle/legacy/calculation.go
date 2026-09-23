package legacy

import (
	"github.com/ritme/backend-go/internal/cycle/recommendation"
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// CalendarFields are the per-day keys a calendar-mode calculation keeps
// (HealthDataEngine::CALENDAR_FIELDS, HealthDataEngine.php:78), in output order.
var CalendarFields = []string{
	"calculation_date",
	"cycle_day",
	"phase",
	"subphase",
	"estimated_ovulation_day",
	"cycle_length_used",
	"is_fertile_window",
	"is_pms_window",
	"is_period_tomorrow",
	"final_probability",
	"cycle_variability",
}

// Calculation is one HealthDataEngine::calculateForDate result.
//
// Its JSON is the PHP array: bilingual text_flags / daily_tips as the engine builds them (the month
// view's shape), the calendar subset when built in calendar mode, or the request-locale shape of
// CycleCalculationController::localizeCalculation after Localize.
type Calculation struct {
	Date civildate.Date
	// Complete is false for the empty calculation (no profile or no last_period_start): every
	// numeric field is then JSON null.
	Complete bool

	// Day is cycle_day (1..CycleLength).
	Day              int
	Phase            enums.CyclePhase
	CurrentSubphase  enums.CycleSubphase
	OvulationDay     int
	CycleLength      int
	IsFertileWindow  bool
	IsPmsWindow      bool
	IsPeriodTomorrow bool
	IsLutealSpotting bool
	// Scores are already rounded as the API emits them (round(x, 4); final_probability is
	// round(p*100, 2)).
	CycleScore       float64
	AgeFactor        float64
	BaseProbability  float64
	SymptomScore     float64
	FinalProbability float64
	Variability      enums.CycleVariability
	UncertaintyRange int

	TextFlags []TextFlag
	DailyTips []recommendation.Tip
	// Profile is source_profile_data (nil for the empty calculation).
	Profile *ProfileSnapshot
	// Log is today's log (nil = none); its Source is source_daily_log_data.
	Log *DailyLog

	calendar  bool
	localized bool
	locale    string
}

// TextFlag is one bilingual `text_flags` entry ({en, fa}, the engine's fixed pair).
type TextFlag struct {
	Key string
	EN  string
	FA  string
}

// lookup is `$value[$key]` on the {en, fa} blob.
func (f TextFlag) lookup(key string) (string, bool) {
	switch key {
	case "en":
		return f.EN, true
	case "fa":
		return f.FA, true
	}
	return "", false
}

// ProfileSnapshot is source_profile_data (HealthDataEngine::getProfileSnapshot).
type ProfileSnapshot struct {
	Birthday        civildate.Date `json:"birthday"`
	PeriodDuration  *int           `json:"period_duration"`
	CycleDuration   *int           `json:"cycle_duration"`
	LastPeriodStart civildate.Date `json:"last_period_start"`
}

// CycleDay implements view.BaseCalc: ok is false for the empty calculation (PHP null).
func (c Calculation) CycleDay() (int, bool) { return c.Day, c.Complete }

// Subphase implements view.BaseCalc.
func (c Calculation) Subphase() enums.CycleSubphase { return c.CurrentSubphase }

// Calendar reports whether c was built in calendar mode (only CalendarFields are emitted).
func (c Calculation) Calendar() bool { return c.calendar }

// Localize is CycleCalculationController::localizeCalculation: text_flags become the locale's
// string when the flag has that key and stay the whole {en, fa} object otherwise; daily_tips go
// through DailyTipLocalizer. PHP: CycleCalculationController.php:507.
func (c Calculation) Localize(locale string) Calculation {
	c.localized, c.locale = true, locale
	return c
}

// MarshalJSON writes the PHP array.
func (c Calculation) MarshalJSON() ([]byte, error) {
	m := jsonx.NewObject()
	m.Set("calculation_date", c.Date)
	if c.Complete {
		m.Set("cycle_day", c.Day).
			Set("phase", c.Phase).
			Set("subphase", c.CurrentSubphase).
			Set("estimated_ovulation_day", c.OvulationDay).
			Set("cycle_length_used", c.CycleLength)
	} else {
		for _, k := range []string{"cycle_day", "phase", "subphase", "estimated_ovulation_day", "cycle_length_used"} {
			m.Set(k, nil)
		}
	}
	m.Set("is_fertile_window", c.IsFertileWindow).
		Set("is_pms_window", c.IsPmsWindow).
		Set("is_period_tomorrow", c.IsPeriodTomorrow)
	if !c.calendar {
		m.Set("is_luteal_spotting", c.IsLutealSpotting)
		setScore(m, c.Complete, "cycle_score", c.CycleScore)
		setScore(m, c.Complete, "age_factor", c.AgeFactor)
		setScore(m, c.Complete, "base_probability", c.BaseProbability)
		setScore(m, c.Complete, "symptom_score", c.SymptomScore)
	}
	setScore(m, c.Complete, "final_probability", c.FinalProbability)
	if c.Complete {
		m.Set("cycle_variability", c.Variability)
	} else {
		m.Set("cycle_variability", nil)
	}
	if c.calendar {
		return m.MarshalJSON()
	}

	if c.Complete {
		m.Set("uncertainty_range", c.UncertaintyRange)
	} else {
		m.Set("uncertainty_range", nil)
	}
	m.Set("text_flags", c.textFlagsJSON())
	m.Set("daily_tips", c.dailyTipsJSON())
	if c.Profile != nil {
		m.Set("source_profile_data", c.Profile)
	} else {
		m.Set("source_profile_data", nil)
	}
	if c.Log != nil {
		m.Set("source_daily_log_data", c.Log.Source)
	} else {
		m.Set("source_daily_log_data", nil)
	}
	return m.MarshalJSON()
}

func setScore(m *jsonx.OrderedMap, complete bool, key string, v float64) {
	if complete {
		m.Set(key, jsonx.Float(v))
	} else {
		m.Set(key, nil)
	}
}

func (c Calculation) textFlagsJSON() *jsonx.OrderedMap {
	flags := jsonx.NewArray()
	for _, f := range c.TextFlags {
		if text, ok := f.lookup(c.locale); c.localized && ok {
			flags.Set(f.Key, text)
			continue
		}
		// Raw engine shape, or isset($value[$locale]) false: the whole bilingual blob.
		flags.Set(f.Key, jsonx.Obj("en", f.EN, "fa", f.FA))
	}
	return flags
}

func (c Calculation) dailyTipsJSON() any {
	if c.localized {
		return recommendation.Localize(c.DailyTips, c.locale)
	}
	if c.DailyTips == nil {
		return []recommendation.Tip{}
	}
	return c.DailyTips
}

// MonthSummary is the month view's `month_summary` (CycleCalculationController::calculateMonthSummary).
type MonthSummary struct {
	FertileDays int `json:"fertile_days"`
	PeriodDays  int `json:"period_days"`
	PmsDays     int `json:"pms_days"`
}

// SummarizeMonth counts fertile, period and PMS days over a month of calculations.
// PHP: CycleCalculationController.php:533.
func SummarizeMonth(calcs []Calculation) MonthSummary {
	var s MonthSummary
	for _, c := range calcs {
		if c.IsFertileWindow {
			s.FertileDays++
		}
		if c.Complete && c.Phase == enums.CyclePhaseMenstruation {
			s.PeriodDays++
		}
		if c.IsPmsWindow {
			s.PmsDays++
		}
	}
	return s
}
