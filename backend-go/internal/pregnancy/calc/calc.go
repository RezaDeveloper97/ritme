// Package calc is PregnancyCalculationService
// (backend/app/Services/PregnancyEngine/PregnancyCalculationService.php): gestational age,
// due date, conception date, current week, fetal-movement thresholds, high-risk rules and
// the /pregnancy/status payload. It is pure: callers load the profile and pass "today".
//
// Typical use (T-M2-17 messages, T-M2-18 home):
//
//	p, _ := pregnancy.LoadProfile(ctx, q, userID)    // nil when the user has no profile
//	c := calc.New(p, locale, civildate.Today(clk))
//	c.CurrentWeek()                                   // 1..40
//	st := c.Status()                                  // getPregnancyStatus(); st.JSON() is the API shape
//
// Quirks kept from PHP: bilingual {en, fa} blobs in `formatted` / `flags` regardless of the
// request locale; the Persian date is Gregorian with Persian month names; getCurrentWeek
// treats "no gestational age" as week 0 (null + 1 = 1).
package calc

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Calculator is one PregnancyCalculationService instance: a user's profile (nil when the
// user has none), the request locale and the request's Tehran "today" (Carbon::today()).
type Calculator struct {
	p      *store.PregnancyProfile
	locale string
	today  civildate.Date
}

// New returns a calculator. p may be nil ($user->pregnancyProfile === null).
func New(p *store.PregnancyProfile, locale string, today civildate.Date) *Calculator {
	return &Calculator{p: p, locale: locale, today: today}
}

// Profile returns the profile the calculator was built with (nil when none).
func (c *Calculator) Profile() *store.PregnancyProfile { return c.p }

// GestationalAge is calculateGestationalAge(): Valid is false for the "empty" result
// (every field null).
type GestationalAge struct {
	Valid       bool
	Weeks       int
	Days        int
	TotalDays   int
	Trimester   int
	Confidence  enums.ConfidenceLevel
	Uncertainty int
	locale      string
}

// JSON is the PHP array (key order kept).
func (g GestationalAge) JSON() *jsonx.OrderedMap {
	if !g.Valid {
		return jsonx.Obj(
			"weeks", nil, "days", nil, "total_days", nil, "trimester", nil,
			"confidence_level", nil, "confidence_label", nil, "uncertainty_days", nil, "formatted", nil,
		)
	}
	return jsonx.Obj(
		"weeks", g.Weeks,
		"days", g.Days,
		"total_days", g.TotalDays,
		"trimester", g.Trimester,
		"confidence_level", string(g.Confidence),
		"confidence_label", g.Confidence.Label(g.locale),
		"uncertainty_days", g.Uncertainty,
		"formatted", jsonx.Obj(
			"en", fmt.Sprintf("%d weeks, %d days", g.Weeks, g.Days),
			"fa", fmt.Sprintf("%d هفته و %d روز", g.Weeks, g.Days),
		),
	)
}

// MarshalJSON implements json.Marshaler.
func (g GestationalAge) MarshalJSON() ([]byte, error) { return json.Marshal(g.JSON()) }

// GestationalAge is calculateGestationalAge(today).
func (c *Calculator) GestationalAge() GestationalAge { return c.GestationalAgeOn(c.today) }

// GestationalAgeOn is calculateGestationalAge($referenceDate).
func (c *Calculator) GestationalAgeOn(ref civildate.Date) GestationalAge {
	if c.p == nil {
		return GestationalAge{}
	}
	switch enums.PregnancyAgeSource(c.p.AgeSource.String) {
	case enums.PregnancyAgeSourceLmp:
		if !c.p.AgeSource.Valid || !c.p.LmpDate.Valid {
			return GestationalAge{}
		}
		return c.format(c.p.LmpDate.Date.DiffDays(ref), enums.ConfidenceLevelMedium, 3)
	case enums.PregnancyAgeSourceUltrasound:
		if !c.p.UltrasoundDate.Valid || !c.p.UltrasoundWeeks.Valid || !c.p.UltrasoundDays.Valid {
			return GestationalAge{}
		}
		atScan := int(c.p.UltrasoundWeeks.Int32)*7 + int(c.p.UltrasoundDays.Int32)
		return c.format(atScan+c.p.UltrasoundDate.Date.DiffDays(ref), enums.ConfidenceLevelHigh, 1)
	case enums.PregnancyAgeSourceManual:
		if !c.p.ManualWeeks.Valid || !c.p.ManualDays.Valid || !c.p.ManualEntryDate.Valid {
			return GestationalAge{}
		}
		atEntry := int(c.p.ManualWeeks.Int32)*7 + int(c.p.ManualDays.Int32)
		return c.format(atEntry+c.p.ManualEntryDate.Date.DiffDays(ref), enums.ConfidenceLevelLow, 5)
	}
	return GestationalAge{}
}

