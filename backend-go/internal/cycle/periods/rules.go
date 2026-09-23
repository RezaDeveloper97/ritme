package periods

import (
	"github.com/ritme/backend-go/internal/enums"
	"github.com/ritme/backend-go/internal/platform/jsonx"
)

// PeriodLogController constants.
const (
	longPeriodDays        = 10
	minPlausibleCycleDays = 21
	maxPlausibleCycleDays = 45
	closeStartDays        = 14
	hardCapDays           = 12
)

// QualityFlags is PeriodLogController::qualityFlags: the stored data_quality_flags.
func QualityFlags(cycleLength, bleedingLength *int) []string {
	flags := []string{}
	if bleedingLength != nil && *bleedingLength > longPeriodDays {
		flags = append(flags, string(enums.DataQualityFlagLongPeriod))
	}
	if bleedingLength != nil && *bleedingLength == 1 {
		flags = append(flags, string(enums.DataQualityFlagShortPeriod))
	}
	if cycleLength != nil && (*cycleLength < minPlausibleCycleDays || *cycleLength > maxPlausibleCycleDays) {
		flags = append(flags, string(enums.DataQualityFlagIrregularCycle))
	}
	return flags
}

// Warnings is PeriodLogController::periodWarnings: `[{type, message}]`, copy hardcoded fa / else
// English exactly as in PHP (`$locale === 'fa'`).
func Warnings(cycleLength, bleedingLength *int, locale string) []any {
	t := func(fa, en string) string {
		if locale == "fa" {
			return fa
		}
		return en
	}
	warnings := []any{}
	add := func(typ, msg string) { warnings = append(warnings, jsonx.Obj("type", typ, "message", msg)) }

	if bleedingLength != nil && *bleedingLength > longPeriodDays {
		add(string(enums.DataQualityFlagLongPeriod), t(
			"این بازه طولانی\u200cتر از معمول است. اگر مطمئنی، می\u200cتوانی آن را ثبت کنی.",
			"This range is longer than usual. You can still log it if you are sure."))
	}
	if bleedingLength != nil && *bleedingLength == 1 {
		add(string(enums.DataQualityFlagShortPeriod), t(
			"آیا این خون\u200cریزی واقعاً پریود بود یا لکه\u200cبینی؟ می\u200cتوانی همچنان ثبتش کنی.",
			"Was this really a period or spotting? You can still log it."))
	}
	switch {
	case cycleLength != nil && *cycleLength > 0 && *cycleLength < closeStartDays:
		add("close_to_previous", t(
			"این تاریخ خیلی نزدیک به پریود قبلی است. مطمئنی این یک پریود جدید است و ادامهٔ خون\u200cریزی قبلی نیست؟",
			"This is very close to your previous period. Are you sure it is a new period and not continued bleeding?"))
	case cycleLength != nil && (*cycleLength < minPlausibleCycleDays || *cycleLength > maxPlausibleCycleDays):
		add(string(enums.DataQualityFlagIrregularCycle), t(
			"این فاصله با الگوی معمول چرخه\u200cها متفاوت است و ممکن است روی دقت پیش\u200cبینی اثر بگذارد.",
			"This gap differs from your usual cycle pattern and may affect prediction accuracy."))
	}
	return warnings
}

// Localised 4xx messages of the controller (hardcoded fa / else English).
func msgPreviousOpen(locale string) string {
	return pick(locale, "پایان پریود قبلی هنوز ثبت نشده است. ابتدا پایان آن را مشخص کن.",
		"Your previous period has no end date yet. Please log its end first.")
}

func msgStartOverlap(locale string) string {
	return pick(locale, "این تاریخ با یک پریود ثبت\u200cشدهٔ دیگر همپوشانی دارد. لطفاً یکی از بازه\u200cها را ویرایش کن.",
		"This date overlaps another logged period. Please edit one of the ranges.")
}

func msgRangeOverlap(locale string) string {
	return pick(locale, "این بازه با یک پریود ثبت\u200cشدهٔ دیگر همپوشانی دارد. لطفاً یکی از بازه\u200cها را ویرایش کن.",
		"This range overlaps another logged period. Please edit one of the ranges.")
}

func msgNoOngoing(locale string) string {
	return pick(locale, "پریود فعالی برای پایان دادن وجود ندارد.", "There is no ongoing period to end.")
}

func msgEndBeforeStart(locale string) string {
	return pick(locale, "تاریخ پایان پریود نمی\u200cتواند قبل از تاریخ شروع باشد.",
		"The period end date cannot be before its start date.")
}

func msgNotFound(locale string) string {
	return pick(locale, "پریود مورد نظر پیدا نشد.", "Period not found.")
}

func pick(locale, fa, en string) string {
	if locale == "fa" {
		return fa
	}
	return en
}
