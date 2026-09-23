// Package alerts holds the rules of PregnancyAlertService
// (backend/app/Services/PregnancyEngine/PregnancyAlertService.php). It is pure: it turns a
// saved log (its attribute values as `$log->attr` returns them, i.e. after Eloquent casts)
// into alert drafts; the pregnancy package persists them.
//
// Quirks kept: titles / messages / actions are stored in the request locale ("fa" →
// Persian, anything else → English); medical_history_flags is `[]` when empty, else an
// object, and is only set on symptom alerts; blood-sugar trigger values are the decimal
// strings of the `decimal:2` cast.
package alerts

import (
	"fmt"

	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

// Attrs reads a model attribute after casts ($log->{$key}); *jsonx.OrderedMap satisfies it.
type Attrs interface {
	Get(key string) (any, bool)
}

// Draft is one PregnancyAlert::create([...]) call. Nil Trigger / MedicalFlags mean the key
// was not part of the create array (column stays NULL, key absent from the JSON).
type Draft struct {
	Level        enums.AlertLevel
	Type         string
	Title        string
	Message      string
	Week         int
	Trigger      *jsonx.OrderedMap
	MedicalFlags any
	Actions      []string
}

// Context is what every rule needs: the request locale, the current pregnancy week
// (calc.CurrentWeek) and the user's profile (nil when none).
type Context struct {
	Locale  string
	Week    int
	Profile *store.PregnancyProfile
}

func (c Context) tr(fa, en string) string {
	if c.Locale == "fa" {
		return fa
	}
	return en
}

func (c Context) trList(fa, en []string) []string {
	if c.Locale == "fa" {
		return fa
	}
	return en
}

// criticalSymptoms are getCriticalSymptoms() in order.
var criticalSymptoms = []string{"spotting", "bleeding", "fluid_leakage", "severe_sudden_pain"}

// severityFields are counted by countSevereSymptoms().
var severityFields = []string{
	"nausea_severity", "vomiting_severity", "fatigue_severity",
	"headache_severity", "dizziness_severity", "breast_pain_severity",
	"lower_abdominal_pain_severity", "cramping_severity",
	"back_pain_severity", "pelvic_pressure_severity",
}

// ForSymptomLog is processSymptomLog().
func ForSymptomLog(ctx Context, log Attrs) []Draft {
	var out []Draft
	for _, s := range criticalSymptoms {
		if !truthy(log, "has_"+s) {
			continue
		}
		sev, _ := log.Get(s + "_severity")
		if d, ok := ctx.symptomAlert(s, sev); ok {
			out = append(out, d)
		}
	}
	severe := 0
	for _, f := range severityFields {
		if v, _ := log.Get(f); v == "severe" {
			severe++
		}
	}
	if severe >= 3 {
		out = append(out, Draft{
			Level:        enums.AlertLevelWarning,
			Type:         "symptom_based",
			Title:        ctx.tr("هشدار: علائم متعدد شدید", "Warning: Multiple Severe Symptoms"),
			Message:      ctx.tr("چندین علامت شدید گزارش شده است. لطفاً با پزشک خود مشورت کنید.", "Multiple severe symptoms reported. Please consult your doctor."),
			Week:         ctx.Week,
			Trigger:      jsonx.Obj("multiple_severe", true),
			MedicalFlags: MedicalHistoryFlags(ctx.Profile),
			Actions: ctx.trList(
				[]string{"با پزشک تماس بگیرید", "استراحت کنید", "علائم را پیگیری کنید"},
				[]string{"Contact your doctor", "Rest", "Monitor symptoms"}),
		})
	}
	return out
}

// symptomAlert is createSymptomAlert() + getSymptomAlertData().
func (ctx Context) symptomAlert(symptom string, severity any) (Draft, bool) {
	d := Draft{
		Type:         "symptom_based",
		Week:         ctx.Week,
		Trigger:      jsonx.Obj(symptom, severity),
		MedicalFlags: MedicalHistoryFlags(ctx.Profile),
	}
	switch symptom {
	case "bleeding":
		if severity == "severe" || calc.IsHighRisk(ctx.Profile) {
			d.Level = enums.AlertLevelEmergency
			d.Title = ctx.tr("اورژانس: خونریزی شدید", "Emergency: Heavy Bleeding")
			d.Message = ctx.tr("خونریزی شدید در دوران بارداری نیاز به توجه فوری پزشکی دارد. لطفاً فوراً به اورژانس مراجعه کنید.",
				"Heavy bleeding during pregnancy requires immediate medical attention. Please go to emergency immediately.")
			d.Actions = ctx.trList([]string{"فوراً به اورژانس مراجعه کنید", "با پزشک تماس بگیرید"},
				[]string{"Go to emergency immediately", "Call your doctor"})
			return d, true
		}
		d.Level = enums.AlertLevelWarning
		d.Title = ctx.tr("هشدار: خونریزی", "Warning: Bleeding")
		d.Message = ctx.tr("خونریزی گزارش شده است. لطفاً با پزشک خود تماس بگیرید.", "Bleeding has been reported. Please contact your doctor.")
		d.Actions = ctx.trList([]string{"با پزشک تماس بگیرید", "استراحت کنید"}, []string{"Contact your doctor", "Rest"})
		return d, true
	case "spotting":
		if ctx.Week <= 12 || severity == "severe" {
			d.Level = enums.AlertLevelWarning
			d.Title = ctx.tr("هشدار: لکه‌بینی", "Warning: Spotting")                                                     //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
			d.Message = ctx.tr("لکه‌بینی گزارش شده است. لطفاً وضعیت را پیگیری کنید و در صورت تداوم با پزشک مشورت کنید.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
				"Spotting has been reported. Please monitor and consult your doctor if it continues.")
			d.Actions = ctx.trList([]string{"وضعیت را پیگیری کنید", "استراحت کنید", "در صورت تداوم با پزشک تماس بگیرید"},
				[]string{"Monitor the situation", "Rest", "Contact doctor if it continues"})
			return d, true
		}
		d.Level = enums.AlertLevelInfo
		d.Title = ctx.tr("توجه: لکه‌بینی خفیف", "Note: Light Spotting")                             //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
		d.Message = ctx.tr("لکه‌بینی خفیف گزارش شده است. این می‌تواند طبیعی باشد اما پیگیری کنید.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
			"Light spotting has been reported. This can be normal but please monitor.")
		d.Actions = ctx.trList([]string{"وضعیت را پیگیری کنید", "در صورت افزایش با پزشک تماس بگیرید"},
			[]string{"Monitor the situation", "Contact doctor if it increases"})
		return d, true
	case "fluid_leakage":
		d.Level = enums.AlertLevelEmergency
		d.Title = ctx.tr("اورژانس: خروج مایع", "Emergency: Fluid Leakage")
		d.Message = ctx.tr("خروج مایع ممکن است نشانه پارگی کیسه آب باشد. لطفاً فوراً به بیمارستان مراجعه کنید.",
			"Fluid leakage may indicate ruptured membranes. Please go to hospital immediately.")
		d.Actions = ctx.trList([]string{"فوراً به بیمارستان مراجعه کنید", "حرکت نکنید و دراز بکشید"},
			[]string{"Go to hospital immediately", "Lie down and avoid movement"})
		return d, true
	case "severe_sudden_pain":
		d.Level = enums.AlertLevelEmergency
		d.Title = ctx.tr("اورژانس: درد شدید", "Emergency: Severe Pain")
		d.Message = ctx.tr("درد شدید و ناگهانی نیاز به بررسی فوری پزشکی دارد.", "Severe sudden pain requires immediate medical evaluation.")
		d.Actions = ctx.trList([]string{"فوراً به اورژانس مراجعه کنید", "با اورژانس ۱۱۵ تماس بگیرید"},
			[]string{"Go to emergency immediately", "Call emergency services"})
		return d, true
	}
	return Draft{}, false
}

// ForWeeklyLog is processWeeklyLog(): high blood pressure (≥140 / ≥90), high blood sugar
// (fasting > 95, post-meal > 140), severe depression feelings.
func ForWeeklyLog(ctx Context, log Attrs) []Draft {
	var out []Draft
	sys, _ := log.Get("systolic_pressure")
	dia, _ := log.Get("diastolic_pressure")
	if HighBloodPressure(sys, dia) {
		out = append(out, Draft{
			Level: enums.AlertLevelWarning,
			Type:  "symptom_based",
			Title: ctx.tr("هشدار: فشار خون بالا", "Warning: High Blood Pressure"),
			Message: ctx.tr(
				fmt.Sprintf("فشار خون شما (%s/%s) بالاتر از حد طبیعی است. لطفاً با پزشک مشورت کنید.", str(sys), str(dia)),
				fmt.Sprintf("Your blood pressure (%s/%s) is above normal. Please consult your doctor.", str(sys), str(dia))),
			Week:    ctx.Week,
			Trigger: jsonx.Obj("systolic", sys, "diastolic", dia),
			Actions: ctx.trList([]string{"با پزشک تماس بگیرید", "استراحت کنید", "مصرف نمک را کاهش دهید"},
				[]string{"Contact your doctor", "Rest", "Reduce salt intake"}),
		})
	}
	fasting, _ := log.Get("fasting_blood_sugar")
	postMeal, _ := log.Get("post_meal_blood_sugar")
	if HighBloodSugar(fasting, postMeal) {
		out = append(out, Draft{
			Level: enums.AlertLevelWarning,
			Type:  "symptom_based",
			Title: ctx.tr("هشدار: قند خون بالا", "Warning: High Blood Sugar"),
			Message: ctx.tr("سطح قند خون شما بالاتر از حد طبیعی است. لطفاً با پزشک مشورت کنید.",
				"Your blood sugar level is above normal. Please consult your doctor."),
			Week:    ctx.Week,
			Trigger: jsonx.Obj("fasting", fasting, "post_meal", postMeal),
			Actions: ctx.trList([]string{"با پزشک تماس بگیرید", "رژیم غذایی را بررسی کنید"},
				[]string{"Contact your doctor", "Review your diet"}),
		})
	}
	if sev, _ := log.Get("depression_severity"); truthy(log, "has_depression_feelings") && sev == "severe" {
		out = append(out, Draft{
			Level: enums.AlertLevelWarning,
			Type:  "symptom_based",
			Title: ctx.tr("توجه: وضعیت روحی", "Note: Mental Health"),
			Message: ctx.tr("احساسات افسردگی شدید گزارش شده است. صحبت با پزشک یا مشاور می‌تواند کمک‌کننده باشد.", //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
				"Severe depression feelings reported. Talking to a doctor or counselor can be helpful."),
			Week: ctx.Week,
			Actions: ctx.trList([]string{"با پزشک یا مشاور صحبت کنید", "از حمایت خانواده بهره بگیرید"},
				[]string{"Talk to a doctor or counselor", "Seek family support"}),
		})
	}
	return out
}

// ForFetalMovement is processFetalMovement(): from week 24, `none` is an emergency and
// `reduced` a warning.
func ForFetalMovement(ctx Context, log Attrs) []Draft {
	status, _ := log.Get("movement_status")
	if ctx.Week < 24 || (status != string(enums.FetalMovementStatusReduced) && status != string(enums.FetalMovementStatusNone)) {
		return nil
	}
	d := Draft{Type: "symptom_based", Week: ctx.Week, Trigger: jsonx.Obj("fetal_movement", status)}
	if status == string(enums.FetalMovementStatusNone) {
		d.Level = enums.AlertLevelEmergency
		d.Title = ctx.tr("اورژانس: عدم حرکت جنین", "Emergency: No Fetal Movement")
		d.Message = ctx.tr("عدم حرکت جنین گزارش شده است. لطفاً فوراً با پزشک تماس بگیرید یا به بیمارستان مراجعه کنید.",
			"No fetal movement reported. Please contact your doctor immediately or go to hospital.")
		d.Actions = ctx.trList([]string{"فوراً به بیمارستان مراجعه کنید", "با پزشک تماس بگیرید"},
			[]string{"Go to hospital immediately", "Contact your doctor"})
	} else {
		d.Level = enums.AlertLevelWarning
		d.Title = ctx.tr("هشدار: کاهش حرکت جنین", "Warning: Reduced Fetal Movement")
		d.Message = ctx.tr("کاهش حرکات جنین گزارش شده است. لطفاً با پزشک خود مشورت کنید.",
			"Reduced fetal movement reported. Please consult your doctor.")
		d.Actions = ctx.trList([]string{"با پزشک تماس بگیرید", "دراز بکشید و حرکات را پیگیری کنید"},
			[]string{"Contact your doctor", "Lie down and monitor movements"})
	}
	return []Draft{d}
}

// MissingData is createMissingDataAlert() (unused by the API, kept for parity).
func MissingData(ctx Context, dataType string, daysMissing int) Draft {
	var title string
	switch dataType {
	case "symptoms":
		title = ctx.tr("یادآوری: ثبت علائم", "Reminder: Log Symptoms")
	case "fetal_movement":
		title = ctx.tr("یادآوری: ثبت حرکات جنین", "Reminder: Log Fetal Movement")
	case "weekly":
		title = ctx.tr("یادآوری: ثبت اطلاعات هفتگی", "Reminder: Log Weekly Data")
	default:
		title = ctx.tr("یادآوری", "Reminder")
	}
	return Draft{
		Level: enums.AlertLevelInfo,
		Type:  "missing_data",
		Title: title,
		Message: ctx.tr(
			fmt.Sprintf("شما %d روز است که داده‌ای ثبت نکرده‌اید. ثبت منظم به پیگیری بهتر سلامت شما کمک می‌کند.", daysMissing), //nolint:staticcheck // ST1018: Persian text (ZWNJ) verbatim from Laravel
			fmt.Sprintf("You haven't logged data for %d days. Regular logging helps track your health better.", daysMissing)),
		Week:    ctx.Week,
		Actions: ctx.trList([]string{"علائم امروز را ثبت کنید"}, []string{"Log today's symptoms"}),
	}
}

// MedicalHistoryFlags is getMedicalHistoryFlags(): `[]` without a profile or flags,
// else {miscarriage_history, high_risk_history, rh_negative, pre_existing_conditions}.
func MedicalHistoryFlags(p *store.PregnancyProfile) *jsonx.OrderedMap {
	flags := jsonx.NewArray()
	if p == nil {
		return flags
	}
	if p.HasMiscarriageHistory.Valid && p.HasMiscarriageHistory.Bool {
		flags.Set("miscarriage_history", true)
	}
	if p.HasHighRiskHistory.Valid && p.HasHighRiskHistory.Bool {
		flags.Set("high_risk_history", true)
	}
	if p.RhFactor.Valid && p.RhFactor.String == "negative" {
		flags.Set("rh_negative", true)
	}
	if p.PreExistingConditions.Valid {
		if v, err := phpval.Decode(p.PreExistingConditions.V); err == nil && phpval.Truthy(v) {
			flags.Set("pre_existing_conditions", p.PreExistingConditions.V)
		}
	}
	return flags
}

// HighBloodPressure is PregnancyWeeklyLog::hasHighBloodPressure().
func HighBloodPressure(systolic, diastolic any) bool {
	if systolic == nil || diastolic == nil {
		return false
	}
	return num(systolic) >= 140 || num(diastolic) >= 90
}

// HighBloodSugar is PregnancyWeeklyLog::hasHighBloodSugar().
func HighBloodSugar(fasting, postMeal any) bool {
	return (fasting != nil && num(fasting) > 95) || (postMeal != nil && num(postMeal) > 140)
}

func truthy(a Attrs, key string) bool {
	v, _ := a.Get(key)
	return phpval.Truthy(plain(v))
}

// plain unwraps the jsonx cast wrappers to PHP scalars.
func plain(v any) any {
	switch x := v.(type) {
	case jsonx.DecimalString:
		if !x.Valid() {
			return nil
		}
		return x.String()
	case jsonx.Float:
		return float64(x)
	case int:
		return int64(x)
	case int32:
		return int64(x)
	}
	return v
}

func num(v any) float64 {
	p := plain(v)
	if phpval.IsNumeric(p) {
		return phpval.ToFloat(p)
	}
	return 0
}

// str is PHP string interpolation of an attribute.
func str(v any) string { return phpval.ToString(plain(v)) }