// format is formatGestationalAge(): weeks = floor(d/7), days = d % 7 (PHP remainder keeps
// the dividend's sign, like Go's %).
func (c *Calculator) format(total int, conf enums.ConfidenceLevel, uncertainty int) GestationalAge {
	weeks := int(math.Floor(float64(total) / 7))
	return GestationalAge{
		Valid:       true,
		Weeks:       weeks,
		Days:        total % 7,
		TotalDays:   total,
		Trimester:   Trimester(weeks),
		Confidence:  conf,
		Uncertainty: uncertainty,
		locale:      c.locale,
	}
}

// Trimester is calculateTrimester(): ≤12 → 1, ≤27 → 2, else 3.
func Trimester(weeks int) int {
	switch {
	case weeks <= 12:
		return 1
	case weeks <= 27:
		return 2
	}
	return 3
}

// DueDate is calculateEDD(): nil when it cannot be computed.
type DueDate struct {
	Date           civildate.Date
	DaysRemaining  int // max(0, today→EDD), Carbon diffInDays on two midnights
	WeeksRemaining int
}

// JSON is the PHP array.
func (d *DueDate) JSON() any {
	if d == nil {
		return nil
	}
	return jsonx.Obj(
		"date", d.Date.String(),
		"days_remaining", d.DaysRemaining,
		"weeks_remaining", d.WeeksRemaining,
		"formatted", jsonx.Obj("en", FormatEnglishDate(d.Date), "fa", FormatPersianDate(d.Date)),
	)
}

// EDDDate is the due date itself (LMP / estimated LMP + 280), or false when the profile
// lacks the data of its age source.
func (c *Calculator) EDDDate() (civildate.Date, bool) {
	if c.p == nil {
		return civildate.Date{}, false
	}
	switch enums.PregnancyAgeSource(c.p.AgeSource.String) {
	case enums.PregnancyAgeSourceLmp:
		if !c.p.AgeSource.Valid || !c.p.LmpDate.Valid {
			return civildate.Date{}, false
		}
		return c.p.LmpDate.Date.AddDays(280), true
	case enums.PregnancyAgeSourceUltrasound:
		if !c.p.UltrasoundDate.Valid || !c.p.UltrasoundWeeks.Valid || !c.p.UltrasoundDays.Valid {
			return civildate.Date{}, false
		}
		atScan := int(c.p.UltrasoundWeeks.Int32)*7 + int(c.p.UltrasoundDays.Int32)
		return c.p.UltrasoundDate.Date.AddDays(-atScan + 280), true
	case enums.PregnancyAgeSourceManual:
		if !c.p.ManualWeeks.Valid || !c.p.ManualDays.Valid || !c.p.ManualEntryDate.Valid {
			return civildate.Date{}, false
		}
		atEntry := int(c.p.ManualWeeks.Int32)*7 + int(c.p.ManualDays.Int32)
		return c.p.ManualEntryDate.Date.AddDays(-atEntry + 280), true
	}
	return civildate.Date{}, false
}

// EDD is calculateEDD().
func (c *Calculator) EDD() *DueDate {
	edd, ok := c.EDDDate()
	if !ok {
		return nil
	}
	remaining := c.today.DiffDays(edd)
	return &DueDate{
		Date:           edd,
		DaysRemaining:  max(0, remaining),
		WeeksRemaining: max(0, int(math.Floor(float64(remaining)/7))),
	}
}

