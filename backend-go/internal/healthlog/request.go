package healthlog

import (
	"time"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/i18n/lang"
	"github.com/ritme/backend-go/internal/platform/validation"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
)

// storeRules is StoreDailyHealthLogRequest::rules(), in its order (= the error order and the
// order of the validated attributes).
func storeRules() validation.Rules {
	in := validation.In
	intensity := enums.IntensityValues()
	pain := enums.PainIntensityValues()
	smell := enums.SmellValues()
	F := validation.F
	return validation.Rules{
		F("log_date", "required", "date", "before_or_equal:today"),

		// 1. Menstruation & Bleeding
		F("bleeding_intensity", "nullable", in(intensity...)),
		F("blood_color", "nullable", in(enums.BloodColorValues()...)),
		F("has_clots", "nullable", "boolean"),
		F("clots_amount", "nullable", in(enums.ClotsAmountValues()...)),
		F("spotting", "nullable", "boolean"),
		F("bleeding_smell", "nullable", in(smell...)),

		// 2. Symptoms - Pains
		F("headache_intensity", "nullable", in(pain...)),
		F("stomach_ache_intensity", "nullable", in(pain...)),
		F("pelvic_pain_intensity", "nullable", in(pain...)),
		F("breast_pain_intensity", "nullable", in(pain...)),
		F("back_pain_intensity", "nullable", in(pain...)),
		F("ovarian_pain_intensity", "nullable", in(pain...)),

		// 2. Symptoms - Digestive
		F("nausea_intensity", "nullable", in(pain...)),
		F("bloating_intensity", "nullable", in(pain...)),
		F("diarrhea", "nullable", "boolean"),
		F("constipation", "nullable", "boolean"),
		F("appetite_change", "nullable", in("loss", "gain", "normal")),
		F("food_craving", "nullable", "boolean"),

		// 2. Symptoms - Breast & Genital
		F("breast_sensitivity_intensity", "nullable", in(pain...)),
		F("vaginal_dryness", "nullable", "boolean"),
		F("vaginal_burning", "nullable", "boolean"),
		F("vaginal_burning_intensity", "nullable", in(pain...)),
		F("vaginal_itching", "nullable", "boolean"),
		F("vaginal_itching_intensity", "nullable", in(pain...)),
		F("vaginal_smell_change", "nullable", "boolean"),
		F("urination_change", "nullable", in("increase", "decrease", "normal")),
		F("urination_burning_intensity", "nullable", in(pain...)),

		// 2. Symptoms - Skin & Hair
		F("acne", "nullable", "boolean"),
		F("oily_skin", "nullable", "boolean"),
		F("hair_loss", "nullable", "boolean"),
		F("swelling", "nullable", "boolean"),

		// 2. Symptoms - Energy & General
		F("fatigue", "nullable", "boolean"),
		F("dizziness", "nullable", "boolean"),
		F("hot_flashes", "nullable", "boolean"),
		F("chills", "nullable", "boolean"),

		// 3. Mood & Emotions
		F("moods", "nullable", "array"),
		F("moods.*", "required", in(enums.MoodValues()...)),

		// 4. Sleep
		F("sleep_duration", "nullable", in(enums.SleepDurationValues()...)),
		F("sleep_quality", "nullable", in(enums.SleepQualityValues()...)),

		// 4b. Exercise
		F("exercise_type", "nullable", "array"),
		F("exercise_type.*", "required", in(enums.ExerciseTypeValues()...)),
		F("exercise_duration", "nullable", "integer", "min:1", "max:600"),
		F("exercise_intensity", "nullable", in(enums.ExerciseIntensityValues()...)),

		// 5. Sexual Activity (the full legacy value list stays accepted)
		F("sexual_activities", "nullable", "array"),
		F("sexual_activities.*", "required", in(enums.SexualActivityValues()...)),
		F("sexual_desire", "nullable", in(enums.SexualDesireValues()...)),
		F("intercourse_type", "nullable", in(enums.IntercourseTypeValues()...)),

		// 6. Vital Signs
		F("weight", "nullable", "numeric", "min:20", "max:300"),
		F("basal_body_temperature", "nullable", "numeric", "min:35", "max:42"),
		F("heart_rate", "nullable", "integer", "min:30", "max:250"),
		F("systolic_pressure", "nullable", "integer", "min:50", "max:300"),
		F("diastolic_pressure", "nullable", "integer", "min:30", "max:200"),
		F("blood_sugar", "nullable", "numeric", "min:20", "max:600"),
		F("energy_level", "nullable", in(enums.EnergyLevelValues()...)),

		// 7. Vaginal Discharge
		F("discharge_color", "nullable", in(enums.DischargeColorValues()...)),
		F("discharge_texture", "nullable", in(enums.DischargeTextureValues()...)),
		F("discharge_amount", "nullable", in(enums.AmountValues()...)),
		F("discharge_smell", "nullable", in(smell...)),
		F("discharge_itching", "nullable", "boolean"),
		F("discharge_burning", "nullable", "boolean"),

		// 8. Digestive & Urinary
		F("frequent_urination", "nullable", "boolean"),

		// 9. Medications & Supplements
		F("medications", "nullable", "array"),
		F("medications.painkillers", "nullable", "string", "max:255"),
		F("medications.hormonal_pills", "nullable", "string", "max:255"),
		F("medications.antibiotics", "nullable", "string", "max:255"),
		F("medications.supplements", "nullable", "string", "max:255"),

		// 10. Notes
		F("notes", "nullable", "string", "max:2000"),
	}
}

