package pregnancy

import (
	v "github.com/ritme/backend-go/internal/platform/validation"

	"github.com/ritme/backend-go/internal/enums"
)

// FormRequests of the pregnancy controllers (backend/app/Http/Requests/Api/V1/*Pregnancy*):
// rules in declaration order and their Persian custom messages.

// storeProfileRules is StorePregnancyProfileRequest (POST /pregnancy/onboarding).
func storeProfileRules() v.Rules {
	return v.Rules{
		v.F("age_source", "required", v.In(enums.PregnancyAgeSourceValues()...)),
		v.F("lmp_date", "nullable|required_if:age_source,lmp|date|before_or_equal:today"),
		v.F("ultrasound_date", "nullable|required_if:age_source,ultrasound|date|before_or_equal:today"),
		v.F("ultrasound_weeks", "nullable|required_if:age_source,ultrasound|integer|min:1|max:42"),
		v.F("ultrasound_days", "nullable|required_if:age_source,ultrasound|integer|min:0|max:6"),
		v.F("manual_weeks", "nullable|required_if:age_source,manual|integer|min:1|max:42"),
		v.F("manual_days", "nullable|required_if:age_source,manual|integer|min:0|max:6"),
		v.F("has_miscarriage_history", "nullable|boolean"),
		v.F("has_high_risk_history", "nullable|boolean"),
		v.F("pre_existing_conditions", "nullable|array"),
		v.F("pre_existing_conditions.*", "required", v.In(enums.PreExistingConditionValues()...)),
		v.F("blood_type", "nullable", v.In(enums.BloodTypeValues()...)),
		v.F("rh_factor", "nullable", v.In(enums.RhFactorValues()...)),
	}
}

var storeProfileMessages = []string{
	"age_source.required", "منبع محاسبه سن بارداری الزامی است",
	"age_source.in", "منبع محاسبه سن بارداری نامعتبر است",
	"lmp_date.required_if", "تاریخ آخرین پریود برای این روش محاسبه الزامی است",
	"lmp_date.date", "فرمت تاریخ نامعتبر است",
	"lmp_date.before_or_equal", "تاریخ نمی‌تواند در آینده باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_date.required_if", "تاریخ سونوگرافی برای این روش محاسبه الزامی است",
	"ultrasound_weeks.required_if", "هفته بارداری اعلام‌شده در سونو الزامی است", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_weeks.min", "هفته بارداری باید حداقل 1 باشد",
	"ultrasound_weeks.max", "هفته بارداری نمی‌تواند بیشتر از 42 باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_days.required_if", "روز بارداری اعلام‌شده در سونو الزامی است", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_days.min", "روز بارداری باید بین 0 تا 6 باشد",
	"ultrasound_days.max", "روز بارداری باید بین 0 تا 6 باشد",
	"manual_weeks.required_if", "هفته بارداری برای ورود دستی الزامی است",
	"manual_days.required_if", "روز بارداری برای ورود دستی الزامی است",
}

// updateProfileRules is UpdatePregnancyProfileRequest (PUT /pregnancy/profile).
func updateProfileRules() v.Rules {
	return v.Rules{
		v.F("age_source", "nullable", v.In(enums.PregnancyAgeSourceValues()...)),
		v.F("lmp_date", "nullable|date|before_or_equal:today"),
		v.F("ultrasound_date", "nullable|date|before_or_equal:today"),
		v.F("ultrasound_weeks", "nullable|integer|min:1|max:42"),
		v.F("ultrasound_days", "nullable|integer|min:0|max:6"),
		v.F("manual_weeks", "nullable|integer|min:1|max:42"),
		v.F("manual_days", "nullable|integer|min:0|max:6"),
		v.F("has_miscarriage_history", "nullable|boolean"),
		v.F("has_high_risk_history", "nullable|boolean"),
		v.F("pre_existing_conditions", "nullable|array"),
		v.F("pre_existing_conditions.*", "required", v.In(enums.PreExistingConditionValues()...)),
		v.F("blood_type", "nullable", v.In(enums.BloodTypeValues()...)),
		v.F("rh_factor", "nullable", v.In(enums.RhFactorValues()...)),
		v.F("fetal_movement_felt", "nullable|boolean"),
		v.F("first_fetal_movement_date", "nullable|date|before_or_equal:today"),
	}
}

var updateProfileMessages = []string{
	"age_source.in", "منبع محاسبه سن بارداری نامعتبر است",
	"lmp_date.date", "فرمت تاریخ نامعتبر است",
	"lmp_date.before_or_equal", "تاریخ نمی‌تواند در آینده باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_weeks.min", "هفته بارداری باید حداقل 1 باشد",
	"ultrasound_weeks.max", "هفته بارداری نمی‌تواند بیشتر از 42 باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"ultrasound_days.min", "روز بارداری باید بین 0 تا 6 باشد",
	"ultrasound_days.max", "روز بارداری باید بین 0 تا 6 باشد",
}