// ConceptionDate is calculateConceptionDate(): today − (total_days − 14); nil without a
// gestational age. JSON: {date, formatted{en, fa}}.
func (c *Calculator) ConceptionDate() any {
	ga := c.GestationalAge()
	if !ga.Valid {
		return nil
	}
	d := c.today.AddDays(-(ga.TotalDays - 14))
	return jsonx.Obj(
		"date", d.String(),
		"formatted", jsonx.Obj("en", FormatEnglishDate(d), "fa", FormatPersianDate(d)),
	)
}

// CurrentWeek is getCurrentWeek(): clamp(weeks + 1, 1, 40); no gestational age → 1.
func (c *Calculator) CurrentWeek() int {
	ga := c.GestationalAge()
	weeks := 0
	if ga.Valid {
		weeks = ga.Weeks
	}
	return min(40, max(1, weeks+1))
}

// FetalMovementTrackingActive is isFetalMovementTrackingActive() (week ≥ 18).
func (c *Calculator) FetalMovementTrackingActive() bool { return c.CurrentWeek() >= 18 }

// FetalMovementRequired is isFetalMovementRequired() (week ≥ 24).
func (c *Calculator) FetalMovementRequired() bool { return c.CurrentWeek() >= 24 }

// highRiskConditions are the pre-existing conditions that make a pregnancy high risk.
var highRiskConditions = []string{"chronic_hypertension", "diabetes"}

// IsHighRisk is isHighRisk(): miscarriage / high-risk history, Rh negative, or a
// chronic_hypertension / diabetes pre-existing condition (loose in_array).
func (c *Calculator) IsHighRisk() bool { return IsHighRisk(c.p) }

// IsHighRisk is PregnancyCalculationService::isHighRisk for a profile (nil → false).
func IsHighRisk(p *store.PregnancyProfile) bool {
	if p == nil {
		return false
	}
	if (p.HasMiscarriageHistory.Valid && p.HasMiscarriageHistory.Bool) ||
		(p.HasHighRiskHistory.Valid && p.HasHighRiskHistory.Bool) {
		return true
	}
	if p.RhFactor.Valid && p.RhFactor.String == "negative" {
		return true
	}
	for _, cond := range Conditions(p) {
		for _, hr := range highRiskConditions {
			if phpval.LooseEqual(cond, hr) {
				return true
			}
		}
	}
	return false
}

// Conditions returns the values of pre_existing_conditions (array cast; a JSON object's
// values in order), or nil when the column is NULL or not an array.
func Conditions(p *store.PregnancyProfile) []any {
	if p == nil || !p.PreExistingConditions.Valid {
		return nil
	}
	v, err := phpval.Decode(p.PreExistingConditions.V)
	if err != nil {
		return nil
	}
	_, vals := phpval.Entries(v)
	return vals
}

// Status is getPregnancyStatus().
type Status struct {
	HasProfile                  bool
	IsActive                    bool
	GestationalAge              GestationalAge
	DueDate                     *DueDate
	CurrentWeek                 int
	AgeSource                   *string
	IsHighRisk                  bool
	FetalMovementTrackingActive bool
	FetalMovementRequired       bool
	Flags                       *jsonx.OrderedMap
}

// Status computes getPregnancyStatus().
func (c *Calculator) Status() Status {
	if c.p == nil {
		return Status{
			Flags: jsonx.Obj("no_profile", jsonx.Obj(
				"en", "Please complete pregnancy onboarding to get started.",
				"fa", "لطفاً آنبوردینگ بارداری را تکمیل کنید.",
			)),
		}
	}
	ga := c.GestationalAge()
	week := c.CurrentWeek()
	st := Status{
		HasProfile:                  true,
		IsActive:                    c.p.PregnancyMode,
		GestationalAge:              ga,
		DueDate:                     c.EDD(),
		CurrentWeek:                 week,
		IsHighRisk:                  c.IsHighRisk(),
		FetalMovementTrackingActive: week >= 18,
		FetalMovementRequired:       week >= 24,
		Flags:                       c.statusFlags(ga, week),
	}
	if c.p.AgeSource.Valid {
		s := c.p.AgeSource.String
		st.AgeSource = &s
	}
	return st
}