// storeMessages is StoreDailyHealthLogRequest::messages() (English in every locale).
var storeMessages = validation.Messages(
	"log_date.required", "Date is required",
	"log_date.date", "Invalid date format",
	"log_date.before_or_equal", "Date cannot be in the future",
	"weight.min", "Weight must be at least 20 kg",
	"weight.max", "Weight cannot exceed 300 kg",
	"basal_body_temperature.min", "Temperature must be at least 35°C",
	"basal_body_temperature.max", "Temperature cannot exceed 42°C",
	"heart_rate.min", "Heart rate must be at least 30 bpm",
	"heart_rate.max", "Heart rate cannot exceed 250 bpm",
	"systolic_pressure.min", "Systolic pressure must be at least 50 mmHg",
	"systolic_pressure.max", "Systolic pressure cannot exceed 300 mmHg",
	"diastolic_pressure.min", "Diastolic pressure must be at least 30 mmHg",
	"diastolic_pressure.max", "Diastolic pressure cannot exceed 200 mmHg",
	"blood_sugar.min", "Blood sugar must be at least 20 mg/dl",
	"blood_sugar.max", "Blood sugar cannot exceed 600 mg/dl",
	"exercise_duration.min", "Exercise duration must be at least 1 minute",
	"exercise_duration.max", "Exercise duration cannot exceed 600 minutes",
	"notes.max", "Notes cannot exceed 2000 characters",
)

// prepareForValidation is StoreDailyHealthLogRequest::prepareForValidation(): a string
// exercise_type becomes a one-element list ("" cannot reach it: ConvertEmptyStringsToNull
// already turned it into null).
func prepareForValidation(input phpval.Map) {
	if t, ok := input.Get("exercise_type"); ok {
		if s, isStr := t.(string); isStr {
			if s == "" {
				input.Set("exercise_type", nil)
			} else {
				input.Set("exercise_type", []any{s})
			}
		}
	}
}

// ValidateStore runs the FormRequest: nil and the validated attributes, or the framework 422.
func ValidateStore(input phpval.Map, locale string, now time.Time) (phpval.Map, error) {
	prepareForValidation(input)
	v := validation.Make(lang.Default(), locale, input, storeRules(), validation.Now(now), storeMessages)
	if v.Fails() {
		return nil, v.Errors()
	}
	return v.Validated(), nil
}
