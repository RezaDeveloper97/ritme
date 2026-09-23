package pregnancy

import (
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// ---- PregnancyProfile ----

var profileCasts = casts{
	"pregnancy_mode": castBool, "cycle_mode": castBool, "is_locked": castBool,
	"has_miscarriage_history": castBool, "has_high_risk_history": castBool,
	"rh_negative_care_flag": castBool, "fetal_movement_felt": castBool, "onboarding_completed": castBool,
	"lmp_date": castDate, "ultrasound_date": castDate, "manual_entry_date": castDate,
	"estimated_due_date": castDate, "estimated_conception_date": castDate, "first_fetal_movement_date": castDate,
	"onboarding_completed_at": castDateTime, "created_at": castDateTime, "updated_at": castDateTime,
	"pre_existing_conditions": castArray,
}

// profileDefaults are the column defaults of a new pregnancy_profiles row.
func profileDefaults(userID uint64) *model {
	return newModel(profileCasts).
		set("user_id", i64(userID)).
		set("pregnancy_mode", int64(1)).set("cycle_mode", int64(0)).set("is_locked", int64(0)).
		set("age_source", nil).set("confidence_level", nil).set("lmp_date", nil).
		set("ultrasound_date", nil).set("ultrasound_weeks", nil).set("ultrasound_days", nil).
		set("manual_weeks", nil).set("manual_days", nil).set("manual_entry_date", nil).
		set("estimated_due_date", nil).set("estimated_conception_date", nil).set("uncertainty_days", int64(3)).
		set("has_miscarriage_history", nil).set("has_high_risk_history", nil).set("pre_existing_conditions", nil).
		set("blood_type", nil).set("rh_factor", nil).set("rh_negative_care_flag", int64(0)).
		set("first_fetal_movement_date", nil).set("fetal_movement_felt", int64(0)).
		set("onboarding_completed", int64(0)).set("onboarding_completed_at", nil).
		set("created_at", nil).set("updated_at", nil)
}

// profileModel is a loaded PregnancyProfile (every column, table order).
func profileModel(p *store.PregnancyProfile) *model {
	return newModel(profileCasts).
		set("id", i64(p.ID)).
		set("user_id", i64(p.UserID)).
		set("pregnancy_mode", rawFlag(p.PregnancyMode)).
		set("cycle_mode", rawFlag(p.CycleMode)).
		set("is_locked", rawFlag(p.IsLocked)).
		set("age_source", rawString(p.AgeSource)).
		set("confidence_level", rawString(p.ConfidenceLevel)).
		set("lmp_date", rawDate(p.LmpDate)).
		set("ultrasound_date", rawDate(p.UltrasoundDate)).
		set("ultrasound_weeks", rawInt(p.UltrasoundWeeks)).
		set("ultrasound_days", rawInt(p.UltrasoundDays)).
		set("manual_weeks", rawInt(p.ManualWeeks)).
		set("manual_days", rawInt(p.ManualDays)).
		set("manual_entry_date", rawDate(p.ManualEntryDate)).
		set("estimated_due_date", rawDate(p.EstimatedDueDate)).
		set("estimated_conception_date", rawDate(p.EstimatedConceptionDate)).
		set("uncertainty_days", int64(p.UncertaintyDays)).
		set("has_miscarriage_history", rawBool(p.HasMiscarriageHistory)).
		set("has_high_risk_history", rawBool(p.HasHighRiskHistory)).
		set("pre_existing_conditions", rawJSON(p.PreExistingConditions)).
		set("blood_type", rawString(p.BloodType)).
		set("rh_factor", rawString(p.RhFactor)).
		set("rh_negative_care_flag", rawFlag(p.RhNegativeCareFlag)).
		set("first_fetal_movement_date", rawDate(p.FirstFetalMovementDate)).
		set("fetal_movement_felt", rawFlag(p.FetalMovementFelt)).
		set("onboarding_completed", rawFlag(p.OnboardingCompleted)).
		set("onboarding_completed_at", rawTime(p.OnboardingCompletedAt)).
		set("created_at", rawTime(p.CreatedAt)).
		set("updated_at", rawTime(p.UpdatedAt))
}

// ProfileJSON is the PregnancyProfile model's toArray() (GET /pregnancy/profile,
// onboarding, profile export): plain `date` casts serialise as the previous day in UTC.
func ProfileJSON(p *store.PregnancyProfile) any {
	if p == nil {
		return nil
	}
	return profileModel(p).JSON()
}

func profileUpdateParams(m *model) (store.UpdateProfileParams, error) {
	var err error
	flag := func(k string) bool {
		b, e := wFlag(k, m.raw(k))
		if e != nil && err == nil {
			err = e
		}
		return b
	}
	unc := wInt(m.raw("uncertainty_days"))
	if !unc.Valid && err == nil {
		err = errNotNull("uncertainty_days")
	}
	p := store.UpdateProfileParams{
		PregnancyMode:           flag("pregnancy_mode"),
		CycleMode:               flag("cycle_mode"),
		IsLocked:                flag("is_locked"),
		AgeSource:               wString(m.raw("age_source")),
		ConfidenceLevel:         wString(m.raw("confidence_level")),
		LmpDate:                 wDate(m.raw("lmp_date")),
		UltrasoundDate:          wDate(m.raw("ultrasound_date")),
		UltrasoundWeeks:         wInt(m.raw("ultrasound_weeks")),
		UltrasoundDays:          wInt(m.raw("ultrasound_days")),
		ManualWeeks:             wInt(m.raw("manual_weeks")),
		ManualDays:              wInt(m.raw("manual_days")),
		ManualEntryDate:         wDate(m.raw("manual_entry_date")),
		EstimatedDueDate:        wDate(m.raw("estimated_due_date")),
		EstimatedConceptionDate: wDate(m.raw("estimated_conception_date")),
		UncertaintyDays:         unc.Int32,
		HasMiscarriageHistory:   wBool(m.raw("has_miscarriage_history")),
		HasHighRiskHistory:      wBool(m.raw("has_high_risk_history")),
		PreExistingConditions:   wJSON(m.raw("pre_existing_conditions")),
		BloodType:               wString(m.raw("blood_type")),
		RhFactor:                wString(m.raw("rh_factor")),
		RhNegativeCareFlag:      flag("rh_negative_care_flag"),
		FirstFetalMovementDate:  wDate(m.raw("first_fetal_movement_date")),
		FetalMovementFelt:       flag("fetal_movement_felt"),
		OnboardingCompleted:     flag("onboarding_completed"),
		OnboardingCompletedAt:   wTime(m.raw("onboarding_completed_at")),
		UpdatedAt:               wTime(m.raw("updated_at")),
	}
	if id := m.raw("id"); id != nil {
		p.ID = uint64(id.(int64)) //nolint:gosec // G115: ids are positive
	}
	return p, err
}

func profileInsertParams(m *model) (store.InsertProfileParams, error) {
	u, err := profileUpdateParams(m)
	return store.InsertProfileParams{
		UserID:                  uint64(m.raw("user_id").(int64)), //nolint:gosec // G115: ids are positive
		PregnancyMode:           u.PregnancyMode,
		CycleMode:               u.CycleMode,
		IsLocked:                u.IsLocked,
		AgeSource:               u.AgeSource,
		ConfidenceLevel:         u.ConfidenceLevel,
		LmpDate:                 u.LmpDate,
		UltrasoundDate:          u.UltrasoundDate,
		UltrasoundWeeks:         u.UltrasoundWeeks,
		UltrasoundDays:          u.UltrasoundDays,
		ManualWeeks:             u.ManualWeeks,
		ManualDays:              u.ManualDays,
		ManualEntryDate:         u.ManualEntryDate,
		EstimatedDueDate:        u.EstimatedDueDate,
		EstimatedConceptionDate: u.EstimatedConceptionDate,
		UncertaintyDays:         u.UncertaintyDays,
		HasMiscarriageHistory:   u.HasMiscarriageHistory,
		HasHighRiskHistory:      u.HasHighRiskHistory,
		PreExistingConditions:   u.PreExistingConditions,
		BloodType:               u.BloodType,
		RhFactor:                u.RhFactor,
		RhNegativeCareFlag:      u.RhNegativeCareFlag,
		FirstFetalMovementDate:  u.FirstFetalMovementDate,
		FetalMovementFelt:       u.FetalMovementFelt,
		OnboardingCompleted:     u.OnboardingCompleted,
		OnboardingCompletedAt:   u.OnboardingCompletedAt,
		CreatedAt:               wTime(m.raw("created_at")),
		UpdatedAt:               u.UpdatedAt,
	}, err
}

// ---- PregnancySymptomLog ----

// symptomNames are the 14 symptoms in column order (has_X, X_severity).
var symptomNames = []string{
	"nausea", "vomiting", "fatigue", "headache", "dizziness", "breast_pain",
	"lower_abdominal_pain", "cramping", "back_pain", "pelvic_pressure",
	"spotting", "bleeding", "fluid_leakage", "severe_sudden_pain",
}

var symptomCasts = func() casts {
	c := casts{"log_date": castYMD, "created_at": castDateTime, "updated_at": castDateTime}
	for _, s := range symptomNames {
		c["has_"+s] = castBool
	}
	return c
}()

func symptomModel(r *store.PregnancySymptomLog) *model {
	pairs := [][2]any{
		{r.HasNausea, r.NauseaSeverity}, {r.HasVomiting, r.VomitingSeverity}, {r.HasFatigue, r.FatigueSeverity},
		{r.HasHeadache, r.HeadacheSeverity}, {r.HasDizziness, r.DizzinessSeverity}, {r.HasBreastPain, r.BreastPainSeverity},
		{r.HasLowerAbdominalPain, r.LowerAbdominalPainSeverity}, {r.HasCramping, r.CrampingSeverity},
		{r.HasBackPain, r.BackPainSeverity}, {r.HasPelvicPressure, r.PelvicPressureSeverity},
		{r.HasSpotting, r.SpottingSeverity}, {r.HasBleeding, r.BleedingSeverity},
		{r.HasFluidLeakage, r.FluidLeakageSeverity}, {r.HasSevereSuddenPain, r.SevereSuddenPainSeverity},
	}
	m := newModel(symptomCasts).set("id", i64(r.ID)).set("user_id", i64(r.UserID)).set("log_date", r.LogDate)
	for i, s := range symptomNames {
		m.set("has_"+s, rawBool(pairs[i][0].(sqlNullBool))).set(s+"_severity", rawString(pairs[i][1].(sqlNullString)))
	}
	return m.set("notes", rawString(r.Notes)).set("created_at", rawTime(r.CreatedAt)).set("updated_at", rawTime(r.UpdatedAt))
}

func symptomUpdateParams(m *model) store.UpdateSymptomLogParams {
	b := func(s string) sqlNullBool { return wBool(m.raw("has_" + s)) }
	v := func(s string) sqlNullString { return wString(m.raw(s + "_severity")) }
	p := store.UpdateSymptomLogParams{
		HasNausea: b("nausea"), NauseaSeverity: v("nausea"),
		HasVomiting: b("vomiting"), VomitingSeverity: v("vomiting"),
		HasFatigue: b("fatigue"), FatigueSeverity: v("fatigue"),
		HasHeadache: b("headache"), HeadacheSeverity: v("headache"),
		HasDizziness: b("dizziness"), DizzinessSeverity: v("dizziness"),
		HasBreastPain: b("breast_pain"), BreastPainSeverity: v("breast_pain"),
		HasLowerAbdominalPain: b("lower_abdominal_pain"), LowerAbdominalPainSeverity: v("lower_abdominal_pain"),
		HasCramping: b("cramping"), CrampingSeverity: v("cramping"),
		HasBackPain: b("back_pain"), BackPainSeverity: v("back_pain"),
		HasPelvicPressure: b("pelvic_pressure"), PelvicPressureSeverity: v("pelvic_pressure"),
		HasSpotting: b("spotting"), SpottingSeverity: v("spotting"),
		HasBleeding: b("bleeding"), BleedingSeverity: v("bleeding"),
		HasFluidLeakage: b("fluid_leakage"), FluidLeakageSeverity: v("fluid_leakage"),
		HasSevereSuddenPain: b("severe_sudden_pain"), SevereSuddenPainSeverity: v("severe_sudden_pain"),
		Notes:     wString(m.raw("notes")),
		UpdatedAt: wTime(m.raw("updated_at")),
	}
	if id := m.raw("id"); id != nil {
		p.ID = uint64(id.(int64)) //nolint:gosec // G115: ids are positive
	}
	return p
}

func symptomInsertParams(m *model, userID uint64) store.InsertSymptomLogParams {
	u := symptomUpdateParams(m)
	return store.InsertSymptomLogParams{
		UserID: userID, LogDate: wDate(m.raw("log_date")).Date,
		HasNausea: u.HasNausea, NauseaSeverity: u.NauseaSeverity,
		HasVomiting: u.HasVomiting, VomitingSeverity: u.VomitingSeverity,
		HasFatigue: u.HasFatigue, FatigueSeverity: u.FatigueSeverity,
		HasHeadache: u.HasHeadache, HeadacheSeverity: u.HeadacheSeverity,
		HasDizziness: u.HasDizziness, DizzinessSeverity: u.DizzinessSeverity,
		HasBreastPain: u.HasBreastPain, BreastPainSeverity: u.BreastPainSeverity,
		HasLowerAbdominalPain: u.HasLowerAbdominalPain, LowerAbdominalPainSeverity: u.LowerAbdominalPainSeverity,
		HasCramping: u.HasCramping, CrampingSeverity: u.CrampingSeverity,
		HasBackPain: u.HasBackPain, BackPainSeverity: u.BackPainSeverity,
		HasPelvicPressure: u.HasPelvicPressure, PelvicPressureSeverity: u.PelvicPressureSeverity,
		HasSpotting: u.HasSpotting, SpottingSeverity: u.SpottingSeverity,
		HasBleeding: u.HasBleeding, BleedingSeverity: u.BleedingSeverity,
		HasFluidLeakage: u.HasFluidLeakage, FluidLeakageSeverity: u.FluidLeakageSeverity,
		HasSevereSuddenPain: u.HasSevereSuddenPain, SevereSuddenPainSeverity: u.SevereSuddenPainSeverity,
		Notes: u.Notes, CreatedAt: wTime(m.raw("created_at")), UpdatedAt: u.UpdatedAt,
	}
}

// ---- PregnancyWeeklyLog ----

var weeklyCasts = casts{
	"log_date": castYMD, "has_swelling": castBool, "has_shortness_of_breath": castBool,
	"has_blood_pressure_device": castBool, "has_anxiety": castBool, "has_mood_swings": castBool,
	"has_depression_feelings": castBool, "weight": castDecimal2, "fasting_blood_sugar": castDecimal2,
	"post_meal_blood_sugar": castDecimal2, "swelling_locations": castArray,
	"created_at": castDateTime, "updated_at": castDateTime,
}

func weeklyModel(r *store.PregnancyWeeklyLog) *model {
	return newModel(weeklyCasts).
		set("id", i64(r.ID)).
		set("user_id", i64(r.UserID)).
		set("log_date", r.LogDate).
		set("pregnancy_week", int64(r.PregnancyWeek)).
		set("weight", rawString(r.Weight)).
		set("swelling_locations", rawJSON(r.SwellingLocations)).
		set("has_swelling", rawBool(r.HasSwelling)).
		set("has_shortness_of_breath", rawBool(r.HasShortnessOfBreath)).
		set("has_blood_pressure_device", rawFlag(r.HasBloodPressureDevice)).
		set("systolic_pressure", rawInt(r.SystolicPressure)).
		set("diastolic_pressure", rawInt(r.DiastolicPressure)).
		set("fasting_blood_sugar", rawString(r.FastingBloodSugar)).
		set("post_meal_blood_sugar", rawString(r.PostMealBloodSugar)).
		set("overall_mood", rawString(r.OverallMood)).
		set("has_anxiety", rawBool(r.HasAnxiety)).
		set("anxiety_severity", rawString(r.AnxietySeverity)).
		set("has_mood_swings", rawBool(r.HasMoodSwings)).
		set("mood_swings_severity", rawString(r.MoodSwingsSeverity)).
		set("has_depression_feelings", rawBool(r.HasDepressionFeelings)).
		set("depression_severity", rawString(r.DepressionSeverity)).
		set("notes", rawString(r.Notes)).
		set("created_at", rawTime(r.CreatedAt)).
		set("updated_at", rawTime(r.UpdatedAt))
}

func weeklyUpdateParams(m *model) (store.UpdateWeeklyLogParams, error) {
	// NOT NULL DEFAULT 0: absent on a new row → default; an explicit null fails like MariaDB.
	bpd, err := false, error(nil)
	if v, ok := m.attrs.Get("has_blood_pressure_device"); ok {
		bpd, err = wFlag("has_blood_pressure_device", v)
	}
	p := store.UpdateWeeklyLogParams{
		LogDate:                wDate(m.raw("log_date")).Date,
		Weight:                 wDecimal(m.raw("weight")),
		SwellingLocations:      wJSON(m.raw("swelling_locations")),
		HasSwelling:            wBool(m.raw("has_swelling")),
		HasShortnessOfBreath:   wBool(m.raw("has_shortness_of_breath")),
		HasBloodPressureDevice: bpd,
		SystolicPressure:       wInt(m.raw("systolic_pressure")),
		DiastolicPressure:      wInt(m.raw("diastolic_pressure")),
		FastingBloodSugar:      wDecimal(m.raw("fasting_blood_sugar")),
		PostMealBloodSugar:     wDecimal(m.raw("post_meal_blood_sugar")),
		OverallMood:            wString(m.raw("overall_mood")),
		HasAnxiety:             wBool(m.raw("has_anxiety")),
		AnxietySeverity:        wString(m.raw("anxiety_severity")),
		HasMoodSwings:          wBool(m.raw("has_mood_swings")),
		MoodSwingsSeverity:     wString(m.raw("mood_swings_severity")),
		HasDepressionFeelings:  wBool(m.raw("has_depression_feelings")),
		DepressionSeverity:     wString(m.raw("depression_severity")),
		Notes:                  wString(m.raw("notes")),
		UpdatedAt:              wTime(m.raw("updated_at")),
	}
	if id := m.raw("id"); id != nil {
		p.ID = uint64(id.(int64)) //nolint:gosec // G115: ids are positive
	}
	return p, err
}

func weeklyInsertParams(m *model, userID uint64) (store.InsertWeeklyLogParams, error) {
	u, err := weeklyUpdateParams(m)
	return store.InsertWeeklyLogParams{
		UserID: userID, LogDate: u.LogDate, PregnancyWeek: wInt(m.raw("pregnancy_week")).Int32,
		Weight: u.Weight, SwellingLocations: u.SwellingLocations, HasSwelling: u.HasSwelling,
		HasShortnessOfBreath: u.HasShortnessOfBreath, HasBloodPressureDevice: u.HasBloodPressureDevice,
		SystolicPressure: u.SystolicPressure, DiastolicPressure: u.DiastolicPressure,
		FastingBloodSugar: u.FastingBloodSugar, PostMealBloodSugar: u.PostMealBloodSugar,
		OverallMood: u.OverallMood, HasAnxiety: u.HasAnxiety, AnxietySeverity: u.AnxietySeverity,
		HasMoodSwings: u.HasMoodSwings, MoodSwingsSeverity: u.MoodSwingsSeverity,
		HasDepressionFeelings: u.HasDepressionFeelings, DepressionSeverity: u.DepressionSeverity,
		Notes: u.Notes, CreatedAt: wTime(m.raw("created_at")), UpdatedAt: u.UpdatedAt,
	}, err
}

// ---- PregnancyFetalMovement ----

var fetalCasts = casts{"log_date": castYMD, "created_at": castDateTime, "updated_at": castDateTime}

func fetalModel(r *store.PregnancyFetalMovement) *model {
	return newModel(fetalCasts).
		set("id", i64(r.ID)).
		set("user_id", i64(r.UserID)).
		set("log_date", r.LogDate).
		set("pregnancy_week", int64(r.PregnancyWeek)).
		set("movement_status", r.MovementStatus).
		set("movement_count", rawInt(r.MovementCount)).
		set("first_movement_time", rawString(r.FirstMovementTime)).
		set("last_movement_time", rawString(r.LastMovementTime)).
		set("notes", rawString(r.Notes)).
		set("created_at", rawTime(r.CreatedAt)).
		set("updated_at", rawTime(r.UpdatedAt))
}

func fetalUpdateParams(m *model) (store.UpdateFetalMovementParams, error) {
	var err error
	week := wInt(m.raw("pregnancy_week"))
	if !week.Valid {
		err = errNotNull("pregnancy_week")
	}
	status := wString(m.raw("movement_status"))
	if !status.Valid && err == nil {
		err = errNotNull("movement_status")
	}
	p := store.UpdateFetalMovementParams{
		PregnancyWeek:     week.Int32,
		MovementStatus:    status.String,
		MovementCount:     wInt(m.raw("movement_count")),
		FirstMovementTime: wString(m.raw("first_movement_time")),
		LastMovementTime:  wString(m.raw("last_movement_time")),
		Notes:             wString(m.raw("notes")),
		UpdatedAt:         wTime(m.raw("updated_at")),
	}
	if id := m.raw("id"); id != nil {
		p.ID = uint64(id.(int64)) //nolint:gosec // G115: ids are positive
	}
	return p, err
}

func fetalInsertParams(m *model, userID uint64) (store.InsertFetalMovementParams, error) {
	u, err := fetalUpdateParams(m)
	return store.InsertFetalMovementParams{
		UserID: userID, LogDate: wDate(m.raw("log_date")).Date, PregnancyWeek: u.PregnancyWeek,
		MovementStatus: u.MovementStatus, MovementCount: u.MovementCount,
		FirstMovementTime: u.FirstMovementTime, LastMovementTime: u.LastMovementTime,
		Notes: u.Notes, CreatedAt: wTime(m.raw("created_at")), UpdatedAt: u.UpdatedAt,
	}, err
}

// ---- PregnancyAlert ----

var alertCasts = casts{
	"is_read": castBool, "is_dismissed": castBool, "read_at": castDateTime, "dismissed_at": castDateTime,
	"trigger_symptoms": castArray, "medical_history_flags": castArray, "recommended_actions": castArray,
	"created_at": castDateTime, "updated_at": castDateTime,
}

func alertModel(r *store.PregnancyAlert) *model {
	return newModel(alertCasts).
		set("id", i64(r.ID)).
		set("user_id", i64(r.UserID)).
		set("alert_level", r.AlertLevel).
		set("alert_type", r.AlertType).
		set("title", r.Title).
		set("message", r.Message).
		set("pregnancy_week", rawInt(r.PregnancyWeek)).
		set("trigger_symptoms", rawJSON(r.TriggerSymptoms)).
		set("medical_history_flags", rawJSON(r.MedicalHistoryFlags)).
		set("is_read", rawFlag(r.IsRead)).
		set("is_dismissed", rawFlag(r.IsDismissed)).
		set("read_at", rawTime(r.ReadAt)).
		set("dismissed_at", rawTime(r.DismissedAt)).
		set("recommended_actions", rawJSON(r.RecommendedActions)).
		set("created_at", rawTime(r.CreatedAt)).
		set("updated_at", rawTime(r.UpdatedAt))
}

// AlertJSON is a loaded PregnancyAlert's toArray().
func AlertJSON(r *store.PregnancyAlert) any {
	if r == nil {
		return nil
	}
	return alertModel(r).JSON()
}

// i64 converts an id for a raw attribute (ids always fit int64).
func i64(u uint64) int64 { return int64(u) } //nolint:gosec // G115: ids fit int64