// JSON is the PHP array of getPregnancyStatus() (the /pregnancy/status data).
func (s Status) JSON() *jsonx.OrderedMap {
	if !s.HasProfile {
		return jsonx.Obj(
			"is_active", false,
			"gestational_age", GestationalAge{}.JSON(),
			"estimated_due_date", nil,
			"current_week", nil,
			"trimester", nil,
			"age_source", nil,
			"confidence_level", nil,
			"is_high_risk", false,
			"fetal_movement_tracking_active", false,
			"fetal_movement_required", false,
			"flags", s.Flags,
		)
	}
	var trimester, confidence, ageSource any
	if s.GestationalAge.Valid {
		trimester = s.GestationalAge.Trimester
		confidence = string(s.GestationalAge.Confidence)
	}
	if s.AgeSource != nil {
		ageSource = *s.AgeSource
	}
	return jsonx.Obj(
		"is_active", s.IsActive,
		"gestational_age", s.GestationalAge.JSON(),
		"estimated_due_date", s.DueDate.JSON(),
		"current_week", s.CurrentWeek,
		"trimester", trimester,
		"age_source", ageSource,
		"confidence_level", confidence,
		"is_high_risk", s.IsHighRisk,
		"fetal_movement_tracking_active", s.FetalMovementTrackingActive,
		"fetal_movement_required", s.FetalMovementRequired,
		"flags", s.Flags,
	)
}

// MarshalJSON implements json.Marshaler.
func (s Status) MarshalJSON() ([]byte, error) { return json.Marshal(s.JSON()) }

// statusFlags is generateStatusFlags(): raw {en, fa} blobs, never localized.
func (c *Calculator) statusFlags(ga GestationalAge, week int) *jsonx.OrderedMap {
	trimester := 1
	if ga.Valid {
		trimester = ga.Trimester
	}
	flags := jsonx.NewArray()
	flags.Set("trimester_info", jsonx.Obj(
		"en", fmt.Sprintf("You are in trimester %d.", trimester),
		"fa", fmt.Sprintf("شما در سه‌ماهه %d هستید.", trimester), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	))
	flags.Set("week_info", jsonx.Obj(
		"en", fmt.Sprintf("Week %d of pregnancy.", week),
		"fa", fmt.Sprintf("هفته %d بارداری.", week),
	))
	if week >= 18 && week < 24 {
		flags.Set("fetal_movement", jsonx.Obj(
			"en", "You may start feeling baby movements. Track them when you notice!",
			"fa", "ممکن است حرکات جنین را احساس کنید. وقتی متوجه شدید ثبت کنید!",
		))
	}
	if week >= 24 {
		flags.Set("daily_tracking", jsonx.Obj(
			"en", "Daily fetal movement tracking is important. Please log movements daily.",
			"fa", "ثبت روزانه حرکات جنین مهم است. لطفاً روزانه ثبت کنید.",
		))
	}
	if c.p != nil && c.p.RhFactor.Valid && c.p.RhFactor.String == "negative" {
		flags.Set("rh_warning", jsonx.Obj(
			"en", "RH negative detected. Special monitoring is recommended.",
			"fa", "RH منفی شناسایی شد. مراقبت ویژه توصیه می‌شود.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
		))
	}
	if c.IsHighRisk() {
		flags.Set("high_risk", jsonx.Obj(
			"en", "Based on your profile, extra monitoring is recommended.",
			"fa", "بر اساس پروفایل شما، مراقبت بیشتر توصیه می‌شود.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
		))
	}
	return flags
}

// persianMonths are the Persian names of the Gregorian months (formatPersianDate).
var persianMonths = [...]string{
	"ژانویه", "فوریه", "مارس", "آوریل", "مه", "ژوئن",
	"ژوئیه", "اوت", "سپتامبر", "اکتبر", "نوامبر", "دسامبر",
}

// FormatPersianDate is formatPersianDate(): "{day} {Persian Gregorian month} {year}" with
// Western digits (no Jalali conversion).
func FormatPersianDate(d civildate.Date) string {
	return fmt.Sprintf("%d %s %d", d.Day, persianMonths[d.Month-1], d.Year)
}

// FormatEnglishDate is Carbon format('F j, Y').
func FormatEnglishDate(d civildate.Date) string {
	return fmt.Sprintf("%s %d, %d", d.Month.String(), d.Day, d.Year)
}