// symptomRules is StorePregnancySymptomLogRequest.
func symptomRules() v.Rules {
	rules := v.Rules{v.F("log_date", "required|date|before_or_equal:today")}
	sev := v.In(enums.SymptomSeverityValues()...)
	for _, s := range symptomNames {
		rules = append(rules, v.F("has_"+s, "nullable|boolean"), v.F(s+"_severity", "nullable", sev))
	}
	return append(rules, v.F("notes", "nullable|string|max:2000"))
}

var symptomMessages = []string{
	"log_date.required", "تاریخ الزامی است",
	"log_date.date", "فرمت تاریخ نامعتبر است",
	"log_date.before_or_equal", "تاریخ نمی‌تواند در آینده باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"notes.max", "یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
}

// weeklyRules is StorePregnancyWeeklyLogRequest.
func weeklyRules() v.Rules {
	sev := v.In(enums.SymptomSeverityValues()...)
	return v.Rules{
		v.F("log_date", "required|date|before_or_equal:today"),
		v.F("pregnancy_week", "required|integer|min:1|max:42"),
		v.F("weight", "nullable|numeric|min:30|max:200"),
		v.F("has_swelling", "nullable|boolean"),
		v.F("swelling_locations", "nullable|array"),
		v.F("swelling_locations.*", "required", v.In(enums.SwellingLocationValues()...)),
		v.F("has_shortness_of_breath", "nullable|boolean"),
		v.F("has_blood_pressure_device", "nullable|boolean"),
		v.F("systolic_pressure", "nullable|integer|min:60|max:250"),
		v.F("diastolic_pressure", "nullable|integer|min:40|max:150"),
		v.F("fasting_blood_sugar", "nullable|numeric|min:40|max:400"),
		v.F("post_meal_blood_sugar", "nullable|numeric|min:40|max:500"),
		v.F("overall_mood", "nullable", v.In(enums.MentalHealthStatusValues()...)),
		v.F("has_anxiety", "nullable|boolean"),
		v.F("anxiety_severity", "nullable", sev),
		v.F("has_mood_swings", "nullable|boolean"),
		v.F("mood_swings_severity", "nullable", sev),
		v.F("has_depression_feelings", "nullable|boolean"),
		v.F("depression_severity", "nullable", sev),
		v.F("notes", "nullable|string|max:2000"),
	}
}

var weeklyMessages = []string{
	"log_date.required", "تاریخ الزامی است",
	"log_date.date", "فرمت تاریخ نامعتبر است",
	"log_date.before_or_equal", "تاریخ نمی‌تواند در آینده باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"pregnancy_week.required", "هفته بارداری الزامی است",
	"pregnancy_week.min", "هفته بارداری باید حداقل 1 باشد",
	"pregnancy_week.max", "هفته بارداری نمی‌تواند بیشتر از 42 باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"weight.min", "وزن باید حداقل 30 کیلوگرم باشد",
	"weight.max", "وزن نمی‌تواند بیشتر از 200 کیلوگرم باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"systolic_pressure.min", "فشار خون سیستولیک نامعتبر است",
	"systolic_pressure.max", "فشار خون سیستولیک نامعتبر است",
	"diastolic_pressure.min", "فشار خون دیاستولیک نامعتبر است",
	"diastolic_pressure.max", "فشار خون دیاستولیک نامعتبر است",
	"notes.max", "یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
}

// fetalRules is StorePregnancyFetalMovementRequest.
func fetalRules() v.Rules {
	return v.Rules{
		v.F("log_date", "required|date|before_or_equal:today"),
		v.F("pregnancy_week", "required|integer|min:1|max:42"),
		v.F("movement_status", "required", v.In(enums.FetalMovementStatusValues()...)),
		v.F("movement_count", "nullable|integer|min:0|max:100"),
		v.F("first_movement_time", "nullable|date_format:H:i"),
		v.F("last_movement_time", "nullable|date_format:H:i"),
		v.F("notes", "nullable|string|max:2000"),
	}
}

var fetalMessages = []string{
	"log_date.required", "تاریخ الزامی است",
	"log_date.date", "فرمت تاریخ نامعتبر است",
	"log_date.before_or_equal", "تاریخ نمی‌تواند در آینده باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"pregnancy_week.required", "هفته بارداری الزامی است",
	"pregnancy_week.min", "هفته بارداری باید حداقل 1 باشد",
	"pregnancy_week.max", "هفته بارداری نمی‌تواند بیشتر از 42 باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"movement_status.required", "وضعیت حرکت جنین الزامی است",
	"movement_status.in", "وضعیت حرکت جنین نامعتبر است",
	"movement_count.min", "تعداد حرکات نمی‌تواند منفی باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
	"movement_count.max", "تعداد حرکات نامعتبر است",
	"first_movement_time.date_format", "فرمت زمان نامعتبر است (HH:MM)",
	"last_movement_time.date_format", "فرمت زمان نامعتبر است (HH:MM)",
	"notes.max", "یادداشت نمی‌تواند بیشتر از 2000 کاراکتر باشد", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
}
