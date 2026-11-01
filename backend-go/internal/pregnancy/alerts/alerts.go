// Package alerts is what is left of the v1 pregnancy alert rules (PregnancyAlertService,
// backend/app/Services/PregnancyEngine/PregnancyAlertService.php): the bleeding alert of a day-log `flow` value in
// pregnancy (ForLoggedFlow, T-M9-28) and its copy. The v1 save-time rules (symptom, weekly and fetal-movement logs) are
// retired (T-M10-08, D-165): every pregnancy log save evaluates the preg.* rules of the message engine
// (messages/pregnancyalerts → the alerts inbox).
//
// Quirks kept: titles / messages / actions are stored in the request locale ("fa" → Persian, anything else →
// English); medical_history_flags is `[]` when empty, else an object.
package alerts

import (
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/civildate"
	"github.com/ritme/backend-go/internal/platform/jsonx"
	"github.com/ritme/backend-go/internal/platform/validation/phpval"
	"github.com/ritme/backend-go/internal/pregnancy/calc"
	"github.com/ritme/backend-go/internal/pregnancy/store"
)

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
	// Day selects the clinical config version (zero = today in Tehran).
	Day civildate.Date
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
